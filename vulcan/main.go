package main

import (
	"github.com/golang/glog"
	"github.com/mailgun/vulcan/service"
	"os"
	"runtime"
)

func main() {
	if os.Getenv("GOMAXPROCS") == "" {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}
	service, err := service.NewService()
	if err != nil {
		glog.Fatalf("Failed to init service, error:", err)
	}

	glog.Fatal(service.Start())
}
