package expr

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type readmeCase struct {
	line    string
	input   string
	want    string
	wantErr string
	skipBug string
}

// readmeCases mirrors every code-block line under "## Expression examples" in
// README.md, in order, replacing placeholders with concrete inputs.
var readmeCases = []readmeCase{
	// Arithmetic
	{line: "today", input: "today", want: "2026-11-14"},
	{line: "now - 1d", input: "now - 1d", want: "2026-11-13T14:30:00-06:00"},
	{line: "today + 3 days", input: "today + 3 days", want: "2026-11-17T14:30:00-06:00"},
	{line: "2026-06-02 + 2d", input: "2026-06-02 + 2d", want: "2026-06-04"},
	// Subtracting 1h from a bare date promotes it to midnight 00:00:00 first.
	{line: "2026-06-02 - 1h", input: "2026-06-02 - 1h", want: "2026-06-01T23:00:00-06:00"},
	// Nov has 30 days: (30 - 14) + 24 = 40 days.
	{line: "date1 - date2            → a length that remembers both ends", input: "2026-12-24 - 2026-11-14", want: "40d"},
	// 2 hours * 3 = 6 hours.
	{line: "2h * 3", input: "2h * 3", want: "6h"},
	{line: "2 * (1h + 30m)", input: "2 * (1h + 30m)", want: "3h"},

	// Relative / natural language
	{line: "yesterday                → today - 1d (a date)", input: "yesterday", want: "2026-11-13"},
	{line: "2d ago                   → now - 2d (keeps the time)", input: "2d ago", want: "2026-11-12T14:30:00-06:00"},
	{line: "last week                → the previous Monday–Sunday (a span)", input: "last week", want: "2026-11-02 → 2026-11-08"},
	{line: "today - 1w, monday       → snap to that week's Monday", input: "today - 1w, monday", want: "2026-11-02"},
	{line: "past monday", input: "past monday", want: "2026-11-09"},
	{line: "closest monday", input: "closest monday", want: "2026-11-16"},
	{line: "end of day, today", input: "end of day, today", want: "2026-11-14T23:59:59-06:00"},
	{line: "EoD                      → end of day (also SoD, EoW, SoW, EoM, SoM, EoQ, SoQ, EoY, SoY)", input: "EoD", want: "2026-11-14T23:59:59-06:00"},
	{line: `end day                  → "of" is optional (start day, end minute, start month, end year)`, input: "end day", want: "2026-11-14T23:59:59-06:00"},
	{line: "end of week, this week", input: "end of week, this week", want: "2026-11-15T23:59:59-06:00"},
	{line: "next month", input: "next month", want: "2026-12-01 → 2026-12-31"},
	{line: "in 3 months", input: "in 3 months", want: "2027-02-14T14:30:00-06:00"},
	{line: "in Q3", input: "in Q3", want: "2026-07-01 → 2026-09-30"},
	{line: "2nd week, H2", input: "2nd week, H2", want: "2026-07-06 → 2026-07-12"},
	{line: "week 2", input: "week 2", want: "2026-01-05 → 2026-01-11"},
	{line: "on aug 20th              → this year", input: "on aug 20th", want: "2026-08-20"},
	{line: "on sept, past year", input: "on sept, past year", want: "2025-09-01 → 2025-09-30"},

	// Durations & ranges
	{line: "2h", input: "2h", want: "2h"},
	{line: "90m                      → 1h 30m", input: "90m", want: "1h 30m"},
	{line: "180s in hours            → 0.05 (direct conversion; also millis, halfs, quarters)", input: "180s in hours", want: "0.05"},
	{line: "yesterday 2pm - now, in hours", input: "yesterday 2pm - now, in hours", want: "-24.5"},
	{line: "during 2 days", input: "during 2 days", want: "2026-11-14 → 2026-11-15"},
	{line: "during 6 weeks, starting …", input: "during 6 weeks, starting next monday", want: "2026-11-16 → 2026-12-27"},
	// Closed inclusive date range 2026-11-16 .. 2026-11-20.
	{line: "<a> .. <b>", input: "2026-11-16 .. 2026-11-20", want: "2026-11-16 → 2026-11-20"},
	{line: "how many <unit> between <a> and <b>", input: "how many days between 2026-11-16 and 2026-11-20", want: "4"},
	{line: "days to 2027-01-01       → shorthand count from today/now", input: "days to 2027-01-01", want: "48"},
	{line: "days to 2027", input: "days to 2027", want: "48"},
	{line: "days from 2026-11-16 to 2026-11-20", input: "days from 2026-11-16 to 2026-11-20", want: "4"},

	// Conditionals & aliases
	{line: "if today is friday, from today to next friday, else from last friday to this friday", input: "if today is friday, from today to next friday, else from last friday to this friday", want: "2026-11-13 → 2026-11-13"},
	{line: "oncall week = if today is friday, from today to next friday, else from last friday to this friday", input: "oncall week = if today is friday, from today to next friday, else from last friday to this friday", want: "2026-11-13 → 2026-11-13"},

	// Recurrence & cron
	{
		line:  "every monday",
		input: "every monday",
		want:  "2026-11-16\n2026-11-23\n2026-11-30\n2026-12-07\n2026-12-14",
	},
	{
		line:  "every monday 9am",
		input: "every monday 9am",
		want:  "2026-11-16T09:00:00-06:00\n2026-11-23T09:00:00-06:00\n2026-11-30T09:00:00-06:00\n2026-12-07T09:00:00-06:00\n2026-12-14T09:00:00-06:00",
	},
	{
		line:  "every 2 days",
		input: "every 2 days",
		want:  "2026-11-16\n2026-11-18\n2026-11-20\n2026-11-22\n2026-11-24",
	},
	{
		line:  "every 2 weeks",
		input: "every 2 weeks",
		want:  "2026-11-28\n2026-12-12\n2026-12-26\n2027-01-09\n2027-01-23",
	},
	{
		line:  "every 90 minutes",
		input: "every 90 minutes",
		want:  "2026-11-14T15:00:00-06:00\n2026-11-14T16:30:00-06:00\n2026-11-14T18:00:00-06:00\n2026-11-14T19:30:00-06:00\n2026-11-14T21:00:00-06:00",
	},
	{
		line:  "every weekday",
		input: "every weekday",
		want:  "2026-11-16\n2026-11-17\n2026-11-18\n2026-11-19\n2026-11-20",
	},
	{
		line:  "every day 2nd",
		input: "every day 2nd",
		want:  "2026-12-02\n2027-01-02\n2027-02-02\n2027-03-02\n2027-04-02",
	},
	{
		line:  "every month 15th",
		input: "every month 15th",
		want:  "2026-11-15\n2026-12-15\n2027-01-15\n2027-02-15\n2027-03-15",
	},
	{
		line:  "every weekday 2nd",
		input: "every weekday 2nd",
		want:  "2026-11-17\n2026-11-24\n2026-12-01\n2026-12-08\n2026-12-15",
	},
	{
		line:  "every day-year 2nd",
		input: "every day-year 2nd",
		want:  "2027-01-02\n2028-01-02\n2029-01-02\n2030-01-02\n2031-01-02",
	},
	{
		line:  "every 2nd monday",
		input: "every 2nd monday",
		want:  "2026-12-14\n2027-01-11\n2027-02-08\n2027-03-08\n2027-04-12",
	},
	{
		line:  "every year aug 20",
		input: "every year aug 20",
		want:  "2027-08-20\n2028-08-20\n2029-08-20\n2030-08-20\n2031-08-20",
	},
	{
		line:  "…, next <n>",
		input: "every monday, next 2",
		want:  "2026-11-16\n2026-11-23",
	},
	{
		line:  "…, to cron",
		input: "every monday, to cron",
		want:  "0 0 * * 1",
	},
	{
		line:  "…, to ics",
		input: "every monday 9am, to ics",
		want:  "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//ahorita//EN\r\nBEGIN:VEVENT\r\nUID:11b234c3677a4ecd1f8cec038e461b07@ahorita\r\nDTSTAMP:20261114T203000Z\r\nDTSTART;TZID=America/Mexico_City:20261116T090000\r\nSUMMARY:every monday 09:00\r\nRRULE:FREQ=WEEKLY;BYDAY=MO;BYHOUR=9;BYMINUTE=0;BYSECOND=0\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n",
	},
	{
		line:  "…, to grid",
		input: "today, to grid",
		want:  "   November 2026\nMo Tu We Th Fr Sa Su\n                   1\n 2  3  4  5  6  7  8\n 9 10 11 12 13 14*15\n16 17 18 19 20 21 22\n23 24 25 26 27 28 29\n30",
	},
	{
		line:  `cron "0 9 * * 1,3"`,
		input: `cron "0 9 * * 1,3"`,
		want:  "2026-11-16T09:00:00-06:00\n2026-11-18T09:00:00-06:00\n2026-11-23T09:00:00-06:00\n2026-11-25T09:00:00-06:00\n2026-11-30T09:00:00-06:00",
	},
	{
		line:  "…, explain",
		input: `cron "0 9 * * 1,3", explain`,
		want:  "At 09:00 on Monday and Wednesday",
	},

	// Timezones
	{line: "<time> @ <place>         → read the time as that place's time", input: "2pm @ Tokyo", want: "2026-11-14T14:00:00+09:00"},
	{line: "<date> in <place>", input: "2026-11-14 14:30 in Tokyo", want: "2026-11-14T14:30:00+09:00"},
	{line: "<date>, <tz>", input: "2026-11-14 14:30, America/New_York", want: "2026-11-14T14:30:00-05:00"},
	{line: "…, to <timezone>         → convert", input: "now, to Asia/Tokyo", want: "2026-11-15T05:30:00+09:00"},
	// 14:30-06:00 equals 20:30Z.
	{line: "…, to UTC", input: "now, to UTC", want: "2026-11-14T20:30:00Z"},
	{line: "now to Tokyo             → places (IANA city names, case- and accent-insensitive)", input: "now to Tokyo", want: "2026-11-15T05:30:00+09:00"},
	{line: "now to Los Angeles       → unquoted multi-word places", input: "now to Los Angeles", want: "2026-11-14T12:30:00-08:00"},
	// CDMX is America/Mexico_City (UTC-6), keeping the 14:30 local wall time.
	{line: "now to CDMX              → aliases (CDMX, NYC, DC, SF, LA, Tokio, Londres, …)", input: "now to CDMX", want: "2026-11-14T14:30:00-06:00"},
	{line: "now to california        → states, countries & major cities (seattle, guadalajara, spain, japan, …)", input: "now to california", want: "2026-11-14T12:30:00-08:00"},
	{line: "now to los_angels        → fuzzy prefix, substring & typo matching (rejected under --strict)", input: "now to los_angels", want: "2026-11-14T12:30:00-08:00"},
	{line: "(now - 1 business day) to los_angeles", input: "(now - 1 business day) to los_angeles", want: "2026-11-13T12:30:00-08:00"},
	// JST is fixed UTC+9: 20:30Z + 9h = next day 05:30+09:00.
	{line: "now to JST               → fixed-offset abbreviations (EST, PST, CET, JST, …; ambiguous like CST error)", input: "now to JST", want: "2026-11-15T05:30:00+09:00"},
	// +09:00 matches JST: 20:30Z + 9h = 2026-11-15T05:30:00+09:00.
	{line: "now to +09:00            → offsets (+09:00, -06:00, +9, UTC+2, GMT-5)", input: "now to +09:00", want: "2026-11-15T05:30:00+09:00"},

	// Business days
	{line: "<range>, to business days", input: "2026-11-20 - 2026-11-16, to business days", want: "4"},
	{line: "<day>, next business day", input: "2026-11-14, next business day", want: "2026-11-17"},
	{line: "next business day on or after <day>", input: "next business day on or after 2026-11-17", want: "2026-11-17"},
	{line: "<day> ± <n> business days", input: "2026-11-13 + 3 business days", want: "2026-11-19"},
	{
		line:  "holidays MX 2026",
		input: "holidays MX 2026",
		want: "2026-01-01  New Year's Day\n" +
			"2026-02-02  Constitution Day\n" +
			"2026-03-16  Benito Juárez's Birthday\n" +
			"2026-05-01  Labour Day\n" +
			"2026-09-16  Independence Day\n" +
			"2026-11-16  Revolution Day\n" +
			"2026-12-25  Christmas Day",
	},

	// Spanish
	{line: "hoy", input: "hoy", want: "2026-11-14"},
	{line: "ahora - 1d", input: "ahora - 1d", want: "2026-11-13T14:30:00-06:00"},
	{line: "ayer", input: "ayer", want: "2026-11-13"},
	// today is 2026-11-14, so tomorrow is 2026-11-15.
	{line: "mañana", input: "mañana", want: "2026-11-15"},
	// now is 2026-11-14T14:30:00-06:00, so 2 days ago keeps 14:30 -> 2026-11-12T14:30:00-06:00.
	{line: "hace 2 días", input: "hace 2 días", want: "2026-11-12T14:30:00-06:00"},
	{line: "la semana pasada", input: "la semana pasada", want: "2026-11-02 → 2026-11-08"},
	{line: "próximo mes", input: "próximo mes", want: "2026-12-01 → 2026-12-31"},
	{line: "en 3 meses", input: "en 3 meses", want: "2027-02-14T14:30:00-06:00"},
	// Bare "fin del día" uses today's date (2026-11-14T23:59:59-06:00).
	{line: "fin del día", input: "fin del día", want: "2026-11-14T23:59:59-06:00"},
	// Saturday 2026-11-14 is not Friday, so takes the "sino lunes" branch -> 2026-11-09.
	{line: "si hoy es viernes, hoy, sino lunes", input: "si hoy es viernes, hoy, sino lunes", want: "2026-11-09"},
}

func TestReadme(t *testing.T) {
	ctx := refCtx(t)
	for _, tc := range readmeCases {
		t.Run(tc.line, func(t *testing.T) {
			if tc.skipBug != "" {
				t.Skip(tc.skipBug)
			}
			v, err := Eval(tc.input, ctx)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("Eval(%q) = %s, want error containing %q", tc.input, v, tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Eval(%q) error = %q, want substring %q", tc.input, err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Eval(%q) error: %v", tc.input, err)
			}
			if got := v.String(); got != tc.want {
				t.Fatalf("Eval(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestReadmeNoDrift(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}

	var got []string
	inSection := false
	inFence := false
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimRight(raw, "\r")
		if !inSection {
			if line == "## Expression examples" {
				inSection = true
			}
			continue
		}
		if strings.HasPrefix(line, "## ") {
			break
		}
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
			continue
		}
		if inFence && strings.TrimSpace(line) != "" {
			got = append(got, line)
		}
	}

	if !inSection {
		t.Fatal("section \"## Expression examples\" not found in README.md")
	}
	if len(got) == 0 {
		t.Fatal("no expression example lines extracted from README.md")
	}

	seen := make(map[string]bool, len(readmeCases))
	for _, tc := range readmeCases {
		seen[tc.line] = true
	}
	for _, line := range got {
		if !seen[line] {
			t.Errorf("README expression example missing from readmeCases: %q", line)
		}
	}
	if len(got) != len(readmeCases) {
		t.Errorf("README expression example count = %d, readmeCases count = %d", len(got), len(readmeCases))
	}
}
