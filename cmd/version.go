package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// These variables are set from main via SetVersion
var (
	version   = "dev"
	buildTime = "unknown"
)

// SetVersion sets the version and build time from main package
func SetVersion(v, bt string) {
	version = v
	buildTime = bt
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of bghelper",
	Long:  `All software has versions. This is bghelper's`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("bghelper version %s (built %s)\n", version, buildTime)
		return nil
	},
}

// GetVersion returns the current version
func GetVersion() string {
	return version
}

// GetBuildTime returns the build time
func GetBuildTime() string {
	return buildTime
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
