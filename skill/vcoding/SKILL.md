---
name: "vcoding"
description: "Multi-model planning pipeline that generates reviewed implementation plans from GitHub issues or spec files. Use when you need to create implementation plans from GitHub issues or specification files, or when the user mentions planning, code planning, or implementation planning."
license: "MIT"
compatibility: "Requires vcoding CLI, git, gh CLI >= 2.0.0 (authenticated), and OPENROUTER_API_KEY environment variable"
metadata:
  author: "futureCreator"
  version: "latest"
  repository: "https://github.com/futureCreator/vcoding"
  install: "curl -fsSL https://raw.githubusercontent.com/futureCreator/vcoding/main/install.sh | bash"
---

# vcoding AgentSkill

Multi-model planning pipeline CLI that orchestrates AI models to generate reviewed implementation plans from GitHub issues or spec files.

## Quick Start

### 1. Install (if not already installed)

```bash
vcoding version
# If "command not found":
curl -fsSL https://raw.githubusercontent.com/futureCreator/vcoding/main/install.sh | bash
# Restart shell or run: source ~/.bashrc (or ~/.zshrc)
```

### 2. Setup (first time only)

```bash
vcoding init                          # Create config
export OPENROUTER_API_KEY="..."       # Get from https://openrouter.ai/keys
gh auth login                         # If not already authenticated
```

### 3. Verify

```bash
vcoding doctor                        # Should print "All checks passed"
```

### 4. Use

```bash
vcoding pick 42                       # Plan from GitHub issue
vcoding do SPEC.md                    # Plan from spec file
```

## Overview

vCoding automates the planning workflow by delegating different tasks to specialized AI models in a fixed 3-step pipeline:

1. **Planner** (`z-ai/glm-5`) — reads ticket + project context + git diff, writes `PLAN.md`
2. **Reviewer** (`deepseek/deepseek-v3.2`) — reads `PLAN.md`, writes `REVIEW.md`
3. **Editor** (`moonshotai/kimi-k2.5`) — reads `PLAN.md` + `REVIEW.md`, overwrites `PLAN.md`

The final output is a reviewed `PLAN.md` file in `.vcoding/runs/latest/` ready for implementation.

All inter-model communication happens through markdown files (file-as-protocol). No shared memory, no message passing. Models are configurable via `.vcoding/config.yaml` or `~/.vcoding/config.yaml`.

## Prerequisites

Run `vcoding doctor` to verify. If any check fails:

| Check | Verify Command | If Failing |
|-------|---------------|------------|
| vcoding CLI | `vcoding version` | Run install command from metadata.install |
| git repository | `git rev-parse --is-inside-work-tree` | Run `git init` or cd to git repo |
| gh CLI >= 2.0 | `gh version` | Install from https://cli.github.com/ |
| GitHub auth | `gh auth status` | Run `gh auth login` |
| OpenRouter API key | `test -n "$OPENROUTER_API_KEY"` | Get key from https://openrouter.ai/keys |
| Clean working dir | `git status --porcelain` | Commit or stash changes |

## Commands

### `vcoding init`

Initialize vCoding global configuration file.

```bash
vcoding init [--minimal]
```

- `--minimal` — Generate config without inline comments
- Creates: `~/.vcoding/config.yaml`
- Exit: 0 = success, 1 = error

---

### `vcoding pick <issue-number>`

Run pipeline on a GitHub issue.

```bash
vcoding pick 42
vcoding pick 42 --verbose
```

- `-p, --pipeline string` — Pipeline to use (default: "default")
- `-v, --verbose` — Show full model output
- Exit: 0 = success, 1 = error
- Output: `.vcoding/runs/latest/PLAN.md`

---

### `vcoding do <spec-file>`

Run pipeline on a local spec file.

```bash
vcoding do SPEC.md
vcoding do specs/feature.md --verbose
```

- `-p, --pipeline string` — Pipeline to use (default: "default")
- `-v, --verbose` — Show full model output
- Exit: 0 = success, 1 = error
- Output: `.vcoding/runs/latest/PLAN.md`

---

### `vcoding stats`

Show cost and run statistics.

```bash
vcoding stats
```

- Exit: 0 = success, 1 = error

---

### `vcoding doctor`

Check prerequisites and configuration.

```bash
vcoding doctor
```

- Exit: 0 = all checks passed, 1 = one or more checks failed

---

### `vcoding version`

Print version information.

```bash
vcoding version
```

- Exit: 0 = success

## Workflow

```
# 1. Ensure vcoding is installed
EXEC('vcoding version')
IF exit_code != 0:
   EXEC('curl -fsSL https://raw.githubusercontent.com/futureCreator/vcoding/main/install.sh | bash')
   RESTART_SHELL_OR_SOURCE_PROFILE()
   EXEC('vcoding version')  # Verify
   IF still fails:
     Report installation failure and halt

# 2. Verify environment
EXEC('vcoding doctor')
IF exit_code != 0:
   Read error output
   Fix specific issue (see Prerequisites table)
   Retry

# 3. Generate plan
RUN 'vcoding pick <issue>' OR 'vcoding do <spec>'

# 4. Read plan
PLAN = READ('.vcoding/runs/latest/PLAN.md')

# 5. Implement
Follow PLAN.md steps

# 6. Handle failures
IF tests fail:
   READ '.vcoding/runs/latest/REVIEW.md' for insights
   Fix manually OR re-run with updated spec
```

### Failure Recovery Scenarios

- **`vcoding version` fails (CLI not installed)**: Run install script from frontmatter's `install` field; if still fails, report manual installation needed
- **`vcoding doctor` fails**: Fix missing prerequisites (API keys, gh auth, etc.) before proceeding
- **`vcoding pick/do` times out**: CLI has no built-in timeout; agent should implement own timeout (e.g., 5 minutes) and retry with exponential backoff
- **PLAN.md missing after `do`**: Check `.vcoding/runs/latest/` for partial artifacts; review `TICKET.md` and `meta.json` for error details
- **Implementation fails tests**: Read `REVIEW.md` for insights; consider re-running with refined spec
- **API key exhausted**: Check `meta.json` for token usage; wait or use different API key

### Security Considerations

- **Credential Management**: `OPENROUTER_API_KEY` must be set in the agent's environment. Never log or expose this key.
- **GitHub Token**: If `GH_TOKEN` is used instead of `gh auth`, ensure it has appropriate repo scope.
- **File Permissions**: All generated files use `0644` permissions. Sensitive data should not be written to `.vcoding/` directory.

## Output Files

Each run creates `.vcoding/runs/<run-id>/`:

| File | Description |
|------|-------------|
| `TICKET.md` | Input issue/spec content |
| `PLAN.md` | Generated implementation plan (final output) |
| `REVIEW.md` | Review output (default pipeline only) |
| `meta.json` | Run metadata (status, cost, tokens, timestamps) |

`.vcoding/runs/latest/` is a symlink to the most recent run.

### meta.json Schema

```json
{
  "started_at": "2024-01-01T00:00:00Z",
  "input_mode": "pick",
  "input_ref": "42",
  "status": "completed",
  "steps": [...],
  "total_cost": 0.035,
  "git_branch": "main",
  "git_commit": "abc1234"
}
```

## Error Handling

| Exit Code | Meaning |
|-----------|---------|
| 0 | Success |
| 1 | Error (config invalid, API error, file not found, etc.) |

### Common Issues

| Symptom | Cause | Fix |
|---------|-------|-----|
| `vcoding: command not found` | Not installed | Run install command |
| `gh: command not found` | gh CLI missing | Install gh CLI |
| `invalid config` | Missing config | Run `vcoding init` |
| `OPENROUTER_API_KEY missing` | Env var not set | `export OPENROUTER_API_KEY=...` |
| `gh auth` error | Not authenticated | Run `gh auth login` |
| Empty PLAN.md | API error | Check API key; retry |

## Security

- Never log or expose `OPENROUTER_API_KEY`
- `GH_TOKEN` must have appropriate repo scope
- Generated files use `0644` permissions
