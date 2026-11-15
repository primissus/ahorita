# Architecture

## In one sentence
A date/time expression engine in Go that reads English and Spanish. A CLI (v1) and a web GUI (later) front it, and both support the same operations: arithmetic, ranges, relative dates, timezones, business days, holidays, recurrence, cron, and formatting.

## Stack
- Language / runtime: Go (local toolchain 1.27)
- CLI: stdlib `flag`
- GUI: web first, a button calculator with date pickers. React + TypeScript with Go engine compiled to WebAssembly (`GOOS=js GOARCH=wasm`).
- Database: none
- Dependencies (planned, added by the plan that needs them): `github.com/rickar/cal/v2` (holiday rules, plan 9), `github.com/teambition/rrule-go` (recurrence, plan 10)
- External services: none

## Folder map
Currently exists: `go.mod`, `.gitignore`, `plans/`, and `expr/` (`token.go`, `lexer.go`, `lexer_test.go`, `normalize.go`, `normalize_test.go`, `vocab_en.go`, `vocab_es.go`). Each plan in `plans/` creates only what it needs. Target:

```
go.mod
expr/            the engine: one package, no I/O, no clock, no env
cmd/ahorita/     the CLI: flags, env, config, stdin/stdout, clipboard
plans/           build plans and their progress trackers
```

File-by-file ownership is in `plans/README.md`.

## Data flow
1. Input string (CLI args joined with spaces, stdin, or GUI field) plus options (`-t` output type, `-f` format, tz, lang).
2. Lexer → tokens.
3. Normalizer → English and Spanish words, accents and multi-word phrases become canonical keywords.
4. Parser → a pipeline of stages (`a, b | c`), each an AST.
5. Evaluator runs the stages left to right against an injected context (now, local tz, config: language, week rules, holiday calendar, strict).
6. Formatter renders the result as `date|datetime|time|range|relative` in the chosen display format and language.

## What does NOT exist (and should not be created)
- No server, daemon, or database.
- No network calls anywhere in v1. Holidays are offline rules.
- No I/O in the engine: the CLI reads the clock, env vars, config and calendar files and passes them in.
- No GUI-only or CLI-only operations. Features live in the engine.
- No subcommands: listings are expressions (`holidays MX 2026`).
