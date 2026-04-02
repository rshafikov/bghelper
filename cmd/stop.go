package cmd

import (
	"fmt"

	"github.com/ra.shafikov/bghelper/internal/process"
	"github.com/ra.shafikov/bghelper/internal/storage"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop <id>",
	Short: "Stop a running background process",
	Long: `Stop a running background process by its ID.
The process will receive SIGTERM signal to allow graceful shutdown.

Example:
  bgh stop 1
  bgh stop 2`,
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

		// Load the process (by ID or name) - status is computed dynamically
		loadedProcess, err := resolveProcessIdentifier(processID, store)
		if err != nil {
			return fmt.Errorf("process not found: %s", processID)
		}

		// Use the actual ID for operations
		actualID := loadedProcess.ID

		// Check if process is already stopped or crashed
		if loadedProcess.Status != process.StatusRunning {
			return fmt.Errorf("process is not running: %s (status: %s)", actualID, loadedProcess.Status)
		}

		// Load process into manager memory
		if err := manager.LoadFromStorage(actualID); err != nil {
			return fmt.Errorf("failed to load process: %w", err)
		}

		// Stop the process
		if err := manager.Stop(actualID); err != nil {
			return fmt.Errorf("failed to stop process: %w", err)
		}

		// Reload to get updated process (status computed dynamically)
		updatedProcess, err := store.Load(actualID)
		if err != nil {
			return fmt.Errorf("failed to verify stop status: %w", err)
		}

		// Output confirmation
		fmt.Printf("Stopped process %s (PID: %d)\n", updatedProcess.ID, updatedProcess.PID)
		fmt.Printf("Command: %s\n", updatedProcess.Command)
		if updatedProcess.ExitCode != nil {
			fmt.Printf("Exit code: %d\n", *updatedProcess.ExitCode)
		}
		fmt.Printf("Status: %s\n", updatedProcess.Status)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)

	// Add completion for process IDs
	stopCmd.ValidArgsFunction = completeProcessIDs
}

