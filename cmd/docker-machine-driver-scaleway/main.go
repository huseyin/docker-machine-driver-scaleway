package main

import (
	"github.com/docker/machine/libmachine/drivers/plugin"
	scaleway "github.com/huseyin/docker-machine-driver-scaleway"
)

// Version defines a driver version number.
var Version = "undefined"

func main() {
	plugin.RegisterDriver(scaleway.NewDriver("", ""))
}
