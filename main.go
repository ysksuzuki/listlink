package main

import (
	"fmt"
	"net"
	"strings"

	"github.com/vishvananda/netlink"
)

var excludedIPs []net.IP

func main() {
	initExcludedIPs()
	ips, err := listLocalAddresses(netlink.FAMILY_V4)
	if err != nil {
		panic(err)
	}

	for _, ip := range ips {
		fmt.Println("local address = " + ip.String())
	}
}


func listLocalAddresses(family int) ([]net.IP, error) {
	addrs, err := netlink.AddrList(nil, family)
	if err != nil {
		return nil, err
	}

	var addresses []net.IP

	for _, addr := range addrs {
		if addr.Scope == int(netlink.SCOPE_LINK) {
			continue
		}
		if isExcluded(excludedIPs, addr.IP) {
			continue
		}
		if addr.IP.IsLoopback() {
			continue
		}

		addresses = append(addresses, addr.IP)
	}

	if hostDevice, err := netlink.LinkByName("cilium_host"); hostDevice != nil && err == nil {
		addrs, err = netlink.AddrList(hostDevice, family)
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			if addr.Scope == int(netlink.SCOPE_LINK) {
				addresses = append(addresses, addr.IP)
			}
		}
	}

	return addresses, nil
}

func isExcluded(excludeList []net.IP, ip net.IP) bool {
	for _, e := range excludeList {
		if e.Equal(ip) {
			return true
		}
	}
	return false
}

func initExcludedIPs() {
	// We exclude below bad device prefixes from address selection ...
	prefixes := []string{
		"docker",
	}
	links, err := netlink.LinkList()
	if err != nil {
		return
	}
	for _, l := range links {
		// ... also all down devices since they won't be reachable.
		if l.Attrs().OperState == netlink.OperUp {
			skip := true
			for _, p := range prefixes {
				if strings.HasPrefix(l.Attrs().Name, p) {
					skip = false
					break
				}
			}
			if skip {
				continue
			}
		}
		addr, err := netlink.AddrList(l, netlink.FAMILY_ALL)
		if err != nil {
			continue
		}
		for _, a := range addr {
			excludedIPs = append(excludedIPs, a.IP)
		}
	}
}