# Code conventions

## Style
- Formatting: `gofmt` (via `go fmt` / editor). [PENDING: golangci-lint config?]
- Naming: standard Go. Short package names, no stutter (`expr.Parse`, not `expr.ParseExpr`).
- Imports: stdlib group, then third-party, then local.

## Patterns we DO use
- Errors as values, wrapped with `fmt.Errorf("...: %w", err)`. Parse errors carry the input position.
- Pass "now" and `*time.Location` explicitly (context struct), never `time.Now()` inside the engine.
- Table-driven tests.

## FORBIDDEN patterns
- No `panic` for user input errors.
- No UI/CLI imports inside the engine.
- No interfaces with a single implementation.

## Tests
- Where they live: `_test.go` next to the code.
- What must always be tested: every grammar form in README "Expression examples", DST transitions, month-end arithmetic (Jan 31 + 1 month), leap years, timezone conversion.

## Commits
- Format: conventional commits (`feat:`, `fix:`, `docs:` …). [PENDING: confirm]
