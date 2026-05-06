package download

import (
	"os"

	"github.com/schollz/progressbar/v3"
)

// Reporter is called periodically to report download progress.
type Reporter interface {
	Report(current, total int64)
}

// ConsoleReporter writes progress to the console using a progress bar.
type ConsoleReporter struct {
	bar *progressbar.ProgressBar
}

// NewConsoleReporter creates a new console progress reporter.
func NewConsoleReporter(total int64) *ConsoleReporter {
	return &ConsoleReporter{
		bar: progressbar.NewOptions64(
			total,
			progressbar.OptionSetWriter(os.Stderr),
			progressbar.OptionEnableColorCodes(true),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetWidth(50),
			progressbar.OptionSetDescription("Downloading"),
		),
	}
}

// Report updates the progress bar.
func (r *ConsoleReporter) Report(current, total int64) {
	r.bar.Set64(current)
}

// ProgressMsg is a JSON-friendly progress update.
type ProgressMsg struct {
	Current int64 `json:"current"`
	Total   int64 `json:"total"`
}

// ServerReporter sends progress updates over a channel.
type ServerReporter struct {
	ch chan<- ProgressMsg
}

// NewServerReporter creates a new server progress reporter.
func NewServerReporter(ch chan<- ProgressMsg) *ServerReporter {
	return &ServerReporter{ch: ch}
}

// Report sends a progress message to the channel (non-blocking).
func (r *ServerReporter) Report(current, total int64) {
	select {
	case r.ch <- ProgressMsg{Current: current, Total: total}:
	default:
	}
}

// NopReporter discards all progress updates.
type NopReporter struct{}

// Report is a no-op.
func (r *NopReporter) Report(current, total int64) {}
