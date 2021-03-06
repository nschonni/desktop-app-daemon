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

package service

import (
	"time"

	"github.com/ivpn/desktop-app-daemon/wifiNotifier"
)

type wifiInfo struct {
	ssid     string
	security wifiNotifier.WiFiSecurity
}

func (inf *wifiInfo) IsInsecure() bool {
	return inf.security == wifiNotifier.WiFiSecurityNone || inf.security == wifiNotifier.WiFiSecurityWEP
}

var lastWiFiInfo *wifiInfo
var timerDelayedNotify *time.Timer

const delayBeforeWiFiChangeNotify = time.Second * 1

func (s *Service) initWiFiFunctionality() error {
	defer func() {
		if r := recover(); r != nil {
			log.Error("initWiFiFunctionality PANIC (recovered): ", r)
		}
	}()

	wifiNotifier.SetWifiNotifier(s.onWiFiChanged)
	return nil
}

func (s *Service) onWiFiChanged(ssid string) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("onWiFiChanged PANIC (recovered): ", r)
		}
	}()

	security := wifiNotifier.GetCurrentNetworkSecurity()

	lastWiFiInfo = &wifiInfo{
		ssid,
		security}

	// do delay before processing wifi change
	// (same wifi change event can occur several times in short period of time)
	if timerDelayedNotify != nil {
		timerDelayedNotify.Stop()
		timerDelayedNotify = nil
	}
	timerDelayedNotify = time.AfterFunc(delayBeforeWiFiChangeNotify, func() {
		if lastWiFiInfo == nil || lastWiFiInfo.ssid != ssid || lastWiFiInfo.security != security {
			return // do nothing (new wifi info available)
		}

		// notify clients about WiFi change
		s._evtReceiver.OnWiFiChanged(ssid, isInsecureWiFi(security))
	})
}

func isInsecureWiFi(security wifiNotifier.WiFiSecurity) bool {
	return security == wifiNotifier.WiFiSecurityNone || security == wifiNotifier.WiFiSecurityWEP
}

// GetWiFiCurrentState returns info about currently connected wifi
func (s *Service) GetWiFiCurrentState() (ssid string, isInsecureNetwork bool) {
	return wifiNotifier.GetCurrentSSID(), isInsecureWiFi(wifiNotifier.GetCurrentNetworkSecurity())
}

// GetWiFiAvailableNetworks returns list of available WIFI networks
func (s *Service) GetWiFiAvailableNetworks() []string {
	return wifiNotifier.GetAvailableSSIDs()
}
