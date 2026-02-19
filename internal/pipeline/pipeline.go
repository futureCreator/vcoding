package pipeline

import (
	"fmt"
	"os"

	"github.com/epmk/vcoding/internal/types"
	"gopkg.in/yaml.v3"
)

// Pipeline represents a named sequence of steps.
type Pipeline struct {
	Name  string       `yaml:"name"`
	Steps []types.Step `yaml:"steps"`
}

// Parse decodes a pipeline from YAML bytes.
func Parse(data []byte) (*Pipeline, error) {
	var p Pipeline
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing pipeline: %w", err)
	}
	if p.Name == "" {
		return nil, fmt.Errorf("pipeline must have a name")
	}
	return &p, nil
}

// ParseFile reads and parses a pipeline YAML file.
func ParseFile(path string) (*Pipeline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading pipeline file %s: %w", path, err)
	}
	return Parse(data)
}
