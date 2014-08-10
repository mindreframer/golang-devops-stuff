package main

import (
	"github.com/abhishekkr/gol/golservice"
	"github.com/abhishekkr/goshare"
)

func main() {
	var goshare_service golservice.Funk
	goshare_service = func() { goshare.GoShare() }
	golservice.Daemon(goshare_service)
}
