package provider

import (
	"fmt"
	"math/big"
	"net"
)

type WireGuardAssignments struct {
	NodeIPs  map[string]string
	AdminIPs map[string]string
}

func assignWireGuardIPs(cidr string, nodeNames []string, adminNames []string, requestedAdminIPs map[string]string) (WireGuardAssignments, error) {
	base, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return WireGuardAssignments{}, err
	}
	base = base.To4()
	if base == nil {
		return WireGuardAssignments{}, fmt.Errorf("wireguard.subnet must be an IPv4 CIDR")
	}
	ones, bits := network.Mask.Size()
	if bits != 32 {
		return WireGuardAssignments{}, fmt.Errorf("wireguard.subnet must be an IPv4 CIDR")
	}
	usable := (uint64(1) << uint64(32-ones)) - 2
	required := uint64(len(nodeNames) + len(adminNames))
	if usable < required {
		return WireGuardAssignments{}, fmt.Errorf("subnet %s has %d usable host addresses, need %d", cidr, usable, required)
	}
	out := WireGuardAssignments{NodeIPs: map[string]string{}, AdminIPs: map[string]string{}}
	used := map[string]bool{}
	for i, name := range nodeNames {
		ip := addIPv4(network.IP.To4(), uint64(i+1)).String()
		out.NodeIPs[name] = ip
		used[ip] = true
	}
	for _, name := range adminNames {
		if requestedAdminIPs[name] != "" {
			ip := net.ParseIP(requestedAdminIPs[name]).To4()
			if ip == nil || !network.Contains(ip) {
				return WireGuardAssignments{}, fmt.Errorf("admin peer %q wg_ip %q is not inside %s", name, requestedAdminIPs[name], cidr)
			}
			if used[ip.String()] || ip.Equal(network.IP.To4()) || ip.Equal(addIPv4(network.IP.To4(), usable+1)) {
				return WireGuardAssignments{}, fmt.Errorf("admin peer %q wg_ip %q is not usable or conflicts with another IP", name, requestedAdminIPs[name])
			}
			out.AdminIPs[name] = ip.String()
			used[ip.String()] = true
		} else {
			assigned := ""
			for offset := usable; offset > 0; offset-- {
				ip := addIPv4(network.IP.To4(), offset).String()
				if !used[ip] {
					assigned = ip
					break
				}
			}
			if assigned == "" {
				return WireGuardAssignments{}, fmt.Errorf("no free IP available for admin peer %q", name)
			}
			out.AdminIPs[name] = assigned
			used[assigned] = true
		}
	}
	return out, nil
}

func addIPv4(ip net.IP, offset uint64) net.IP {
	x := big.NewInt(0).SetBytes(ip.To4())
	x.Add(x, big.NewInt(0).SetUint64(offset))
	b := x.Bytes()
	out := make([]byte, 4)
	copy(out[4-len(b):], b)
	return net.IP(out)
}
