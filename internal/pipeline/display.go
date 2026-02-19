package pipeline

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
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

// Header prints the pipeline header.
func (d *Display) Header() {
	fmt.Fprintf(d.w, "\n🐙 vCoding — %s\n", d.title)
	fmt.Fprintln(d.w, strings.Repeat("─", 45))
}

// StepStart prints a step-in-progress line.
func (d *Display) StepStart(name string) {
	fmt.Fprintf(d.w, "⏳ %-12s %-28s\n", name, "running...")
}

// StepDone prints a completed step line.
func (d *Display) StepDone(name, detail string, cost float64, duration time.Duration) {
	costStr := "—"
	if cost > 0 {
		costStr = fmt.Sprintf("$%.4f", cost)
	}
	fmt.Fprintf(d.w, "✅ %-12s %-28s %-8s %.1fs\n",
		name, detail, costStr, duration.Seconds())
}

// StepFailed prints a failed step line.
func (d *Display) StepFailed(name string, err error) {
	fmt.Fprintf(d.w, "❌ %-12s %s\n", name, err.Error())
}

// Summary prints the final run summary.
func (d *Display) Summary(totalCost float64, totalDuration time.Duration, prURL string) {
	fmt.Fprintln(d.w, strings.Repeat("─", 45))
	if prURL != "" {
		fmt.Fprintf(d.w, "✅ Done  $%.4f  %.0fs  %s\n", totalCost, totalDuration.Seconds(), prURL)
	} else {
		fmt.Fprintf(d.w, "✅ Done  $%.4f  %.0fs\n", totalCost, totalDuration.Seconds())
	}
	fmt.Fprintln(d.w)
}

// Failed prints a failure summary.
func (d *Display) Failed(err error) {
	fmt.Fprintln(d.w, strings.Repeat("─", 45))
	fmt.Fprintf(d.w, "❌ Failed: %s\n\n", err.Error())
}
