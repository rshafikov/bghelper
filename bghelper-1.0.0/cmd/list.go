package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/ra.shafikov/bghelper/internal/output"
	"github.com/ra.shafikov/bghelper/internal/process"
	"github.com/ra.shafikov/bghelper/internal/storage"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all processes",
	Long: `List all processes (running and stopped) in a formatted table.

Processes are sorted by creation date (newest first). Running processes
are shown in green, stopped in yellow, and crashed in red.

Example:
  bgh list
  bgh list --format json
  bgh list --format yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get format flag
		format, _ := cmd.Flags().GetString("format")

		// Get storage directory
		storageDir, err := getStorageDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to get storage directory: %v\n", err)
			os.Exit(1)
		}

		// Initialize storage
		store := storage.NewFileStore(storageDir)

		// Load all processes - status is computed dynamically for each
		processes, err := store.LoadAll()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to list processes: %v\n", err)
			os.Exit(1)
		}

		// Sort by created_at descending (newest first)
		sort.Slice(processes, func(i, j int) bool {
			return processes[i].CreatedAt.After(processes[j].CreatedAt)
		})

		// Output based on format
		switch format {
		case "json":
			outputJSONList(processes)
		case "yaml":
			outputYAMLList(processes)
		case "table", "":
			formatter := output.NewTableFormatter(os.Stdout)
			formatter.FormatProcessList(processes)
		default:
			fmt.Fprintf(os.Stderr, "Error: invalid format '%s'. Use: table, json, or yaml\n", format)
			os.Exit(2)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("format", "f", "table", "Output format (table, json, yaml)")
}

// outputJSONList outputs processes as JSON
func outputJSONList(processes []*process.Process) {
	data, err := json.MarshalIndent(processes, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to marshal JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

// outputYAMLList outputs processes as YAML
func outputYAMLList(processes []*process.Process) {
	data, err := yaml.Marshal(processes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to marshal YAML: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(string(data))
}
