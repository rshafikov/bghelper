package output

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/ra.shafikov/bghelper/internal/process"
	"golang.org/x/term"
)

// TableFormatter formats processes as a table
type TableFormatter struct {
	writer io.Writer
	width  int
}

// NewTableFormatter creates a new table formatter
func NewTableFormatter(w io.Writer) *TableFormatter {
	width := getTerminalWidth()
	return &TableFormatter{
		writer: w,
		width:  width,
	}
}

// getTerminalWidth returns the terminal width or a default
func getTerminalWidth() int {
	// Try to get terminal width
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		if w < 60 {
			return 60
		}
		return w
	}
	// Default width
	return 80
}

// FormatProcessList formats a list of processes as a table with smart column widths
func (tf *TableFormatter) FormatProcessList(processes []*process.Process) {
	if len(processes) == 0 {
		fmt.Fprintln(tf.writer, "No processes found.")
		return
	}

	// Calculate smart column widths based on actual data
	idWidth := 2       // Minimum: "ID"
	nameWidth := 4     // Minimum: "NAME"
	statusWidth := 6   // Minimum: "STATUS"
	timeWidth := 11    // Fixed: "MM-DD HH:MM"
	commandWidth := 7  // Minimum: "COMMAND"

	// Find maximum widths based on actual data
	for _, p := range processes {
		// ID width
		if len(p.ID) > idWidth {
			idWidth = len(p.ID)
		}
		// Name width
		nameLen := len(p.Name)
		if nameLen == 0 {
			nameLen = 1 // for "-"
		}
		if nameLen > nameWidth {
			nameWidth = nameLen
		}
		// Status width (running=7, stopped=7, crashed=7)
		if len(p.Status) > statusWidth {
			statusWidth = len(p.Status)
		}
		// Command width
		if len(p.Command) > commandWidth {
			commandWidth = len(p.Command)
		}
	}

	// Cap widths at reasonable maxima
	if nameWidth > 15 {
		nameWidth = 15
	}
	maxCommandWidth := tf.width - idWidth - nameWidth - statusWidth - (timeWidth * 2) - 14 // 14 for borders and padding
	if maxCommandWidth < 15 {
		maxCommandWidth = 15
	}
	if commandWidth > maxCommandWidth {
		commandWidth = maxCommandWidth
	}

	// Print top border
	fmt.Fprintf(tf.writer, "+%s+%s+%s+%s+%s+%s+\n",
		strings.Repeat("-", idWidth+2),
		strings.Repeat("-", nameWidth+2),
		strings.Repeat("-", statusWidth+2),
		strings.Repeat("-", timeWidth+2),
		strings.Repeat("-", timeWidth+2),
		strings.Repeat("-", commandWidth+2))

	// Print header
	fmt.Fprintf(tf.writer, "| %-*s | %-*s | %-*s | %-*s | %-*s | %-*s |\n",
		idWidth, "ID",
		nameWidth, "NAME",
		statusWidth, "STATUS",
		timeWidth, "CREATED",
		timeWidth, "UPDATED",
		commandWidth, "COMMAND")

	// Print header/data separator
	fmt.Fprintf(tf.writer, "+%s+%s+%s+%s+%s+%s+\n",
		strings.Repeat("=", idWidth+2),
		strings.Repeat("=", nameWidth+2),
		strings.Repeat("=", statusWidth+2),
		strings.Repeat("=", timeWidth+2),
		strings.Repeat("=", timeWidth+2),
		strings.Repeat("=", commandWidth+2))

	// Print rows
	for _, p := range processes {
		tf.printRow(p, idWidth, nameWidth, statusWidth, commandWidth, timeWidth)
	}

	// Print bottom border
	fmt.Fprintf(tf.writer, "+%s+%s+%s+%s+%s+%s+\n",
		strings.Repeat("-", idWidth+2),
		strings.Repeat("-", nameWidth+2),
		strings.Repeat("-", statusWidth+2),
		strings.Repeat("-", timeWidth+2),
		strings.Repeat("-", timeWidth+2),
		strings.Repeat("-", commandWidth+2))
}

// printRow prints a single process row
func (tf *TableFormatter) printRow(p *process.Process, idWidth, nameWidth, statusWidth, commandWidth, timeWidth int) {
	// Truncate command if needed
	cmd := truncateString(p.Command, commandWidth)

	// Format name (show "-" if empty)
	name := p.Name
	if name == "" {
		name = "-"
	}
	name = truncateString(name, nameWidth)

	// Format timestamps (compact format)
	createdAt := formatTimeCompact(p.CreatedAt)
	updatedAt := formatTimeCompact(p.UpdatedAt)

	// Color-code status
	var statusStr string
	switch p.Status {
	case process.StatusRunning:
		statusStr = color.GreenString("%-*s", statusWidth, string(p.Status))
	case process.StatusStopped:
		statusStr = color.YellowString("%-*s", statusWidth, string(p.Status))
	case process.StatusCrashed:
		statusStr = color.RedString("%-*s", statusWidth, string(p.Status))
	default:
		statusStr = fmt.Sprintf("%-*s", statusWidth, string(p.Status))
	}

	// Print row with borders
	fmt.Fprintf(tf.writer, "| %-*s | %-*s | %s | %-*s | %-*s | %-*s |\n",
		idWidth, p.ID,
		nameWidth, name,
		statusStr,
		timeWidth, createdAt,
		timeWidth, updatedAt,
		commandWidth, cmd)
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	// Handle multi-byte characters properly
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}

// formatTimeCompact formats a time value in compact form (MM-DD HH:MM)
func formatTimeCompact(t time.Time) string {
	if t.IsZero() {
		return "N/A"
	}
	return t.Format("01-02 15:04")
}
