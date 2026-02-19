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
	w     io.Writer
	title string
}

// NewDisplay creates a display that writes to stdout.
func NewDisplay(title string) *Display {
	return &Display{w: os.Stdout, title: title}
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

// StepStart prints a step-in-progress line.
func (d *Display) StepStart(name, model string) {
	model = truncateModel(model)
	fmt.Fprintf(d.w, "⏳ %-12s %-30s %s\n", name, model, "running...")
}

// StepDone prints a completed step line.
func (d *Display) StepDone(name, model, detail string, cost float64, duration time.Duration) {
	model = truncateModel(model)
	costStr := "—"
	if cost > 0 {
		costStr = fmt.Sprintf("$%.4f", cost)
	}
	fmt.Fprintf(d.w, "✅ %-12s %-30s %-28s %-10s %.1fs\n",
		name, model, detail, costStr, duration.Seconds())
}

// StepFailed prints a failed step line.
func (d *Display) StepFailed(name, model string, err error) {
	model = truncateModel(model)
	fmt.Fprintf(d.w, "❌ %-12s %-30s %s\n", name, model, err.Error())
}

// Summary prints the final run summary.
func (d *Display) Summary(totalCost float64, totalDuration time.Duration, prURL string) {
	fmt.Fprintln(d.w, strings.Repeat("─", 76))
	if prURL != "" {
		fmt.Fprintf(d.w, "✅ Done  $%.4f  %.0fs  %s\n", totalCost, totalDuration.Seconds(), prURL)
	} else {
		fmt.Fprintf(d.w, "✅ Done  $%.4f  %.0fs\n", totalCost, totalDuration.Seconds())
	}
	fmt.Fprintln(d.w)
}

// Failed prints a failure summary.
func (d *Display) Failed(err error) {
	fmt.Fprintln(d.w, strings.Repeat("─", 76))
	fmt.Fprintf(d.w, "❌ Failed: %s\n\n", err.Error())
}
