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
}
