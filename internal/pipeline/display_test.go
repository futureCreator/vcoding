package pipeline

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func newTestDisplay(buf *bytes.Buffer) *Display {
	return &Display{w: buf, title: "test"}
}

func TestStepStart_ContainsModel(t *testing.T) {
	var buf bytes.Buffer
	d := newTestDisplay(&buf)
	d.StepStart("plan", "anthropic/claude-opus-4-6")
	out := buf.String()
	if !strings.Contains(out, "anthropic/claude-opus-4-6") {
		t.Errorf("StepStart output missing model: %q", out)
	}
	if !strings.Contains(out, "plan") {
		t.Errorf("StepStart output missing step name: %q", out)
	}
}

func TestStepDone_ContainsModel(t *testing.T) {
	var buf bytes.Buffer
	d := newTestDisplay(&buf)
	d.StepDone("review", "openai/gpt-4o", "REVIEW.md", 0.0012, 3*time.Second, "")
	out := buf.String()
	if !strings.Contains(out, "openai/gpt-4o") {
		t.Errorf("StepDone output missing model: %q", out)
	}
	if !strings.Contains(out, "$0.0012") {
		t.Errorf("StepDone output missing cost: %q", out)
	}
}

func TestStepDone_ZeroCostShowsDash(t *testing.T) {
	var buf bytes.Buffer
	d := newTestDisplay(&buf)
	d.StepDone("build", "api", "", 0, time.Second, "")
	out := buf.String()
	if !strings.Contains(out, "—") {
		t.Errorf("StepDone expected dash for zero cost, got: %q", out)
	}
}

func TestStepDone_ArtifactPreview(t *testing.T) {
	var buf bytes.Buffer
	d := newTestDisplay(&buf)
	content := "line1\nline2\nline3\n"
	d.StepDone("plan", "model", "PLAN.md", 0, time.Second, content)
	out := buf.String()
	for _, line := range []string{"line1", "line2", "line3"} {
		if !strings.Contains(out, line) {
			t.Errorf("StepDone preview missing %q: %q", line, out)
		}
	}
	if !strings.Contains(out, "│") {
		t.Errorf("StepDone preview missing │ prefix: %q", out)
	}
}

func TestStepDone_ArtifactPreviewTruncated(t *testing.T) {
	var buf bytes.Buffer
	d := newTestDisplay(&buf)
	// Build 15-line content; only first 10 should appear, then truncation note.
	var lines []string
	for i := 1; i <= 15; i++ {
		lines = append(lines, fmt.Sprintf("line%d", i))
	}
	d.StepDone("plan", "model", "PLAN.md", 0, time.Second, strings.Join(lines, "\n"))
	out := buf.String()
	if !strings.Contains(out, "5 more lines") {
		t.Errorf("StepDone should show truncation note, got: %q", out)
	}
	if strings.Contains(out, "line15") {
		t.Errorf("StepDone should not show line15: %q", out)
	}
}

func TestStepFailed_ContainsModel(t *testing.T) {
	var buf bytes.Buffer
	d := newTestDisplay(&buf)
	d.StepFailed("edit", "anthropic/claude-sonnet-4-6", errors.New("timed out"))
	out := buf.String()
	if !strings.Contains(out, "anthropic/claude-sonnet-4-6") {
		t.Errorf("StepFailed output missing model: %q", out)
	}
	if !strings.Contains(out, "timed out") {
		t.Errorf("StepFailed output missing error: %q", out)
	}
}

func TestTruncateModel_ShortName(t *testing.T) {
	got := truncateModel("anthropic/claude-3")
	if got != "anthropic/claude-3" {
		t.Errorf("expected no truncation, got %q", got)
	}
}

func TestTruncateModel_LongName(t *testing.T) {
	long := "some-provider/some-very-long-model-name-v1.2.3-beta"
	got := truncateModel(long)
	if len([]rune(got)) > modelColumnWidth {
		t.Errorf("truncateModel did not truncate: len=%d, got %q", len([]rune(got)), got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("truncated model should end with ellipsis, got %q", got)
	}
}

func TestTruncateModel_ExactWidth(t *testing.T) {
	// A model name exactly at modelColumnWidth should not be truncated.
	exact := strings.Repeat("a", modelColumnWidth)
	got := truncateModel(exact)
	if got != exact {
		t.Errorf("exact-width model should not be truncated, got %q", got)
	}
}

func TestSanitizeModel_StripsANSI(t *testing.T) {
	input := "\x1b[31mmalicious\x1b[0m"
	got := sanitizeModel(input)
	if strings.Contains(got, "\x1b") {
		t.Errorf("sanitizeModel did not strip ANSI: %q", got)
	}
	if got != "malicious" {
		t.Errorf("expected 'malicious', got %q", got)
	}
}

func TestSanitizeModel_StripsControlChars(t *testing.T) {
	input := "model\x00name\x1f"
	got := sanitizeModel(input)
	if strings.Contains(got, "\x00") || strings.Contains(got, "\x1f") {
		t.Errorf("sanitizeModel did not strip control chars: %q", got)
	}
}

func TestTruncateModel_Unicode(t *testing.T) {
	// Unicode model name with multi-byte runes (CJK characters).
	cjk := strings.Repeat("模", 35) // 35 CJK chars, should be truncated
	got := truncateModel(cjk)
	if len([]rune(got)) > modelColumnWidth {
		t.Errorf("unicode truncation failed: len=%d", len([]rune(got)))
	}
}
