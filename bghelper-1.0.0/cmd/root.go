package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bgh",
	Short: "bghelper - Background process manager",
	Long: `bghelper is a CLI tool for managing background processes.
It helps you start, stop, and track long-running commands like SSH tunnels,
dev servers, and other background tasks.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Add subcommands here later
}
