# vCoding

Multi-model issue-to-PR pipeline CLI that orchestrates different AI models to take an issue or spec from input to pull request automatically.

## Overview

vCoding automates the software development workflow by delegating different tasks to specialized AI models:

- **Planner** (Claude Opus): Creates detailed implementation plans from issues/specs
- **Reviewer** (Kimi): Reviews plans for completeness and correctness
- **Editor** (Claude Sonnet): Revises plans based on review feedback
- **Auditor** (Codex): Reviews code changes after implementation
- **Claude Code**: Handles actual code implementation

## Features

- **Multi-model orchestration**: Leverages strengths of different AI models for specific tasks
- **File-as-protocol**: All inter-model communication happens through markdown files
- **Pipeline-based**: Configurable YAML pipelines define the workflow
- **Run isolation**: Each execution gets its own timestamped directory with immutable snapshots
- **Cost tracking**: Monitors API usage and costs across runs
- **GitHub integration**: Automatically creates pull requests
- **No loops**: Fixed linear sequence for predictable cost and time

## Installation

### Prerequisites

- Go 1.22+
- `gh` CLI (GitHub CLI) - for PR creation and issue fetching
- `claude` CLI (Claude Code) - for implementation
- OpenRouter API key - for accessing AI models

### Build from source

```bash
git clone https://github.com/futureCreator/vcoding.git
cd vcoding
go build ./cmd/vcoding
```

Optionally, move the binary to your PATH:

```bash
mv vcoding /usr/local/bin/
```

## Quick Start

### 1. Initialize configuration

```bash
vcoding init
```

This creates a `.vcoding/config.yaml` file with default settings.

### 2. Set up your OpenRouter API key

```bash
export OPENROUTER_API_KEY="your-api-key-here"
```

Or add to your shell profile for persistence.

### 3. Verify prerequisites

```bash
vcoding doctor
```

### 4. Run a pipeline

**From a GitHub issue:**
```bash
vcoding pick 123
```

**From a spec file:**
```bash
vcoding do specs/feature-xyz.md
```

## Commands

| Command | Description |
|---------|-------------|
| `vcoding init` | Initialize vCoding configuration |
| `vcoding pick <issue>` | Run pipeline on a GitHub issue |
| `vcoding do <spec-file>` | Run pipeline on a local spec file |
| `vcoding stats` | Show cost and run statistics |
| `vcoding doctor` | Check prerequisites and configuration |
| `vcoding version` | Print version information |

### Command Options

**pick** - Run pipeline on GitHub issue
```bash
vcoding pick <issue-number> [flags]
  -p, --pipeline string   Pipeline to use (default "default")
      --force             Skip dirty working tree check
```

**do** - Run pipeline on spec file
```bash
vcoding do <spec-file> [flags]
  -p, --pipeline string   Pipeline to use (default "default")
      --force             Skip dirty working tree check
```

## Configuration

Configuration is loaded in the following priority order:
1. Project `.vcoding/config.yaml`
2. User `~/.vcoding/config.yaml`
3. Built-in defaults

### Example configuration

```yaml
default_pipeline: default

provider:
  endpoint: https://openrouter.ai/api/v1
  api_key_env: OPENROUTER_API_KEY

roles:
  planner: anthropic/claude-opus-4-6
  reviewer: moonshotai/kimi-k2.5
  editor: anthropic/claude-sonnet-4-6
  auditor: openai/codex-5.3

github:
  default_repo: owner/repo
  base_branch: main

executors:
  claude-code:
    command: claude
    args: ["-p", "--output-format", "json"]
    timeout: 300s

language:
  artifacts: en
  normalize_ticket: true

project_context:
  max_files: 20
  max_file_size: 50KB
  include_patterns:
    - "*.go"
    - "*.rs"
    - "*.ts"
    - "*.py"
    - "*.md"
  exclude_patterns:
    - "vendor/"
    - "node_modules/"
    - ".git/"
    - ".vcoding/"

max_context_tokens: 80000
log_level: info
```

## Pipelines

Pipelines define the sequence of steps executed during a run. Two built-in pipelines are included:

### default
The full workflow with review and audit cycles:
1. **Plan** - Create implementation plan
2. **Review** - Review the plan
3. **Revise** - Revise based on review
4. **Implement** - Claude Code implements the plan
5. **Test** - Run tests
6. **Audit** - Review code changes
7. **Fix** - Apply fixes from audit
8. **PR** - Create pull request

### quick
A streamlined workflow for faster turnaround:
1. **Plan** - Create implementation plan
2. **Implement** - Claude Code implements the plan
3. **PR** - Create pull request

### Custom pipelines

You can create custom pipeline YAML files in `~/.vcoding/pipelines/`:

```yaml
name: custom

steps:
  - name: Plan
    executor: api
    model: anthropic/claude-sonnet-4-6
    prompt_template: plan
    input: [TICKET.md]
    output: PLAN.md

  - name: Implement
    executor: claude-code
    input: [PLAN.md]

  - name: PR
    type: github-pr
    title_from: TICKET.md
    body_template: pr-summary
```

### Executors

- **api** - Call AI models via OpenRouter API
- **claude-code** - Execute Claude Code CLI for implementation
- **shell** - Run shell commands
- **github-pr** - Create GitHub pull requests

## Project Structure

```
.vcoding/
├── config.yaml          # Project configuration
├── pipelines/           # Custom pipeline definitions
└── runs/               # Run directories (timestamped)
    ├── 20240219120000-feature-x/
    │   ├── meta.json       # Run metadata
    │   ├── TICKET.md       # Input issue/spec
    │   ├── PLAN.md         # Generated plan
    │   ├── REVIEW.md       # Review output
    │   ├── TEST.md         # Test results
    │   └── REVIEW-CODE.md  # Code audit
    └── ...
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `OPENROUTER_API_KEY` | Required for API executor |
| `GH_TOKEN` | GitHub token for CI environments (bypasses `gh auth login`) |
| `GITHUB_TOKEN` | Alternative to `GH_TOKEN`; `GH_TOKEN` takes precedence |

## CI Usage

In CI environments, authenticate with GitHub by setting `GH_TOKEN` instead of running `gh auth login`:

```yaml
# GitHub Actions — gh is pre-installed on ubuntu-latest/macos-latest/windows-latest
env:
  GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  OPENROUTER_API_KEY: ${{ secrets.OPENROUTER_API_KEY }}
```

For other CI systems (GitLab CI, CircleCI, etc.), install the `gh` CLI and set `GH_TOKEN` in the job environment. In Docker/container environments, set `GH_TOKEN` via environment variable rather than mounting credential files.

## Cost Tracking

vCoding tracks API costs for each run. View statistics with:

```bash
vcoding stats
```

## Development

### Running tests

```bash
go test ./...
```

### Building

```bash
go build ./cmd/vcoding
```

## Architecture

### Design Decisions

1. **File-as-Protocol**: All inter-model communication happens through markdown files in run directories. No shared memory, no message passing.

2. **Single API Gateway**: All models are called through OpenRouter's OpenAI-compatible endpoint.

3. **No Loops**: The pipeline is a fixed linear sequence for predictable cost and time.

4. **Executor Abstraction**: vCoding orchestrates but does not implement agentic coding loops. Claude Code handles implementation.

5. **Run Isolation**: Each pipeline execution gets its own timestamped directory with immutable snapshots.

### Key Components

- **Pipeline** - YAML-defined workflow with steps
- **Engine** - Step orchestrator that executes pipelines
- **Executor** - Interface for different execution types (API, Claude Code, shell)
- **Source** - Input abstraction (GitHub issues, spec files)
- **Context** - File-based context management between steps

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
