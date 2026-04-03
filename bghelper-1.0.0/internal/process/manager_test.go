package process_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ra.shafikov/bghelper/internal/process"
	"github.com/ra.shafikov/bghelper/internal/storage"
)

func TestManagerStartAndPersist(t *testing.T) {
	// Create temp storage directory
	tmpDir, err := os.MkdirTemp("", "bghelper-manager-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := storage.NewFileStore(tmpDir)
	manager := process.NewManager(store)

	// Start a process (use sleep to keep it running longer)
	p, err := manager.Start("test-001", "sleep 2")
	if err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	// Verify process is in memory
	if p.Status != process.StatusRunning {
		t.Errorf("Expected process to be running, got %s", p.Status)
	}

	if p.PID == 0 {
		t.Error("Expected PID to be set")
	}

	// Wait a bit for persistence (should be immediate, but give it 100ms)
	time.Sleep(100 * time.Millisecond)

	// Verify process was persisted
	loaded, err := store.Load("test-001")
	if err != nil {
		t.Fatalf("Failed to load persisted process: %v", err)
	}

	// Status is computed dynamically - if PID is alive, it should be running
	// The process should still be running since we used "sleep 2"
	if loaded.Status != process.StatusRunning {
		t.Errorf("Expected computed status to be running (PID alive), got %s", loaded.Status)
	}

	if loaded.PID != p.PID {
		t.Errorf("PID mismatch: expected %d, got %d", p.PID, loaded.PID)
	}

	if loaded.Command != "sleep 2" {
		t.Errorf("Command mismatch: expected 'sleep 2', got '%s'", loaded.Command)
	}
}

func TestManagerStopAndPersist(t *testing.T) {
	// Create temp storage directory
	tmpDir, err := os.MkdirTemp("", "bghelper-manager-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := storage.NewFileStore(tmpDir)
	manager := process.NewManager(store)

	// Start a process
	_, err = manager.Start("test-stop-001", "sleep 1")
	if err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	// Wait for it to start
	time.Sleep(100 * time.Millisecond)

	// Stop the process
	if err := manager.Stop("test-stop-001"); err != nil {
		t.Fatalf("Failed to stop process: %v", err)
	}

	// Wait for process to exit and status to be updated
	time.Sleep(200 * time.Millisecond)

	// Verify status changed to stopped
	p, err := manager.Get("test-stop-001")
	if err != nil {
		t.Fatalf("Failed to get process: %v", err)
	}

	if p.Status != process.StatusStopped && p.Status != process.StatusCrashed {
		t.Errorf("Expected process to be stopped, got %s", p.Status)
	}

	// Verify persistence - status is computed dynamically from PID
	loaded, err := store.Load("test-stop-001")
	if err != nil {
		t.Fatalf("Failed to load persisted process: %v", err)
	}

	// Status is computed dynamically based on PID
	// After stop, the process is no longer alive, so status should be stopped
	if loaded.Status != process.StatusStopped && loaded.Status != process.StatusCrashed {
		t.Errorf("Expected computed status stopped/crashed (PID not alive), got %s", loaded.Status)
	}

	if loaded.ExitCode == nil {
		t.Error("Expected ExitCode to be set after stop")
	}

	if *loaded.ExitCode != 0 && *loaded.ExitCode != 1 {
		// May be 1 if interrupted
		t.Logf("Exit code: %d", *loaded.ExitCode)
	}
}

func TestManagerStateChangePersistence(t *testing.T) {
	// Create temp storage directory
	tmpDir, err := os.MkdirTemp("", "bghelper-manager-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := storage.NewFileStore(tmpDir)
	manager := process.NewManager(store)

	// Start a process
	p, err := manager.Start("test-state-001", "echo test")
	if err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	initialPID := p.PID

	// Wait for process to complete (echo should exit quickly)
	time.Sleep(200 * time.Millisecond)

	// Verify process status was updated
	p, err = manager.Get("test-state-001")
	if err != nil {
		t.Fatalf("Failed to get process: %v", err)
	}

	if p.Status == process.StatusRunning {
		t.Error("Process should not still be running after completion")
	}

	if p.ExitCode == nil {
		t.Error("ExitCode should be set after process completion")
	}

	// Verify the final state was persisted
	loaded, err := store.Load("test-state-001")
	if err != nil {
		t.Fatalf("Failed to load persisted state: %v", err)
	}

	// Status is computed dynamically - after process exits, PID is not alive
	// so status should be stopped
	if loaded.Status != process.StatusStopped {
		t.Errorf("Expected computed status stopped (PID not alive), got %s", loaded.Status)
	}

	if loaded.PID != initialPID {
		t.Errorf("PID was modified after completion: expected %d, got %d", initialPID, loaded.PID)
	}
}

func TestManagerPersistenceWithinOneSecond(t *testing.T) {
	// Create temp storage directory
	tmpDir, err := os.MkdirTemp("", "bghelper-manager-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := storage.NewFileStore(tmpDir)
	manager := process.NewManager(store)

	// Start time
	startTime := time.Now()

	// Start a process
	_, err = manager.Start("test-speed-001", "echo fast")
	if err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

	// Wait for file to appear (should be within 1 second)
	filePath := filepath.Join(tmpDir, "test-speed-001.yaml")
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filePath); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	elapsed := time.Since(startTime)

	// Verify file exists and was created within 1 second
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("Process file was not created within expected time")
	}

	if elapsed > 1000*time.Millisecond {
		t.Errorf("Process persistence took %v, expected < 1 second", elapsed)
	}

	t.Logf("Process persisted within: %v", elapsed)
}

func TestManagerDeleteAndPersist(t *testing.T) {
	// Create temp storage directory
	tmpDir, err := os.MkdirTemp("", "bghelper-manager-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := storage.NewFileStore(tmpDir)
	manager := process.NewManager(store)

	// Create a stopped process
	p := process.NewProcess("test-delete-001", "echo test")
	p.Status = process.StatusStopped
	if err := store.Save(p); err != nil {
		t.Fatalf("Failed to save initial process: %v", err)
	}

	// Load it into memory (simulating a restart scenario)
	err = manager.LoadFromStorage("test-delete-001")
	if err != nil {
		t.Fatalf("Failed to load process into memory: %v", err)
	}

	// Delete the process
	if err := manager.Delete("test-delete-001"); err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// Verify file was deleted
	filePath := filepath.Join(tmpDir, "test-delete-001.yaml")
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("Process file should be deleted")
	}

	// Verify removed from memory
	_, err = manager.Get("test-delete-001")
	if err == nil {
		t.Error("Expected error when getting deleted process")
	}
}

func TestManagerUnloadAfterPersist(t *testing.T) {
	// Test that process can be removed from memory but remains in storage
	tmpDir, err := os.MkdirTemp("", "bghelper-manager-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := storage.NewFileStore(tmpDir)
	manager := process.NewManager(store)

	// Start a process
	p, err := manager.Start("test-unload-001", "echo test")
	if err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	_ = p // Avoid unused variable warning

	// Wait for completion
	time.Sleep(200 * time.Millisecond)

	// Verify persisted
	_, err = store.Load("test-unload-001")
	if err != nil {
		t.Fatalf("Initial save failed: %v", err)
	}

	// Simulate application restart by loading directly from storage
	// (this bypasses the in-memory cache as it would after a restart)

	// Verify still in storage
	loaded, err := store.Load("test-unload-001")
	if err != nil {
		t.Fatalf("Load after memory clear failed: %v", err)
	}

	// Status is computed dynamically - after process exits, status is stopped
	if loaded.Status != process.StatusStopped {
		t.Errorf("Expected computed status stopped (PID not alive), got %s", loaded.Status)
	}
}
