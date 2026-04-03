package cmd

import (
	"fmt"

	"github.com/ra.shafikov/bghelper/internal/process"
	"github.com/ra.shafikov/bghelper/internal/storage"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a process from storage",
	Long: `Delete a stopped process from storage by its ID.
Running processes cannot be deleted unless --force is used.

Example:
  bgh delete 1
  bgh delete 2
  bgh delete --force 3`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		processID := args[0]

		// Get force flag
		force, _ := cmd.Flags().GetBool("force")

		// Get storage directory
		storageDir, err := getStorageDir()
		if err != nil {
			return fmt.Errorf("failed to get storage directory: %w", err)
		}

		// Initialize storage
		store := storage.NewFileStore(storageDir)

		// Load the process (by ID or name) - status is computed dynamically
		proc, err := resolveProcessIdentifier(processID, store)
		if err != nil {
			return fmt.Errorf("process not found: %s", processID)
		}

		// Use the actual ID for operations
		actualID := proc.ID

		// Check if process is running (status is already computed dynamically)
		if proc.Status == process.StatusRunning {
			if !force {
				return fmt.Errorf("cannot delete running process (id=%s), stop it first or use --force", actualID)
			}

			// Force delete: stop the process first
			fmt.Printf("Stopping running process %s (PID: %d)...\n", proc.ID, proc.PID)

			// Create manager and load process
			manager := process.NewManager(store)
			if err := manager.LoadFromStorage(actualID); err != nil {
				return fmt.Errorf("failed to load process for stopping: %w", err)
			}

			// Stop the process
			if err := manager.Stop(actualID); err != nil {
				return fmt.Errorf("failed to stop process: %w", err)
			}
		}

		// Delete the process
		if err := store.Delete(actualID); err != nil {
			return fmt.Errorf("failed to delete process: %w", err)
		}

		// Output confirmation
		fmt.Printf("Deleted process %s\n", actualID)
		fmt.Printf("Command: %s\n", proc.Command)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	// Add flags
	deleteCmd.Flags().BoolP("force", "f", false, "Force delete running processes (stops them first)")

	// Add completion for process IDs
	deleteCmd.ValidArgsFunction = completeProcessIDs
}
