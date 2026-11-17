package expr

import (
	"testing"
	"time"
	_ "time/tzdata"
)

func refCtx(t *testing.T) Context {
	t.Helper()
	loc, err := time.LoadLocation("America/Mexico_City")
	if err != nil {
		t.Fatalf("load reference location: %v", err)
	}
	now := time.Date(2026, time.November, 14, 14, 30, 0, 0, loc)
	return Context{
		Now: now,
		Loc: loc,
		Cfg: DefaultConfig(),
	}
}

func TestCases(t *testing.T) {
	ctx := refCtx(t)

	cases := []struct {
		id         string
		input      string
		opts       Options
		noHolidays bool
		want       string
		skip       string
	}{
		{id: "1", input: "today", want: "2026-11-14"},
		{id: "2", input: "now - 1d", want: "2026-11-13T14:30:00-06:00"},
		{id: "3", input: "2026-06-02 + 2d", want: "2026-06-04"},
		{id: "4", input: "2026-06-02 - 1h", want: "2026-06-01T23:00:00-06:00"},
		{id: "5", input: "yesterday", want: "2026-11-13"},
		{id: "6", input: "2d ago", want: "2026-11-12T14:30:00-06:00"},
		{id: "7", input: "last week", want: "2026-11-02 → 2026-11-08"},
		{id: "8", input: "today - 1w", want: "2026-11-07"},
		{id: "9", input: "closest monday", want: "2026-11-16"},
		{id: "10", input: "today - 1w, monday", want: "2026-11-02"},
		{id: "11", input: "past monday", want: "2026-11-09"},
		{id: "12", input: "end of day, today", want: "2026-11-14T23:59:59-06:00"},
		{id: "13", input: "end of week, this week", want: "2026-11-15T23:59:59-06:00"},
		{id: "14", input: "next month", want: "2026-12-01 → 2026-12-31"},
		{id: "15", input: "in 3 months", want: "2027-02-14T14:30:00-06:00"},
		{id: "16", input: "in Q3", want: "2026-07-01 → 2026-09-30"},
		{id: "17", input: "2nd week, h2", want: "2026-07-06 → 2026-07-12"},
		{id: "17b", input: "week 2", want: "2026-01-05 → 2026-01-11"},
		{id: "18", input: "on aug 20th", want: "2026-08-20"},
		{id: "19", input: "on sept, past year", want: "2025-09-01 → 2025-09-30"},
		{id: "20", input: "2h", want: "2h"},
		{id: "20b", input: "90m", want: "1h 30m"},
		{id: "21", input: "during 2 days", want: "2026-11-14 → 2026-11-15"},
		{id: "22", input: "during 6 weeks, starting next monday", want: "2026-11-16 → 2026-12-27"},
		{id: "23", input: "every monday", want: "2026-11-16\n2026-11-23\n2026-11-30\n2026-12-07\n2026-12-14"},
		{id: "24", input: "every day 2nd", want: "2026-12-02\n2027-01-02\n2027-02-02\n2027-03-02\n2027-04-02"},
		{id: "25", input: "every monday, to cron", want: "0 0 * * 1"},
		{id: "26", input: "2026-12-24 - 2026-11-14", want: "40d"},
		{id: "26b", input: "2026-11-14 - 2026-12-24", want: "-40d"},
		{id: "27", input: "how many days between 2026-11-16 and 2026-11-20", want: "4"},
		{id: "27a", input: "how many days between 2026-11-16 and 2026-11-20 inclusive", want: "5"},
		{id: "27b", input: "how many mondays between today and 2026-12-31", want: "7"},
		{id: "28", input: "2026-11-14 14:30 in Tokyo", want: "2026-11-14T14:30:00+09:00"},
		{id: "28b", input: "now in Tokyo", want: "2026-11-15T05:30:00+09:00"},
		{id: "28c", input: "2pm @ Tokyo", want: "2026-11-14T14:00:00+09:00"},
		{id: "29", input: "2026-11-14 14:30, America/New_York", want: "2026-11-14T14:30:00-05:00"},
		{id: "30", input: "now to Asia/Tokyo", want: "2026-11-15T05:30:00+09:00"},
		{id: "31", input: "now to UTC", want: "2026-11-14T20:30:00Z"},
		{id: "32", input: "2026-11-20 - 2026-11-16, to business days", want: "4"},
		{id: "32_no_holidays", input: "2026-11-20 - 2026-11-16, to business days", noHolidays: true, want: "4"},
		{id: "32a", input: "2026-11-20 - 2026-11-16, to business days inclusive", want: "4"},
		{id: "32a_no_holidays", input: "2026-11-20 - 2026-11-16, to business days inclusive", noHolidays: true, want: "5"},
		{id: "33", input: "2026-11-14, next business day", want: "2026-11-17"},
		{id: "33_no_holidays", input: "2026-11-14, next business day", noHolidays: true, want: "2026-11-16"},
		{id: "33b", input: "2026-11-17, next business day", want: "2026-11-18"},
		{id: "33b_no_holidays", input: "2026-11-17, next business day", noHolidays: true, want: "2026-11-18"},
		{id: "34", input: "2026-11-13 + 3 business days", want: "2026-11-19"},
		{id: "34_no_holidays", input: "2026-11-13 + 3 business days", noHolidays: true, want: "2026-11-18"},
		{id: "34b", input: "2026-11-18 - 2 business days", want: "2026-11-13"},
		{id: "34b_no_holidays", input: "2026-11-18 - 2 business days", noHolidays: true, want: "2026-11-16"},
		{id: "34c", input: "2026-11-14 + 1 business day", want: "2026-11-17"},
		{id: "34c_no_holidays", input: "2026-11-14 + 1 business day", noHolidays: true, want: "2026-11-16"},
		{id: "35", input: "now + 2days", want: "2026-11-16T14:30:00-06:00"},
		{id: "36", input: "now", opts: Options{Type: "date"}, want: "2026-11-14"},
		{id: "37", input: "now", opts: Options{Type: "datetime"}, want: "2026-11-14T14:30:00-06:00"},
		{id: "38", input: "now", opts: Options{Type: "time"}, want: "14:30"},
		{id: "39", input: "now", opts: Options{Type: "range"}, want: "0s"},
		{id: "40", input: "now", opts: Options{Type: "relative"}, want: "now"},
		{id: "40b", input: "today + 2d", opts: Options{Type: "relative"}, want: "in 2 days"},
		{id: "41", input: "2026-06-02", opts: Options{Pattern: "dd-MM-yyyy"}, want: "02-06-2026"},
		{id: "42", input: "2026-06-02", opts: Options{Pattern: "d/M/yy"}, want: "2/6/26"},
		{id: "43", input: "(now - 1 business day) to los_angeles", want: "2026-11-13T12:30:00-08:00"},
		{id: "43_human", input: "(now - 1 business day) to los_angeles", opts: Options{Pattern: "human"}, want: "Fri, Nov 13, 2026, 12:30 PM"},
		{id: "44", input: "now to Los Angeles", want: "2026-11-14T12:30:00-08:00"},
		{id: "45", input: "now to LA", want: "2026-11-14T12:30:00-08:00"},
		{id: "46", input: "now to california", want: "2026-11-14T12:30:00-08:00"},
		{id: "47", input: "now to los_angels", want: "2026-11-14T12:30:00-08:00"},
		{id: "48", input: "2 * (1h + 30m)", want: "3h"},
	}

	for _, tc := range cases {
		t.Run(tc.id+"_"+tc.input, func(t *testing.T) {
			if tc.skip != "" {
				t.Skip(tc.skip)
			}
			c := ctx
			if tc.noHolidays {
				c.Cfg.Country = ""
			}
			got, err := Eval(tc.input, c)
			if err != nil {
				t.Fatalf("case %s (%q) unexpected error: %v", tc.id, tc.input, err)
			}
			out, err := Format(got, c, tc.opts)
			if err != nil {
				t.Fatalf("case %s (%q) Format error: %v", tc.id, tc.input, err)
			}
			if out != tc.want {
				t.Fatalf("case %s (%q) = %q, want %q", tc.id, tc.input, out, tc.want)
			}
		})
	}
}
