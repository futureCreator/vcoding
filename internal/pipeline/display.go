package pipeline

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Display handles terminal progress output for the pipeline.
type Display struct {
	w       io.Writer
	title   string
	verbose bool
	stop    chan struct{}
	done    chan struct{}
}

// NewDisplay creates a display that writes to stdout.
func NewDisplay(title string, verbose bool) *Display {
	return &Display{w: os.Stdout, title: title, verbose: verbose}
}

// modelColumnWidth is the fixed display width reserved for the model/executor column.
var modelColumnWidth = 30

// ansiEscapeRe matches ANSI terminal escape sequences and C0/DEL control characters.
var ansiEscapeRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|[\x00-\x1f\x7f]`)

// sanitizeModel strips ANSI escape sequences and control characters from a model name.
func sanitizeModel(name string) string {
	return ansiEscapeRe.ReplaceAllString(name, "")
}

// truncateModel sanitizes and truncates model to fit within modelColumnWidth runes,
// appending an ellipsis if truncation occurs.
func truncateModel(model string) string {
	model = sanitizeModel(model)
	if utf8.RuneCountInString(model) <= modelColumnWidth {
		return model
	}
	runes := []rune(model)
	return string(runes[:modelColumnWidth-1]) + "…"
}

// Header prints the pipeline header.
func (d *Display) Header() {
	fmt.Fprintf(d.w, "\n🐙 vCoding — %s\n", d.title)
	fmt.Fprintln(d.w, strings.Repeat("─", 76))
}

// StepStart prints a step-in-progress line and starts an elapsed time ticker.
// In non-verbose mode, the line is updated in place every second with elapsed time.
// In verbose mode, a plain line is printed (executor output follows on subsequent lines).
func (d *Display) StepStart(name, model string) {
	model = truncateModel(model)
	if d.verbose {
		fmt.Fprintf(d.w, "⏳ %-12s %-30s running...\n", name, model)
		return
	}
	// Print without trailing newline so the ticker can overwrite in place.
	fmt.Fprintf(d.w, "⏳ %-12s %-30s running...", name, model)

	stop := make(chan struct{})
	done := make(chan struct{})
	d.stop = stop
	d.done = done
	start := time.Now()

	go func() {
		defer close(done)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				fmt.Fprintf(d.w, "\r⏳ %-12s %-30s running... %.0fs",
					name, model, time.Since(start).Seconds())
			}
		}
	}()
}

// stopTicker stops the elapsed time goroutine and waits for it to finish.
func (d *Display) stopTicker() {
	if d.stop != nil {
		close(d.stop)
		<-d.done
		d.stop = nil
		d.done = nil
	}
}

// maxPreviewLines is the default number of artifact lines shown after step completion.
const maxPreviewLines = 10

// StepDone prints a completed step line, overwriting the running line in non-verbose mode.
// artifactContent, when non-empty, is shown as a preview (first maxPreviewLines lines).
func (d *Display) StepDone(name, model, detail string, cost float64, duration time.Duration, artifactContent string) {
	d.stopTicker()
	model = truncateModel(model)
	costStr := "—"
	if cost > 0 {
		costStr = fmt.Sprintf("$%.4f", cost)
	}
	prefix := "\r"
	if d.verbose {
		prefix = ""
	}
	fmt.Fprintf(d.w, "%s✅ %-12s %-30s %-28s %-10s %.1fs\n",
		prefix, name, model, detail, costStr, duration.Seconds())

	// Artifact preview
	if artifactContent != "" {
		lines := strings.Split(artifactContent, "\n")
		// Drop the trailing empty element that Split adds for a newline-terminated string.
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		previewLines := lines
		truncated := false
		if len(lines) > maxPreviewLines {
			previewLines = lines[:maxPreviewLines]
			truncated = true
		}
		for _, l := range previewLines {
			fmt.Fprintf(d.w, "  │ %s\n", l)
		}
		if truncated {
			fmt.Fprintf(d.w, "  │ ... (%d more lines)\n", len(lines)-maxPreviewLines)
		}
	}
}

// StepFailed prints a failed step line, overwriting the running line in non-verbose mode.
func (d *Display) StepFailed(name, model string, err error) {
	d.stopTicker()
	model = truncateModel(model)
	prefix := "\r"
	if d.verbose {
		prefix = ""
	}
	fmt.Fprintf(d.w, "%s❌ %-12s %-30s %s\n", prefix, name, model, err.Error())
}

// Summary prints the final run summary.
func (d *Display) Summary(totalCost float64, totalDuration time.Duration) {
	fmt.Fprintln(d.w, strings.Repeat("─", 76))
	fmt.Fprintf(d.w, "✅ Done  $%.4f  %.0fs\n", totalCost, totalDuration.Seconds())
	fmt.Fprintln(d.w)
}

// Failed prints a failure summary.
func (d *Display) Failed(err error) {
	fmt.Fprintln(d.w, strings.Repeat("─", 76))
	fmt.Fprintf(d.w, "❌ Failed: %s\n\n", err.Error())
}
