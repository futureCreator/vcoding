package run

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Run represents a single pipeline execution.
type Run struct {
	ID   string
	Dir  string
	Meta Meta
}

// Meta holds metadata about a run, persisted to meta.json.
type Meta struct {
	StartedAt time.Time    `json:"started_at"`
	InputMode string       `json:"input_mode"` // "pick" | "do"
	InputRef  string       `json:"input_ref"`  // issue number or spec path
	Status    string       `json:"status"`     // "running" | "completed" | "failed"
	Steps     []StepResult `json:"steps"`
	TotalCost float64      `json:"total_cost"`
	Error     string       `json:"error,omitempty"`
	GitBranch string       `json:"git_branch"`
	GitCommit string       `json:"git_commit"`
}

// StepResult records the outcome of a single step.
type StepResult struct {
	Name       string  `json:"name"`
	Status     string  `json:"status"` // "completed" | "failed" | "skipped"
	Cost       float64 `json:"cost"`
	TokensIn   int     `json:"tokens_in"`
	TokensOut  int     `json:"tokens_out"`
	DurationMS int64   `json:"duration_ms"`
	Error      string  `json:"error,omitempty"`
}

// New creates a new run directory under .vcoding/runs/.
func New(mode, ref, slug, gitBranch, gitCommit string) (*Run, error) {
	now := time.Now()
	ms := now.UnixMilli() % 1000
	id := fmt.Sprintf("%s-%03d-%s",
		now.Format("20060102-150405"),
		ms,
		sanitizeSlug(slug),
	)

	baseDir := filepath.Join(".vcoding", "runs")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("creating runs dir: %w", err)
	}

	dir := filepath.Join(baseDir, id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating run dir: %w", err)
	}

	r := &Run{
		ID:  id,
		Dir: dir,
		Meta: Meta{
			StartedAt: now,
			InputMode: mode,
			InputRef:  ref,
			Status:    "running",
			GitBranch: gitBranch,
			GitCommit: gitCommit,
		},
	}

	if err := r.SaveMeta(); err != nil {
		return nil, err
	}

	if err := updateLatestLink(baseDir, id); err != nil {
		return nil, err
	}

	return r, nil
}

// SaveMeta writes meta.json to the run directory.
func (r *Run) SaveMeta() error {
	data, err := json.MarshalIndent(r.Meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling meta: %w", err)
	}
	path := filepath.Join(r.Dir, "meta.json")
	return os.WriteFile(path, data, 0644)
}

// AddStepResult appends a step result and updates total cost.
func (r *Run) AddStepResult(sr StepResult) error {
	r.Meta.Steps = append(r.Meta.Steps, sr)
	r.Meta.TotalCost += sr.Cost
	return r.SaveMeta()
}

// Complete marks the run as completed.
func (r *Run) Complete() error {
	r.Meta.Status = "completed"
	return r.SaveMeta()
}

// Fail marks the run as failed with an error message.
func (r *Run) Fail(msg string) error {
	r.Meta.Status = "failed"
	r.Meta.Error = msg
	return r.SaveMeta()
}

// FilePath returns the absolute path to a file within this run directory.
func (r *Run) FilePath(name string) string {
	return filepath.Join(r.Dir, name)
}

// WriteFile writes content to a named file in the run directory.
func (r *Run) WriteFile(name, content string) error {
	return os.WriteFile(r.FilePath(name), []byte(content), 0644)
}

// ReadFile reads a named file from the run directory.
func (r *Run) ReadFile(name string) (string, error) {
	data, err := os.ReadFile(r.FilePath(name))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// updateLatestLink atomically updates the "latest" symlink.
func updateLatestLink(baseDir, id string) error {
	latestPath := filepath.Join(baseDir, "latest")
	tmpPath := latestPath + ".tmp"

	// Remove any stale tmp link
	os.Remove(tmpPath)

	if err := os.Symlink(id, tmpPath); err != nil {
		return fmt.Errorf("creating temp symlink: %w", err)
	}
	if err := os.Rename(tmpPath, latestPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("updating latest symlink: %w", err)
	}
	return nil
}

var nonAlphanumRe = regexp.MustCompile(`[^a-z0-9]+`)

// sanitizeSlug converts a string to a URL-friendly slug.
func sanitizeSlug(s string) string {
	s = strings.ToLower(s)
	s = nonAlphanumRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 40 {
		s = s[:40]
		s = strings.TrimRight(s, "-")
	}
	if s == "" {
		s = "run"
	}
	return s
}
