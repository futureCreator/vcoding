package source

import "context"

// Input is the normalized input passed to the pipeline.
type Input struct {
	Title string
	Body  string
	Slug  string
	Mode  string // "pick" | "do"
	Ref   string // issue number or file path
}

// Source fetches input from an external source and normalizes it.
type Source interface {
	Fetch(ctx context.Context) (*Input, error)
}
