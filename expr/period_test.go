package expr

import (
	"strings"
	"testing"
	"time"
)

func TestPeriod(t *testing.T) {
	baseCtx := testRefContext(t)

	tests := []struct {
		name      string
		input     string
		weekStart *time.Weekday
		now       string
		want      string
		wantErr   string
	}{
		{name: "last week", input: "last week", want: "2026-11-02 → 2026-11-08"},
		{name: "this week", input: "this week", want: "2026-11-09 → 2026-11-15"},
		{name: "next month", input: "next month", want: "2026-12-01 → 2026-12-31"},
		{name: "last year", input: "last year", want: "2025-01-01 → 2025-12-31"},
		{name: "in Q3", input: "in Q3", want: "2026-07-01 → 2026-09-30"},
		{name: "h2", input: "h2", want: "2026-07-01 → 2026-12-31"},
		{name: "past monday", input: "past monday", want: "2026-11-09"},
		{name: "next friday", input: "next friday", want: "2026-11-20"},
		{name: "next saturday strictly after today", input: "next saturday", want: "2026-11-21"},
		{name: "this monday", input: "this monday", want: "2026-11-09"},
		{name: "monday", input: "monday", want: "2026-11-09"},
		{name: "today minus 1w then monday", input: "today - 1w, monday", want: "2026-11-02"},
		{name: "closest monday", input: "closest monday", want: "2026-11-16"},
		{name: "closest saturday", input: "closest saturday", want: "2026-11-14"},
		{name: "closest monday to date", input: "closest monday to 2026-12-03", want: "2026-11-30"},
		{name: "on aug 20th", input: "on aug 20th", want: "2026-08-20"},
		{name: "next aug 20", input: "next aug 20", want: "2027-08-20"},
		{name: "sept", input: "sept", want: "2026-09-01 → 2026-09-30"},
		{name: "on sept, past year", input: "on sept, past year", want: "2025-09-01 → 2025-09-30"},
		{name: "2026-06-02, next month", input: "2026-06-02, next month", want: "2026-07-01 → 2026-07-31"},
		{
			name:      "WeekStart Sunday this week",
			input:     "this week",
			weekStart: ptrWeekday(time.Sunday),
			want:      "2026-11-08 → 2026-11-14",
		},
		{
			name:      "WeekStart Sunday work week",
			input:     "this work week",
			weekStart: ptrWeekday(time.Sunday),
			want:      "2026-11-09 → 2026-11-13",
		},
		{name: "this work week Monday start", input: "this work week", want: "2026-11-09 → 2026-11-13"},
		{name: "next day span", input: "next day", want: "2026-11-15 → 2026-11-15"},
		{name: "last month year boundary", input: "2026-01-15, last month", want: "2025-12-01 → 2025-12-31"},
		{name: "next quarter year boundary", input: "next quarter", want: "2027-01-01 → 2027-03-31"},
		{name: "last half", input: "last half", want: "2026-01-01 → 2026-06-30"},
		{name: "month with year", input: "february 2028", want: "2028-02-01 → 2028-02-29"},
		{name: "month day year", input: "aug 20 2028", want: "2028-08-20"},
		{name: "Q1 with year", input: "Q1 2025", want: "2025-01-01 → 2025-03-31"},
		{name: "next feb 29 skips non-leap years", input: "2025-01-01, next feb 29", want: "2028-02-29"},
		{name: "last feb 29 skips non-leap years", input: "2026-11-14, last feb 29", want: "2024-02-29"},
		{name: "next business day skips weekend and holiday", input: "next business day", want: "2026-11-17"},
		{name: "this business day errors", input: "this business day", wantErr: "next or last"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := baseCtx
			if tc.weekStart != nil {
				ctx.Cfg.WeekStart = *tc.weekStart
			}
			if tc.now != "" {
				parsed, err := time.Parse(time.RFC3339, tc.now)
				if err != nil {
					t.Fatalf("bad test now %q: %v", tc.now, err)
				}
				ctx.Now = parsed
			}
			got, err := Eval(tc.input, ctx)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("Eval(%q) = %s, want error containing %q", tc.input, got.String(), tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Eval(%q) error = %q, want substring %q", tc.input, err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Eval(%q) unexpected error: %v", tc.input, err)
			}
			if got.String() != tc.want {
				t.Fatalf("Eval(%q) = %q, want %q", tc.input, got.String(), tc.want)
			}
		})
	}
}

func TestWeek(t *testing.T) {
	baseCtx := testRefContext(t)

	tests := []struct {
		name    string
		input   string
		weekOne string
		want    string
		wantErr bool
	}{
		{name: "week 2", input: "week 2", want: "2026-01-05 → 2026-01-11"},
		{name: "week 1", input: "week 1", want: "2025-12-29 → 2026-01-04"},
		{name: "week 53", input: "week 53", want: "2026-12-28 → 2027-01-03"},
		{name: "week 54 error", input: "week 54", wantErr: true},
		{name: "week 0 error", input: "week 0", wantErr: true},
		{name: "2nd week, h2", input: "2nd week, h2", want: "2026-07-06 → 2026-07-12"},
		{name: "2nd week of march", input: "2nd week of march", want: "2026-03-09 → 2026-03-15"},
		{name: "WeekOne days 2nd week h2", input: "2nd week, h2", weekOne: "days", want: "2026-07-08 → 2026-07-14"},
		{name: "WeekOne full 2nd week h2", input: "2nd week, h2", weekOne: "full", want: "2026-07-13 → 2026-07-19"},
		{name: "WeekOne first-day 2nd week march", input: "2nd week of march", weekOne: "first-day", want: "2026-03-02 → 2026-03-08"},
		{name: "monday of next week", input: "monday of next week", want: "2026-11-16"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := baseCtx
			if tc.weekOne != "" {
				ctx.Cfg.WeekOne = tc.weekOne
			}
			got, err := Eval(tc.input, ctx)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Eval(%q) = %s, want error", tc.input, got.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("Eval(%q) unexpected error: %v", tc.input, err)
			}
			if got.String() != tc.want {
				t.Fatalf("Eval(%q) = %q, want %q", tc.input, got.String(), tc.want)
			}
		})
	}
}

func TestBound(t *testing.T) {
	baseCtx := testRefContext(t)

	tests := []struct {
		name    string
		input   string
		strict  *bool
		want    string
		wantErr bool
	}{
		{name: "end of day, today", input: "end of day, today", want: "2026-11-14T23:59:59-06:00"},
		{name: "end of week, this week", input: "end of week, this week", want: "2026-11-15T23:59:59-06:00"},
		{name: "end of work week", input: "end of work week", want: "2026-11-13T23:59:59-06:00"},
		{name: "start of month", input: "start of month", want: "2026-11-01"},
		{name: "end of next month", input: "end of next month", want: "2026-12-31T23:59:59-06:00"},
		{name: "last week, end of day", input: "last week, end of day", want: "2026-11-08T23:59:59-06:00"},
		{name: "today | end of day | start of hour", input: "today | end of day | start of hour", want: "2026-11-14T23:00:00-06:00"},
		{name: "start of quarter", input: "start of quarter", want: "2026-10-01"},
		{name: "start of year", input: "start of year", want: "2026-01-01"},
		{name: "end of hour", input: "end of hour", want: "2026-11-14T14:59:59-06:00"},
		{name: "start of minute", input: "start of minute", want: "2026-11-14T14:30:00-06:00"},
		{name: "end of year", input: "end of year", want: "2026-12-31T23:59:59-06:00"},
		{name: "non-strict fin de semana", input: "fin de semana", strict: ptrBool(false), want: "2026-11-15T23:59:59-06:00"},
		{name: "strict fin de semana errors", input: "fin de semana", strict: ptrBool(true), wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := baseCtx
			if tc.strict != nil {
				ctx.Cfg.Strict = *tc.strict
			}
			got, err := Eval(tc.input, ctx)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Eval(%q) = %s, want error", tc.input, got.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("Eval(%q) unexpected error: %v", tc.input, err)
			}
			if got.String() != tc.want {
				t.Fatalf("Eval(%q) = %q, want %q", tc.input, got.String(), tc.want)
			}
		})
	}
}

func ptrWeekday(w time.Weekday) *time.Weekday {
	return &w
}

func ptrBool(b bool) *bool {
	return &b
}

func TestSpan(t *testing.T) {
	baseCtx := testRefContext(t)

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{name: "during 2 days", input: "during 2 days", want: "2026-11-14 → 2026-11-15"},
		{name: "during 6 weeks starting next monday", input: "during 6 weeks, starting next monday", want: "2026-11-16 → 2026-12-27"},
		{name: "during 2h", input: "during 2h", want: "2026-11-14T14:30:00-06:00 → 2026-11-14T16:30:00-06:00"},
		{name: "span date .. date", input: "2026-11-16 .. 2026-11-20", want: "2026-11-16 → 2026-11-20"},
		{name: "span same date", input: "2026-11-16 .. 2026-11-16", want: "2026-11-16 → 2026-11-16"},
		{name: "from .. to dates", input: "from 2026-11-16 to 2026-11-20", want: "2026-11-16 → 2026-11-20"},
		{name: "between .. and datetimes", input: "between 2026-11-16 09:00 and 2026-11-16 17:00", want: "2026-11-16T09:00:00-06:00 → 2026-11-16T17:00:00-06:00"},
		{name: "span clocks on today", input: "09:00 .. 17:00", want: "2026-11-14T09:00:00-06:00 → 2026-11-14T17:00:00-06:00"},
		{name: "during 3 months starting jan 31", input: "during 3 months, starting 2026-01-31", want: "2026-01-31 → 2026-04-29"},
		{name: "last week starting 2026-12-01 preserves length", input: "last week, starting 2026-12-01", want: "2026-12-01 → 2026-12-07"},
		{name: "during 2h starting 3pm", input: "during 2h, starting 3pm", want: "2026-11-14T15:00:00-06:00 → 2026-11-14T17:00:00-06:00"},
		{name: "span end before start error", input: "2026-11-20 .. 2026-11-16", wantErr: "before start"},
		{name: "starting on non-span error", input: "today, starting 2026-11-16", wantErr: "requires a span"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Eval(tc.input, baseCtx)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("Eval(%q) = %s, want error containing %q", tc.input, got.String(), tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Eval(%q) error = %q, want substring %q", tc.input, err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Eval(%q) unexpected error: %v", tc.input, err)
			}
			if got.String() != tc.want {
				t.Fatalf("Eval(%q) = %q, want %q", tc.input, got.String(), tc.want)
			}
		})
	}
}

func TestCount(t *testing.T) {
	baseCtx := testRefContext(t)
	nyLoc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("load NY location: %v", err)
	}

	tests := []struct {
		name           string
		input          string
		countInclusive bool
		loc            *time.Location
		now            string
		want           string
		wantErr        string
	}{
		{name: "how many days between", input: "how many days between 2026-11-16 and 2026-11-20", want: "4"},
		{name: "how many days inclusive", input: "how many days between 2026-11-16 and 2026-11-20 inclusive", want: "5"},
		{name: "CountInclusive true", input: "how many days between 2026-11-16 and 2026-11-20", countInclusive: true, want: "5"},
		{name: "span to days already inclusive", input: "2026-11-16 .. 2026-11-20, to days", want: "5"},
		{name: "span to days ignores CountInclusive", input: "2026-11-16 .. 2026-11-20, to days", countInclusive: true, want: "5"},
		{name: "how many mondays", input: "how many mondays between today and 2026-12-31", want: "7"},
		{name: "how many saturdays inclusive", input: "how many saturdays between today and 2026-11-21 inclusive", want: "2"},
		{name: "how many weeks", input: "how many weeks between 2026-11-16 and 2026-11-30", want: "2"},
		{name: "diff to hours", input: "2026-12-24 - 2026-11-14, to hours", want: "960"},
		{name: "negative diff to days", input: "2026-11-14 - 2026-12-24, to days", want: "-40"},
		{name: "90m to hours", input: "90m to hours", want: "1.5"},
		{name: "2d to hours", input: "2d to hours", want: "48"},
		{name: "3 months to days error", input: "3 months to days", wantErr: "during"},
		{name: "during 3 months to days", input: "during 3 months, to days", want: "92"},
		{name: "next month to days", input: "next month, to days", want: "31"},
		{name: "2026 to months", input: "2026, to months", want: "12"},
		{name: "how many months clamping", input: "how many months between 2026-01-31 and 2026-04-30", want: "3"},
		{
			name:  "NY DST spring forward span to days is 2",
			input: "2026-03-07 .. 2026-03-08, to days",
			loc:   nyLoc,
			now:   "2026-03-07T12:00:00-05:00",
			want:  "2",
		},
		{
			name:  "NY DST spring forward span to hours is 47",
			input: "2026-03-07 .. 2026-03-08, to hours",
			loc:   nyLoc,
			now:   "2026-03-07T12:00:00-05:00",
			want:  "47",
		},
		{name: "to business day error without dates", input: "2d to business days", wantErr: "give a span"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := baseCtx
			ctx.Cfg.CountInclusive = tc.countInclusive
			if tc.loc != nil {
				ctx.Loc = tc.loc
			}
			if tc.now != "" {
				parsed, err := time.Parse(time.RFC3339, tc.now)
				if err != nil {
					t.Fatalf("bad test now %q: %v", tc.now, err)
				}
				ctx.Now = parsed
			}
			got, err := Eval(tc.input, ctx)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("Eval(%q) = %s, want error containing %q", tc.input, got.String(), tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Eval(%q) error = %q, want substring %q", tc.input, err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Eval(%q) unexpected error: %v", tc.input, err)
			}
			if got.String() != tc.want {
				t.Fatalf("Eval(%q) = %q, want %q", tc.input, got.String(), tc.want)
			}
		})
	}

	t.Run("to key returns not available yet", func(t *testing.T) {
		ev := &evaluator{ctx: baseCtx, loc: baseCtx.Loc, cfgLang: baseCtx.Cfg.Lang}
		_, err := ev.evalTo(Node{
			Op:   "to",
			Pos:  0,
			Args: []Node{{Op: "key", Pos: 3, Text: "cron"}},
		}, Value{Kind: KRecurrence}, false)
		if err == nil || !strings.Contains(err.Error(), "aren't available yet") {
			t.Fatalf("evalTo key error = %v, want 'aren't available yet'", err)
		}
	})
}
