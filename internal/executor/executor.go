package executor

import (
	"context"
	"time"

	"github.com/epmk/vcoding/internal/types"
)

// Executor runs a single pipeline step.
type Executor interface {
	Execute(ctx context.Context, req *Request) (*Result, error)
}

// Request carries all inputs for a step execution.
type Request struct {
	Step       types.Step
	RunDir     string
	InputFiles map[string]string // filename → content
	Verbose    bool              // stream executor output to terminal
}

// Result holds the output of a step execution.
type Result struct {
	Output    string
	Cost      float64
	Duration  time.Duration
	TokensIn  int
	TokensOut int
}
