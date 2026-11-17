# Decisions made

> One entry per decision. What matters is the "why" and what was rejected.
> IDs like `3.5` point at `QUESTIONS.md`; `C.2`, `N7`, `F.1` point at `QUESTIONS-2.md`.
> Status: `current` (settled), `proposed` (Claude's reading of an answer, waiting for the user's confirmation), `deferred`, `open`.

## Project

### 2026-11-14 · Go for the core and CLI
- **Decision:** Write ahorita in Go.
- **Why:** Fast startup and a single static binary make the CLI easy to call from shell snippets and text expanders.
- **Rejected:** TypeScript/Node + React. Slower CLI startup and a runtime dependency. Still possible for a web GUI later.
- **Status:** current

### 2026-11-14 · One engine, two front ends
- **Decision:** CLI and GUI share one expression engine and the same set of operations.
- **Why:** Avoids feature drift between the two.
- **Rejected:** [PENDING]
- **Status:** current

### 2026-11-14 · 1.1 Expression grammar approach
- **Decision:** Hand-written lexer and recursive-descent parser (Pratt-style for operators). Pipeline: lex → normalize words → parse → evaluate → format.
- **Why:** The user accepted either option and asked for the faster MVP and the easier maintenance. By hand the v1 grammar is a few hundred lines of plain Go, with no generator step, exact error positions, and "expected next" for free. Two keyword languages and loose word order fit generator grammars poorly.
- **Rejected:** goyacc, participle. If a library is ever wanted, participle (no code generation).
- **Status:** current (C.1 ★ accepted in chat 2026-11-15)

### 2026-11-14 · 1.2 v1 scope
- **Decision:** v1 = engine + CLI covering arithmetic, relative dates, ranges, timezones and display formats. Then business days + holidays, then recurrence + cron, then the GUI.
- **Why:** A usable CLI ships first; each later milestone adds one feature family.
- **Rejected:** Building the whole feature list before a first release.
- **Status:** current

### 2026-11-14 · 1.3 Audience
- **Decision:** Personal tool first, kept open to going public later.
- **Why:** Defaults follow the user's preferences (MX, Monday weeks), but nothing is hard-coded that config can't change.
- **Rejected:** Personal-only; public from day one.
- **Status:** current

### 2026-11-14 · 1.4 Input language
- **Decision:** English and Spanish input in v1.
- **Why:** User's choice, against the recommendation. It widens v1: word tables, accents and Spanish word order (see F.1–F.5).
- **Rejected:** English only in v1.
- **Status:** current

### 2026-11-14 · 1.5 Module path, license, git init
- **Decision:** Module path `github.com/primissus/ahorita`. No license yet (README stays "TBD"). OK to `git init`.
- **Why:** Matches the user's GitHub login and their other Go module; a license waits until the tool goes public (1.3). Answered in chat, not in `QUESTIONS.md`.
- **Rejected:** A bare local module name (`ahorita`); MIT or Apache-2.0 now.
- **Status:** current

## Grammar

### 2026-11-14 · 2.1 The comma is a pipe
- **Decision:** `a, b, c` is a left-to-right pipeline: each stage is applied to the previous result. `|` is the same operator (`today | end of day | start of hour | to UTC`).
- **Why:** The user's proposal. One rule covers narrowing, converting and setting a zone.
- **Rejected:** The comma as "narrow in either order" only; the comma meaning three different things.
- **Status:** current. What happens to the notebook's right-anchored forms (`end of day, today`) is proposed in C.2.

### 2026-11-14 · 2.2 Timezone operators
- **Decision:** `@` interprets (`2pm @ Tokyo`), `to` converts (`now to Tokyo`), and `in <place>` is accepted too. What follows `in` decides its meaning: a duration (`in 3 months`), a period (`in Q3`) or a place (`in Tokyo`).
- **Why:** User chose A and B together: explicit operators plus the natural-language form.
- **Rejected:** `in` as the only spelling.
- **Status:** current. Whether `in <place>` interprets or converts is proposed in C.3.

### 2026-11-14 · 2.3 Periods are spans
- **Decision:** `last week`, `next month`, `in Q3`, `on sept`, `2nd week, h2` give a span. `-t date` shows its start, `-t range` the whole span, `end of …` its end. `today - 1w` stays a single date.
- **Why:** A period has two ends; picking one silently loses the other.
- **Rejected:** `next month` = today + 1 month.
- **Status:** current

### 2026-11-14 · 2.4 Unit letters are case-sensitive
- **Decision:** `m` = minute, `M` = month. Full words always work (`min`, `month`, `months`). Combined forms work (`1h30m`).
- **Why:** User's choice.
- **Rejected:** `mo` as the only month letter.
- **Status:** current. The full letter table and ISO 8601 durations are proposed in C.4.

### 2026-11-14 · 2.5 Accepted input formats
- **Decision:** ISO date, ISO/RFC 3339 date-time, `14:30`, `2pm`/`2:30pm`, Unix seconds (`@1700000000`), Unix milliseconds, month names (`aug 20`, `20 aug`, `aug 20th`), slash dates. The active language decides which words are accepted.
- **Why:** All ticked by the user.
- **Rejected:** none
- **Status:** current. Slash-date order and how milliseconds are told from seconds are proposed in C.5.

## Meaning

### 2026-11-14 · 3.1 Which weekday
- **Decision:** `last X` = the most recent X strictly before today. `next X` = the first X strictly after today. `this X` = the X in the current week. `closest X` = the nearest X in either direction, today if today is X.
- **Why:** "Strictly before/after" never returns today, which is the least surprising reading.
- **Rejected:** `next X` = the X in next week.
- **Status:** current

### 2026-11-14 · 3.2 Dates without a year
- **Decision:** `on aug 20th` is this year even if it has passed. `next aug 20` gives the next one.
- **Why:** Matches the notebook.
- **Rejected:** Always the next occurrence.
- **Status:** current

### 2026-11-14 · 3.3 Week numbers
- **Decision:** `2nd week` / `week 2` follow ISO week numbering. Week rules (first day of the week and the like) are configurable.
- **Why:** User's choice.
- **Rejected:** Days 8–14 of the period; the second full week.
- **Status:** current. How ISO numbering applies inside a period that isn't a year (`2nd week, h2`) is proposed in C.6.

### 2026-11-14 · 3.4 Week start
- **Decision:** Weeks start on Monday by default; configurable. `end of week` = end of the week's last day (Sunday by default). `end of work week` = end of Friday.
- **Why:** ISO convention and the user's locale.
- **Rejected:** Sunday start as the default.
- **Status:** current

### 2026-11-14 · 3.5 Month-end arithmetic
- **Decision:** `2026-01-31 + 1M` clamps to the last day of the month (`2026-02-28`). Configurable back to Go's overflow.
- **Why:** "One month after Jan 31" landing in March surprises people.
- **Rejected:** Go's `AddDate` overflow (`2026-03-03`) as the default.
- **Status:** current

### 2026-11-14 · 3.6 Mixing types
- **Decision:** (a) Date ± hours becomes a date-time starting at 00:00 (`2026-06-02 - 1h` → `2026-06-01T23:00`). (b) `date1 - date2` keeps its sign (`-40d`). (c) Only duration × number is allowed; any other multiplication is an error.
- **Why:** Each is the reading that loses no information.
- **Rejected:** Erroring on date ± hours; always-positive differences.
- **Status:** current

### 2026-11-14 · 3.7 Counting between two dates
- **Decision:** End dates are not counted by default (Monday → Friday = 4). The `inclusive` keyword counts them. The default is configurable.
- **Why:** Matches plain date subtraction.
- **Rejected:** Always inclusive; inclusive only for business days.
- **Status:** current

### 2026-11-14 · 3.8 Day words vs "ago"
- **Decision:** `yesterday`, `tomorrow` are dates (`today ± 1d`). `2d ago` keeps the time of day (`now - 2d`).
- **Why:** A day word names a day; "ago" measures back from this moment.
- **Rejected:** `yesterday` = `now - 1d` (the notebook's reading).
- **Status:** current. The user also asked to "consider the format"; the reading is proposed in C.7.

### 2026-11-14 · 3.9 Default output type
- **Decision:** With no `-t`, the output follows the result: `today` → date, `now` → date-time, `a - b` → range.
- **Why:** No flag needed for the common case.
- **Rejected:** Always date-time.
- **Status:** current

### 2026-11-14 · 3.10 Ambiguous input and `--strict`
- **Decision:** Make the best guess quietly; `--explain` shows how the input was read. `--strict` turns every ambiguity into an error.
- **Why:** Quiet by default suits snippets; strict suits scripts.
- **Rejected:** Warnings on stderr by default.
- **Status:** current

### 2026-11-14 · 3.11 Converting calendar units
- **Decision:** `3 months to days` is an error unless there's a start date.
- **Why:** The answer is 89–92 depending on the start.
- **Rejected:** Counting from today silently; average month length.
- **Status:** current. The exact anchored syntax is proposed in N7.

## Timezones

### 2026-11-15 · 4.1 Turning a place name into a timezone
- **Decision:** Match the city part of standard IANA zone names (`Tokyo` → `Asia/Tokyo`), plus a small alias list (`CDMX`, `NYC`, `SF`, etc.).
- **Why:** Fast, offline, zero heavy dependencies; covers major cities and common shorthand. Answered in chat (★ accepted).
- **Rejected:** Embedding GeoNames; online geocoding.
- **Status:** current

### 2026-11-15 · 4.2 Timezone abbreviations
- **Decision:** Accept a fixed list of unambiguous abbreviations (`JST`, `EST`, `PST`, etc.) as fixed offsets. Ambiguous abbreviations (`CST`, `CDT`, `IST`, `BST`, `AST`) return an error listing candidates.
- **Why:** Prevents silent misinterpretation of `CST` or `IST` across regions. Answered in chat (★ accepted).
- **Rejected:** Config mapping per abbreviation; rejecting all abbreviations.
- **Status:** current

### 2026-11-15 · 4.3 Default zone
- **Decision:** Use the system's local timezone by default, overridable with `TZ`, `AHORITA_TZ`, config `tz`, or `--tz`.
- **Why:** Matches standard CLI expectations while remaining deterministic when overridden or injected. Answered in chat (★ accepted).
- **Rejected:** Always defaulting to UTC.
- **Status:** current

### 2026-11-15 · 4.4 Timezone data
- **Decision:** Embed Go's timezone database (`_ "time/tzdata"`).
- **Why:** Ensures consistent timezone resolution across OSes and WebAssembly (about 450 KB binary cost). Answered in chat (★ accepted).
- **Rejected:** Relying solely on OS zoneinfo files.
- **Status:** current

### 2026-11-17 · 4.5 Fuzzy timezone dictionary, unquoted article preservation, and parentheses grouping (Plan 12)
- **Decision:**
  1. Export `LookupZone(name string, strict bool)` with an expanded place dictionary (US/MX/CA states and regions, countries, and non-IANA major cities/exonyms) and deterministic tiered fuzzy matching for non-slashed queries of length ≥ 4 (Tier 1 word-prefix → Tier 2 substring → Tier 3 Optimal String Alignment Damerau-Levenshtein with `maxDist = 1`, or `2` only when both query and candidate have length ≥ 9 and share/transpose the first character). Unambiguous matches auto-resolve when `strict == false` and return `did you mean: ...` under `--strict` or when multiple distinct zones match. `--tz` falls back to `LookupZone(value, false)` when `time.LoadLocation` fails (while still rejecting `"Local"`).
  2. Preserve unquoted place names starting with Spanish articles (`Los Angeles`, `LA`, `La Paz`, `El Salvador`, `Las Vegas`) and following Spanish `a` (`ahora a LA`) in `Normalize` by only dropping `"a la"`/`"a las"` before `Clock`/`Number`/`Quantity` and only dropping single-word `"el"`/`"la"`/`"los"`/`"las"` when followed by a Spanish-vocab/special-phrase/number/quantity token.
  3. Support `"(" span-expr ")"` grouping in the parser (`parsePrimary`), allow `(today - 1d) 2pm` via `canTakeClock`, and parenthesize lower-precedence sub-expressions in `Expr.String()` for clean AST round-tripping.
- **Why:** Allows natural unquoted place names, common states/countries/cities, mild typos, and grouped sub-expressions like `(now - 1 business day) to los_angeles` and `2 * (1h + 30m)` while preserving offline determinism, `--strict` guarantees, and all Spanish grammar.
- **Rejected:** Dropping Spanish articles unconditionally (breaks `Los Angeles`/`LA`); allowing edit distance 2 on short words (causes false positives like `Atlantis` → `Atlanta`).
- **Status:** current

## CLI

### 2026-11-14 · 5.1 Unquoted input
- **Decision:** `ahorita now + 2d` works: arguments are joined with spaces. Flags go before the expression.
- **Why:** Less quoting in snippets. (`*` and `|` still need quotes in a shell.)
- **Rejected:** Requiring one quoted argument.
- **Status:** current

### 2026-11-14 · 5.2 Display format syntax
- **Decision:** Java-style tokens (`dd-MM-yyyy HH:mm`) plus named presets (`iso`, `rfc3339`, `unix`, `human`). English order is month-first (`MM-dd-yyyy`); Spanish order is day-first.
- **Why:** `mm` must mean minutes; Java-style tokens are the most widely known.
- **Rejected:** strftime, Go layouts, the lowercase `dd-mm-yyyy` style.
- **Status:** current. Where the language-dependent order applies is proposed in C.8.

### 2026-11-14 · 5.3 Default display
- **Decision:** ISO (`2026-11-16`, `2026-11-16T14:30:00-06:00`).
- **Why:** Suits scripts and sorts correctly.
- **Rejected:** Human-readable by default.
- **Status:** current

### 2026-11-14 · 5.4 Flags
- **Decision:** `-n` (no trailing newline), `-c` (copy to clipboard), `--json`, `--explain`, `--strict`, `--config-file`, `-f/--format`, `--calendar`, `--day-week-start`, `-o/--output <file>`, plus `-t/--type`.
- **Why:** User's list.
- **Rejected:** none
- **Status:** current. What `--calendar` takes is proposed in C.10.

### 2026-11-14 · 5.5 Listings are expressions
- **Decision:** Holiday lists, cron explanations and calendars are expressions (`holidays MX 2026`), not subcommands.
- **Why:** The GUI gets them for free.
- **Rejected:** Subcommands.
- **Status:** current

### 2026-11-14 · 5.6 Bare `ahorita`
- **Decision:** Prints `now`.
- **Rejected:** Interactive prompt; help.
- **Status:** current

### 2026-11-14 · 5.7 Interactive mode with variables
- **Decision:** Later.
- **Status:** deferred

### 2026-11-14 · 5.8 Settings
- **Decision:** Env vars and flags (`AHORITA_TZ`, `AHORITA_FORMAT`, …). Precedence: flag > env > default.
- **Why:** User chose A (no config file in v1).
- **Rejected:** A config file in v1.
- **Status:** current, but it conflicts with 3.3, 5.4 (`--config-file`) and 6.2, which all ask for a config file. Resolution proposed in C.9.

### 2026-11-14 · 5.9 Distribution
- **Decision:** `go install` only, for now.
- **Rejected:** Homebrew tap, goreleaser.
- **Status:** current

### 2026-11-14 · CLI flag parsing
- **Decision:** stdlib `flag`.
- **Why:** The stdlib-first rule in CLAUDE.md. Each flag is registered under its short and long name.
- **Rejected:** cobra, kong (no subcommands to justify them, see 5.5).
- **Status:** current

## Business days & holidays

### 2026-11-14 · 6.1 Holiday source
- **Decision:** Offline rules from `github.com/rickar/cal/v2` (v2.1.33 has an `mx` package). No network.
- **Why:** Instant and works offline. Dependency reason: holiday rules for ~40 countries are data we shouldn't maintain by hand.
- **Rejected:** Online API with a cache; offline-then-online. Revisit after v1.
- **Status:** current

### 2026-11-14 · 6.2 Default region
- **Decision:** MX. Settable by env var (and config file, see C.9). Official public holidays only. State-level holidays later.
- **Status:** current (state-level: deferred)

### 2026-11-14 · 6.3 Weekend
- **Decision:** Saturday and Sunday; configurable.
- **Rejected:** Per-country weekends.
- **Status:** current

### 2026-11-14 · 6.4 Own days off
- **Decision:** In v1 of the business-days milestone: passed as a parameter that points at a calendar.
- **Why:** User's choice.
- **Rejected:** Later.
- **Status:** current. The file format is proposed in C.10.

### 2026-11-14 · 6.5 Starting on a non-business day
- **Decision:** `saturday + 1 business day` = Monday. The start day isn't counted, as in spreadsheets' WORKDAY.
- **Rejected:** Roll to Monday first, then add.
- **Status:** current

### 2026-11-14 · 6.6 `next business day`
- **Decision:** Strictly after: the following business day even when today is one. `on or after` keeps a business day as it is.
- **Rejected:** Same day if it's a business day.
- **Status:** current

## Recurrence, calendars, cron

### 2026-11-14 · 7.1 `every day 2nd`
- **Decision:** The 2nd of every month. Every other day is `every 2 days`.
- **Status:** current. The user's other forms (`every weekday 2nd`, `every day-year 2nd`) are proposed in C.11.

### 2026-11-14 · 7.2 Calendar outputs
- **Decision:** A list of the next N dates, an `.ics` file, and a month grid in the terminal.
- **Why:** iCalendar (`.ics`, RFC 5545) is the one globally used calendar format; the others (jCal, xCal, CSV imports) are re-encodings of it and aren't needed.
- **Status:** current

### 2026-11-14 · 7.3 Recurrence format
- **Decision:** RFC 5545 RRULE via `github.com/teambition/rrule-go` (v1.8.2).
- **Why:** The format calendars use; converts to a date list, `.ics` and cron. Dependency reason: correct RRULE expansion is more than a few lines.
- **Rejected:** A format of our own (the user is fine with one if RRULE proves heavy).
- **Status:** current

### 2026-11-14 · 7.4 Cron operations
- **Decision:** Text → cron, cron → next run times, cron → plain-language explanation.
- **Status:** current

### 2026-11-14 · 7.5 Cron format
- **Decision:** Standard 5-field.
- **Rejected:** Optional seconds; Quartz/AWS/GitHub variants.
- **Status:** current

### 2026-11-14 · 7.6 Schedules cron can't express
- **Decision:** Several cron lines when that works, otherwise an error. Under `--strict`, always an error.
- **Rejected:** Approximations with a warning.
- **Status:** current

### 2026-11-15 · 7.7 Cron and timezones
- **Decision:** Output cron in the given (or local) zone. When converting a recurrence to another zone (`to <zone>`) and then `to cron`, shift the time/weekday into that zone and attach a warning note if the weekday shifts or DST differs between the zones.
- **Why:** Keeps local schedules accurate by default while making UTC/cross-zone drift explicit. Answered in chat (★ accepted).
- **Rejected:** Silent UTC conversion; rejecting zone conversions on recurrences.
- **Status:** current

### 2026-11-15 · 7.8 Cron parsing and evaluation
- **Decision:** Parse and evaluate 5-field cron expressions via `github.com/robfig/cron/v3` (`v3.0.1`).
- **Why:** Provides standard 5-field cron parsing (`ParseStandard`), range/step/list/month/weekday validation, and timezone-aware `Schedule.Next` stepping in a small focused dependency. Dependency reason: complete cron bitmask/calendar field evaluation is error-prone to duplicate.
- **Rejected:** A hand-written cron matcher.
- **Status:** current

## GUI

### 2026-11-14 · 8.1 Kind of GUI
- **Decision:** A button calculator with date pickers.
- **Rejected:** Notepad sheet, launcher popup, terminal UI (for now).
- **Status:** current

### 2026-11-14 · 8.2 Platforms
- **Decision:** Web first.
- **Status:** current

### 2026-11-15 · 8.3 GUI framework
- **Decision:** React + TypeScript, with the Go engine compiled to WebAssembly (`GOOS=js GOARCH=wasm`).
- **Why:** Calls the exact same `expr.Run` engine in the browser, preserving the one-engine rule with no duplicate date logic (C.12 ★ accepted in chat).
- **Rejected:** Kotlin Multiplatform (cannot call Go directly without WebAssembly or duplicating the engine).
- **Status:** current
