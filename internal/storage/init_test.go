package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ra.shafikov/bghelper/internal/process"
)

func TestAutomaticDirectoryCreation(t *testing.T) {
	// Use a non-existent directory to test automatic creation
	tmpDir, err := os.MkdirTemp("", "bghelper-init-test-")
	if err != nil {
		t.Fatalf("Failed to create temp root: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Point to a subdirectory that doesn't exist yet
	storageDir := filepath.Join(tmpDir, ".bghelper", "processes")

	// Verify directory doesn't exist initially
	if _, err := os.Stat(storageDir); !os.IsNotExist(err) {
		t.Fatal("Storage directory should not exist initially")
	}

	// Create FileStore - this should trigger directory creation
	store := NewFileStore(storageDir)

	// Create a process and save it
	p := process.NewProcess("init-test-001", "echo test")
	if err := store.Save(p); err != nil {
		t.Fatalf("Save should create directories automatically: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(storageDir); os.IsNotExist(err) {
		t.Fatal("Storage directory was not created automatically")
	}

	// Verify correct permissions (0700)
	info, err := os.Stat(storageDir)
	if err != nil {
		t.Fatalf("Failed to stat storage directory: %v", err)
	}

	// Check file permissions
	mode := info.Mode().Perm()
	if mode != 0700 {
		t.Errorf("Expected directory permissions 0700, got %#o", mode)
	}

	// Verify parent directory also exists
	parentDir := filepath.Dir(storageDir)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		t.Fatal("Parent .bghelper directory was not created")
	}

	// Verify file was created and can be loaded
	loaded, err := store.Load("init-test-001")
	if err != nil {
		t.Fatalf("Failed to load process: %v", err)
	}

	if loaded.ID != p.ID {
		t.Errorf("Process ID mismatch: expected %s, got %s", p.ID, loaded.ID)
	}
}

func TestMultipleDirectoryCreationAttempts(t *testing.T) {
	// Test that multiple calls to Save don't cause issues
	tmpDir, err := os.MkdirTemp("", "bghelper-multi-test-")
	if err != nil {
		t.Fatalf("Failed to create temp root: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storageDir := filepath.Join(tmpDir, ".bghelper", "processes")
	store := NewFileStore(storageDir)

	// Save multiple processes - directory creation should handle this gracefully
	for i := 1; i <= 3; i++ {
		p := process.NewProcess(getTestID(i), "echo test")
		if err := store.Save(p); err != nil {
			t.Fatalf("Save %d failed: %v", i, err)
		}
	}

	// Verify all files exist
	for i := 1; i <= 3; i++ {
		filePath := filepath.Join(storageDir, getTestID(i)+".yaml")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Process file %s was not created", filePath)
		}
	}

	// List should work correctly
	ids, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(ids) != 3 {
		t.Errorf("Expected 3 processes in list, got %d", len(ids))
	}
}

func TestApplicationContinuesWithoutErrors(t *testing.T) {
	// Simulate a real-world scenario where directory doesn't exist
	tmpDir, err := os.MkdirTemp("", "bghelper-continue-test-")
	if err != nil {
		t.Fatalf("Failed to create temp root: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storageDir := filepath.Join(tmpDir, ".bghelper", "processes")
	store := NewFileStore(storageDir)

	// Application should continue without errors
	p1 := process.NewProcess("continue-001", "ls -la")
	if err := store.Save(p1); err != nil {
		t.Fatalf("Expected no error when saving to non-existent directory: %v", err)
	}

	// Should be able to immediately save another process
	p2 := process.NewProcess("continue-002", "pwd")
	if err := store.Save(p2); err != nil {
		t.Fatalf("Expected no error on second save: %v", err)
	}

	// All operations should work (Load, List, Delete)
	_, err = store.Load("continue-001")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	ids, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(ids) != 2 {
		t.Errorf("Expected 2 processes, got %d", len(ids))
	}

	if err := store.Delete("continue-001"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify application state is consistent
	remaining, _ := store.List()
	if len(remaining) != 1 {
		t.Errorf("Expected 1 remaining process, got %d", len(remaining))
	}
}

func TestFirstRunScenario(t *testing.T) {
	// Simulate a user's first run experience
	tmpDir := os.TempDir()
	storageDir := filepath.Join(tmpDir, ".bghelper-test", "processes")

	// Clean up any previous test remnants
	defer os.RemoveAll(filepath.Join(tmpDir, ".bghelper-test"))

	// Verify clean state
	if _, err := os.Stat(storageDir); !os.IsNotExist(err) {
		os.RemoveAll(storageDir)
	}

	store := NewFileStore(storageDir)

	// User's first command: save a process
	// Note: Status is computed dynamically, not stored
	p := process.NewProcess("first-run", "ssh -L 8080:localhost:8080 user@server")
	p.PID = 0 // PID 0 means not running

	// Should succeed without manual setup
	if err := store.Save(p); err != nil {
		t.Fatalf("First run failed - directories not created automatically: %v", err)
	}

	// Verify application can be used immediately
	loaded, err := store.Load("first-run")
	if err != nil {
		t.Fatalf("First run load failed: %v", err)
	}

	// Status is computed dynamically from PID
	// PID 0 means the process is not running
	if loaded.Status != process.StatusStopped {
		t.Errorf("Expected process status to be stopped (PID 0), got %s", loaded.Status)
	}

	if loaded.PID != 0 {
		t.Errorf("Expected PID 0, got %d", loaded.PID)
	}
}

func TestConcurrentDirectoryCreation(t *testing.T) {
	// Test that concurrent Save operations don't race
	tmpDir, err := os.MkdirTemp("", "bghelper-concurrent-test-")
	if err != nil {
		t.Fatalf("Failed to create temp root: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storageDir := filepath.Join(tmpDir, ".bghelper", "processes")
	store := NewFileStore(storageDir)

	// Save multiple processes concurrently
	done := make(chan bool, 3)
	errors := make(chan error, 3)

	for i := 1; i <= 3; i++ {
		go func(id string) {
			p := process.NewProcess(id, "echo concurrent")
			if err := store.Save(p); err != nil {
				errors <- err
			} else {
				done <- true
			}
		}(getTestID(i))
	}

	// Wait for completion
	for i := 0; i < 3; i++ {
		select {
		case err := <-errors:
			t.Errorf("Concurrent save failed: %v", err)
		case <-done:
			// Success
		}
	}

	// Verify all processes saved
	ids, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(ids) != 3 {
		t.Errorf("Expected 3 processes, got %d", len(ids))
	}
}

// Helper function for generating test IDs
func getTestID(num int) string {
	ids := []string{"", "test-001", "test-002", "test-003"}
	if num >= 0 && num < len(ids) {
		return ids[num]
	}
	return "test-001"
}
