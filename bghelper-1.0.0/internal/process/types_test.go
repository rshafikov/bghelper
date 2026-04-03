package process

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestProcessMarshalYAML(t *testing.T) {
	// Create a test process
	exitCode := 0
	p := &Process{
		ID:        "001",
		Command:   "ssh -L 8080:localhost:8080 user@server",
		Status:    StatusRunning,
		PID:       12345,
		CreatedAt: time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC),
		StartedAt: time.Date(2026, 3, 30, 10, 5, 0, 0, time.UTC),
		ExitCode:  &exitCode,
	}

	// Marshal to YAML
	data, err := yaml.Marshal(p)
	if err != nil {
		t.Fatalf("Failed to marshal process: %v", err)
	}

	// Expected YAML structure - note that Status is NOT serialized (yaml:"-")
	expected := `id: "001"
command: ssh -L 8080:localhost:8080 user@server
pid: 12345
created_at: 2026-03-30T10:00:00Z
started_at: 2026-03-30T10:05:00Z
exit_code: 0
`

	if string(data) != expected {
		t.Errorf("Marshaled YAML doesn't match expected format.\nGot:\n%s\nExpected:\n%s", string(data), expected)
	}
}

func TestProcessUnmarshalYAML(t *testing.T) {
	// Note: status is NOT serialized - it's computed dynamically
	yamlData := `id: "002"
command: vkey start
pid: 0
created_at: 2026-03-30T11:00:00Z
started_at: "0001-01-01T00:00:00Z"
`

	var p Process
	err := yaml.Unmarshal([]byte(yamlData), &p)
	if err != nil {
		t.Fatalf("Failed to unmarshal process: %v", err)
	}

	// Verify all fields
	if p.ID != "002" {
		t.Errorf("Expected ID '002', got '%s'", p.ID)
	}

	if p.Command != "vkey start" {
		t.Errorf("Expected Command 'vkey start', got '%s'", p.Command)
	}

	// Status is computed dynamically, not loaded from YAML
	// After unmarshal, Status will be empty string (zero value)
	// It should be computed by calling RefreshStatus()
	if p.Status != "" {
		t.Errorf("Expected Status to be empty after unmarshal (computed dynamically), got '%s'", p.Status)
	}

	if p.PID != 0 {
		t.Errorf("Expected PID 0, got %d", p.PID)
	}

	expectedCreated := time.Date(2026, 3, 30, 11, 0, 0, 0, time.UTC)
	if !p.CreatedAt.Equal(expectedCreated) {
		t.Errorf("Expected CreatedAt %v, got %v", expectedCreated, p.CreatedAt)
	}

	// ExitCode should be nil (omitempty)
	if p.ExitCode != nil {
		t.Errorf("Expected ExitCode nil, got %v", *p.ExitCode)
	}
}

func TestProcessWithNilExitCode(t *testing.T) {
	// Test that a process with nil ExitCode marshals correctly (omitempty)
	p := NewProcess("003", "ls -la")
	p.Status = StatusRunning // This won't be serialized (yaml:"-")
	p.PID = 99999

	data, err := yaml.Marshal(p)
	if err != nil {
		t.Fatalf("Failed to marshal process: %v", err)
	}

	// Should not contain exit_code field when nil
	if contains(string(data), "exit_code") {
		t.Errorf("ExitCode should be omitted when nil, but found in YAML:\n%s", string(data))
	}

	// Should not contain status field (yaml:"-" means it's not serialized)
	if contains(string(data), "status") {
		t.Errorf("Status should not be serialized, but found in YAML:\n%s", string(data))
	}
}

func TestStatusConstants(t *testing.T) {
	if StatusRunning != "running" {
		t.Errorf("Expected StatusRunning to be 'running', got '%s'", StatusRunning)
	}

	if StatusStopped != "stopped" {
		t.Errorf("Expected StatusStopped to be 'stopped', got '%s'", StatusStopped)
	}

	if StatusCrashed != "crashed" {
		t.Errorf("Expected StatusCrashed to be 'crashed', got '%s'", StatusCrashed)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && hasSubstring(s, substr))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
