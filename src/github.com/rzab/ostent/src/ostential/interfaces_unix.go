package ostential
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

#else
#include <linux/if_link.h>
u_int32_t Ibytes(void *data) { return ((struct rtnl_link_stats *)data)->rx_bytes; }
u_int32_t Obytes(void *data) { return ((struct rtnl_link_stats *)data)->tx_bytes; }
#endif

char ADDR[INET_ADDRSTRLEN];
*/
import "C"
import "unsafe"

type InterfaceTotal struct{
	Name string
	In   uint
	Out  uint
}

func NewInterfaces() ([]InterfaceTotal, string) {
	var ifaces *C.struct_ifaddrs
	if getrc, _ := C.getifaddrs(&ifaces); getrc != 0 {
		return []InterfaceTotal{}, ""
	}
	defer C.freeifaddrs(ifaces)

	ifs := []InterfaceTotal{}
	IP  := ""

	for fi := ifaces; fi != nil; fi = fi.ifa_next {
		if fi.ifa_addr == nil {
			continue
		}

		ifa_name := C.GoString(fi.ifa_name)
		if IP == "" &&
			fi.ifa_addr.sa_family == C.AF_INET   &&
			                    ifa_name != "lo" &&
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
		if  C.Ibytes(data) == 0 &&
			C.Obytes(data) == 0 {
			continue
		}
		ifs = append(ifs, InterfaceTotal{
			Name: ifa_name,
			In:   uint(C.Ibytes(data)),
			Out:  uint(C.Obytes(data)),
		})
	}
	return ifs, IP
}
