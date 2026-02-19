# CLAUDE.md — vCoding Project Instructions

## Project Overview

vCoding is a multi-model issue-to-PR pipeline CLI written in Go. It orchestrates different AI models (Planner, Reviewer, Editor, Auditor) via markdown files as the communication protocol, then delegates implementation to Claude Code and creates a PR automatically.

## Tech Stack

- **Language**: Go 1.23+
- **CLI Framework**: cobra
- **HTTP**: net/http (OpenRouter, OpenAI-compatible API)
- **GitHub**: gh CLI (shelling out)
- **Config**: YAML (gopkg.in/yaml.v3)
- **Embedding**: Go embed for prompts and default pipelines

## Project Structure

```
cmd/vcoding/main.go          # Entry point
internal/
  config/config.go            # Config loading (~/.vcoding/config.yaml)
  pipeline/
    pipeline.go               # Pipeline YAML parsing
    engine.go                 # Step orchestrator
    context.go                # File-based context between steps
    display.go                # Terminal progress display
  executor/
    executor.go               # Executor interface
    api.go                    # OpenRouter API executor
    claudecode.go             # Claude Code CLI executor
    shell.go                  # Shell command executor
  source/
    source.go                 # Input source interface
    github.go                 # GitHub issue source (pick)
    spec.go                   # Spec file source (do)
  github/
    issue.go                  # Fetch issues via gh CLI
    pr.go                     # Create PRs via gh CLI
  project/
    scanner.go                # Project file scanner for context
    git.go                    # Git operations (diff, branch info)
  run/
    run.go                    # Run directory management, meta.json
  cost/
    tracker.go                # Cost tracking from API responses
prompts/                      # Embedded prompt templates (English)
pipelines/                    # Embedded default pipeline YAMLs
```

## Build & Run

```bash
go build ./cmd/vcoding
go test ./...
```

## Conventions

- **Language**: Code, comments, and variable names in English. User-facing strings in English.
- **Error handling**: Return errors, don't panic. Wrap errors with `fmt.Errorf("context: %w", err)`.
- **Packages**: Keep packages small and focused. Use interfaces at package boundaries.
- **Testing**: Table-driven tests. Test files next to source (`foo_test.go`).
- **No external frameworks** for HTTP or DI. Keep dependencies minimal.
- **Prompts and pipeline YAMLs** are embedded via `//go:embed` and can be overridden by user files in `~/.vcoding/`.

## Key Design Decisions

1. **File-as-Protocol**: All inter-model communication happens through markdown files in `.vcoding/runs/<run-id>/`. No shared memory, no message passing.
2. **Single API Gateway**: All models are called through OpenRouter's OpenAI-compatible endpoint. One HTTP client implementation serves all models.
3. **No loops**: The pipeline is a fixed linear sequence (Audit 1x -> Fix 1x -> PR). No retry loops for predictable cost and time.
4. **Executor abstraction**: vCoding orchestrates but does not implement agentic coding loops. Claude Code handles implementation via `claude -p`.
5. **Run isolation**: Each pipeline execution gets its own timestamped directory. Inputs are immutable snapshots.

## Common Patterns

### Adding a new executor

1. Implement `executor.Executor` interface in `internal/executor/`
2. Register in executor factory (engine.go)

### Adding a new pipeline step

1. Add step definition to pipeline YAML
2. If new executor type needed, implement it
3. If new prompt template needed, add to `prompts/`

### Config resolution order

Project `.vcoding/config.yaml` > User `~/.vcoding/config.yaml` > Built-in defaults

## Environment Variables

- `OPENROUTER_API_KEY` — Required for API executor
- `GITHUB_TOKEN` — Required for GitHub operations (or gh CLI auth)
