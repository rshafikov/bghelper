package cmd

import (
	"fmt"

	"github.com/ra.shafikov/bghelper/internal/process"
	"github.com/ra.shafikov/bghelper/internal/storage"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart <id>",
	Short: "Restart a process",
	Long: `Restart a process by its ID.

The process will be stopped if running and started again with the same ID.
This does NOT create a new process - it reuses the existing one.

The log file is cleared on restart for a fresh start.

Example:
  bgh restart 1
  bgh restart my-tunnel`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		processID := args[0]

		// Get storage directory
		storageDir, err := getStorageDir()
		if err != nil {
			return fmt.Errorf("failed to get storage directory: %w", err)
		}

		// Initialize storage and manager
		store := storage.NewFileStore(storageDir)
		manager := process.NewManager(store)

		// Verify process exists
		proc, err := resolveProcessIdentifier(processID, store)
		if err != nil {
			return fmt.Errorf("process not found: %s", processID)
		}

		// Restart the process
		p, err := manager.Restart(proc.ID)
		if err != nil {
			return fmt.Errorf("failed to restart process: %w", err)
		}

		// Output confirmation
		if p.Name != "" {
			fmt.Printf("Restarted process %s (%s) (PID: %d)\n", p.ID, p.Name, p.PID)
		} else {
			fmt.Printf("Restarted process %s (PID: %d)\n", p.ID, p.PID)
		}
		fmt.Printf("Command: %s\n", p.Command)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)

	// Add completion for process IDs
	restartCmd.ValidArgsFunction = completeProcessIDs
}
