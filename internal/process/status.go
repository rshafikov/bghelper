package process

import (
	"os"
	"syscall"
)

// IsProcessAlive checks if a process with the given PID is still running
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	// Try to find the process
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process is alive
	// This doesn't actually send a signal, just checks if process exists
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

// RefreshStatus checks if a process is still alive and updates its status.
// Status is computed dynamically and NOT persisted to storage.
// Returns true if status was changed from running to stopped/crashed.
func RefreshStatus(p *Process) bool {
	// If PID is 0, process is not running
	if p.PID == 0 {
		p.Status = StatusStopped
		return false
	}

	// Check if PID is still alive
	if IsProcessAlive(p.PID) {
		p.Status = StatusRunning
		return false
	}

	// Process is not alive anymore - determine status based on exit code
	// If we have no exit code set, assume it exited cleanly
	if p.ExitCode == nil {
		exitCode := 0
		p.ExitCode = &exitCode
		p.Status = StatusStopped
	} else if *p.ExitCode == 0 {
		p.Status = StatusStopped
	} else {
		p.Status = StatusCrashed
	}

	return true
}
