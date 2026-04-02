package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ra.shafikov/bghelper/internal/process"
	"gopkg.in/yaml.v3"
)

// FileStore implements the process.Store interface using the filesystem
type FileStore struct {
	baseDir string
}

// NewFileStore creates a new FileStore with the given base directory
func NewFileStore(baseDir string) *FileStore {
	return &FileStore{
		baseDir: baseDir,
	}
}

// EnsureStorageDir creates the storage directory if it doesn't exist
func (s *FileStore) EnsureStorageDir() error {
	if err := os.MkdirAll(s.baseDir, 0700); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}
	return nil
}

// Save persists a process to disk using atomic writes
func (s *FileStore) Save(p *process.Process) error {
	// Ensure storage directory exists
	if err := s.EnsureStorageDir(); err != nil {
		return err
	}

	// Marshal process to YAML
	data, err := yaml.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to marshal process: %w", err)
	}

	// Create temp file
	tempFile, err := os.CreateTemp(s.baseDir, fmt.Sprintf(".%s-*.yaml", p.ID))
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()

	// Write data to temp file
	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	// Close temp file
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Destination file path
	destPath := filepath.Join(s.baseDir, fmt.Sprintf("%s.yaml", p.ID))

	// Rename temp file to destination (atomic operation)
	if err := os.Rename(tempPath, destPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// Load reads a process from disk and computes its status dynamically
func (s *FileStore) Load(id string) (*process.Process, error) {
	filePath := filepath.Join(s.baseDir, fmt.Sprintf("%s.yaml", id))

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("process not found: %s", id)
	}

	// Read file content
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read process file: %w", err)
	}

	// Unmarshal YAML to Process
	var p process.Process
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("failed to unmarshal process: %w", err)
	}

	// Compute status dynamically based on PID
	process.RefreshStatus(&p)

	return &p, nil
}

// LoadByName finds a process by its name (if set)
func (s *FileStore) LoadByName(name string) (*process.Process, error) {
	// Get all process IDs
	ids, err := s.List()
	if err != nil {
		return nil, err
	}

	// Search for process with matching name
	for _, id := range ids {
		p, err := s.Load(id)
		if err != nil {
			continue // Skip processes that fail to load
		}
		if p.Name == name {
			return p, nil
		}
	}

	return nil, fmt.Errorf("process not found with name: %s", name)
}

// List returns all process IDs in storage
func (s *FileStore) List() ([]string, error) {
	// Ensure directory exists
	if err := s.EnsureStorageDir(); err != nil {
		return nil, err
	}

	// Read directory contents
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage directory: %w", err)
	}

	// Collect process IDs (extract from filename.yaml)
	var ids []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, ".yaml") && !strings.HasPrefix(name, ".") {
			// Extract ID from filename
			id := strings.TrimSuffix(name, ".yaml")
			ids = append(ids, id)
		}
	}

	return ids, nil
}

// Delete removes a process file from storage
func (s *FileStore) Delete(id string) error {
	filePath := filepath.Join(s.baseDir, fmt.Sprintf("%s.yaml", id))

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("process not found: %s", id)
	}

	// Delete the file
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete process file: %w", err)
	}

	// Also delete log file if it exists
	logsPath := s.GetLogsPath(id)
	if _, err := os.Stat(logsPath); err == nil {
		os.Remove(logsPath) // Ignore errors for log file deletion
	}

	return nil
}

// GetLogsPath returns the path to the log file for a process
func (s *FileStore) GetLogsPath(id string) string {
	return filepath.Join(s.baseDir, fmt.Sprintf("%s.log", id))
}

// LoadAll loads all processes from storage
func (s *FileStore) LoadAll() ([]*process.Process, error) {
	// Get all process IDs
	ids, err := s.List()
	if err != nil {
		return nil, err
	}

	// Load each process
	var processes []*process.Process
	for _, id := range ids {
		p, err := s.Load(id)
		if err != nil {
			// Skip processes that fail to load
			continue
		}
		processes = append(processes, p)
	}

	return processes, nil
}
