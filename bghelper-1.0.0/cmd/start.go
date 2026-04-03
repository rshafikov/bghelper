package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ra.shafikov/bghelper/internal/process"
	"github.com/ra.shafikov/bghelper/internal/storage"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start <command>",
	Short: "Start a new background process",
	Long: `Start a new background process with the given command.
The process will run in the background and its state will be persisted.

Processes are assigned sequential IDs (1, 2, 3, etc.) for easy reference.
You can optionally give a process a friendly name using --name flag.

Example:
  bgh start "ssh -L 8080:localhost:8080 user@server"
  bgh start --name "dev-server" "python3 -m http.server 8000"
  bgh start -n "tunnel" "ssh -L 5432:localhost:5432 user@db"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get the full command (all arguments joined)
		fullCommand := ""
		for i, arg := range args {
			if i > 0 {
				fullCommand += " "
			}
			fullCommand += arg
		}

		// Get optional name flag
		name, _ := cmd.Flags().GetString("name")

		// Get storage directory
		storageDir, err := getStorageDir()
		if err != nil {
			return fmt.Errorf("failed to get storage directory: %w", err)
		}

		// Initialize storage and manager
		store := storage.NewFileStore(storageDir)
		manager := process.NewManager(store)

		// Generate unique process ID
		processID, err := generateUniqueID(manager)
		if err != nil {
			return fmt.Errorf("failed to generate process ID: %w", err)
		}

		// Start the process
		p, err := manager.Start(processID, fullCommand)
		if err != nil {
			return fmt.Errorf("failed to start process: %w", err)
		}

		// Set name if provided
		if name != "" {
			p.Name = name
			// Persist the updated process with name
			if err := store.Save(p); err != nil {
				// Log warning but don't fail
				fmt.Fprintf(os.Stderr, "Warning: failed to save process name: %v\n", err)
			}
		}

		// Output success message
		if name != "" {
			fmt.Printf("Started process %s (%s) (PID: %d)\n", p.ID, name, p.PID)
		} else {
			fmt.Printf("Started process %s (PID: %d)\n", p.ID, p.PID)
		}
		fmt.Printf("Command: %s\n", p.Command)
		fmt.Printf("Status: %s\n", p.Status)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().StringP("name", "n", "", "Optional friendly name for the process")
}

// generateUniqueID generates a unique process ID using sequential integers
func generateUniqueID(manager *process.Manager) (string, error) {
	// Get all existing process IDs
	ids, err := manager.List()
	if err != nil {
		// If we can't list, start with 1
		return "1", nil
	}

	// Find the maximum numeric ID
	maxID := 0
	for _, id := range ids {
		// Try to parse as integer
		num, err := strconv.Atoi(id)
		if err == nil && num > maxID {
			maxID = num
		}
	}

	// Return next sequential ID
	return strconv.Itoa(maxID + 1), nil
}
