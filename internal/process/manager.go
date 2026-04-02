package process

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// Manager handles process lifecycle and state management
type Manager struct {
	store     Store
	processes map[string]*Process
	mu        sync.RWMutex
}

// NewManager creates a new process manager
func NewManager(store Store) *Manager {
	return &Manager{
		store:     store,
		processes: make(map[string]*Process),
	}
}

// Start creates and starts a new background process
func (m *Manager) Start(id string, command string) (*Process, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if process already exists
	if _, exists := m.processes[id]; exists {
		return nil, fmt.Errorf("process with id %s already exists", id)
	}

	// Create new process
	p := NewProcess(id, command)
	p.Status = StatusRunning
	p.StartedAt = time.Now()

	// Get log file path
	p.LogsPath = m.store.GetLogsPath(id)

	// Ensure the log file directory exists
	logsDir := filepath.Dir(p.LogsPath)
	if err := os.MkdirAll(logsDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Create/open log file for writing
	logFile, err := os.OpenFile(p.LogsPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	// Start the command
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	// Set PID
	p.PID = cmd.Process.Pid

	// Track in memory
	m.processes[id] = p

	// Persist immediately
	if err := m.store.Save(p); err != nil {
		// Don't fail the start if save fails, but log the error
		fmt.Fprintf(os.Stderr, "Warning: failed to persist process: %v\n", err)
	}

	// Monitor the process in a goroutine
	go m.monitorProcess(id, cmd, logFile)

	return p, nil
}

// Stop stops a running process
func (m *Manager) Stop(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, exists := m.processes[id]
	if !exists {
		return fmt.Errorf("process not found: %s", id)
	}

	if p.Status != StatusRunning {
		return fmt.Errorf("process is not running: %s", id)
	}

	// Send SIGTERM to the process
	proc, err := os.FindProcess(p.PID)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Update status to signal we're stopping
	originalStatus := p.Status
	p.Status = StatusStopped

	// Send interrupt signal
	if err := proc.Signal(os.Interrupt); err != nil {
		// If graceful shutdown fails, force kill
		p.Status = originalStatus // Reset status if can't signal
		if err := proc.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
		// If kill succeeds, update status to stopped anyway
		p.Status = StatusStopped
	}

	// Set exit code to indicate interrupted
	exitCode := 130 // 128 + 2 (SIGINT)
	p.ExitCode = &exitCode

	// Persist the updated state immediately
	if saveErr := m.store.Save(p); saveErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to persist process state: %v\n", saveErr)
	}

	return nil
}

// monitorProcess watches a process and updates its status when it exits
func (m *Manager) monitorProcess(id string, cmd *exec.Cmd, logFile io.Closer) {
	// Wait for the process to exit
	err := cmd.Wait()

	// Close log file after process exits
	if logFile != nil {
		logFile.Close()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	p, exists := m.processes[id]
	if !exists {
		// Process was deleted while monitoring
		return
	}

	// Update status based on exit code
	if err != nil {
		// Process exited with error
		exitCode := 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		p.Status = StatusCrashed
		p.ExitCode = &exitCode
	} else {
		// Process exited cleanly
		exitCode := 0
		p.Status = StatusStopped
		p.ExitCode = &exitCode
	}

	// Persist the updated state immediately
	if saveErr := m.store.Save(p); saveErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to persist process state: %v\n", saveErr)
	}
}

// Get retrieves a process by ID
func (m *Manager) Get(id string) (*Process, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, exists := m.processes[id]
	if !exists {
		return nil, fmt.Errorf("process not found: %s", id)
	}

	return p, nil
}

// LoadFromStorage loads a process from storage into memory
func (m *Manager) LoadFromStorage(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already in memory
	if _, exists := m.processes[id]; exists {
		return fmt.Errorf("process already in memory: %s", id)
	}

	// Load from storage
	p, err := m.store.Load(id)
	if err != nil {
		return fmt.Errorf("failed to load from storage: %w", err)
	}

	// Compute status dynamically
	RefreshStatus(p)

	// Add to memory
	m.processes[id] = p
	return nil
}

// Delete removes a process from storage and memory
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if process exists in memory
	p, exists := m.processes[id]
	if !exists {
		// Try to delete from storage anyway (might be a stale file)
		if err := m.store.Delete(id); err != nil {
			return fmt.Errorf("process not found: %s", id)
		}
		return nil
	}

	// Don't delete running processes
	if p.Status == StatusRunning {
		return fmt.Errorf("cannot delete running process: %s", id)
	}

	// Delete from storage
	if err := m.store.Delete(id); err != nil {
		return fmt.Errorf("failed to delete from storage: %w", err)
	}

	// Remove from memory
	delete(m.processes, id)

	return nil
}

// Restart restarts a process - stops it if running and starts again with same ID
func (m *Manager) Restart(id string) (*Process, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get existing process from memory or storage
	p, exists := m.processes[id]
	if !exists {
		// Try to load from storage
		var err error
		p, err = m.store.Load(id)
		if err != nil {
			return nil, fmt.Errorf("process not found: %s", id)
		}
		// Add to memory
		m.processes[id] = p
	}

	// If process is running, stop it first
	if p.Status == StatusRunning && p.PID > 0 {
		proc, err := os.FindProcess(p.PID)
		if err == nil {
			// Try graceful shutdown first
			if err := proc.Signal(os.Interrupt); err != nil {
				// If graceful fails, force kill
				proc.Kill()
			}
			// Wait a moment for process to exit
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Clear log file for fresh start
	if p.LogsPath != "" {
		os.Truncate(p.LogsPath, 0)
	}

	// Reset process state for restart
	p.Status = StatusRunning
	p.StartedAt = time.Now()
	p.ExitCode = nil

	// Ensure log file exists
	if p.LogsPath == "" {
		p.LogsPath = m.store.GetLogsPath(id)
	}
	logsDir := filepath.Dir(p.LogsPath)
	if err := os.MkdirAll(logsDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Create/open log file
	logFile, err := os.OpenFile(p.LogsPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	// Start the command
	cmd := exec.Command("bash", "-c", p.Command)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	// Update PID
	p.PID = cmd.Process.Pid

	// Persist the updated state
	if err := m.store.Save(p); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to persist process: %v\n", err)
	}

	// Monitor the process in a goroutine
	go m.monitorProcess(id, cmd, logFile)

	return p, nil
}

// List returns all process IDs
func (m *Manager) List() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Also list from storage to get stopped processes
	storedIDs, err := m.store.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list from storage: %w", err)
	}

	// Combine with in-memory processes
	idSet := make(map[string]bool)

	// Add stored IDs
	for _, id := range storedIDs {
		idSet[id] = true
	}

	// Add in-memory IDs
	for id := range m.processes {
		idSet[id] = true
	}

	// Convert to slice
	var ids []string
	for id := range idSet {
		ids = append(ids, id)
	}

	return ids, nil
}

// ListAll returns all processes (from memory and storage)
func (m *Manager) ListAll() ([]*Process, error) {
	ids, err := m.List()
	if err != nil {
		return nil, err
	}

	var processes []*Process
	for _, id := range ids {
		p, err := m.Get(id)
		if err != nil {
			// If not in memory, try loading from storage
			stored, loadErr := m.store.Load(id)
			if loadErr != nil {
				continue // Skip if can't load
			}
			processes = append(processes, stored)
		} else {
			processes = append(processes, p)
		}
	}

	return processes, nil
}
