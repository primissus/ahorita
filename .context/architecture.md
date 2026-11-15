# Architecture

## In one sentence
A date/time expression engine in Go that reads English and Spanish. A CLI (v1) and a web GUI (later) front it, and both support the same operations: arithmetic, ranges, relative dates, timezones, business days, holidays, recurrence, cron, and formatting.

## Stack
- Language / runtime: Go (local toolchain 1.27)
- CLI: stdlib `flag`
- GUI: web first, a button calculator with date pickers. React + TypeScript with Go engine compiled to WebAssembly (`GOOS=js GOARCH=wasm`).
- Database: none
- Dependencies: `github.com/rickar/cal/v2` (holiday rules, plan 9), `github.com/teambition/rrule-go` (recurrence rules and `.ics` expansion, plan 10), `github.com/robfig/cron/v3` (5-field cron parsing and stepping, plan 10)
- External services: none

## Folder map
Currently exists: `go.mod`, `.gitignore`, `README.md`, `CLAUDE.md`, `CASES.md`, `QUESTIONS.md`, `QUESTIONS-2.md`, `.context/`, `plans/`, `expr/` (`token.go`, `lexer.go`, `lexer_test.go`, `normalize.go`, `normalize_test.go`, `vocab_en.go`, `vocab_es.go`, `ast.go`, `parser.go`, `parser_test.go`, `print.go`, `print_test.go`, `value.go`, `value_test.go`, `context.go`, `eval.go`, `eval_test.go`, `arith.go`, `arith_test.go`, `period.go`, `period_test.go`, `format.go`, `format_test.go`, `pattern.go`, `pattern_test.go`, `result.go`, `result_test.go`, `zone.go`, `zones_data.go` [generated], `zones_gen.go`, `zone_test.go`, `business.go`, `business_test.go`, `recur.go`, `recur_test.go`, `cron.go`, `cron_test.go`, `calendar.go`, `calendar_test.go`, `cases_test.go`, `readme_test.go`, `spanish_test.go`, `edge_test.go`, `strict_test.go`), `internal/settings/` (`settings.go`, `settings_test.go`), `cmd/ahorita/` (`config.go`, `config_test.go`, `main.go`, `main_test.go`), `cmd/wasm/` (`bridge.go`, `main.go`, `main_stub.go`, `main_test.go`), and `web/` (`DESIGN.md`, `build-wasm.sh`, `smoke.mjs`, `public/ahorita.wasm`, `public/wasm_exec.js` — Step 4+ React/Vite scaffold pending `web/DESIGN.md` approval). Each plan in `plans/` creates only what it needs. Target:

```
go.mod
expr/               the engine: one package, no I/O, no clock, no env
internal/settings/  shared settings validation & days-off parsing (no file I/O)
cmd/ahorita/        the CLI: flags, env, config, stdin/stdout, clipboard
cmd/wasm/           the JS/WASM bridge exposing ahoritaRun(input, optionsJSON)
web/                the web GUI (build-wasm.sh, smoke.mjs, DESIGN.md; React/Vite app pending approval)
plans/              build plans and their progress trackers
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
