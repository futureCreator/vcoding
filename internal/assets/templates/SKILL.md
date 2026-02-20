---
name: "vcoding"
description: "Multi-model planning pipeline that generates reviewed implementation plans from GitHub issues or spec files. Use when you need to create implementation plans from GitHub issues or specification files, or when the user mentions planning, code planning, or implementation planning."
license: "MIT"
protocol: "file"
compatibility: "Requires vcoding CLI, git, gh CLI >= 2.0.0 (authenticated), and OPENROUTER_API_KEY environment variable"
metadata:
  author: "futureCreator"
  version: "{{ .Version }}"
  repository: "https://github.com/futureCreator/vcoding"
  install: "curl -fsSL https://raw.githubusercontent.com/futureCreator/vcoding/main/install.sh | bash"
---

# vcoding AgentSkill

Multi-model planning pipeline CLI that orchestrates AI models to generate reviewed implementation plans from GitHub issues or spec files.

## Overview

vCoding automates the planning workflow by delegating different tasks to specialized AI models in a fixed 3-step pipeline:

1. **Planner** (`z-ai/glm-5`) — reads ticket + project context + git diff, writes `PLAN.md`
2. **Reviewer** (`deepseek/deepseek-v3.2`) — reads `PLAN.md`, writes `REVIEW.md`
3. **Editor** (`moonshotai/kimi-k2.5`) — reads `PLAN.md` + `REVIEW.md`, overwrites `PLAN.md`

The final output is a reviewed `PLAN.md` file in `.vcoding/runs/latest/` ready for implementation.

All inter-model communication happens through markdown files (file-as-protocol). No shared memory, no message passing. Models are configurable via `.vcoding/config.yaml` or `~/.vcoding/config.yaml`.

## Prerequisites

### Prerequisites Verification

Before using vCoding, verify all prerequisites:

1. **vcoding CLI on PATH**: `vcoding version` returns successfully
2. **git repository**: `git rev-parse --is-inside-work-tree` returns "true"
3. **gh CLI (>= 2.0.0)**: `gh version | grep "gh version 2\."` matches
4. **GitHub authentication**: `gh auth status` shows authenticated, or `GH_TOKEN` env var is set
5. **OpenRouter API key**: `test -n "$OPENROUTER_API_KEY"` returns 0
6. **Clean working directory** (recommended): `git status --porcelain` returns empty

Run `vcoding doctor` for automated environment verification.

## Commands

### `vcoding init`

Initialize vCoding global configuration file.

**Usage:**
```
vcoding init [--minimal]
```

**Flags:**
- `--minimal` — Generate config without inline comments

**Creates:**
- `~/.vcoding/config.yaml` — Global configuration (with model roles, token limits, etc.)

**Exit codes:** 0 = success, 1 = error

**Example:**
```bash
vcoding init
```

---

### `vcoding pick`

Run pipeline on a GitHub issue.

**Usage:**
```
vcoding pick <issue-number> [flags]
```

**Arguments:**
- `issue-number` — GitHub issue number (required)

**Flags:**
- `-p, --pipeline string` — Pipeline to use (default: "default")
- `-v, --verbose` — Show full model output

**Exit codes:** 0 = success, 1 = error

**Output:** `.vcoding/runs/<run-id>/PLAN.md` (symlinked from `.vcoding/runs/latest/`)

**Example:**
```bash
vcoding pick 42
vcoding pick 42 --verbose
```

---

### `vcoding do`

Run pipeline on a local spec file.

**Usage:**
```
vcoding do <spec-file> [flags]
```

**Arguments:**
- `spec-file` — Path to spec/markdown file (required)

**Flags:**
- `-p, --pipeline string` — Pipeline to use (default: "default")
- `-v, --verbose` — Show full model output

**Exit codes:** 0 = success, 1 = error

**Output:** `.vcoding/runs/<run-id>/PLAN.md` (symlinked from `.vcoding/runs/latest/`)

**Example:**
```bash
vcoding do SPEC.md
vcoding do specs/feature-auth.md --verbose
```

---

### `vcoding stats`

Show cost and run statistics.

**Usage:**
```
vcoding stats
```

**Exit codes:** 0 = success, 1 = error

**Example:**
```bash
vcoding stats
```

---

### `vcoding doctor`

Check prerequisites and configuration.

**Usage:**
```
vcoding doctor
```

**Exit codes:** 0 = all checks passed, 1 = one or more checks failed

**Example:**
```bash
vcoding doctor
```

---

### `vcoding version`

Print version information.

**Usage:**
```
vcoding version
```

**Exit codes:** 0 = success

**Example:**
```bash
vcoding version
```

## Workflow

### Standard Workflow

1. **Verify Environment**
   ```
   EXEC('vcoding doctor')
   IF exit_code != 0:
     Report environment issue and halt
   ```

2. **Generate Plan**
   ```
   EXEC('vcoding pick <issue-number>')
   -- OR --
   EXEC('vcoding do <spec-file>')
   ```

3. **Read Generated Plan**
   ```
   PLAN = READ('.vcoding/runs/latest/PLAN.md')
   IF PLAN is missing:
     Check stderr output from previous command
     Report plan generation failure
   ```

4. **Implement According to Plan**
   - Follow steps in PLAN.md
   - Write code changes
   - Run tests to verify

5. **Handle Implementation Failures**
   ```
   IF tests fail:
     REVIEW = READ('.vcoding/runs/latest/REVIEW.md')
     Analyze review insights
     Option A: Fix issues manually based on review
     Option B: Re-run with modified spec: EXEC('vcoding do <updated-spec>')
   ```

6. **Report Cost (Optional)**
   ```
   EXEC('vcoding stats')
   ```

### Failure Recovery Scenarios

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

vCoding uses a file-as-protocol contract. All outputs are written to the run directory.

### Run Directory

Each invocation creates `.vcoding/runs/<run-id>/` with:

| File | Description |
|------|-------------|
| `TICKET.md` | Input issue/spec content |
| `PLAN.md` | Generated implementation plan (final output) |
| `REVIEW.md` | Review output (default pipeline only) |
| `meta.json` | Run metadata (status, cost, tokens, timestamps) |

### Latest Symlink

`.vcoding/runs/latest/` always points to the most recent run directory. Use this to read outputs without knowing the run ID:

```
.vcoding/runs/latest/PLAN.md    # Latest plan
.vcoding/runs/latest/REVIEW.md  # Latest review
.vcoding/runs/latest/meta.json  # Latest run metadata
```

### File Formats

- `PLAN.md`, `REVIEW.md`, `TICKET.md` — Markdown
- `meta.json` — JSON with fields: `started_at`, `input_mode`, `input_ref`, `status`, `steps`, `total_cost`, `git_branch`, `git_commit`

### `meta.json` Schema

```json
{
  "started_at": "2024-01-01T00:00:00Z",
  "input_mode": "pick",
  "input_ref": "42",
  "status": "completed",
  "steps": [
    {
      "name": "Plan",
      "status": "completed",
      "cost": 0.012,
      "tokens_in": 1500,
      "tokens_out": 800,
      "duration_ms": 4200
    }
  ],
  "total_cost": 0.035,
  "git_branch": "main",
  "git_commit": "abc1234"
}
```

## Error Handling

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (config invalid, API error, file not found, usage error, etc.) |

### Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|------------|
| `vcoding pick` exits 1 with "gh: command not found" | gh CLI not installed | Install gh CLI |
| `vcoding pick` exits 1 with "invalid config" | Missing or malformed config.yaml | Run `vcoding init` and set API key |
| `vcoding do` exits 0 but PLAN.md is empty | Model returned empty response | Check API key validity; retry |
| `vcoding doctor` reports OPENROUTER_API_KEY missing | Environment variable not set | `export OPENROUTER_API_KEY=...` |
| Pipeline times out | Network or API latency | Implement agent-side 5-minute timeout with retry |

## Examples

### Example 1: Plan from GitHub Issue

```bash
# Verify environment
vcoding doctor
# Output: "All checks passed" or list of issues

# Generate plan from issue
vcoding pick 42
# Creates: .vcoding/runs/<run-id>/
# Updates: .vcoding/runs/latest -> <run-id>

# Read the plan
# PLAN = READ('.vcoding/runs/latest/PLAN.md')

# Implement according to plan steps
# ... agent makes code changes ...

# Check cost
vcoding stats
```

### Example 2: Plan from Spec File

```bash
# Create spec file
# WRITE('spec.md', 'Add user authentication with OAuth2')

# Generate plan
vcoding do spec.md

# Read and implement
# PLAN = READ('.vcoding/runs/latest/PLAN.md')
# ... implementation ...
```

### Example 3: Verbose Output for Debugging

```bash
# Use verbose mode to see full model output
vcoding pick 15 --verbose

# Read the plan
# PLAN = READ('.vcoding/runs/latest/PLAN.md')
```
