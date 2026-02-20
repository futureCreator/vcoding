# AGENTS.md

This project uses **vcoding** to generate AI-driven implementation plans from GitHub issues or local spec files.

## How to Implement

When asked to implement a feature or fix:

1. Check if `.vcoding/runs/latest/PLAN.md` exists.
2. **If it exists**: Read it and implement according to the plan — follow the listed files to change, implementation steps, and edge cases exactly.
3. **If it does not exist**: Do not proceed with implementation. Ask the user to first generate a plan:
   - `vcoding pick <issue-number>` — generate a plan from a GitHub issue
   - `vcoding do <spec-file>` — generate a plan from a local spec file

## Workflow

```
vcoding pick 42       # Fetch issue #42 → produce .vcoding/runs/latest/PLAN.md
vcoding do SPEC.md    # Use local spec  → produce .vcoding/runs/latest/PLAN.md
# Then implement according to .vcoding/runs/latest/PLAN.md
```

## Plan Structure

`.vcoding/runs/latest/PLAN.md` contains:

- **Goal** — one-paragraph summary of what needs to be done
- **Files to Change** — specific files and what to change in each
- **Implementation Steps** — numbered, concrete, actionable steps; follow them in order
- **Edge Cases** — constraints and edge cases to handle

## Notes

- `.vcoding/runs/latest/PLAN.md` always points to the latest auto-generated plan — do not edit manually.
- If the plan seems incomplete or incorrect, ask the user to re-run `vcoding pick` or `vcoding do`.
