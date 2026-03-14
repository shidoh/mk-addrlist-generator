package main

import (
	"mk-addrlist-generator/cmd"
)

// Build information (set at compile time via ldflags)
var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	// Set version information
	cmd.SetVersionInfo(version, buildTime, gitCommit)

	// Execute root command
	cmd.Execute()
}
