package storage

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ra.shafikov/bghelper/internal/process"
)

func TestFileStoreSaveAndLoad(t *testing.T) {
	// Create temp directory for testing
	tmpDir, err := ioutil.TempDir("", "bghelper-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewFileStore(tmpDir)

	// Create a test process with PID 0 (stopped)
	p := process.NewProcess("test-001", "ls -la")
	p.PID = 0 // PID 0 means not running

	// Save the process
	if err := store.Save(p); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	filePath := filepath.Join(tmpDir, "test-001.yaml")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("Process file was not created at %s", filePath)
	}

	// Load the process
	loaded, err := store.Load("test-001")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify all fields match
	if loaded.ID != p.ID {
		t.Errorf("ID mismatch: expected %s, got %s", p.ID, loaded.ID)
	}

	if loaded.Command != p.Command {
		t.Errorf("Command mismatch: expected %s, got %s", p.Command, loaded.Command)
	}

	// Status is computed dynamically based on PID
	// PID 0 means stopped
	if loaded.Status != process.StatusStopped {
		t.Errorf("Status mismatch: expected %s (computed from PID 0), got %s", process.StatusStopped, loaded.Status)
	}

	if loaded.PID != p.PID {
		t.Errorf("PID mismatch: expected %d, got %d", p.PID, loaded.PID)
	}
}

func TestFileStoreList(t *testing.T) {
	// Create temp directory for testing
	tmpDir, err := ioutil.TempDir("", "bghelper-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewFileStore(tmpDir)

	// Create multiple processes
	processes := []struct {
		id      string
		command string
	}{
		{"proc-001", "command1"},
		{"proc-002", "command2"},
		{"proc-003", "command3"},
	}

	for _, procData := range processes {
		p := process.NewProcess(procData.id, procData.command)
		if err := store.Save(p); err != nil {
			t.Fatalf("Failed to save process %s: %v", procData.id, err)
		}
	}

	// List all processes
	ids, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Verify we have 3 processes
	if len(ids) != 3 {
		t.Errorf("Expected 3 processes, got %d", len(ids))
	}

	// Verify all IDs are present
	expectedIDs := map[string]bool{
		"proc-001": true,
		"proc-002": true,
		"proc-003": true,
	}

	for _, id := range ids {
		if !expectedIDs[id] {
			t.Errorf("Unexpected process ID: %s", id)
		}
		delete(expectedIDs, id)
	}

	// Ensure all expected IDs were found
	if len(expectedIDs) > 0 {
		t.Errorf("Not all expected IDs were found. Missing: %v", expectedIDs)
	}
}

func TestFileStoreDelete(t *testing.T) {
	// Create temp directory for testing
	tmpDir, err := ioutil.TempDir("", "bghelper-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewFileStore(tmpDir)

	// Create and save a process
	p := process.NewProcess("to-delete", "echo test")
	if err := store.Save(p); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	filePath := filepath.Join(tmpDir, "to-delete.yaml")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("Process file should exist before delete")
	}

	// Delete the process
	if err := store.Delete("to-delete"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify file no longer exists
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatal("Process file should not exist after delete")
	}

	// Verify process cannot be loaded
	_, err = store.Load("to-delete")
	if err == nil {
		t.Fatal("Expected error when loading deleted process")
	}
}

func TestFileStoreLoadNotFound(t *testing.T) {
	// Create temp directory for testing
	tmpDir, err := ioutil.TempDir("", "bghelper-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewFileStore(tmpDir)

	// Try to load non-existent process
	_, err = store.Load("non-existent")
	if err == nil {
		t.Fatal("Expected error when loading non-existent process")
	}

	expectedErr := "process not found: non-existent"
	if err.Error() != expectedErr {
		t.Errorf("Expected error message '%s', got '%s'", expectedErr, err.Error())
	}
}

func TestFileStoreEnsureStorageDir(t *testing.T) {
	// Create temp directory for testing
	tmpDir, err := ioutil.TempDir("", "bghelper-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Use a subdirectory that doesn't exist yet
	storageDir := filepath.Join(tmpDir, "bghelper", "processes")
	store := NewFileStore(storageDir)

	// Ensure storage directory should create the path
	if err := store.EnsureStorageDir(); err != nil {
		t.Fatalf("EnsureStorageDir failed: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(storageDir); os.IsNotExist(err) {
		t.Fatal("Storage directory was not created")
	}

	// Verify correct permissions
	info, err := os.Stat(storageDir)
	if err != nil {
		t.Fatalf("Failed to stat storage directory: %v", err)
	}

	// Check permissions are 0700 (owner only)
	mode := info.Mode().Perm()
	if mode != 0700 {
		t.Errorf("Expected permissions 0700, got %#o", mode)
	}
}

func TestFileStoreAtomicWrites(t *testing.T) {
	// Create temp directory for testing
	tmpDir, err := ioutil.TempDir("", "bghelper-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewFileStore(tmpDir)

	// Create and save a process (with PID 0 - not running)
	p := process.NewProcess("atomic-test", "sleep 10")
	p.PID = 0

	if err := store.Save(p); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify the final file exists (should be renamed from temp)
	filePath := filepath.Join(tmpDir, "atomic-test.yaml")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("Final process file was not created")
	}

	// Verify temp files don't remain
	entries, err := ioutil.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read storage directory: %v", err)
	}

	// Check for temp files (should not exist)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") && strings.HasSuffix(entry.Name(), ".yaml") {
			t.Errorf("Temp file was not cleaned up: %s", entry.Name())
		}
	}

	// Verify data integrity
	loaded, err := store.Load("atomic-test")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.PID != 0 {
		t.Errorf("Expected PID 0, got %d", loaded.PID)
	}

	// Status should be computed as stopped since PID is 0
	if loaded.Status != process.StatusStopped {
		t.Errorf("Expected status %s, got %s", process.StatusStopped, loaded.Status)
	}
}

func TestFileStoreEmptyList(t *testing.T) {
	// Create temp directory for testing
	tmpDir, err := ioutil.TempDir("", "bghelper-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewFileStore(tmpDir)

	// List should return empty slice (no error)
	ids, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(ids) != 0 {
		t.Errorf("Expected empty list, got %d items", len(ids))
	}
}
