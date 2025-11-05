package dhcp

import (
	"net"
	"strings"
)

func cloneIP(ip net.IP) net.IP {
	if ip == nil {
		return nil
	}
	dup := make(net.IP, len(ip))
	copy(dup, ip)
	return dup
}

func incrementIP(ip net.IP) net.IP {
	if ip == nil {
		return nil
	}
	res := cloneIP(ip)
	for i := len(res) - 1; i >= 0; i-- {
		res[i]++
		if res[i] != 0 {
			break
		}
	}
	return res
}

func compareIP(a, b net.IP) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}
	aa := a.To4()
	bb := b.To4()
	if aa == nil || bb == nil {
		return strings.Compare(a.String(), b.String())
	}
	for i := range aa {
		if aa[i] < bb[i] {
			return -1
		}
		if aa[i] > bb[i] {
			return 1
		}
	}
	return 0
}
