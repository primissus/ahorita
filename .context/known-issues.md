# Known issues (gotchas)

> The traps that already bit you. Each one saves an hour of the agent's time
> (and yours).

Known domain traps and v1 gotchas caught during design and acceptance testing:

## Month arithmetic overflows
- **Happens when:** `2026-01-31 + 1M`
- **Real cause:** Go's `time.AddDate` normalizes Feb 31 → Mar 3.
- **Fix:** clamp to the last day of the month (decision 3.5). Never call `AddDate` with months directly; go through the engine's month-add helper. Go's behavior stays available as a config option.

## DST gaps and overlaps
- **Happens when:** adding hours vs days across a DST change, or parsing a local time that doesn't exist or happens twice.
- **Real cause:** `+1d` (calendar) ≠ `+24h` (fixed) on DST days.
- **Fix:** keep calendar and fixed durations as separate fields. The reference zone (`America/Mexico_City`) has no DST, so DST tests must use another zone (`America/New_York`).

## Ambiguous numeric dates
- **Happens when:** `02/06/2026`, which reads as Feb 6 or Jun 2.
- **Fix:** [PENDING: C.5. Proposed: the active language decides (en = month-first, es = day-first); `--strict` rejects it when both readings are valid.]

## `m` vs `M`
- **Happens when:** `2m` (2 minutes) vs `2M` (2 months).
- **Real cause:** unit letters are case-sensitive (decision 2.4).
- **Fix:** the lexer keeps the original case; never lowercase a quantity's unit. Words are matched case-insensitively, single letters are not.

## Words with two meanings
- **Happens when:** `ago` (English "ago" / Spanish abbreviation of agosto), `mar` (March / martes), `second`/`segundo` (unit / ordinal), `mañana` (tomorrow / morning), `in` (offset / period / place), `to` (convert zone / convert unit / range end).
- **Fix:** each is resolved by position in the normalizer or parser, and each tie-break rule needs its own test. Under `--strict` an unresolved tie is an error.

## Shell characters
- **Happens when:** `ahorita today | end of day` or `ahorita 2h * 3` typed without quotes.
- **Real cause:** the shell eats `|` and expands `*` before ahorita sees them.
- **Fix:** the comma is the shell-safe pipe. Document that `|` and `*` need quotes.

## Flags after the expression
- **Happens when:** `ahorita now + 2d -t date`.
- **Real cause:** stdlib `flag` stops at the first non-flag argument, and `-` is also the minus operator.
- **Fix:** flags go before the expression. Intentional.

## MX holiday rules (implemented in `expr/business.go`)
- **Why custom rules were needed:** in `github.com/rickar/cal/v2` v2.1.33, the upstream `mx` package keeps Constitution Day on Feb 5 and Juárez's birthday on Mar 21 (the law moved both to Mondays), shifts weekend holidays to a weekday (Mexico doesn't), and marks Christmas as a bank holiday.
- **What was built (plan 9):** `expr/business.go` defines MX holidays directly from Ley Federal del Trabajo art. 74 (including the October 1 presidential inauguration every 6 years starting in 2024) with both English and Spanish names, and does not use `cal/v2/mx`. Election days (LFT art. 74, IX) have no fixed rule and can be supplied via a `--calendar` days-off file. All 46 other country packages in `rickar/cal/v2` that export a top-level `Holidays` list are wired and filtered by `ObservancePublic` (upstream `rickar/cal/v2` caveats remain in some countries: e.g., `GB` marks all bank holidays `ObservanceBank`, `IE`/`BR`/`PL`/`LT`/`NC` omit `Type` on some public holidays, `CA`/`NZ`/`ZA` mark Christmas/Boxing Day/Day of Goodwill `ObservanceBank`, `GR` names Jan 1 `"Xristougenna"`, and `AU` exposes only per-state slices rather than a national `Holidays` variable).

## Things that look broken but are intentional
- `yesterday` has no time of day; `2d ago` does (decision 3.8).
- `last week` is a span, not a date (decision 2.3).
- `on aug 20th` can be in the past (decision 3.2).
- `20 ago` under English is not a strict-mode word tie: the normalizer resolves `ago` next to a bare number as `august` (`2026-08-20`), while bare `ago` sets `Tie` (plan 2 step 5).

## `time.Date` normalizes a DST gap backward in non-strict mode
- **Happens when:** evaluating `2026-03-08 02:30` in `America/New_York` (02:00 → 03:00 jump) with `Strict: false`.
- **Real cause:** Go's `time.Date` chooses the pre-transition offset and yields `2026-03-08T01:30:00-05:00` instead of `03:30:00-04:00`.
- **Fix:** documented for non-strict mode; `--strict` (`wall()` in `expr/eval.go`) errors with `Pos` pointing at the time literal (`11`).

## Recurrence and cron keywords aren't recognized before plan 10
- **Happens when:** evaluating `every monday`, `every day 2nd`, or `every monday, to cron` in v1.
- **Real cause:** plan 3's parser left `every` and `cron` out of the keyword tables until plan 10, so those README examples fail with unknown-word or parse errors instead of `"aren't available yet"`.
- **Fix:** tracked as 3 `BUG:` skips in `expr/readme_test.go` until plan 10 adds recurrence and cron syntax.
