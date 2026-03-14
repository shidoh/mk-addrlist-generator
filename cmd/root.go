package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version information (set at compile time)
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"

	// Global flags
	configPath string
	verbose    bool
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "mk-addrlist-generator",
	Short: "MikroTik Address List Generator",
	Long: `MikroTik Address List Generator - A service that generates MikroTik 
address lists from various sources (URLs, files, static addresses) 
and provides them via HTTP API.

Supports multiple output formats:
  - mikrotik: MikroTik RouterOS script format (default)
  - plain: Plain text format (one IP/network per line)
  - json: JSON format
  - nftables: nftables set format`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "config.yaml", "Path to configuration file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}

// SetVersionInfo sets the version information for the CLI
func SetVersionInfo(version, buildTime, gitCommit string) {
	Version = version
	BuildTime = buildTime
	GitCommit = gitCommit
}
