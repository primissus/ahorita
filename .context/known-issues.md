# Known issues (gotchas)

> The traps that already bit you. Each one saves an hour of the agent's time
> (and yours).

Nothing has bitten yet. These are known domain traps to design around:

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

## MX holiday rules in `rickar/cal`
- **Happens when:** asking for Mexican public holidays.
- **Real cause:** in v2.1.33 the `mx` package keeps Constitution Day on Feb 5 and Juárez's birthday on Mar 21 (the law moved both to Mondays), shifts weekend holidays to a weekday (Mexico doesn't), and marks Christmas as a bank holiday.
- **Fix:** plan 9 defines the MX list itself from Ley Federal del Trabajo art. 74 and uses the library for other countries only.

## Things that look broken but are intentional
- `yesterday` has no time of day; `2d ago` does (decision 3.8).
- `last week` is a span, not a date (decision 2.3).
- `on aug 20th` can be in the past (decision 3.2).
