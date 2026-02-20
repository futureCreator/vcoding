# vCoding

Multi-model planning pipeline CLI that orchestrates different AI models to generate reviewed implementation plans from issues or specs.

## Overview

vCoding automates the planning workflow by delegating different tasks to specialized AI models:

- **Planner** (Claude Opus): Creates detailed implementation plans from issues/specs
- **Reviewer** (DeepSeek R1): Reviews plans for completeness and correctness
- **Editor** (GLM-5): Revises plans based on review feedback

The final output is a reviewed `PLAN.md` file ready for implementation.

## Features

- **Multi-model orchestration**: Leverages strengths of different AI models for specific tasks
- **File-as-protocol**: All inter-model communication happens through markdown files
- **Pipeline-based**: Configurable YAML pipelines define the workflow
- **Run isolation**: Each execution gets its own timestamped directory with immutable snapshots
- **Cost tracking**: Monitors API usage and costs across runs
- **No loops**: Fixed linear sequence for predictable cost and time
- **AI agent ready**: Generated instruction files enable autonomous execution by Claude Code, Cursor, and other AI assistants

## Installation

### Prerequisites

- Go 1.22+
- `gh` CLI (GitHub CLI) - for issue fetching (optional)
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

This creates:
- `~/.vcoding/config.yaml` - Global configuration with default settings
- `SKILL.md` - ClawHub-compatible AgentSkill definition for vcoding

The `SKILL.md` file enables AI agents to understand and run vcoding pipelines autonomously without manual shell execution.

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

The output will be a reviewed `PLAN.md` file in `.vcoding/runs/latest/`.

## Commands

| Command | Description |
|---------|-------------|
| `vcoding init` | Initialize vCoding configuration and agent instruction files |
| `vcoding pick <issue>` | Run pipeline on a GitHub issue |
| `vcoding do <spec-file>` | Run pipeline on a local spec file |
| `vcoding stats` | Show cost and run statistics |
| `vcoding doctor` | Check prerequisites and configuration |
| `vcoding migrate-config` | Remove deprecated GitHub token fields from config files |
| `vcoding version` | Print version information |

### Command Options

**pick** - Run pipeline on GitHub issue
```bash
vcoding pick <issue-number> [flags]
  -p, --pipeline string   Pipeline to use (default "default")
  -v, --verbose           Stream executor output to terminal
```

**do** - Run pipeline on spec file
```bash
vcoding do <spec-file> [flags]
  -p, --pipeline string   Pipeline to use (default "default")
  -v, --verbose           Stream executor output to terminal
```

## Configuration

Configuration is loaded in the following priority order:
1. Project `.vcoding/config.yaml`
2. User `~/.vcoding/config.yaml`
3. Built-in defaults

### Example configuration

```yaml
provider:
  endpoint: https://openrouter.ai/api/v1
  api_key_env: OPENROUTER_API_KEY

roles:
  planner: anthropic/claude-opus-4-6
  reviewer: deepseek/deepseek-r1
  editor: z-ai/glm-5

github:
  default_repo: owner/repo
  base_branch: main

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

Pipelines define the sequence of steps executed during a run.

### default
The built-in planning workflow with review cycle:
1. **Plan** - Create implementation plan from ticket and project context
2. **Review** - Review the plan
3. **Revise** - Revise based on review

Models are referenced by role (`$planner`, `$reviewer`, `$editor`) and resolved from config at runtime.

### Custom pipelines

You can create custom pipeline YAML files in `~/.vcoding/pipelines/` or `.vcoding/pipelines/`:

```yaml
name: custom

steps:
  - name: Plan
    executor: api
    model: $planner
    prompt_template: plan
    input: [TICKET.md, project:context]
    output: PLAN.md

  - name: Review
    executor: api
    model: $reviewer
    prompt_template: review
    input: [PLAN.md]
    output: REVIEW.md

  - name: Revise
    executor: api
    model: $editor
    prompt_template: revise
    input: [PLAN.md, REVIEW.md]
    output: PLAN.md
```

### Executors

- **api** - Call AI models via OpenRouter API
- **shell** - Run shell commands

## Project Structure

```
.vcoding/
├── config.yaml          # Project configuration
├── pipelines/           # Custom pipeline definitions
└── runs/               # Run directories (timestamped)
    ├── 20240219120000-feature-x/
    │   ├── meta.json       # Run metadata
    │   ├── TICKET.md       # Input issue/spec
    │   ├── PLAN.md         # Generated plan (final output)
    │   └── REVIEW.md       # Review output
    ├── latest -> 20240219120000-feature-x/  # symlink to most recent run
    └── ...

# Agent instruction file (created by vcoding init)
SKILL.md                 # ClawHub-compatible AgentSkill definition
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `OPENROUTER_API_KEY` | Required for API executor |
| `GH_TOKEN` | GitHub token for fetching issues via `gh` CLI |
| `GITHUB_TOKEN` | Alternative to `GH_TOKEN`; `GH_TOKEN` takes precedence |

## CI Usage

In CI environments, set `GH_TOKEN` instead of running `gh auth login`:

```yaml
env:
  GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  OPENROUTER_API_KEY: ${{ secrets.OPENROUTER_API_KEY }}
```

## Cost Tracking

vCoding tracks API costs for each run. View statistics with:

```bash
vcoding stats
```

## AI Agent Integration

After running `vcoding init`, the generated `SKILL.md` file enables AI agents to autonomously execute vcoding pipelines. It follows the ClawHub AgentSkill format, allowing agents to discover and run `vcoding pick` or `vcoding do` without manual shell execution.

### ClawHub Publishing

`SKILL.md` can be published to the ClawHub skill registry:

1. Ensure `SKILL.md` is committed to your repository
2. Submit the repository URL to the ClawHub registry

Other agents can then discover and use vcoding by reading the `SKILL.md` definition.

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

4. **Run Isolation**: Each pipeline execution gets its own timestamped directory with immutable snapshots.

### Key Components

- **Pipeline** - YAML-defined workflow with steps
- **Engine** - Step orchestrator that executes pipelines
- **Executor** - Interface for different execution types (API, shell)
- **Source** - Input abstraction (GitHub issues, spec files)
- **Context** - File-based context management between steps

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
