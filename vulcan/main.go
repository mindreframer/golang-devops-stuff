package main

import (
	"github.com/golang/glog"
	"github.com/mailgun/vulcan/service"
)

func main() {
	service, err := service.NewService()
	if err != nil {
		glog.Fatalf("Failed to init service, error:", err)
	}

	glog.Fatal(service.Start())
}
