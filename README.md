# vCoding

A multi-model issue-to-PR pipeline CLI. vCoding orchestrates different AI models—Planner, Reviewer, Editor, Auditor—to take a GitHub issue or spec file all the way to a merged pull request, automatically.

## How it works

Each pipeline run creates a timestamped directory (`.vcoding/runs/<run-id>/`) where every model communicates through markdown files. No shared memory, no hidden state—just transparent, inspectable artifacts at every step.

**Default pipeline:**

```
Plan → Review → Revise → Implement → Test → Audit → Fix → PR
```

1. **Plan** — Planner model reads the issue and produces `PLAN.md`
2. **Review** — Reviewer model critiques the plan and produces `REVIEW.md`
3. **Revise** — Editor model incorporates feedback into a revised `PLAN.md`
4. **Implement** — Claude Code executes the plan against your codebase
5. **Test** — Runs your test suite and captures results
6. **Audit** — Auditor model reviews the actual git diff against the plan
7. **Fix** — Claude Code applies any fixes from the audit
8. **PR** — Creates a GitHub pull request automatically

## Requirements

- Go 1.23+
- [`gh` CLI](https://cli.github.com/) (authenticated)
- [`claude` CLI](https://claude.ai/download) (for the Implement and Fix steps)
- An [OpenRouter](https://openrouter.ai/) API key

## Installation

```bash
go install github.com/your-org/vcoding/cmd/vcoding@latest
```

Or build from source:

```bash
git clone https://github.com/your-org/vcoding
cd vcoding
go build ./cmd/vcoding
```

## Setup

```bash
# Create ~/.vcoding/config.yaml with defaults
vcoding init

# Set your OpenRouter API key
export OPENROUTER_API_KEY="sk-or-..."
```

## Usage

### Pick a GitHub issue

```bash
vcoding pick 42
```

Fetches issue #42, runs the full pipeline, and opens a PR.

### Run against a local spec file

```bash
vcoding do SPEC.md
```

Use a markdown file as the input instead of a GitHub issue.

### Use the quick pipeline (no review or audit)

```bash
vcoding pick 42 --pipeline quick
vcoding do SPEC.md --pipeline quick
```

The `quick` pipeline skips the Review, Revise, Audit, and Fix steps:

```
Plan → Implement → PR
```

### Check run history and costs

```bash
vcoding stats
```

```
Runs: 5 total, 4 completed, 1 failed
Total cost: $1.84
Average cost: $0.37

Run ID                                    Status     Cost        Mode
────────────────────────────────────────────────────────────────────
20260219-151430-042-add-user-auth        completed  $0.73       pick
20260219-140522-001-refactor-db-layer    completed  $0.45       do
...
```

### Flags

| Flag | Commands | Description |
|------|----------|-------------|
| `--pipeline`, `-p` | `pick`, `do` | Pipeline name or path (default: `default`) |
| `--force` | `pick`, `do` | Skip dirty working tree check |

## Configuration

`~/.vcoding/config.yaml` (user-level) or `.vcoding/config.yaml` (project-level):

```yaml
default_pipeline: default

provider:
  endpoint: https://openrouter.ai/api/v1
  api_key_env: OPENROUTER_API_KEY

roles:
  planner:  anthropic/claude-opus-4-6
  reviewer: moonshotai/kimi-k2.5
  editor:   anthropic/claude-sonnet-4-6
  auditor:  openai/codex-5.3

github:
  base_branch: main

executors:
  claude-code:
    command: claude
    timeout: 300s

project_context:
  max_files: 20
  max_file_size: 50KB
  include_patterns: ["*.go", "*.rs", "*.ts", "*.py", "*.md"]
  exclude_patterns: ["vendor/", "node_modules/", ".git/", ".vcoding/"]

max_context_tokens: 80000
```

**Resolution order:** project config > user config > built-in defaults

## Pipelines

Pipelines are YAML files that define the sequence of steps. You can create custom pipelines:

```yaml
# .vcoding/pipelines/my-pipeline.yaml
name: my-pipeline

steps:
  - name: Plan
    executor: api
    model: anthropic/claude-opus-4-6
    prompt_template: plan
    input: [TICKET.md]
    output: PLAN.md

  - name: Implement
    executor: claude-code
    input: [PLAN.md]

  - name: Test
    executor: shell
    command: "go test ./..."
    output: TEST.md

  - name: PR
    executor: github-pr
    title_from: TICKET.md
    body_template: pr-summary
```

**Executor types:**

| Type | Description |
|------|-------------|
| `api` | Calls an LLM via OpenRouter (requires `model` and `prompt_template`) |
| `claude-code` | Delegates to the Claude Code CLI |
| `shell` | Runs an arbitrary shell command |
| `github-pr` | Creates a GitHub pull request |

**Input sources:**

- File names relative to the run directory: `[TICKET.md, PLAN.md]`
- Live git diff: `["git:diff"]`

Custom pipelines go in `.vcoding/pipelines/` (project) or `~/.vcoding/pipelines/` (user) and take precedence over the built-in ones.

## Prompt templates

Built-in templates (embedded in the binary):

| Template | Purpose |
|----------|---------|
| `plan` | Reads the ticket and produces a detailed implementation plan |
| `review` | Critically reviews the plan for gaps and issues |
| `revise` | Incorporates review feedback into a revised plan |
| `code-review` | Audits actual code changes against the plan |
| `fix` | Applies fixes from the code review |
| `pr-summary` | Generates a concise PR description |

Override any template by placing a file at `~/.vcoding/prompts/<name>.md` or `.vcoding/prompts/<name>.md`.

## Run artifacts

Each run produces:

```
.vcoding/runs/20260219-151430-042-add-user-auth/
├── meta.json         # Status, cost, per-step metrics
├── TICKET.md         # Normalized input (immutable)
├── PLAN.md           # Implementation plan
├── REVIEW.md         # Plan review
├── REVIEW-CODE.md    # Code audit
├── TEST.md           # Test output
└── PR.md             # PR description
.vcoding/runs/latest  # Symlink to most recent run
```

`meta.json` records status, total cost, git branch, and per-step token usage and duration.

## Environment variables

| Variable | Description |
|----------|-------------|
| `OPENROUTER_API_KEY` | Required for all API executor steps |

## Development

```bash
go build ./cmd/vcoding
go test ./...
```

See [CLAUDE.md](CLAUDE.md) for architecture notes and contribution guidelines.
