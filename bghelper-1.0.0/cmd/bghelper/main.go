package main

import (
	"github.com/ra.shafikov/bghelper/cmd"
)

// Version is set at build time via ldflags
var Version = "dev"

// BuildTime is set at build time via ldflags
var BuildTime = "unknown"

func main() {
	cmd.SetVersion(Version, BuildTime)
	cmd.Execute()
}
