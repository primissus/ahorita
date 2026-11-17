package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/primissus/ahorita/expr"
)

func testRuntime(defaultTZ string) runtimeEnv {
	loc, err := time.LoadLocation("America/Mexico_City")
	if err != nil {
		loc = time.FixedZone("CST", -6*3600)
	}
	fixedNow := time.Date(2026, time.November, 14, 14, 30, 0, 0, loc)
	return runtimeEnv{
		now:       func() time.Time { return fixedNow },
		defaultTZ: func() string { return defaultTZ },
	}
}

func TestRunBridge(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		optsJSON  string
		defaultTZ string
		wantKind  string
		wantText  string
		wantNote  string
		wantErr   string
		wantPos   *int
		wantExp   bool
		wantSteps int
	}{
		{
			name:      "date arithmetic with default tz",
			input:     "2026-06-02 + 2d",
			optsJSON:  "{}",
			defaultTZ: "America/Mexico_City",
			wantKind:  "date",
			wantText:  "2026-06-04",
		},
		{
			name:      "empty options string uses default tz",
			input:     "today",
			optsJSON:  "",
			defaultTZ: "America/Mexico_City",
			wantKind:  "date",
			wantText:  "2026-11-14",
		},
		{
			name:      "missing tz without default is error not UTC",
			input:     "now",
			optsJSON:  "{}",
			defaultTZ: "",
			wantErr:   "missing tz",
		},
		{
			name:      "explicit empty tz is error",
			input:     "now",
			optsJSON:  `{"tz":""}`,
			defaultTZ: "America/Mexico_City",
			wantErr:   "invalid tz",
		},
		{
			name:      "explicit Local tz is rejected",
			input:     "now",
			optsJSON:  `{"tz":"Local"}`,
			defaultTZ: "America/Mexico_City",
			wantErr:   "invalid tz",
		},
		{
			name:      "default Local tz is rejected",
			input:     "now",
			optsJSON:  `{}`,
			defaultTZ: "Local",
			wantErr:   "invalid tz",
		},
		{
			name:      "explicit now and tz with explain and format",
			input:     "today - 1w, monday",
			optsJSON:  `{"now":"2026-11-14T14:30:00-06:00","tz":"America/Mexico_City","format":"short","explain":true}`,
			wantKind:  "date",
			wantText:  "11-02-2026",
			wantSteps: 2,
		},
		{
			name:     "show_timezone appends timezone",
			input:    "now",
			optsJSON: `{"tz":"America/Mexico_City","format":"human","show_timezone":true}`,
			wantKind: "datetime",
			wantText: "Sat, Nov 14, 2026, 2:30 PM America/Mexico_City",
		},
		{
			name:      "type relative in Spanish",
			input:     "hace 2 días",
			optsJSON:  `{"type":"relative","lang":"es"}`,
			defaultTZ: "America/Mexico_City",
			wantKind:  "datetime",
			wantText:  "hace 2 días",
		},
		{
			name:      "strict rejects ambiguous stage order",
			input:     "end of day, today",
			optsJSON:  `{"strict":true}`,
			defaultTZ: "America/Mexico_City",
			wantErr:   "out of order",
		},
		{
			name:      "strict string boolean accepted",
			input:     "end of day, today",
			optsJSON:  `{"strict":"true"}`,
			defaultTZ: "America/Mexico_City",
			wantErr:   "out of order",
		},
		{
			name:      "business days with country none and weekend array and days_off",
			input:     "2026-11-13 + 2 business days",
			optsJSON:  `{"country":"none","weekend":["sunday"],"days_off":["2026-11-14"]}`,
			defaultTZ: "America/Mexico_City",
			wantKind:  "date",
			wantText:  "2026-11-17",
		},
		{
			name:      "business days with MX default skips Nov 16",
			input:     "2026-11-13 + 3 business days",
			optsJSON:  `{}`,
			defaultTZ: "America/Mexico_City",
			wantKind:  "date",
			wantText:  "2026-11-19",
		},
		{
			name:     "cron zone note propagated",
			input:    "every monday 8pm, to UTC, to cron",
			optsJSON: `{"tz":"America/Mexico_City"}`,
			wantKind: "text",
			wantText: "0 2 * * 2",
			wantNote: "weekday shifts from Monday to Tuesday in UTC",
		},
		{
			name:      "month overflow and count inclusive and week settings",
			input:     "2026-01-31 + 1M",
			optsJSON:  `{"month_overflow":"overflow","week_start":"sunday","week_one":"first-day","count_inclusive":true}`,
			defaultTZ: "America/Mexico_City",
			wantKind:  "date",
			wantText:  "2026-03-03",
		},
		{
			name:      "parse error includes pos and expected",
			input:     "now +",
			optsJSON:  `{}`,
			defaultTZ: "America/Mexico_City",
			wantErr:   "unexpected end of input",
			wantPos:   ptrInt(5),
			wantExp:   true,
		},
		{
			name:      "unknown JSON option key errors",
			input:     "today",
			optsJSON:  `{"bogus":"x"}`,
			defaultTZ: "America/Mexico_City",
			wantErr:   "invalid options JSON",
		},
		{
			name:      "calendar path in browser rejected",
			input:     "today",
			optsJSON:  `{"calendar":"/tmp/off.txt"}`,
			defaultTZ: "America/Mexico_City",
			wantErr:   "days_off",
		},
		{
			name:      "invalid now timestamp rejected",
			input:     "today",
			optsJSON:  `{"now":"2026-11-14"}`,
			defaultTZ: "America/Mexico_City",
			wantErr:   "invalid now",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw := runBridge(tc.input, tc.optsJSON, testRuntime(tc.defaultTZ))
			var res expr.Result
			if err := json.Unmarshal([]byte(raw), &res); err != nil {
				t.Fatalf("invalid JSON returned: %v (%s)", err, raw)
			}
			if tc.wantErr != "" {
				if res.Error == nil {
					t.Fatalf("expected error containing %q, got success %+v", tc.wantErr, res)
				}
				if !strings.Contains(res.Error.Msg, tc.wantErr) {
					t.Errorf("error.message = %q, want substring %q", res.Error.Msg, tc.wantErr)
				}
				if tc.wantPos != nil && res.Error.Pos != *tc.wantPos {
					t.Errorf("error.pos = %d, want %d", res.Error.Pos, *tc.wantPos)
				}
				if tc.wantExp && len(res.Error.Expected) == 0 {
					t.Errorf("error.expected = empty, want suggestions")
				}
				return
			}
			if res.Error != nil {
				t.Fatalf("unexpected error: %+v", res.Error)
			}
			if res.Kind != tc.wantKind {
				t.Errorf("kind = %q, want %q", res.Kind, tc.wantKind)
			}
			if res.Text != tc.wantText {
				t.Errorf("text = %q, want %q", res.Text, tc.wantText)
			}
			if tc.wantNote != "" && !strings.Contains(res.Note, tc.wantNote) {
				t.Errorf("note = %q, want substring %q", res.Note, tc.wantNote)
			}
			if tc.wantSteps > 0 && len(res.Explain) != tc.wantSteps {
				t.Errorf("explain len = %d, want %d", len(res.Explain), tc.wantSteps)
			}
		})
	}
}

func ptrInt(v int) *int { return &v }
