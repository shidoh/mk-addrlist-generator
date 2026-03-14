package cmd

import (
	"fmt"

	"mk-addrlist-generator/pkg/config"

	"github.com/spf13/cobra"
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the configuration file",
	Long: `Validate the configuration file without starting the server.

This command checks:
  - YAML syntax
  - Required fields
  - Timeout format
  - List source definitions

Examples:
  # Validate default config file
  mk-addrlist-generator validate

  # Validate specific config file
  mk-addrlist-generator validate -c /path/to/config.yaml`,
	RunE: runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	fmt.Printf("Validating configuration file: %s\n", configPath)

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Print summary
	fmt.Printf("\n✓ Configuration is valid\n\n")
	fmt.Printf("Summary:\n")
	fmt.Printf("  Default timeout: %s\n", cfg.Config.Timeout)
	fmt.Printf("  Default comment prefix: %s\n", cfg.Config.CommentPrefix)
	fmt.Printf("  Number of lists: %d\n\n", len(cfg.Lists))

	fmt.Printf("Lists:\n")
	for name, list := range cfg.Lists {
		sourceCount := len(list.URLs) + len(list.Files) + len(list.Addresses)
		timeout := list.Timeout
		if timeout == "" {
			timeout = cfg.Config.Timeout + " (default)"
		}

		fmt.Printf("  - %s:\n", name)
		fmt.Printf("      Timeout: %s\n", timeout)
		fmt.Printf("      URLs: %d\n", len(list.URLs))
		fmt.Printf("      Files: %d\n", len(list.Files))
		fmt.Printf("      Static addresses: %d\n", len(list.Addresses))
		fmt.Printf("      Total sources: %d\n", sourceCount)
	}

	return nil
}
