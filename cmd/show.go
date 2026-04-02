package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ra.shafikov/bghelper/internal/process"
	"github.com/ra.shafikov/bghelper/internal/storage"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show details of a process",
	Long: `Display detailed information about a specific process by its ID.

The output shows all process details in a table format including:
- ID: Unique process identifier
- Command: The shell command being executed
- Status: Current process status (running, stopped, crashed)
- PID: Process ID (0 if not running)
- CreatedAt: When the process was created
- StartedAt: When the process was last started
- ExitCode: Exit code (if process has stopped/crashed)

Example:
  bgh show 1
  bgh show 2 --format json`,
	Args:          cobra.ExactArgs(1),
	SilenceErrors: true,
	SilenceUsage:  true,
	Run: func(cmd *cobra.Command, args []string) {
		processID := args[0]

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

		// Load the process (by ID or name) - status is computed dynamically
		p, err := resolveProcessIdentifier(processID, store)
		if err != nil {
			// Process not found - exit with code 3
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(3)
		}

		// Output based on format
		switch format {
		case "json":
			outputJSONProcess(p)
		case "yaml":
			outputYAMLProcess(p)
		case "table", "":
			// Display process details in a table
			displayProcessDetails(p)
		default:
			fmt.Fprintf(os.Stderr, "Error: invalid format '%s'. Use: table, json, or yaml\n", format)
			os.Exit(2)
		}
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.Flags().StringP("format", "f", "table", "Output format (table, json, yaml)")

	// Add completion for process IDs
	showCmd.ValidArgsFunction = completeProcessIDs
}

// outputJSONProcess outputs a process as JSON
func outputJSONProcess(p *process.Process) {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to marshal JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

// outputYAMLProcess outputs a process as YAML
func outputYAMLProcess(p *process.Process) {
	data, err := yaml.Marshal(p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to marshal YAML: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(string(data))
}

// displayProcessDetails shows process information in a formatted table
func displayProcessDetails(p *process.Process) {
	fmt.Println("+------------+----------------------------------------------------------+")
	fmt.Printf("| %-10s | %-56s |\n", "ID", p.ID)
	if p.Name != "" {
		fmt.Printf("| %-10s | %-56s |\n", "Name", p.Name)
	}
	fmt.Printf("| %-10s | %-56s |\n", "Command", truncateString(p.Command, 56))
	fmt.Printf("| %-10s | %-56s |\n", "Status", string(p.Status))
	fmt.Printf("| %-10s | %-56d |\n", "PID", p.PID)
	fmt.Printf("| %-10s | %-56s |\n", "CreatedAt", formatTime(p.CreatedAt))
	fmt.Printf("| %-10s | %-56s |\n", "StartedAt", formatTime(p.StartedAt))

	// Add exit code if available
	if p.ExitCode != nil {
		fmt.Printf("| %-10s | %-56d |\n", "ExitCode", *p.ExitCode)
	} else {
		fmt.Printf("| %-10s | %-56s |\n", "ExitCode", "N/A")
	}

	// Add logs path if available
	if p.LogsPath != "" {
		fmt.Printf("| %-10s | %-56s |\n", "LogsPath", truncateString(p.LogsPath, 56))
	}
	fmt.Println("+------------+----------------------------------------------------------+")
}

// formatTime formats a time value for display
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "N/A"
	}
	return t.Format("2006-01-02 15:04:05 MST")
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
