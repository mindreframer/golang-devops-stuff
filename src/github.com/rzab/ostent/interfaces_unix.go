package ostent
/*
#include <sys/socket.h>
#include <net/if.h>
#include <arpa/inet.h>
#include <ifaddrs.h>

#ifndef AF_LINK
#define AF_LINK AF_PACKET
#endif

#ifndef __linux__ // NOT LINUX
u_int32_t Ibytes(void *data) { return ((struct if_data *)data)->ifi_ibytes; }
u_int32_t Obytes(void *data) { return ((struct if_data *)data)->ifi_obytes; }

u_int32_t Ipackets(void *data) { return ((struct if_data *)data)->ifi_ipackets; }
u_int32_t Opackets(void *data) { return ((struct if_data *)data)->ifi_opackets; }

u_int32_t Ierrors(void *data) { return ((struct if_data *)data)->ifi_ierrors; }
u_int32_t Oerrors(void *data) { return ((struct if_data *)data)->ifi_oerrors; }

#else
#include <linux/if_link.h>
u_int32_t Ibytes(void *data) { return ((struct rtnl_link_stats *)data)->rx_bytes; }
u_int32_t Obytes(void *data) { return ((struct rtnl_link_stats *)data)->tx_bytes; }

u_int32_t Ipackets(void *data) { return ((struct rtnl_link_stats *)data)->rx_packets; }
u_int32_t Opackets(void *data) { return ((struct rtnl_link_stats *)data)->tx_packets; }

u_int32_t Ierrors(void *data) { return ((struct rtnl_link_stats *)data)->rx_errors; }
u_int32_t Oerrors(void *data) { return ((struct rtnl_link_stats *)data)->tx_errors; }
#endif

char ADDR[INET_ADDRSTRLEN];
*/
import "C"
import "unsafe"

type InterfaceInfo struct{
	Name string
	 InBytes   uint
	OutBytes   uint
	 InPackets uint
	OutPackets uint
	 InErrors  uint
	OutErrors  uint
}

type InterfacesInfo struct {
	List []InterfaceInfo
	IP string
}

func NewInterfaces(CH chan InterfacesInfo) {
	var ifaces *C.struct_ifaddrs
	if getrc, _ := C.getifaddrs(&ifaces); getrc != 0 {
		CH <- InterfacesInfo{}
		return
	}
	defer C.freeifaddrs(ifaces)

	ifs := []InterfaceInfo{}
	IP  := ""

	for fi := ifaces; fi != nil; fi = fi.ifa_next {
		if fi.ifa_addr == nil {
			continue
		}

		ifa_name := C.GoString(fi.ifa_name)
		if IP == "" &&
			fi.ifa_addr.sa_family == C.AF_INET   &&
			!rx_lo.Match([]byte(ifa_name))       &&
			realInterfaceName(  ifa_name  ) {

			sa_in := (*C.struct_sockaddr_in)(unsafe.Pointer(fi.ifa_addr))
			if C.inet_ntop(
				C.int(fi.ifa_addr.sa_family), // C.AF_INET,
				unsafe.Pointer(&sa_in.sin_addr),
				&C.ADDR[0],
				C.socklen_t(unsafe.Sizeof(C.ADDR))) != nil {

					IP = C.GoString((*C.char)(unsafe.Pointer(&C.ADDR)))
				}
		}

		if fi.ifa_addr.sa_family != C.AF_LINK {
			continue
		}

		data := fi.ifa_data
		it := InterfaceInfo{
			Name: ifa_name,
			 InBytes:   uint(C.Ibytes(data)),
			OutBytes:   uint(C.Obytes(data)),
			 InPackets: uint(C.Ipackets(data)),
			OutPackets: uint(C.Opackets(data)),
			 InErrors:  uint(C.Ierrors(data)),
			OutErrors:  uint(C.Oerrors(data)),
		}
		if  it. InBytes   == 0 &&
			it.OutBytes   == 0 &&
			it. InPackets == 0 &&
			it.OutPackets == 0 &&
			it. InErrors  == 0 &&
			it.OutErrors  == 0 {
			continue
		}
		ifs = append(ifs, it)
	}
	CH <- InterfacesInfo{
		List: ifs,
		IP: IP,
	}
}
