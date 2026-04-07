package cmd

import (
	"fmt"
	"os"

	"mk-addrlist-generator/pkg/config"
	"mk-addrlist-generator/pkg/generator"

	"github.com/spf13/cobra"
)

var (
	outputFile   string
	outputFormat string
	listName     string
	aggregate    bool
	deduplicate  bool
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate address lists without starting the server",
	Long: `Generate address lists and output to stdout or a file.

This command is useful for generating address lists for:
  - One-time generation without running a server
  - Scripting and automation
  - Testing configuration files

Examples:
  # Generate all lists in MikroTik format
  mk-addrlist-generator generate -c config.yaml

  # Generate a specific list in plain format
  mk-addrlist-generator generate -c config.yaml -n mylist -f plain

  # Generate with CIDR aggregation and save to file
  mk-addrlist-generator generate -c config.yaml -a -o output.txt

  # Generate in JSON format
  mk-addrlist-generator generate -c config.yaml -f json`,
	RunE: runGenerate,
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: stdout)")
	generateCmd.Flags().StringVarP(&outputFormat, "format", "f", "mikrotik", "Output format (mikrotik, plain, json, nftables)")
	generateCmd.Flags().StringVarP(&listName, "name", "n", "", "Generate only this list (default: all lists)")
	generateCmd.Flags().BoolVarP(&aggregate, "aggregate", "a", false, "Enable CIDR aggregation")
	generateCmd.Flags().BoolVarP(&deduplicate, "deduplicate", "d", true, "Enable deduplication")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	if err := config.ValidateConfig(cfg); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Create generator
	gen := generator.NewGenerator(cfg)

	// Parse format and per-request options
	format := generator.ParseFormat(outputFormat)
	reqOpts := generator.RequestOptions{
		Deduplicate: deduplicate,
		Aggregate:   aggregate,
	}

	var output string

	if listName != "" {
		// Generate specific list
		list, exists := cfg.Lists[listName]
		if !exists {
			return fmt.Errorf("list %q not found", listName)
		}
		output, err = gen.GenerateListWithFormat(listName, list, format, reqOpts)
	} else {
		// Generate all lists
		output, err = gen.GenerateAllWithFormat(format, reqOpts)
	}

	if err != nil {
		return fmt.Errorf("failed to generate: %w", err)
	}

	// Output
	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(output), 0600); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		if verbose {
			fmt.Printf("Output written to %s\n", outputFile)
		}
	} else {
		fmt.Print(output)
	}

	return nil
}
