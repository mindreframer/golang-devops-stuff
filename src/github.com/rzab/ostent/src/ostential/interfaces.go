package ostential
import (
	"regexp"
	"runtime"
)

var (
	rx_lo      = regexp.MustCompile("lo\\d+") // used in interfaces_unix.go, sortable.go
	RX_fw      = regexp.MustCompile("fw\\d+")
	RX_gif     = regexp.MustCompile("gif\\d+")
	RX_stf     = regexp.MustCompile("stf\\d+")
	RX_bridge  = regexp.MustCompile("bridge\\d+")
	RX_vboxnet = regexp.MustCompile("vboxnet\\d+")
	RX_airdrop = regexp.MustCompile("p2p\\d+")
)

func realInterfaceName(name string) bool {
	bname := []byte(name)
	if  RX_bridge .Match(bname) ||
		RX_vboxnet.Match(bname) {
		return false
	}
	is_darwin := runtime.GOOS == "darwin"
	if is_darwin {
		if  RX_fw     .Match(bname) ||
			RX_gif    .Match(bname) ||
			RX_stf    .Match(bname) ||
			RX_airdrop.Match(bname) {
			return false
		}
	}
	return true
}

/*
func netinterface_ipaddr() (string, error) {
	// list of the system's network interfaces.
	list_iface, err := net.Interfaces()
	// var ifaces ost_api.Interfaces
	if err != nil {
		return "", err
	}

	var addr []string

	for _, iface := range list_iface {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if !realInterfaceName(iface.Name) {
			continue
		}
		if aa, err := iface.Addrs(); err == nil {
			if len(aa) == 0 {
				continue
			}
			for _, a := range aa {
				ipnet, ok := a.(*net.IPNet)
				if !ok {
					return "", fmt.Errorf("Not an IP: %v", a)
					continue
				}
				if ipnet.IP.IsLinkLocalUnicast() {
					continue
				}
				addr = append(addr, ipnet.IP.String())
			}
		}
	}
	if len(addr) == 0 {
		return "", nil
	}
	return addr[0], nil
} // */

func filterInterfaces(ifs []InterfaceTotal) []InterfaceTotal {
	fifs := []InterfaceTotal{}
	for _, fi := range ifs {
		if !realInterfaceName(fi.Name) {
			continue
		}
		fifs = append(fifs, fi)
	}
	return fifs
}
