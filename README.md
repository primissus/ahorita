# ahorita

A date & time calculator. Type an expression in English or Spanish, get a date back.

```sh
ahorita now + 2d
ahorita "2026-06-02 - 1h"
ahorita -t date next month
ahorita "2026-12-24 - today"          # → 40d
ahorita now, to UTC
ahorita hace 2 días
```

Two front ends, one engine:

- **CLI**: fast, scriptable, pipe-friendly. Meant to be dropped into shell snippets and text expanders. This is v1.
- **GUI**: a web calculator with buttons and date pickers for the same operations. *(planned, after the CLI)*

> Status: the v1 engine and CLI work (arithmetic, relative dates, ranges, timezones, business days and holidays, recurrence, cron, `.ics` calendars, month grids, display formats, English and Spanish). Still missing: the web GUI ([plan 11](plans/plan-11-gui.md)). The build plan is in [`plans/`](plans/README.md).

## Usage

```sh
ahorita                       # prints now
ahorita now + 2d              # unquoted expression arguments are joined with spaces
ahorita -t date next month    # flags go before the expression
ahorita -- -06:00             # use -- before expressions starting with -
ahorita "today | end of day"  # quote | and * in a shell; commas don't need quotes
printf "today\ntomorrow\n" | ahorita  # evaluates non-empty lines from stdin
ahorita -o standup.ics 'every monday 9am, to ics'
```

- **Flags before the expression:** flag parsing stops at the first non-flag argument, so `ahorita now + 2d -t date` treats `-t date` as part of the expression. Put flags first (`ahorita -t date now + 2d`).
- **Expressions starting with `-`:** pass `--` to end flag parsing (`ahorita -- -06:00`).
- **Shell quoting:** `|` and `*` are interpreted by shells (`ahorita "today | end of day"`, `ahorita "2h * 3"`). Use `,` as a shell-safe pipe (`ahorita today, end of day`).
- **Stdin:** when no expression arguments are passed and stdin is not a terminal, every non-empty line on stdin is evaluated and printed on its own line. In a terminal with no arguments, `ahorita` evaluates `now`.

### Flags

| Flag | Config key | Values | Effect |
|---|---|---|---|
| `-t`, `--type` | `type` | `date`, `datetime`, `time`, `range`, `relative` | Output value type |
| `-f`, `--format` | `format` | pattern or preset (`iso`, `rfc3339`, `unix`, `human`, `short`) | Display format |
| `--tz` | `tz` | IANA zone (e.g. `America/Mexico_City`, `UTC`) | Default timezone (defaults to system zone) |
| `--lang` | `lang` | `en`, `es` | Default language (defaults from `LC_ALL` / `LANG`, else `en`) |
| `--strict` | `strict` | `true`, `false` (flag alone sets `true`) | Error on ambiguous input or out-of-order stages |
| `--day-week-start` | `week_start` | weekday name in English or Spanish | First day of the week (default `monday`) |
| `--week-one` | `week_one` | `iso`, `first-day`, `full`, `days` | First week of period rule (default `iso`) |
| `--month-overflow` | `month_overflow` | `clamp`, `overflow` | Month arithmetic end-of-month rule (default `clamp`) |
| `--count-inclusive` | `count_inclusive` | `true`, `false` (flag alone sets `true`) | Include both end dates when counting |
| `--country` | `country` | two-letter country code, or `none` | Holiday calendar (default `MX`; `none` disables holidays) |
| `--weekend` | `weekend` | weekday names (EN or ES, comma-separated), or `none` | Non-working weekend days (default `saturday,sunday`) |
| `--calendar` | `calendar` | path *(repeatable flag; comma-separated in env/config)* | Days-off file(s) (`#` comments, one date or whole-day span per line) |
| `-n` | — | — | Omit trailing newline |
| `-c` | — | — | Copy output (without trailing newline) to clipboard |
| `--json` | — | — | Print JSON (`Result` object, one per line) |
| `--explain` | — | — | Print evaluation stages to stderr (or inside JSON with `--json`) |
| `--config-file` | — | path | Config file path (errors if missing) |
| `-o`, `--output` | — | path | Write output to file instead of stdout |

### Environment variables & config file

Precedence is **flag > environment variable > config file > default**.

- Every setting in the table has an environment variable named `AHORITA_` + the config key in upper case: `AHORITA_TYPE`, `AHORITA_FORMAT`, `AHORITA_TZ`, `AHORITA_LANG`, `AHORITA_STRICT`, `AHORITA_WEEK_START`, `AHORITA_WEEK_ONE`, `AHORITA_MONTH_OVERFLOW`, `AHORITA_COUNT_INCLUSIVE`, `AHORITA_COUNTRY`, `AHORITA_WEEKEND`, `AHORITA_CALENDAR`.
- `AHORITA_CONFIG` overrides the config file path (errors if missing). Standard `TZ`, `LC_ALL`, and `LANG` (values starting with `es` default to Spanish) are also honored.
- Default config file path: `$XDG_CONFIG_HOME/ahorita/config.toml`, or `~/.config/ahorita/config.toml`. Missing default files are ignored.
- **Days-off files (`--calendar`):** plain text with one ahorita expression per line evaluating to a date (`2026-11-02`, `next friday`) or a whole-day span (`2026-12-24 .. 2027-01-02`), or an `.ics` file (all-day events and RRULEs within 1 year before to 5 years after `now`). Blank lines and `#` comments (full-line or inline) are ignored in text files. Passing `--calendar` on the CLI overrides `AHORITA_CALENDAR` and `calendar` in config.

Sample `~/.config/ahorita/config.toml`:

```toml
# Output formatting
# type = "date"
# format = "human"

# Locale and timezone
# tz = "America/Mexico_City"
# lang = "es"

# Evaluation rules
# strict = false
# week_start = "monday"
# week_one = "iso"
# month_overflow = "clamp"
# count_inclusive = false

# Business days & holidays
# country = "MX"
# weekend = "saturday,sunday"
# calendar = "~/.config/ahorita/days-off.txt"
```

## Output types

`-t, --type` picks the output shape. Without it, the type follows the result.

| Type       | Example output              |
|------------|-----------------------------|
| `date`     | `2026-11-16`                |
| `datetime` | `2026-11-16T14:30:00-06:00` |
| `time`     | `14:30`                     |
| `range`    | `2d 4h` / start → end       |
| `relative` | `in 2 days` / `3 weeks ago` |

The default display is ISO. `-f, --format` takes a pattern (`dd-MM-yyyy`, `d/M/yy`, `HH:mm`) or a preset (`iso`, `rfc3339`, `unix`, `human`, `short`).

**Presets**

| Preset    | Date (EN / ES)                          | Date-time (EN / ES)                                     | Time (EN / ES)      |
|-----------|-----------------------------------------|---------------------------------------------------------|---------------------|
| `iso`     | `2026-11-14`                            | `2026-11-14T14:30:00-06:00`                             | `14:30`             |
| `rfc3339` | `2026-11-14T00:00:00-06:00`             | `2026-11-14T14:30:00-06:00`                             | — *(error)*         |
| `unix`    | `1794636000` *(00:00 that day)*         | `1794688200`                                            | — *(error)*         |
| `human`   | `Sat, Nov 14, 2026` / `sáb 14 nov 2026` | `Sat, Nov 14, 2026, 2:30 PM` / `sáb 14 nov 2026, 14:30` | `2:30 PM` / `14:30` |
| `short`   | `11-14-2026` / `14-11-2026`             | `11-14-2026 14:30` / `14-11-2026 14:30`                 | `14:30`             |

**Pattern tokens**

| Token                       | Meaning      | Example                                      |
|-----------------------------|--------------|----------------------------------------------|
| `yyyy` / `yy`               | Year         | `2026` / `26`                                |
| `MMMM` / `MMM` / `MM` / `M` | Month        | `June` / `Jun` / `06` / `6`                  |
| `dd` / `d`                  | Day of month | `02` / `2`                                   |
| `EEEE` / `EEE`              | Weekday      | `Tuesday` / `Tue` *(ES: `martes` / `mar`)*   |
| `HH` / `H`                  | Hour (0–23)  | `09` / `9`                                   |
| `hh` / `h`                  | Hour (1–12)  | `02` / `2`                                   |
| `mm` / `m`                  | Minute       | `05` / `5`                                   |
| `ss` / `s`                  | Second       | `07` / `7`                                   |
| `a`                         | AM / PM      | `PM` *(ES: `p. m.`)*                         |
| `XXX`                       | Zone offset  | `-06:00`, `Z`                                |
| `'text'`                    | Literal text | `'T'`, `'de'` (`''` for `'`)                 |

## Expression examples

Stages are separated by `,` or `|` and run left to right, each applied to the previous result: `today | end of day | to UTC`. In a shell, `|` and `*` need quotes; the comma doesn't.

Unit letters: `s`, `m` (minute), `h`, `d`, `w`, `M` (month), `y`. Words always work (`min`, `months`, `días`).

**Arithmetic**
```
today
now - 1d
2026-06-02 + 2d
2026-06-02 - 1h
date1 - date2            → a length that remembers both ends
2h * 3
```

**Relative / natural language**
```
yesterday                → today - 1d (a date)
2d ago                   → now - 2d (keeps the time)
last week                → the previous Monday–Sunday (a span)
today - 1w, monday       → snap to that week's Monday
past monday
closest monday
end of day, today
end of week, this week
next month
in 3 months
in Q3
2nd week, H2
week 2
on aug 20th              → this year
on sept, past year
```

**Durations & ranges**
```
2h
90m                      → 1h 30m
during 2 days
during 6 weeks, starting …
<a> .. <b>
how many <unit> between <a> and <b>
```

**Recurrence & cron**
```
every monday
every monday 9am
every 2 days
every 2 weeks
every 90 minutes
every weekday
every day 2nd
every month 15th
every weekday 2nd
every day-year 2nd
every 2nd monday
every year aug 20
…, next <n>
…, to cron
…, to ics
…, to grid
cron "0 9 * * 1,3"
…, explain
```
By default `every …` and `cron "…"` list the next 5 occurrences; `next N` takes 1–1000. Export directly to a file with `ahorita -o standup.ics 'every monday 9am, to ics'`.

**Timezones**
```
<time> @ <place>         → read the time as that place's time
<date> in <place>
<date>, <tz>
…, to <timezone>         → convert
…, to UTC
now to Tokyo             → places (IANA city names, case- and accent-insensitive)
now to CDMX              → aliases (CDMX, NYC, DC, SF, LA, Tokio, Londres, …)
now to JST               → fixed-offset abbreviations (EST, PST, CET, JST, …; ambiguous like CST error)
now to +09:00            → offsets (+09:00, -06:00, +9, UTC+2, GMT-5)
```
Names containing a keyword (`"Port of Spain"`, `"Ciudad de México"`) need quotes.

**Business days**
```
<range>, to business days
<day>, next business day
next business day on or after <day>
<day> ± <n> business days
holidays MX 2026
```
By default, `MX` follows Federal Labour Law (LFT art. 74), including the October 1 presidential inauguration every 6 years starting in 2024; pass `--country none` (or another two-letter code) to change it. Election days are also legal rest days in Mexico (LFT art. 74, IX) with no fixed calendar rule — add those dates to a `--calendar` days-off file.

**Spanish**
```
hoy
ahora - 1d
ayer
mañana
hace 2 días
la semana pasada
próximo mes
en 3 meses
fin del día
```

## Feature list

- Add / subtract dates and times
- Diff two dates into a range
- Multiply durations (days, hours)
- Unit conversion (e.g. `90m` → `1h 30m`)
- Timezone conversion
- Relative dates
- English and Spanish input
- Business days
- Holidays lookup
- Recurring events / calendars
- Cron expression generation
- Custom display formats

## Build

Written in Go.

```sh
go build ./...
go test ./...
```

## License

TBD
