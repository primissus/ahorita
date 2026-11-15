package expr

import (
	"strings"
	"testing"
	"time"
)

func TestICS(t *testing.T) {
	ctx := recurRefCtx(t, time.Monday)

	t.Run("every_monday_9am_to_ics", func(t *testing.T) {
		v1, err := Eval("every monday 9am, to ics", ctx)
		if err != nil {
			t.Fatal(err)
		}
		v2, err := Eval("every monday 9am, to ics", ctx)
		if err != nil {
			t.Fatal(err)
		}
		if v1.Kind != KText {
			t.Fatalf("Kind = %v, want KText", v1.Kind)
		}
		s := v1.String()
		if s != v2.String() {
			t.Fatalf("ICS output not deterministic across runs")
		}
		if !strings.HasPrefix(s, "BEGIN:VCALENDAR\r\n") {
			t.Errorf("missing BEGIN:VCALENDAR\\r\\n:\n%s", s)
		}
		if !strings.HasSuffix(s, "END:VCALENDAR\r\n") {
			t.Errorf("missing trailing END:VCALENDAR\\r\\n")
		}
		for _, want := range []string{
			"VERSION:2.0\r\n",
			"PRODID:-//ahorita//EN\r\n",
			"DTSTAMP:20261114T203000Z\r\n",
			"DTSTART;TZID=America/Mexico_City:20261116T090000\r\n",
			"SUMMARY:every monday 09:00\r\n",
			"RRULE:FREQ=WEEKLY;BYDAY=MO",
			"@ahorita\r\n",
		} {
			if !strings.Contains(s, want) {
				t.Errorf("missing %q in:\n%s", want, s)
			}
		}
	})

	t.Run("every_monday_all_day", func(t *testing.T) {
		v, err := Eval("every monday, to ics", ctx)
		if err != nil {
			t.Fatal(err)
		}
		s := v.String()
		if !strings.Contains(s, "DTSTART;VALUE=DATE:20261116\r\n") {
			t.Errorf("missing VALUE=DATE DTSTART in:\n%s", s)
		}
		if !strings.Contains(s, "RRULE:FREQ=WEEKLY;BYDAY=MO\r\n") {
			t.Errorf("missing RRULE in:\n%s", s)
		}
	})

	t.Run("single_date_to_ics", func(t *testing.T) {
		v, err := Eval("2026-12-24, to ics", ctx)
		if err != nil {
			t.Fatal(err)
		}
		s := v.String()
		if !strings.Contains(s, "DTSTART;VALUE=DATE:20261224\r\n") {
			t.Errorf("missing DTSTART;VALUE=DATE:20261224 in:\n%s", s)
		}
		if strings.Contains(s, "RRULE:") {
			t.Errorf("unexpected RRULE in single date:\n%s", s)
		}
	})

	t.Run("whole_day_span_to_ics", func(t *testing.T) {
		v, err := Eval("2026-12-24 .. 2026-12-26, to ics", ctx)
		if err != nil {
			t.Fatal(err)
		}
		s := v.String()
		for _, want := range []string{
			"DTSTART;VALUE=DATE:20261224\r\n",
			"DTEND;VALUE=DATE:20261227\r\n",
			"SUMMARY:2026-12-24 .. 2026-12-26\r\n",
		} {
			if !strings.Contains(s, want) {
				t.Errorf("missing %q in:\n%s", want, s)
			}
		}
	})

	t.Run("datetime_to_ics", func(t *testing.T) {
		v, err := Eval("2026-11-16 09:00, to ics", ctx)
		if err != nil {
			t.Fatal(err)
		}
		s := v.String()
		for _, want := range []string{
			"DTSTART;TZID=America/Mexico_City:20261116T090000\r\n",
			"DTEND;TZID=America/Mexico_City:20261116T090000\r\n",
		} {
			if !strings.Contains(s, want) {
				t.Errorf("missing %q in:\n%s", want, s)
			}
		}
	})

	t.Run("errors", func(t *testing.T) {
		if _, err := Eval(`cron "0 9 * * 1", to ics`, ctx); err == nil {
			t.Errorf("expected error for cron to ics")
		}
		if _, err := Eval("2h, to ics", ctx); err == nil {
			t.Errorf("expected error for duration to ics")
		}
	})
}

func TestGrid(t *testing.T) {
	ctx := recurRefCtx(t, time.Monday)

	t.Run("today_to_grid", func(t *testing.T) {
		v, err := Eval("today, to grid", ctx)
		if err != nil {
			t.Fatal(err)
		}
		s := v.String()
		if !strings.Contains(s, "November 2026") {
			t.Errorf("missing title November 2026 in:\n%s", s)
		}
		if !strings.Contains(s, "Mo Tu We Th Fr Sa Su") {
			t.Errorf("missing Monday header in:\n%s", s)
		}
		if !strings.Contains(s, "14*") {
			t.Errorf("missing marked 14* in:\n%s", s)
		}
	})

	t.Run("every_monday_to_grid", func(t *testing.T) {
		v, err := Eval("every monday, to grid", ctx)
		if err != nil {
			t.Fatal(err)
		}
		s := v.String()
		for _, want := range []string{"November 2026", "December 2026", "16*", "23*", "30*", "\n\n"} {
			if !strings.Contains(s, want) {
				t.Errorf("missing %q in:\n%s", want, s)
			}
		}
	})

	t.Run("sunday_week_start", func(t *testing.T) {
		sunCtx := recurRefCtx(t, time.Sunday)
		v, err := Eval("today, to grid", sunCtx)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(v.String(), "Su Mo Tu We Th Fr Sa") {
			t.Errorf("expected Sunday-first header:\n%s", v.String())
		}
	})

	t.Run("spanish_grid", func(t *testing.T) {
		v, err := Eval("hoy, a calendario", ctx)
		if err != nil {
			t.Fatal(err)
		}
		s := v.String()
		if !strings.Contains(s, "noviembre 2026") {
			t.Errorf("missing Spanish month title:\n%s", s)
		}
		if !strings.Contains(s, "lu ma mi ju vi sá do") {
			t.Errorf("missing Spanish header:\n%s", s)
		}
	})

	t.Run("span_to_grid", func(t *testing.T) {
		v, err := Eval("2026-11-29 .. 2026-12-02, to grid", ctx)
		if err != nil {
			t.Fatal(err)
		}
		s := v.String()
		for _, want := range []string{"November 2026", "December 2026", "29*", "30*", " 1*", " 2*"} {
			if !strings.Contains(s, want) {
				t.Errorf("missing %q in:\n%s", want, s)
			}
		}

		if _, err := Eval("2026-01-01 .. 2027-02-01, to grid", ctx); err == nil {
			t.Errorf("expected error for >12 month span grid")
		}
	})
}

func TestDaysFromICS(t *testing.T) {
	loc, err := time.LoadLocation("America/Mexico_City")
	if err != nil {
		t.Fatal(err)
	}
	from := time.Date(2025, 11, 14, 0, 0, 0, 0, loc)
	to := time.Date(2031, 11, 14, 0, 0, 0, 0, loc)

	ics := strings.Join([]string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"BEGIN:VEVENT",
		"DTSTART;VALUE=DATE:20261224",
		"SUMMARY:Christmas Eve",
		"END:VEVENT",
		"BEGIN:VEVENT",
		"DTSTART;VALUE=DATE:20261228",
		"DTEND;VALUE=DATE:20261231",
		"SUMMARY:Winter",
		" Break",
		"BEGIN:VALARM",
		"TRIGGER:-PT15M",
		"END:VALARM",
		"END:VEVENT",
		"BEGIN:VEVENT",
		"DTSTART;TZID=America/Mexico_City:20261116T090000",
		"DTEND;TZID=America/Mexico_City:20261116T100000",
		"SUMMARY:Timed meeting (must be skipped)",
		"END:VEVENT",
		"BEGIN:VEVENT",
		"DTSTART;VALUE=DATE:20260101",
		"RRULE:FREQ=YEARLY;BYMONTH=1;BYMONTHDAY=1",
		"SUMMARY:New Year",
		"END:VEVENT",
		"END:VCALENDAR",
		"",
	}, "\r\n")

	vals, err := DaysFromICS(ics, from, to)
	if err != nil {
		t.Fatal(err)
	}
	// Expect: 2026-12-24 (date), 2026-12-28..2026-12-30 (span), plus 2026-01-01..2031-01-01 (6 years).
	if len(vals) != 8 {
		t.Fatalf("got %d values, want 8: %v", len(vals), vals)
	}
	if vals[0].Kind != KDate || vals[0].String() != "2026-12-24" {
		t.Errorf("vals[0] = %v (%s), want date 2026-12-24", vals[0].Kind, vals[0].String())
	}
	if vals[1].Kind != KSpan || !vals[1].Days || vals[1].String() != "2026-12-28 → 2026-12-30" {
		t.Errorf("vals[1] = %v (%s), want span 2026-12-28 → 2026-12-30", vals[1].Kind, vals[1].String())
	}
	if vals[2].String() != "2026-01-01" || vals[7].String() != "2031-01-01" {
		t.Errorf("yearly RRULE bounds: first=%s last=%s", vals[2].String(), vals[7].String())
	}

	t.Run("errors", func(t *testing.T) {
		if _, err := DaysFromICS("not an ics file", from, to); err == nil {
			t.Errorf("expected error on missing VCALENDAR")
		}
		badRule := "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nDTSTART;VALUE=DATE:20261224\r\nRRULE:BAD=RULE\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
		if _, err := DaysFromICS(badRule, from, to); err == nil {
			t.Errorf("expected error on bad RRULE")
		}
	})
}
