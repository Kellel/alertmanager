// Copyright 2018 Prometheus Team
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cluster

import (
	"net"

	"github.com/hashicorp/go-sockaddr"
	"github.com/pkg/errors"
)

// calculateAdvertiseAddress attempts to clone logic from deep within memberlist
// (NetTransport.FinalAdvertiseAddr) in order to surface its conclusions to the
// application, so we can provide more actionable error messages if the user has
// inadvertantly misconfigured their cluster.
//
// https://github.com/hashicorp/memberlist/blob/022f081/net_transport.go#L126
func calculateAdvertiseAddress(bindAddr, advertiseAddr string) (net.IP, error) {
	if advertiseAddr != "" {
		ip := net.ParseIP(advertiseAddr)
		if ip == nil {
			return nil, errors.Errorf("failed to parse advertise addr '%s'", advertiseAddr)
		}
		if ip4 := ip.To4(); ip4 != nil {
			ip = ip4
		}
		return ip, nil
	}

	if bindAddr == "0.0.0.0" {
		privateIP, err := sockaddr.GetPrivateIP()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get private IP")
		}
		if privateIP == "" {
			return nil, errors.Wrap(err, "no private IP found, explicit advertise addr not provided")
		}
		ip := net.ParseIP(privateIP)
		if ip == nil {
			return nil, errors.Errorf("failed to parse private IP '%s'", privateIP)
		}
		return ip, nil
	}

	ip := net.ParseIP(bindAddr)
	if ip == nil {
		return nil, errors.Errorf("failed to parse bind addr '%s'", bindAddr)
	}
	return ip, nil
}
