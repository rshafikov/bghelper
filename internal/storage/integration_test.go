package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ra.shafikov/bghelper/internal/process"
)

func TestFileStoreIntegration(t *testing.T) {
	// Use actual home directory for integration test
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("Could not get home directory: %v", err)
	}

	storageDir := filepath.Join(homeDir, ".bghelper", "processes")

	// Clean up test files after
	defer func() {
		testFiles := []string{"test-001.yaml", "test-002.yaml"}
		for _, file := range testFiles {
			os.Remove(filepath.Join(storageDir, file))
		}
	}()

	store := NewFileStore(storageDir)

	// Create and save processes
	proc1 := process.NewProcess("test-001", "ssh -L 8080:localhost:8080 user@server")
	proc1.Status = process.StatusRunning
	proc1.PID = 12345

	if err := store.Save(proc1); err != nil {
		t.Fatalf("Failed to save process 1: %v", err)
	}

	proc2 := process.NewProcess("test-002", "vkey start")
	proc2.Status = process.StatusStopped

	if err := store.Save(proc2); err != nil {
		t.Fatalf("Failed to save process 2: %v", err)
	}

	// Verify files were created in the correct location
	file1Path := filepath.Join(storageDir, "test-001.yaml")
	if _, err := os.Stat(file1Path); os.IsNotExist(err) {
		t.Errorf("Process file not created at expected location: %s", file1Path)
	}

	// List processes
	ids, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Verify both test processes are in the list
	found1 := false
	found2 := false
	for _, id := range ids {
		if id == "test-001" {
			found1 = true
		}
		if id == "test-002" {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Errorf("Not all test processes found in list: test-001=%v, test-002=%v", found1, found2)
	}

	// Load and verify
	loaded, err := store.Load("test-001")
	if err != nil {
		t.Fatalf("Failed to load process: %v", err)
	}

	if loaded.Command != proc1.Command {
		t.Errorf("Command mismatch: expected %s, got %s", proc1.Command, loaded.Command)
	}

	// Clean up
	if err := store.Delete("test-001"); err != nil {
		t.Fatalf("Failed to delete process: %v", err)
	}

	if err := store.Delete("test-002"); err != nil {
		t.Fatalf("Failed to delete process: %v", err)
	}
}
