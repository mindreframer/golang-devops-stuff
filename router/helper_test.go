package router_test

import (
	"github.com/cloudfoundry/gorouter/registry"
	"github.com/cloudfoundry/gorouter/test"

	"net"
	"time"
)

func waitMsgReceived(registry *registry.RouteRegistry, app *test.TestApp, expectedToBeFound bool, timeout time.Duration) bool {
	interval := time.Millisecond * 50
	repetitions := int(timeout / interval)

	for j := 0; j < repetitions; j++ {
		if j > 0 {
			time.Sleep(interval)
		}

		received := true
		for _, url := range app.Urls() {
			pool := registry.Lookup(url)
			if expectedToBeFound && pool == nil {
				received = false
				break
			} else if !expectedToBeFound && pool != nil {
				received = false
				break
			}
		}
		if received {
			return true
		}
	}

	return false
}

func waitAppRegistered(registry *registry.RouteRegistry, app *test.TestApp, timeout time.Duration) bool {
	return waitMsgReceived(registry, app, true, timeout)
}

func waitAppUnregistered(registry *registry.RouteRegistry, app *test.TestApp, timeout time.Duration) bool {
	return waitMsgReceived(registry, app, false, timeout)
}

func timeoutDialler() func(net, addr string) (c net.Conn, err error) {
	return func(netw, addr string) (net.Conn, error) {
		c, err := net.DialTimeout(netw, addr, 10*time.Second)
		c.SetDeadline(time.Now().Add(2 * time.Second))
		return c, err
	}
}
