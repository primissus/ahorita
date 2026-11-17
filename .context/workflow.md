# Workflow

## Before touching anything
1. Read `.context/decisions.md`. Don't reopen settled choices.
2. Read `plans/README.md`. Work happens plan by plan; pick up the plan you were given and its `.PROGRESS.md` tracker.
3. Check README "Expression examples". New syntax goes there first.
4. Commit policy (`W.1`): commit progress using conventional commits (`feat:`, `fix:`, `test:`, `docs:`) after each plan/milestone, and keep `.context/` updated throughout execution (confirmed in chat 2026-11-15).

## To make a change
1. Add table-driven test cases for the expression(s), with a fixed `now` and tz.
2. Implement in the engine.
3. Expose it through the CLI (and later the GUI) without adding logic there.

## Open questions
- `QUESTIONS.md`: the original batches. Still blank: batch 4, 7.7.
- `QUESTIONS-2.md`: gaps and follow-ups. New questions found while building go to its Follow-ups section, in the same style.
- An unanswered question stays `[PENDING]` in `.context/`. Plans say which ★ default to build meanwhile.

## Before calling something done
- [ ] `go build ./...` passes
- [ ] `go test ./...` green
- [ ] `go vet ./...` clean
- [ ] `cd web && npm run build && npm test` passes
- [ ] README examples updated if syntax changed
- [ ] The plan's tracker and the status table in `plans/README.md` are up to date

## Deploy
`go install github.com/primissus/ahorita/cmd/ahorita@latest` (decision 5.9). No release tooling.
