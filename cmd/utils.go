package cmd

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/ra.shafikov/bghelper/internal/process"
	"github.com/ra.shafikov/bghelper/internal/storage"
	"github.com/spf13/cobra"
)

// getStorageDir returns the storage directory path (~/.bghelper/processes)
func getStorageDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}

	homeDir := usr.HomeDir
	if homeDir == "" {
		// Fallback to checking HOME environment variable
		homeDir = os.Getenv("HOME")
		if homeDir == "" {
			return "", fmt.Errorf("cannot determine home directory")
		}
	}

	return filepath.Join(homeDir, ".bghelper", "processes"), nil
}

// completeProcessIDs provides shell completion for process IDs
func completeProcessIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Get storage directory
	storageDir, err := getStorageDir()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	// Initialize storage
	store := storage.NewFileStore(storageDir)

	// Load all processes
	processes, err := store.LoadAll()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	// Extract process IDs and names
	var ids []string
	for _, p := range processes {
		// Filter by prefix if user has started typing
		if toComplete == "" || strings.HasPrefix(p.ID, toComplete) || strings.HasPrefix(p.Name, toComplete) {
			// Add ID with description for better completion experience
			var description string
			if p.Name != "" {
				description = fmt.Sprintf("%s (%s - %s)", p.Command, p.Name, p.Status)
			} else {
				description = fmt.Sprintf("%s (%s)", p.Command, p.Status)
			}
			ids = append(ids, fmt.Sprintf("%s\t%s", p.ID, description))

			// Also add name if available
			if p.Name != "" {
				ids = append(ids, fmt.Sprintf("%s\t%s (%s)", p.Name, p.Command, p.Status))
			}
		}
	}

	return ids, cobra.ShellCompDirectiveNoFileComp
}

// resolveProcessIdentifier resolves a process by ID or name
// Returns the process if found, or an error if not found
func resolveProcessIdentifier(idOrName string, store *storage.FileStore) (*process.Process, error) {
	// First try to load by ID
	p, err := store.Load(idOrName)
	if err == nil {
		return p, nil
	}

	// If not found by ID, try to load by name
	p, err = store.LoadByName(idOrName)
	if err != nil {
		// Return original error (process not found by ID)
		return nil, fmt.Errorf("process not found: %s", idOrName)
	}

	return p, nil
}
