# ahorita Web GUI — Screen & Interaction Design (`web/DESIGN.md`)

> Status: **Awaiting user approval before Step 4 (App scaffold)**  
> Architecture: React + TypeScript + Go engine compiled to WebAssembly (`web/public/ahorita.wasm` via `ahoritaRun(input, optionsJSON)`).  
> Rule: Zero date math in JavaScript. The UI builds expression strings and option payloads, sends them to `ahoritaRun`, and renders `Result`.

---

## 1. Layout Overview

### Desktop (`>= 960px`): Two-Pane Calculator Workspace

```
+----------------------------------------------------------------------------------+
|  ahorita   [America/Mexico_City v]        [EN | ES]  [Theme: Auto]  [Settings]   |
+----------------------------------------------------------------------------------+
|  EXPRESSION & RESULT (Left / Top)         |  GROUPED KEYPAD (Right)              |
|  +-------------------------------------+  |  [Core] [Rel/Range] [Zones]          |
|  | > now + 2d                      [C] |  |  [Periods] [BizDays] [Recur/Cron]    |
|  |   ~~~~~~~~~~~~~~~~~~~~~~~~~~        |  |  +--------------------------------+  |
|  | Hint: complete expression           |  |  | Values: now today tomorrow yest  |  |
|  +-------------------------------------+  |  | Ops:    +  -  *  ,  |  ..  @  to |  |
|  | [Copy Expr] [Copy Result] [Share]   |  |  | Units:  s  m  h  d  w  M  q  y   |  |
|  +-------------------------------------+  |  |         business days  work week |  |
|  | RESULT                   [datetime] |  |  | Pad:    7  8  9  :  /            |  |
|  | 2026-11-16T14:30:00-06:00           |  |  |         4  5  6  "  ⌫            |  |
|  | ISO: ...  |  Unix: 1794861000       |  |  |         1  2  3  0  .            |  |
|  +-------------------------------------+  |  +--------------------------------+  |
|  | ! Note (if any): DST/weekday shift  |  |  | Active Category Tray (e.g.       |  |
|  +-------------------------------------+  |  | Rel/Range, BizDays, Recur/Cron)  |  |
|  | Type: [Auto][date][dt][time][rng][rel] |  +--------------------------------+  |
|  | Format: [iso v]  [ ] Explain        |  |  | Quick Pickers:                   |  |
|  +-------------------------------------+  |  | [Date Picker] [Time Picker]      |  |
|  | EXPLAIN DRAWER (collapsible)        |  |  | [Zone Insert Picker]             |  |
|  | 1. now        -> 2026-11-14T14:30   |  +--+--------------------------------+  |
|  | 2. + 2d       -> 2026-11-16T14:30   |                                         |
|  +-------------------------------------+                                         |
+----------------------------------------------------------------------------------+
```

### Phone (`360px – 639px`): Stacked Single-Column

- **Sticky top card**: compact header (`ahorita`, zone chip, `EN/ES`, theme, settings) + expression input + live result box (always visible without scrolling past buttons).
- **Horizontally scrollable pill row**: Type (`Auto`, `date`, `datetime`, `time`, `range`, `relative`) + Format selector + Explain toggle.
- **Pickers row**: compact Date, Time, and Zone buttons opening native/popover pickers.
- **Keypad**: 4-column touch-friendly grid (`>= 44×44px` targets) with segmented category tabs (`Core`, `Rel/Range`, `Periods`, `Zones`, `BizDays`, `Recur`) so the user never has to scroll across 70+ buttons to reach `every` or `business days`.

---

## 2. Display, Result Card, Notes & Explain Drawer

### 2.1 Expression Bar
- Single editable `<input type="text">` (or auto-growing single/two-line `<textarea>` without newlines) in monospace font (`font-variant-numeric: tabular-nums`), supporting normal typing, selection, caret movement, paste, and keyboard editing.
- **Button/picker insertion rule**: inserts at the current selection/caret range (with automatic spacing around word tokens so clicking `today`, `+`, `2`, `d` produces `today + 2d` cleanly, while digits + units like `2` followed by `d` join as `2d`).
- **Error caret / underline**: when `Result.Error` has `pos >= 0`, render a subtle synchronized underline/marker under the character/token at UTF-8 byte offset `pos` (converted safely from UTF-8 bytes to UTF-16 code units in JS).

### 2.2 Result Card
- **Primary output**: `Result.Text` in crisp monospace.
  - Single values (`date`, `datetime`, `time`, `duration`, `span`, `number`): large primary line.
  - Multi-line values (`list` such as `holidays MX 2026` or `next 5`, and `text` such as `to grid` or `to ics`): preformatted scrollable `<pre>` block preserving line alignment (`to grid` requires monospace alignment).
  - Empty expression `""`: shows friendly placeholder (`Type or tap an expression, e.g. "now + 2d"`), no red error.
- **Kind badge**: displays `Result.Kind` (`date`, `datetime`, `time`, `duration`, `span`, `number`, `list`, `text`, `recurrence`).
- **Secondary metadata line** (compact, muted):
  - `ISO`: shown when `Result.ISO` differs from `Result.Text` (e.g., when `-f human` or `-t relative` is active).
  - `Unix`: shown when `Result.Unix !== undefined`.
  - `Range bounds`: `Result.Start → Result.End` when present (spans and date-diff durations).

### 2.3 Incomplete vs. Error Feedback
- **Incomplete expression** (e.g., `"now +"`, `"today,"`, `"2026-06-02 to"`, `"how many"`):
  - `Result.Error` has `Expected` non-empty and the error position is at/near the end of input (`"unexpected end of input"`).
  - Styled as a **neutral/informational syntax hint** (`Expected: a value` / clickable suggestion chips), **not** a harsh error banner.
- **Invalid expression / semantic error** (e.g., `"now + 2x"`, `"today + today"`, `"3 months to days"`):
  - Styled as a clear **error alert** showing `Result.Error.message`, position underline, and `Expected` chips when provided.

### 2.4 Notes Banner (Plan 10 DST / Cross-Zone Warnings)
- Whenever `Result.Note` is non-empty (e.g., `every monday 8pm, to UTC, to cron` -> `"weekday shifts from Monday to Tuesday in UTC; America/Mexico_City does not observe DST while UTC does"` or DST differences), display an amber informational banner directly beneath the result card with an info icon and `role="status"`.

### 2.5 Explain Drawer
- Toggled via the **Explain** switch (or `Alt+E`).
- Sets `options.explain = true` in `ahoritaRun`.
- Renders `Result.Explain` as an ordered table/list: `Stage` (`today - 1w`) -> `Value` (`2026-11-07`), also showing how lenient pipe reordering (`C.2`) interpreted out-of-order stages.

---

## 3. Grouped Keypad Specification

Buttons always insert valid engine syntax. Pitfall rule: **display labels can be typographic (`×`, `−`, `bd`), while inserted text MUST match engine grammar (`*`, `-`, `business days`).**

### Group A — Values & Core Pad (always pinned / `Core` tab)
| Group | Button Label | Inserts | Notes |
|---|---|---|---|
| Values | `now`, `today`, `tomorrow`, `yesterday` | word + space (ES mode optional, see §11 Q1) | Base anchors |
| Numbers | `0`–`9`, `.` | digit / `.` (no leading space after digit) | Joins with units (`2d`) |
| Date/Time Punct | `:`, `/`, `"` | `:`, `/`, `"` | Clock `14:30`, slash `02/06/2026`, quoted cron/place |
| Operators | `+`, `−`, `×` | ` + `, ` - `, ` * ` | ASCII `-` and `*` |
| Pipeline / Span | `,`, `\|`, `..` | `, `, ` \| `, ` .. ` | Stage pipe and literal span |
| Editing | `Space`, `⌫`, `Clear` | space, delete token/char, reset | Always enabled |

### Group B — Units (`Core` tab)
| Button Label | Inserts | Meaning |
|---|---|---|
| `s` | `s` | seconds |
| `m` | `m` | **minutes** (lowercase `m`) |
| `h` | `h` | hours |
| `d` | `d` | days |
| `w` | `w` | weeks |
| `M` | `M` | **months** (uppercase `M` — tooltip + `aria-label` highlight `m` vs `M`) |
| `q` | `q` | quarters |
| `y` | `y` | years |
| `bd` | `business days` | business day unit (`"business day"` / `"business days"` — engine has no `bd` letter suffix) |
| `ww` | `work week` | Monday–Friday work week |

### Group C — Relative, Boundaries & Ranges (`Rel / Range` tab)
- **Relative**: `last`, `next`, `this`, `closest`, `ago`, `in`, `inclusive`
- **Boundaries**: `start of`, `end of`
- **Ranges & Counting**: `from`, `to`, `between`, `and`, `during`, `starting`, `how many`

### Group D — Periods, Weekdays & Months (`Periods` tab)
- **Period units**: `second`, `minute`, `hour`, `day`, `week`, `month`, `quarter`, `half`, `year`
- **Quarters / Halves**: `Q1`, `Q2`, `Q3`, `Q4`, `H1`, `H2`
- **Weekdays**: `Mon`, `Tue`, `Wed`, `Thu`, `Fri`, `Sat`, `Sun` (inserts `monday`..`sunday`)
- **Months**: `Jan`..`Dec` (inserts `january`..`december`)
- **Ordinals**: `1st`, `2nd`, `3rd`, `4th`, `5th`, `of` (for `2nd week, H2` and `every 2nd monday`)

### Group E — Timezones & Conversions (`Zones` tab)
- **Operators**: `@` (interpret as zone), `to` (convert / stage target), `in` (interpret or convert by LHS)
- **Quick Zones**: `UTC`, `CDMX`, `NYC`, `Tokyo`, `London`, plus common fixed offsets (`-06:00`, `+09:00`) and the **Zone Search Picker** (`Intl.supportedValuesOf('timeZone')`).

### Group F — Business Days & Holidays (`BizDays` tab — Plan 9)
- **Keywords**: `business day`, `business days`, `work week`, `on or after`, `on or before`, `holidays`
- **Quick Templates** (optional chips, see §11 Q2):
  - `next business day on or after`
  - `last business day on or before`
  - `, to business days`
  - `, next business day`
  - `holidays MX 2026` (or active configured country + current year)

### Group G — Recurrence & Cron (`Recur / Cron` tab — Plan 10)
- **Recurrence**: `every`, `weekday`, `day-year`, `cron "..."` (inserts `cron ""` and places caret inside quotes)
- **Pipeline Outputs**: `next 5` (editable count), `explain`, `to cron`, `to ics`, `to grid`

---

## 4. Pickers & Synchronization with Options / Tokens

1. **Date Picker (`<input type="date">`)**:
   - Picking a date (e.g., `2026-06-02`) inserts `2026-06-02` at the expression caret (or replaces an immediately surrounding/selected `YYYY-MM-DD` token if the user clicked the picker while the cursor is inside a date token).
   - Never computes dates in JS; the native calendar only emits an ISO date string.
2. **Time Picker (`<input type="time">`)**:
   - Picking `14:30` inserts `14:30` at the caret (or replaces an existing `HH:MM` token under the caret).
3. **Timezone Picker (Searchable Combobox)**:
   - Populated from `Intl.supportedValuesOf('timeZone')` (plus `UTC` and common aliases like `CDMX`, `NYC`).
   - **Header Zone Selector**: sets `options.tz` (default evaluation timezone, persisted to `localStorage`). Never falls back to `"Local"` or silent UTC; initializes from `Intl.DateTimeFormat().resolvedOptions().timeZone` (or `"UTC"` ONLY if browser `Intl` returns undefined, with an explicit visible notice).
   - **Expression Zone Picker (in Zones tab / toolbar)**: inserts the chosen IANA name (quoted if containing spaces/keywords, e.g., `"Port of Spain"`) at the caret — e.g., after `to ` or `@ `.
4. **Output Options (`Type`, `Format`, `Lang`)**:
   - **Type (`options.type`)**: `""` (Auto) | `"date"` | `"datetime"` | `"time"` | `"range"` | `"relative"` — updates `optionsJSON` immediately and triggers `ahoritaRun`.
   - **Format (`options.format`)**: `""` / `"iso"` | `"rfc3339"` | `"unix"` | `"human"` | `"short"` | custom Java-style pattern string (e.g. `dd-MM-yyyy HH:mm`). Selecting custom reveals a pattern text box + clickable token chips (`yyyy`, `MM`, `dd`, `EEE`, `HH`, `mm`, `a`, `XXX`).
   - **Language (`options.lang`)**: `"en"` | `"es"` — sets `options.lang` (affects tie-breaking like `ago`, slash-date order `MM/DD` vs `DD/MM`, and output language when expression has no language-specific words).
5. **Settings Drawer (`localStorage`-persisted config)**:
   - Exposes all remaining CLI config keys directly matching `internal/settings`:
     - `week_start` (`monday`..`sunday`, default `monday`)
     - `week_one` (`iso` | `first-day` | `full` | `days`, default `iso`)
     - `month_overflow` (`clamp` | `overflow`, default `clamp`)
     - `count_inclusive` (checkbox, default `false`)
     - `strict` (checkbox, default `false`)
     - `country` (2-letter code list + `none`, default `MX`)
     - `weekend` (weekday multi-select toggles, default `saturday,sunday`; empty -> `"none"`)
     - `days_off` (multiline textarea, one expression/range per line, e.g. `2026-12-24 .. 2027-01-02`, sent in `options.days_off`)

---

## 5. `Result.Error.Expected`-Driven Button Enabling

The engine (`expr/parser.go`) populates `Result.Error.Expected` on syntax errors with exact strings. The keypad controller uses two modes:

1. **When `input.trim() === ""` or `Result.Error` has `Expected` non-empty**:
   Enable buttons matching the union of `Result.Error.Expected` tokens:

   | `Result.Error.Expected` value | Enabled keypad sets / buttons |
   |---|---|
   | `"a value"` | Values (`now`, `today`, ...), Numbers (`0–9`), Date/Time pickers, Relative (`last`, `next`, `this`, `closest`, `ago`, `in`), Boundaries (`start of`, `end of`), Periods/Weekdays/Months, `during`, `from`, `between`, `how many`, `holidays`, `every`, `cron "..."` |
   | `"a unit"` | Units (`s`, `m`, `h`, `d`, `w`, `M`, `q`, `y`, `business days`, `work week`) + Period unit names (`second`..`year`) |
   | `"a timezone"` | `UTC`, quick zone chips, Zone insert picker, offset signs (`+`, `-`) & digits |
   | `"a unix timestamp"` | Digits `0–9` |
   | `"a duration"` | Digits `0–9`, Units (`s..y`, `business days`) |
   | `"a period"` | `week`, `month`, `quarter`, `half`, `year`, `Q1–Q4`, `H1–H2`, Weekdays, Months |
   | `"a weekday"` | `monday`..`sunday` |
   | `"a month"` | `january`..`december` |
   | `"a week number"` | Digits `0–9` |
   | `"a time"` | Digits `0–9`, `:`, `am`/`pm`, Time picker |
   | `"a quoted 5-field cron string"` | `"` quote key + quick cron preset hints (`"0 9 * * 1,3"`) |
   | `"last"`, `"next"`, `"this"`, `"week"`, `"of"`, `"to"`, `"and"`, `"between"`, `"from"` | Those exact keyword buttons |

   *Fallback*: if `Result.Error` is a **mid-expression or semantic error** (`Expected` empty, e.g., `"unknown unit"` or `"cannot add a date to a date"`), keep editing keys (`⌫`, `Clear`, Digits, Operators) enabled so the user isn't locked out while fixing the expression.

2. **When `Result.Error == null` (complete valid expression)**:
   Enable the fixed **"after a complete value"** set:
   - Binary/pipeline operators: `+`, `-`, `*`, `,`, `|`, `..`, `@`, `to`, `in`, `inclusive`
   - Modifiers valid after values: `ago` (after durations), clock/time picker (after dates), `explain`, `next <N>`, `to cron`, `to ics`, `to grid`, `to business days`
   - Starting value buttons (`today`, `tomorrow`, `every`, etc.) are disabled *unless* a trailing operator/separator (`,`, `|`, `+`, `-`, `..`, `to`, `from`, `between`, `and`) is present, preventing accidental `today tomorrow`.
   - **Crucial rule**: Keyboard typing in the expression input is **never** blocked, regardless of button disabled states.

---

## 6. Copy, `.ics` Download & Shareable URL State

- **Copy Expression**: copies `input` to `navigator.clipboard`.
- **Copy Result**: copies `Result.Text` to `navigator.clipboard`.
- **Download `.ics`**:
  - When `Result.Kind === "text"` and `Result.Text.startsWith("BEGIN:VCALENDAR")`, display a primary **Download `.ics`** button that downloads `ahorita.ics` (`text/calendar;charset=utf-8`).
  - When `Result.Kind` is `"recurrence"`, `"date"`, `"datetime"`, or `"span"`, offer a secondary **Export `.ics`** button that runs `ahoritaRun(input + ", to ics", opts)` and triggers the download directly without mutating the user's expression (or appends `, to ics` — see §11 Q4).
- **Share Link**:
  - Syncs expression to URL query parameter `?q=<encoded>` via `history.replaceState` (and optionally non-default `t`, `f`, `tz`, `lang` query params — see §11 Q3).
  - Clicking **Share** copies the full URL and announces `"Share link copied"`.

---

## 7. Light / Dark Theme

- CSS variables in `src/index.css` for surfaces, borders, monospace display, accent, warning note (`Note`), syntax hint, and error colors.
- Three-state theme preference (`system` default -> `light` -> `dark`), stored in `localStorage`.
- `system` follows `prefers-color-scheme`; manual choice sets `data-theme="light" | "dark"` on `<html>`.
- Contrast target: >= 4.5:1 text contrast and >= 3:1 interactive border/focus-ring contrast in both themes.

---

## 8. Keyboard Navigation & Shortcuts

- Full native typing, caret movement, selection, and clipboard in the expression field.
- Shortcuts (ignored when inside modal inputs except `Escape`, or modifier-based):
  - `/` or `Ctrl/Cmd + K`: focus expression bar
  - `Ctrl/Cmd + Shift + C`: copy `Result.Text`
  - `Escape`: close active popover/drawer first; if none open, clear expression
  - `Alt + E`: toggle Explain drawer
- Logical tab order: Header controls -> Expression bar -> Action buttons -> Output options -> Pickers -> Keypad tabs (`role="tablist"`) -> Active keypad buttons (`ArrowLeft/Right/Up/Down` optional roving or standard Tab).

---

## 9. Screen-Reader Accessibility

- Semantic landmarks: `<header>`, `<main>`, `<nav aria-label="Keypad categories">`, `<section aria-label="Expression and result">`.
- `aria-live="polite"` status region announcing evaluated `Result.Text` (debounced ~250ms while typing to avoid chatter) and error/copy announcements.
- Every symbol/abbreviated button has explicit `aria-label` (e.g., `×` -> `"Multiply"`, `−` -> `"Minus"`, `M` -> `"Months (capital M)"`, `m` -> `"Minutes (lowercase m)"`, `|` -> `"Pipe stage"`, `..` -> `"Range separator"`, `⌫` -> `"Backspace"`).
- Disabled keypad buttons use `disabled` or `aria-disabled="true"` with clear visual contrast + reason hint when hovered/focused; visible `:focus-visible` outline (`2px solid`, `2px` offset) everywhere.

---

## 10. Responsive Breakpoints (down to `360px`)

- `>= 960px` (Desktop): 2-column grid (`minmax(360px, 1.1fr) minmax(360px, 0.9fr)`), Display/Result/Options on left, Keypad/Pickers on right.
- `640px – 959px` (Tablet): single centered column (`max-width: 720px`), Display/Result on top, Pickers + Keypad below.
- `360px – 639px` (Phone): compact sticky Display + Result header, horizontal scroll pill bars for Type/Format and Keypad tabs, 4-column core pad with `>= 44px` button height, `box-sizing: border-box` and `overflow-x: hidden` tested at `360px` width.

---

## 11. Open UI Design Questions for User Review

Please confirm or pick a preference on these before Step 4 starts:

1. **Keypad button language when `lang = es`:**
   - **A ★** Keypad labels and inserted keywords follow the active `lang` setting (`hoy`, `mañana`, `hace`, `días hábiles`, `cada`, `calendario`) — with English always still accepted if typed.
   - **B** Keypad always inserts canonical English (`today`, `tomorrow`, `business days`, `every`); Spanish works when typed.
2. **Compound template chips vs. atomic keyword buttons only:**
   - **A ★** Show atomic keyword buttons by default, plus a compact row of 3–4 helpful template chips in `BizDays` / `Recur` (e.g., `next business day on or after`, `holidays MX 2026`, `every monday 9am`).
   - **B** Strict atomic token buttons only (no multi-word template chips).
3. **What the Share URL carries:**
   - **A ★** Carry `?q=<expr>` plus any non-default display options (`&t=`, `&f=`, `&tz=`, `&lang=`) so a shared link looks identical for the recipient.
   - **B** Carry only `?q=<expr>` (keeps URLs short; recipient uses their own local settings/timezone).
4. **`.ics` Export button on non-`.ics` results (e.g., `every monday 9am`):**
   - **A ★** Offer a one-click **"Download `.ics`"** action in the result action bar (evaluates `expr + ", to ics"` under the hood without altering the user's expression box), in addition to showing it whenever the result itself is `to ics`.
   - **B** Show **Download `.ics`** only when the expression explicitly ends with `, to ics`.
5. **Clicking a starting value (`today`) while a complete expression (`now`) is already in the box:**
   - **A ★** Disable starting-value buttons when the expression is complete without a trailing operator (strict `expected`-driven behavior from `plan-11-gui.md`), leaving `Clear` or operators active.
   - **B** Keep starting-value buttons enabled and automatically clear/replace the expression when clicked after a complete result (like pressing a number on a pocket calculator after `=`).
