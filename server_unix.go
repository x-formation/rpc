// +build !windows

package rpc

import "net"

func addrToIP(addr net.Addr) net.IP {
	if ip, _, err := net.ParseCIDR(addr.String()); err == nil {
		return ip
	}
	return nil
}
