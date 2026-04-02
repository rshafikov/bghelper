package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ra.shafikov/bghelper/internal/storage"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs <id>",
	Short: "View logs of a process",
	Long: `Display the logs (stdout and stderr) of a process.

The logs command shows the output captured from the process.
Use --follow (-f) to stream logs in real-time for running processes.
Use --tail to show only the last N lines.

Example:
  bgh logs 1
  bgh logs 2 --follow
  bgh logs 3 --tail 50`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		processID := args[0]

		// Get flags
		follow, _ := cmd.Flags().GetBool("follow")
		tail, _ := cmd.Flags().GetInt("tail")

		// Get storage directory
		storageDir, err := getStorageDir()
		if err != nil {
			return fmt.Errorf("failed to get storage directory: %w", err)
		}

		// Initialize storage
		store := storage.NewFileStore(storageDir)

		// Load the process (by ID or name)
		proc, err := resolveProcessIdentifier(processID, store)
		if err != nil {
			return fmt.Errorf("process not found: %s", processID)
		}

		// Get log file path
		logsPath := store.GetLogsPath(proc.ID)

		// Check if log file exists
		if _, err := os.Stat(logsPath); os.IsNotExist(err) {
			return fmt.Errorf("no logs available for process %s", proc.ID)
		}

		// Open log file
		logFile, err := os.Open(logsPath)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		defer func(logFile *os.File) {
			_ = logFile.Close()
		}(logFile)

		if follow {
			// Follow mode: stream logs in real-time
			return streamLogs(logFile, proc.ID)
		}

		if tail > 0 {
			// Tail mode: show last N lines
			return tailLogs(logFile, tail)
		}

		// Default: show all logs
		_, err = io.Copy(os.Stdout, logFile)
		return err
	},
}

// streamLogs streams log output in real-time (like tail -f)
func streamLogs(logFile *os.File, processID string) error {
	// First, output existing content
	_, err := io.Copy(os.Stdout, logFile)
	if err != nil {
		return err
	}

	// Then follow for new content
	reader := bufio.NewReader(logFile)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// Wait a bit and try again
				// In a production system, we'd use fsnotify or similar
				continue
			}
			return err
		}
		fmt.Print(line)
	}
}

// tailLogs shows the last N lines of the log file
func tailLogs(logFile *os.File, n int) error {
	// Read all content
	content, err := io.ReadAll(logFile)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")

	// Get last n lines (excluding empty trailing line if present)
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	start := 0
	if len(lines) > n {
		start = len(lines) - n
	}

	for i := start; i < len(lines); i++ {
		fmt.Println(lines[i])
	}

	return nil
}

func init() {
	rootCmd.AddCommand(logsCmd)

	// Add flags
	logsCmd.Flags().BoolP("follow", "f", false, "Follow log output (stream in real-time)")
	logsCmd.Flags().IntP("tail", "t", 0, "Number of lines to show from the end of the logs (default: all)")

	// Add completion for process IDs
	logsCmd.ValidArgsFunction = completeProcessIDs
}
