package network

import (
	"net"
)

type Network interface {
	HostIP() net.IP
	ContainerIP() net.IP
	String() string
}
