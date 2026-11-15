package expr

import (
	"testing"
	"time"
)

func recurRefCtx(t *testing.T, weekStart time.Weekday) Context {
	t.Helper()
	loc, err := time.LoadLocation("America/Mexico_City")
	if err != nil {
		t.Fatal(err)
	}
	return Context{
		Now: time.Date(2026, 11, 14, 14, 30, 0, 0, loc),
		Loc: loc,
		Cfg: Config{
			WeekStart: weekStart,
			Lang:      EN,
		},
	}
}

func TestParseEvery(t *testing.T) {
	cases := []struct {
		input     string
		lang      Lang
		canonical string // empty means equal to input
	}{
		{input: "every monday"},
		{input: "cada lunes", lang: ES, canonical: "every monday"},
		{input: "todos los lunes", lang: ES, canonical: "every monday"},
		{input: "every day"},
		{input: "every 2 days", canonical: "every 2 day"},
		{input: "every 2 weeks", canonical: "every 2 week"},
		{input: "every 90 minutes", canonical: "every 90 minute"},
		{input: "every weekday"},
		{input: "every weekdays", canonical: "every weekday"},
		{input: "every día de semana", lang: ES, canonical: "every weekday"},
		{input: "every day 2nd"},
		{input: "every month 15th"},
		{input: "every month on the 15th", canonical: "every month 15th"},
		{input: "every weekday 2nd"},
		{input: "every day-year 2nd"},
		{input: "every 2nd monday"},
		{input: "every second monday", canonical: "every 2nd monday"},
		{input: "every year aug 20", canonical: "every year august 20"},
		{input: "every year 20 aug", canonical: "every year august 20"},
		{input: "every monday 9am", canonical: "every monday 09:00"},
		{input: "every day 14:30"},
		{input: "every month 15th 9am", canonical: "every month 15th 09:00"},
		{input: "every monday 9am, next 3", canonical: "every monday 09:00 | next 3"},
		{input: "every monday, to cron", canonical: "every monday | to cron"},
		{input: "every monday, to ics", canonical: "every monday | to ics"},
		{input: "every monday, to grid", canonical: "every monday | to grid"},
		{input: `cron "0 9 * * 1,3", explain`, canonical: `cron "0 9 * * 1,3" | explain`},
		{input: `cron "0 9 * * 1", explica`, canonical: `cron "0 9 * * 1" | explain`},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			e, err := Parse(tc.input, tc.lang)
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tc.input, err)
			}
			want := tc.canonical
			if want == "" {
				want = tc.input
			}
			if got := e.String(); got != want {
				t.Errorf("Parse(%q).String() = %q, want %q", tc.input, got, want)
			}
			rt, err := Parse(e.String(), EN)
			if err != nil {
				t.Fatalf("round-trip Parse(%q) failed: %v", e.String(), err)
			}
			if rt.String() != e.String() {
				t.Errorf("round-trip String() = %q, want %q", rt.String(), e.String())
			}
			if !nodesEqualIgnorePos(rt.Stages, e.Stages) {
				t.Errorf("round-trip AST mismatch:\n got: %#v\nwant: %#v", rt.Stages, e.Stages)
			}
		})
	}

	t.Run("every_year_aug_20_shape", func(t *testing.T) {
		e, err := Parse("every year aug 20", EN)
		if err != nil {
			t.Fatal(err)
		}
		if len(e.Stages) != 1 || e.Stages[0].Op != "every" || len(e.Stages[0].Args) != 2 {
			t.Fatalf("got AST %#v", e.Stages)
		}
		if e.Stages[0].Args[0].Op != "unit" || e.Stages[0].Args[0].Text != "year" {
			t.Errorf("arg[0] = %#v, want unit year", e.Stages[0].Args[0])
		}
		if e.Stages[0].Args[1].Op != "month" || e.Stages[0].Args[1].Text != "august" {
			t.Errorf("arg[1] = %#v, want month august", e.Stages[0].Args[1])
		}
	})

	t.Run("errors", func(t *testing.T) {
		bad := []string{
			"every",
			"every 2",
			"every blue",
			"every 0 days",
			"every 2 monday",
			"every 2nd",
			"every day-year",
			"every year aug",
			"every year aug 20 2026",
			"every 30 seconds",
			"cron",
			"cron monday",
		}
		for _, in := range bad {
			if e, err := Parse(in, EN); err == nil {
				t.Errorf("Parse(%q) unexpectedly succeeded: %s", in, e.String())
			}
		}
	})
}

func TestRecur(t *testing.T) {
	ctx := recurRefCtx(t, time.Monday)

	cases := []struct {
		input string
		want  string
	}{
		{"every monday", "2026-11-16\n2026-11-23\n2026-11-30\n2026-12-07\n2026-12-14"},
		{"cada lunes", "2026-11-16\n2026-11-23\n2026-11-30\n2026-12-07\n2026-12-14"},
		{"every day 2nd", "2026-12-02\n2027-01-02\n2027-02-02\n2027-03-02\n2027-04-02"},
		{"every month 15th, next 2", "2026-11-15\n2026-12-15"},
		{"every monday, next 2", "2026-11-16\n2026-11-23"},
		{"every 2 weeks, next 2", "2026-11-28\n2026-12-12"},
		{"every 2 days, next 3", "2026-11-16\n2026-11-18\n2026-11-20"},
		{"every weekday, next 3", "2026-11-16\n2026-11-17\n2026-11-18"},
		{"every weekday 2nd, next 2", "2026-11-17\n2026-11-24"},
		{"every day-year 2nd, next 2", "2027-01-02\n2028-01-02"},
		{"every 2nd monday, next 2", "2026-12-14\n2027-01-11"},
		{"every year aug 20, next 1", "2027-08-20"},
		{"every monday 9am, next 1", "2026-11-16T09:00:00-06:00"},
		{"every saturday 9am, next 1", "2026-11-21T09:00:00-06:00"},
		{"every saturday 8pm, next 1", "2026-11-14T20:00:00-06:00"},
		{"every 90 minutes, next 2", "2026-11-14T15:00:00-06:00\n2026-11-14T16:30:00-06:00"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			v, err := Eval(tc.input, ctx)
			if err != nil {
				t.Fatalf("Eval(%q) unexpected error: %v", tc.input, err)
			}
			if got := v.String(); got != tc.want {
				t.Errorf("Eval(%q).String() =\n%s\nwant:\n%s", tc.input, got, tc.want)
			}
		})
	}

	t.Run("sunday_week_start", func(t *testing.T) {
		sunCtx := recurRefCtx(t, time.Sunday)
		v, err := Eval("every weekday 2nd, next 2", sunCtx)
		if err != nil {
			t.Fatalf("Eval unexpected error: %v", err)
		}
		want := "2026-11-16\n2026-11-23"
		if got := v.String(); got != want {
			t.Errorf("got\n%s\nwant\n%s", got, want)
		}
	})

	t.Run("errors", func(t *testing.T) {
		bad := []string{
			"every monday, next 0",
			"every monday, next 1001",
			"today, next 2",
			"every weekday 8th",
			"every day 32nd",
			"every 6th monday",
			"every year feb 30",
			"every day-year 367th",
		}
		for _, in := range bad {
			if _, err := Eval(in, ctx); err == nil {
				t.Errorf("Eval(%q) unexpectedly succeeded", in)
			}
		}
	})
}
