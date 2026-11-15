# Glossary & entities

## Domain terms
- **Expression** → the input string, e.g. `today - 1w, monday`.
- **Pipeline** → stages separated by `,` or `|`, run left to right; each stage is applied to the previous result: `today - 1w, monday`, `now | to UTC`. The first stage starts from `now`.
- **Stage** → one step of a pipeline. Either a value (`today - 1w`) or a modifier (`monday`, `end of day`, `to UTC`).
- **Snap** → a modifier that moves to a boundary or named point: `monday`, `end of day`, `end of week`, `closest monday`.
- **`today` vs `now`** → `today` is a date. `now` is an instant. `yesterday` = `today - 1d` (a date). `2d ago` = `now - 2d` (keeps the time).
- **Period** → a named stretch of the calendar (`last week`, `next month`, `Q3`, `sept`). Always evaluates to a span.
- **Interpret vs convert** → `2pm @ Tokyo` reads the clock time as Tokyo time. `now to Tokyo` shows the same instant on Tokyo's clock.
- **Output type** → `-t date|datetime|time|range|relative`. Without it, the type follows the result.
- **Display format** → Java-style pattern (`dd-MM-yyyy HH:mm`) or a preset (`iso`, `rfc3339`, `unix`, `human`). Default is ISO.
- **Business day** → a day that isn't a weekend day (Saturday and Sunday by default), a holiday in the active country (MX by default), or one of the user's own days off.
- **Strict mode** → `--strict`: any ambiguous input is an error instead of a best guess.
- **Canonical keyword** → the English word the normalizer maps every synonym and Spanish word to (`pasado` → `last`).
- **Lang** → the active language (`en` or `es`). Both vocabularies are always read; `lang` breaks ties and picks the output language.

## Main entities
- **Instant** → a point in time with a timezone.
- **Date** → a calendar day without a time.
- **Time** → a time of day without a date.
- **Duration** → calendar part (months, days) and fixed part (hours, minutes, seconds), kept apart. Calendar units (month, year, quarter) aren't fixed lengths.
- **Span (range)** → start + end (from `last week`, `during 2 days`, `in Q3`, `2nd week`, `A .. B`).
- **Recurrence** → a repeating rule (`every monday`, `every day 2nd`), stored as an RRULE. Can be converted to a cron expression.
- **Holiday calendar** → a set of non-working dates for a country (`MX` by default via LFT art. 74; other countries from `rickar/cal/v2`, or empty when `none`).
- **Days-off file (`--calendar`)** → plain-text file of custom non-working dates or whole-day spans evaluated line-by-line with `expr.Eval` (ignoring blank lines and `#` comments).

## Unit letters
Case-sensitive where it matters: `m` = minute, `M` = month. Others: `s`, `h`, `d`, `w`, `q`, `y` (case-insensitive per C.4 ★).

## Acronyms & internal names
- **Q1–Q4** → calendar quarters.
- **H1/H2** → calendar half-years.
- **DoW** → day of week.
- **tz** → IANA timezone (`America/Mexico_City`), or a place name, alias, abbreviation, or fixed offset resolved to one.
- **★** → the recommended option in a questions file.
