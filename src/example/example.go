package main

import (
	errplane "../.."
	"math/rand"
	"os"
	"time"
)

const (
	appKey      = ""
	apiKey      = ""
	environment = ""
	proxy       = ""
)

func main() {
	rand.Seed(time.Now().UnixNano())

	ep := errplane.New(appKey, environment, apiKey)
	if proxy != "" {
		ep.SetProxy(proxy)
	}

	ep.SetHttpHost("w.apiv3.errplane.com")       // optional (this is the default value)
	ep.SetUdpAddr("udp.apiv3.errplane.com:8126") // optional (this is the default value)

	err := ep.Report("some_metric", 123.4, time.Now(), "some_context", errplane.Dimensions{
		"foo": "bar",
	})
	if err != nil {
		panic(err)
	}

	for i := 0; i < 10; i++ {
		value := rand.Float64() * 100
		err = ep.Aggregate("some_aggregate", value, "", nil)
		if err != nil {
			panic(err)
		}
	}

	for i := 0; i < 10; i++ {
		value := rand.Float64() * 100
		err = ep.Sum("another_sum", value, "", nil)
		if err != nil {
			panic(err)
		}
	}

	ep.Close()

	time.Sleep(10 * time.Millisecond)

	os.Exit(0)
}
