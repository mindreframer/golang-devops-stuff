package listener

import (
	"flag"
	"os"
	"strconv"
	"strings"
)

const (
	defaultPort    = 80
	defaultAddress = "0.0.0.0"

	defaultReplayAddress = "localhost:28020"
)

// ListenerSettings contain all the needed configuration for setting up the listener
type ListenerSettings struct {
	Port    int
	Address string

	ReplayAddress   string
	FileToReplayPath string

	ReplayLimit int

	Verbose bool
}

var Settings ListenerSettings = ListenerSettings{}

// ReplayServer generates ReplayLimit and ReplayAddress settings out of the replayAddress
func (s *ListenerSettings) Parse() {
	host_info := strings.Split(s.ReplayAddress, "|")

	if len(host_info) > 1 {
		s.ReplayLimit, _ = strconv.Atoi(host_info[1])
	}

	s.ReplayAddress = host_info[0]
}

func init() {
	if len(os.Args) < 2 || os.Args[1] != "listen" {
		return
	}

	flag.IntVar(&Settings.Port, "p", defaultPort, "Specify the http server port whose traffic you want to capture")
	flag.StringVar(&Settings.Address, "ip", defaultAddress, "Specify IP address to listen")

	flag.StringVar(&Settings.ReplayAddress, "r", defaultReplayAddress, "Address of replay server.")

	flag.StringVar(&Settings.FileToReplayPath, "file", "", "File to store captured requests")

	flag.BoolVar(&Settings.Verbose, "verbose", false, "Log requests")
}
