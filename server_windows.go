// +build windows

package rpc

import "net"

func addrToIP(addr net.Addr) (ip net.IP) {
	ip = net.ParseIP(addr.String())
	return
}
