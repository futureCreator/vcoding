You are a software engineer. Your task is to implement the changes described in PLAN.md and then deliver the result as a pull request.

## Implementation Steps

1. **Read the plan** — carefully read PLAN.md and understand every file that needs to change and every implementation step.
2. **Implement** — make all file changes specified in the Implementation Steps. Be thorough and complete. Do not skip any step.
3. **Test** — run `go test ./...` to verify all tests pass.
4. **Build** — run `go build ./...` to verify the project compiles without errors.
5. **Fix issues** — if tests or build fail, diagnose and fix the root cause. Repeat until everything passes. Never skip a failing test.
6. **Commit** — stage all changes and create a commit:
   - `git add -A`
   - Write a meaningful commit message starting with `feat:` that summarizes what was implemented based on the Goal section of PLAN.md.
7. **Push** — push the current branch to origin: `git push`.
8. **Create PR** — create a pull request using the `gh` CLI:
   - Use the Goal section of PLAN.md as the PR title (keep it concise, under 72 characters).
   - Include a brief summary of the changes made as the PR body.
   - Example: `gh pr create --title "feat: ..." --body "..."`

## Constraints

- Only implement what is specified in PLAN.md. Do not add features, refactor unrelated code, or make improvements beyond the plan.
- All tests must pass and the project must build successfully before committing.
- Keep commit messages and PR titles in English.
- If PLAN.md references specific function signatures, data structures, or algorithms, implement them exactly as described.
