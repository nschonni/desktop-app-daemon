//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app-daemon
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2020 Privatus Limited.
//
//  This file is part of the Daemon for IVPN Client Desktop.
//
//  The Daemon for IVPN Client Desktop is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The Daemon for IVPN Client Desktop is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the Daemon for IVPN Client Desktop. If not, see <https://www.gnu.org/licenses/>.
//

package wireguard

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ivpn/desktop-app-daemon/service/dns"
	"github.com/ivpn/desktop-app-daemon/shell"
	"github.com/ivpn/desktop-app-daemon/vpn"
)

type operation int

const (
	disconnect operation = iota
	resume     operation = iota
)

// internalVariables of wireguard implementation for Linux
type internalVariables struct {
	manualDNS            net.IP
	isRunning            bool
	isPaused             bool
	resumeDisconnectChan chan operation // control connection pause\resume or disconnect from paused state
}

func (wg *WireGuard) init() error {
	// It can happen that ivpn-daemon was not correctly stopped during WireGuard connection
	// (e.g. process was terminated)
	// In such situation, the 'wgivpn' keeps active.
	// We should close it in this case. Otherwise, new connection would not be established
	wgInterfaceName := filepath.Base(wg.configFilePath)
	wgInterfaceName = strings.TrimSuffix(wgInterfaceName, path.Ext(wgInterfaceName))
	// stop current WG connection (if exists)
	i, _ := net.InterfaceByName(wgInterfaceName)
	if i != nil {
		log.Info(fmt.Sprintf("Stopping WireGuard interface ('%s' expected to be stopped before the new connection)...", wgInterfaceName))
		err := shell.Exec(log, "ip", "link", "set", "down", wgInterfaceName) // command: sudo ip link set down wgivpn
		if err != nil {
			log.Warning(err)
		}
		err = shell.Exec(log, "ip", "link", "delete", wgInterfaceName) // command: sudo ip link delete wgivpn
		if err != nil {
			log.Warning(err)
		}
	}

	return nil
}

// connect - SYNCHRONOUSLY execute openvpn process (wait until it finished)
func (wg *WireGuard) connect(stateChan chan<- vpn.StateInfo) error {

	wg.internals.isRunning = true
	defer func() {
		wg.internals.isRunning = false
		// do not forget to remove config file after finishing configuration
		if err := os.Remove(wg.configFilePath); err != nil {
			log.Warning(fmt.Sprintf("failed to remove WG configuration: %s", err))
		}
	}()

	wg.internals.resumeDisconnectChan = make(chan operation, 1)

	// loop connection initialisation (required for pause\resume functionality)
	// on 'pause' - we stopping WG interface but not exiting this (connect) method
	// (method 'connect' is synchronous, must NOT exit on pause)
	for true {
		// generate configuration
		err := wg.generateAndSaveConfigFile(wg.configFilePath)
		if err != nil {
			return fmt.Errorf("failed to save WG config file: %w", err)
		}

		// start WG
		log.Info("Shell exec: ", wg.binaryPath, " up ", wg.configFilePath)
		cmd := exec.Command(wg.binaryPath, "up", wg.configFilePath)
		outBytes, err := cmd.CombinedOutput()
		if err != nil {
			if len(outBytes) > 0 {
				log.Error(fmt.Sprintf("'%s' error. Output: %s", wg.binaryPath, string(outBytes)))
			}
			return fmt.Errorf("failed to start WireGuard: %w", err)
		}

		err = func() error {
			// do not forget to restore DNS
			defer func() {
				// restore DNS configuration
				if err := dns.DeleteManual(nil); err != nil {
					log.Warning(fmt.Sprintf("failed to restore DNS configuration: %s", err))
				}
			}()
			// update DNS configuration
			dnsIP := wg.connectParams.hostLocalIP
			if wg.internals.manualDNS != nil {
				dnsIP = wg.internals.manualDNS
			}
			if err := dns.SetManual(dnsIP, nil); err != nil {
				return fmt.Errorf("failed to set DNS: %w", err)
			}

			// notify connected
			stateChan <- vpn.NewStateInfoConnected(wg.connectParams.clientLocalIP, wg.connectParams.hostIP)

			wgInterfaceName := filepath.Base(wg.configFilePath)
			wgInterfaceName = strings.TrimSuffix(wgInterfaceName, path.Ext(wgInterfaceName))
			// wait until wireguard interface is available
			for {
				time.Sleep(time.Millisecond * 500)
				i, err := net.InterfaceByName(wgInterfaceName)
				if err != nil {
					fmt.Println(err)
					break
				}
				if i == nil {
					break
				}
			}
			return nil
		}()

		if err != nil {
			return err
		}

		// if connection not PAUSED - exit
		if wg.isPaused() {
			log.Info("Paused")
			// wait for resume or disconnect request
			op := <-wg.internals.resumeDisconnectChan
			if op != resume {
				break
			}
			log.Info("Resuming...")
		} else {
			break
		}
	}
	return nil
}

func (wg *WireGuard) disconnect() error {

	select {
	case wg.internals.resumeDisconnectChan <- disconnect:
	default:
	}

	if wg.isPaused() {
		// wg interface already 'down'
		return wg.resume()
	}
	return wg.internalDisconnect()
}

func (wg *WireGuard) internalDisconnect() error {
	err := shell.Exec(log, wg.binaryPath, "down", wg.configFilePath)
	if err != nil {
		return fmt.Errorf("failed to stop WireGuard: %w", err)
	}
	return nil
}

func (wg *WireGuard) isPaused() bool {
	return wg.internals.isPaused
}

func (wg *WireGuard) pause() error {
	if wg.internals.isRunning == false {
		return nil
	}
	wg.internals.isPaused = true
	return wg.internalDisconnect()
}

func (wg *WireGuard) resume() error {
	if wg.internals.isPaused == false || wg.internals.isRunning == false {
		return nil
	}
	wg.internals.isPaused = false
	select {
	case wg.internals.resumeDisconnectChan <- resume:
	default:
	}
	return nil
}

func (wg *WireGuard) setManualDNS(addr net.IP) error {
	// set DNS called outside
	wg.internals.manualDNS = addr

	if wg.isPaused() || wg.internals.isRunning == false {
		return nil
	}
	return dns.SetManual(addr, nil)
}

func (wg *WireGuard) resetManualDNS() error {
	// reset DNS called outside
	wg.internals.manualDNS = nil
	if wg.isPaused() {
		return nil
	}

	if wg.internals.isRunning {
		// changing DNS to default value for current WireGuard connection
		return dns.SetManual(wg.connectParams.hostLocalIP, nil)
	}
	return dns.DeleteManual(nil)
}

func (wg *WireGuard) getOSSpecificConfigParams() (interfaceCfg []string, peerCfg []string) {
	interfaceCfg = append(interfaceCfg, "Address = "+wg.connectParams.clientLocalIP.String()+"/32")
	interfaceCfg = append(interfaceCfg, "SaveConfig = true")

	peerCfg = append(peerCfg, "AllowedIPs = 0.0.0.0/0")
	return interfaceCfg, peerCfg
}
