# ahorita

Date/time calculator in Go. One expression engine, two front ends: a CLI (`ahorita "now + 2d"`) and a GUI calculator (later). See README.md for the target feature set and expression examples.

Status: v1 (engine + CLI, plans 1–8) works and is verified; business days (plan 9), recurrence and cron (plan 10), and the web GUI (plan 11) are next. See `plans/README.md`.

Open design questions live in `QUESTIONS.md` (original batches) and `QUESTIONS-2.md` (gaps and follow-ups). When answers arrive, record them in `.context/` and update the status tables. Don't edit the user's answers. An unanswered question stays `[PENDING]` in `.context/`.

## Rules

- The engine (parse → evaluate → format) has no I/O or UI imports. CLI and GUI only call it.
- Every case in `CASES.md` and every expression in README.md "Expression examples" must eventually have a test case. Expected values in `CASES.md` follow the reference "now" at its top.
- Tests must inject "now" and the local timezone. Never read the wall clock in the engine.
- Use stdlib first (`time`, `flag`, `testing`). Add a dependency only with a reason recorded in `.context/decisions.md`.

## Project context

- @.context/architecture.md
- @.context/conventions.md
- @.context/decisions.md
- @.context/glossary.md
- @.context/workflow.md
- @.context/known-issues.md
