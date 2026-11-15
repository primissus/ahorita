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

> Status: the lexer (and starter normalizer) exists and nothing else does yet. Below is the target feature set. The build plan is in [`plans/`](plans/README.md).

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
every day 2nd
…, to cron
```

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
<day> ± <n> business days
```

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
