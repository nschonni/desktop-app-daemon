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

package types

import "github.com/ivpn/desktop-app-daemon/vpn"

// Hello is an initial request
type Hello struct {
	CommandBase
	// connected client version
	Version string
	Secret  uint64

	// GetServersList == true - client requests to send back info about all servers
	GetServersList bool

	// GetStatus == true - client requests current status (Vpn connection, Firewal... etc.)
	GetStatus bool

	// GetConfigParams == true - client requests config parameters (user-defined OpevVPN file location ... etc.)
	GetConfigParams bool

	//	KeepDaemonAlone == false (default) - VPN disconnects when client disconnects from a daemon
	//	KeepDaemonAlone == true - do nothing when client disconnects from a daemon (if VPN is connected - do not disconnect)
	KeepDaemonAlone bool

	// Register credentials (if not logged in)
	// Used when updating from an old client version
	SetRawCredentials RawCredentials
}

// RawCredentials - RAW credentials
type RawCredentials struct {
	AccountID      string
	Session        string
	OvpnUser       string
	OvpnPass       string
	WgPublicKey    string
	WgPrivateKey   string
	WgLocalIP      string
	WgKeyGenerated int64 // Unix time
}

// GetServers request servers list
type GetServers struct {
	CommandBase
}

// PingServers request to ping servers
type PingServers struct {
	CommandBase
	RetryCount int
	TimeOutMs  int
}

// KillSwitchSetAllowLANMulticast enable\disable LAN multicast acces for kill-switch
type KillSwitchSetAllowLANMulticast struct {
	CommandBase
	AllowLANMulticast bool

	// When true - deamon returns empty response as confirmation
	// Needed for supporting old UI clients which are don't require confirmation
	Synchronously bool
}

// KillSwitchSetAllowLAN enable\disable LAN acces for kill-switch
type KillSwitchSetAllowLAN struct {
	CommandBase
	AllowLAN bool

	// When true - deamon returns empty response as confirmation
	// Needed for supporting old UI clients which are don't require confirmation
	Synchronously bool
}

// KillSwitchSetEnabled request to enable\disable kill-switch
type KillSwitchSetEnabled struct {
	CommandBase
	IsEnabled bool
}

// KillSwitchGetStatus get full killswitch status
type KillSwitchGetStatus struct {
	CommandBase
}

// KillSwitchSetIsPersistent request to mark kill-switch persistant
type KillSwitchSetIsPersistent struct {
	CommandBase
	IsPersistent bool
}

// SetPreference sets daemon configuration parameter
type SetPreference struct {
	CommandBase
	Key   string
	Value string
}

// SetAlternateDns request to set custom DNS
type SetAlternateDns struct {
	CommandBase
	DNS string
}

// WGHost is a WireGuard host description
type WGHost struct {
	Host      string
	PublicKey string `json:"public_key"`
	LocalIP   string `json:"local_ip"`
}

// Connect request to establish new VPN connection
type Connect struct {
	CommandBase
	VpnType                    vpn.Type
	CurrentDNS                 string
	FirewallOnDuringConnection bool

	WireGuardParameters struct {
		Port struct {
			Port int
		}

		EntryVpnServer struct {
			Hosts []WGHost
		}
	}

	OpenVpnParameters struct {
		EntryVpnServer struct {
			IPAddresses []string `json:"ip_addresses"`
		}

		MultihopExitSrvID string
		ProxyType         string
		ProxyAddress      string
		ProxyPort         int
		ProxyUsername     string
		ProxyPassword     string

		Port struct {
			Port     int
			Protocol int
		}
	}
}

// Disconnect disconnect active VPN connection
type Disconnect struct {
	CommandBase
}

// GetVPNState request daemon to provive current VPN connection state
type GetVPNState struct {
	CommandBase
}

// SessionNew - create new session
//
// When force is set to true - all active sessions will be deleted prior to creating a new one if user reached session limit.
// Initial call to /sessin/new should always be performed with force set to false, to display special form, when sessions limit is reached.
// IVPN client apps have to set force to true only when customer clicks Log all other clients button.
type SessionNew struct {
	CommandBase
	AccountID  string
	ForceLogin bool
}

// SessionDelete logout from current device
type SessionDelete struct {
	CommandBase
}

// AccountStatus get account status
type AccountStatus struct {
	CommandBase
}

// WireGuardGenerateNewKeys - generate WG keys
type WireGuardGenerateNewKeys struct {
	CommandBase
	OnlyUpdateIfNecessary bool
}

// WireGuardSetKeysRotationInterval -  change WG keys rotation interval
type WireGuardSetKeysRotationInterval struct {
	CommandBase
	Interval int64
}

// WiFiAvailableNetworks - get list of available WIFI networks
type WiFiAvailableNetworks struct {
	CommandBase
}

// WiFiCurrentNetwork - request info about connected WIFI
type WiFiCurrentNetwork struct {
	CommandBase
}

// APIRequest do custom request to API
type APIRequest struct {
	CommandBase
	APIPath         string
	HTTPMethod      string
	HTTPContentType string
	RequestData     string
}
