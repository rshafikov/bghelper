package process

import (
	"time"
)

// Status represents the current state of a process
type Status string

const (
	StatusRunning Status = "running"
	StatusStopped Status = "stopped"
	StatusCrashed Status = "crashed"
)

// Store defines the interface for process persistence
type Store interface {
	Save(p *Process) error
	Load(id string) (*Process, error)
	LoadByName(name string) (*Process, error)
	List() ([]string, error)
	Delete(id string) error
	GetLogsPath(id string) string
}

// Process represents a background process managed by bghelper
type Process struct {
	// ID is a unique identifier for the process
	ID string `yaml:"id"`

	// Name is an optional friendly name for the process
	Name string `yaml:"name,omitempty"`

	// Command is the shell command to execute
	Command string `yaml:"command"`

	// Status is the current state of the process (computed dynamically, not stored)
	Status Status `yaml:"-"`

	// PID is the operating system process ID (0 if not running)
	PID int `yaml:"pid"`

	// CreatedAt is when the process definition was created
	CreatedAt time.Time `yaml:"created_at"`

	// StartedAt is when the process was last started (zero if never started)
	StartedAt time.Time `yaml:"started_at"`

	// ExitCode is the exit code from the last run (nil if still running or never exited)
	ExitCode *int `yaml:"exit_code,omitempty"`

	// LogsPath is the path to the log file for this process
	LogsPath string `yaml:"logs_path,omitempty"`
}

// NewProcess creates a new Process with default values
func NewProcess(id, command string) *Process {
	return &Process{
		ID:        id,
		Command:   command,
		Status:    StatusStopped,
		PID:       0,
		CreatedAt: time.Now(),
		StartedAt: time.Time{},
		ExitCode:  nil,
	}
}
