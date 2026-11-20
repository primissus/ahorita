package expr

import (
	"errors"
	"strings"
	"testing"
	"time"
	_ "time/tzdata"
)

func testRefContext(t *testing.T) Context {
	t.Helper()
	mx, err := time.LoadLocation("America/Mexico_City")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}
	return Context{
		Now: time.Date(2026, 11, 14, 14, 30, 0, 0, mx),
		Loc: mx,
		Cfg: DefaultConfig(),
	}
}

func TestEval(t *testing.T) {
	ctx := testRefContext(t)

	tests := []struct {
		name  string
		input string
		cfg   *Config
		want  string
	}{
		{name: "today", input: "today", want: "2026-11-14"},
		{name: "now", input: "now", want: "2026-11-14T14:30:00-06:00"},
		{name: "yesterday", input: "yesterday", want: "2026-11-13"},
		{name: "tomorrow", input: "tomorrow", want: "2026-11-15"},
		{name: "date literal", input: "2026-06-02", want: "2026-06-02"},
		{name: "date and clock", input: "2026-11-14 14:30", want: "2026-11-14T14:30:00-06:00"},
		{name: "datetime with offset", input: "2026-11-14T14:30:00-06:00", want: "2026-11-14T14:30:00-06:00"},
		{name: "datetime without offset", input: "2026-11-14T14:30", want: "2026-11-14T14:30:00-06:00"},
		{name: "clock", input: "2:30pm", want: "14:30"},
		{name: "clock with seconds", input: "14:30:05", want: "14:30:05"},
		{name: "unix seconds", input: "@1700000000", want: "2023-11-14T16:13:20-06:00"},
		{name: "unix milliseconds", input: "@1700000000000", want: "2023-11-14T16:13:20-06:00"},
		{
			name:  "slash date EN",
			input: "02/06/2026",
			cfg:   func() *Config { c := DefaultConfig(); c.Lang = EN; return &c }(),
			want:  "2026-02-06",
		},
		{
			name:  "slash date ES",
			input: "02/06/2026",
			cfg:   func() *Config { c := DefaultConfig(); c.Lang = ES; return &c }(),
			want:  "2026-06-02",
		},
		{
			name:  "slash date unambiguous under strict",
			input: "13/06/2026",
			cfg:   func() *Config { c := DefaultConfig(); c.Strict = true; return &c }(),
			want:  "2026-06-13",
		},
		{
			name:  "slash date same day and month under strict",
			input: "06/06/2026",
			cfg:   func() *Config { c := DefaultConfig(); c.Strict = true; return &c }(),
			want:  "2026-06-06",
		},
		{name: "number", input: "42", want: "42"},
		{name: "year span", input: "2026", want: "2026-01-01 → 2026-12-31"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := ctx
			if tc.cfg != nil {
				c.Cfg = *tc.cfg
			}
			got, err := Eval(tc.input, c)
			if err != nil {
				t.Fatalf("Eval(%q) error: %v", tc.input, err)
			}
			if got.String() != tc.want {
				t.Errorf("Eval(%q) = %q, want %q", tc.input, got.String(), tc.want)
			}
		})
	}

	t.Run("errors", func(t *testing.T) {
		errCases := []struct {
			name    string
			input   string
			strict  bool
			wantPos int
			wantMsg string
		}{
			{
				name:    "invalid date",
				input:   "2026-13-45",
				wantPos: 0,
				wantMsg: "invalid date",
			},
			{
				name:    "invalid feb 29 non-leap",
				input:   "2026-02-29",
				wantPos: 0,
				wantMsg: "invalid date",
			},
			{
				name:    "invalid clock",
				input:   "25:00",
				wantPos: 0,
				wantMsg: "invalid time",
			},
			{
				name:    "ambiguous slash date under strict",
				input:   "02/06/2026",
				strict:  true,
				wantPos: 0,
				wantMsg: "ambiguous",
			},
			{
				name:    "ambiguous word under strict",
				input:   "fin de semana",
				strict:  true,
				wantPos: 0,
				wantMsg: "ambiguous word",
			},
		}

		for _, tc := range errCases {
			t.Run(tc.name, func(t *testing.T) {
				c := ctx
				c.Cfg.Strict = tc.strict
				_, err := Eval(tc.input, c)
				if err == nil {
					t.Fatalf("Eval(%q) succeeded, expected error", tc.input)
				}
				var evalErr *Error
				if !errors.As(err, &evalErr) {
					t.Fatalf("error = %T, want *Error", err)
				}
				if evalErr.Pos != tc.wantPos {
					t.Errorf("Pos = %d, want %d (msg: %q)", evalErr.Pos, tc.wantPos, evalErr.Msg)
				}
				if !strings.Contains(evalErr.Msg, tc.wantMsg) {
					t.Errorf("Msg = %q, want substring %q", evalErr.Msg, tc.wantMsg)
				}
			})
		}
	})
}

func TestStageOrder(t *testing.T) {
	ctx := testRefContext(t)

	t.Run("canonical reorderings in Explain", func(t *testing.T) {
		parsed, err := Parse("today | tomorrow", EN)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if _, err := orderStages(parsed.Stages, false); err == nil {
			t.Fatal("expected error for two value stages")
		}

		// Test orderStages directly on various pipelines before steps 4-6 exist too.
		orderTests := []struct {
			input string
			want  string
		}{
			{"end of day, today", "today | end of day"},
			{"end of week, this week", "this week | end of week"},
			{"2nd week, h2", "h2 | week 2"},
			{"on sept, past year", "last year | september"},
			{"to UTC, start of hour, today", "today | to UTC | start of hour"},
			{"today - 1w, monday", "today - 1w | monday"},
		}
		for _, tc := range orderTests {
			e, err := Parse(tc.input, EN)
			if err != nil {
				t.Fatalf("Parse(%q): %v", tc.input, err)
			}
			ordered, err := orderStages(e.Stages, false)
			if err != nil {
				t.Fatalf("orderStages(%q): %v", tc.input, err)
			}
			got := Expr{Stages: ordered}.String()
			if got != tc.want {
				t.Errorf("orderStages(%q) = %q, want %q", tc.input, got, tc.want)
			}
		}
	})

	t.Run("strict rejects out of order stages", func(t *testing.T) {
		c := ctx
		c.Cfg.Strict = true
		_, err := Eval("end of day, today", c)
		if err == nil {
			t.Fatal("expected error under strict")
		}
		if !strings.Contains(err.Error(), "stages are out of order; write: today | end of day") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Explain on ordered stages", func(t *testing.T) {
		steps, err := Explain("2026-11-14", ctx)
		if err != nil {
			t.Fatalf("Explain error: %v", err)
		}
		if len(steps) != 1 || steps[0].Text != "2026-11-14" || steps[0].Value.String() != "2026-11-14" {
			t.Errorf("Explain = %#v", steps)
		}
	})
}

func TestZone(t *testing.T) {
	baseCtx := testRefContext(t)
	nyLoc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("load NY location: %v", err)
	}

	tests := []struct {
		name    string
		input   string
		loc     *time.Location
		now     string
		strict  bool
		want    string
		wantErr string
	}{
		{name: "now to UTC", input: "now to UTC", want: "2026-11-14T20:30:00Z"},
		{name: "now to utc lowercase", input: "now to utc", want: "2026-11-14T20:30:00Z"},
		{name: "now to Asia/Tokyo", input: "now to Asia/Tokyo", want: "2026-11-15T05:30:00+09:00"},
		{name: "date time comma bare zone", input: "2026-11-14 14:30, America/New_York", want: "2026-11-14T14:30:00-05:00"},
		{name: "2pm @ Asia/Tokyo", input: "2pm @ Asia/Tokyo", want: "2026-11-14T14:00:00+09:00"},
		{name: "2026-11-14 14:30 in Asia/Tokyo", input: "2026-11-14 14:30 in Asia/Tokyo", want: "2026-11-14T14:30:00+09:00"},
		{name: "now in Asia/Tokyo converts", input: "now in Asia/Tokyo", want: "2026-11-15T05:30:00+09:00"},
		{name: "today | end of day | start of hour | to UTC", input: "today | end of day | start of hour | to UTC", want: "2026-11-15T05:00:00Z"},
		{name: "today to UTC uses 00:00 in ctx zone", input: "today to UTC", want: "2026-11-14T06:00:00Z"},
		{name: "2pm to UTC uses ctx today", input: "2pm to UTC", want: "2026-11-14T20:00:00Z"},
		{name: "offset datetime in Asia/Tokyo converts", input: "2026-11-14T14:30:00-06:00 in Asia/Tokyo", want: "2026-11-15T05:30:00+09:00"},
		{name: "offset datetime @ Asia/Tokyo reinterprets wall clock", input: "2026-11-14T14:30:00-06:00 @ Asia/Tokyo", want: "2026-11-14T14:30:00+09:00"},
		{name: "now to Atlantis error", input: "now to Atlantis", wantErr: "Atlantis"},
		{name: "bare zone after clock, to local CST", input: "sat 12:30 pm pst, to cst", want: "2026-11-14T14:30:00-06:00"},
		{name: "spanish bare zone after clock", input: "sáb 12:30 pm pst, to cst", want: "2026-11-14T14:30:00-06:00"},
		{name: "CST ambiguous outside local zone", input: "now to cst", loc: nyLoc, now: "2026-11-14T14:30:00-05:00", wantErr: "ambiguous"},
		{name: "CST ambiguous under strict", input: "now to cst", strict: true, wantErr: "ambiguous"},
		{name: "now to Local rejected", input: "now to Local", wantErr: "Local"},
		{
			name:    "strict NY DST spring forward nonexistent wall time",
			input:   "2026-03-08 02:30",
			loc:     nyLoc,
			strict:  true,
			wantErr: "does not exist",
		},
		{
			name:   "strict NY ctx but 02:30 in Asia/Tokyo succeeds",
			input:  "2026-03-08 02:30 in Asia/Tokyo",
			loc:    nyLoc,
			now:    "2026-03-08T12:00:00-04:00",
			strict: true,
			want:   "2026-03-08T02:30:00+09:00",
		},
		{
			name:    "strict NY DST fall back ambiguous wall time",
			input:   "2026-11-01 01:30",
			loc:     nyLoc,
			strict:  true,
			wantErr: "ambiguous",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := baseCtx
			ctx.Cfg.Strict = tc.strict
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
}
