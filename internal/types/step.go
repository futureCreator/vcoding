// Package types holds shared data structures used across packages.
package types

// Step is a single unit of work in a pipeline.
type Step struct {
	Name           string   `yaml:"name"`
	Executor       string   `yaml:"executor"`
	Model          string   `yaml:"model,omitempty"`
	PromptTemplate string   `yaml:"prompt_template,omitempty"`
	Input          []string `yaml:"input"`
	Output         string   `yaml:"output,omitempty"`
	Command        string   `yaml:"command,omitempty"`
	Type           string   `yaml:"type,omitempty"`
	TitleFrom      string   `yaml:"title_from,omitempty"`
	BodyTemplate   string   `yaml:"body_template,omitempty"`
}
