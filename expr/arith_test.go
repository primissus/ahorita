package expr

import (
	"errors"
	"strings"
	"testing"
	"time"
	_ "time/tzdata"
)

func TestArith(t *testing.T) {
	ctx := testRefContext(t)
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}

	tests := []struct {
		name    string
		input   string
		loc     *time.Location
		cfgMod  func(*Config)
		want    string
		wantErr string
	}{
		{name: "date plus days", input: "2026-06-02 + 2d", want: "2026-06-04"},
		{name: "date minus hour", input: "2026-06-02 - 1h", want: "2026-06-01T23:00:00-06:00"},
		{name: "now minus day", input: "now - 1d", want: "2026-11-13T14:30:00-06:00"},
		{name: "2d ago", input: "2d ago", want: "2026-11-12T14:30:00-06:00"},
		{name: "in 3 months", input: "in 3 months", want: "2027-02-14T14:30:00-06:00"},
		{name: "jan 31 plus 1M clamped", input: "2026-01-31 + 1M", want: "2026-02-28"},
		{
			name:   "jan 31 plus 1M overflow",
			input:  "2026-01-31 + 1M",
			cfgMod: func(c *Config) { c.MonthOverflow = true },
			want:   "2026-03-03",
		},
		{name: "leap year jan 31 plus 1M", input: "2028-01-31 + 1M", want: "2028-02-29"},
		{name: "mar 31 minus 1M", input: "2026-03-31 - 1M", want: "2026-02-28"},
		{name: "leap feb 29 plus 1y", input: "2028-02-29 + 1y", want: "2029-02-28"},
		{name: "date diff positive", input: "2026-12-24 - 2026-11-14", want: "40d"},
		{name: "date diff negative", input: "2026-11-14 - 2026-12-24", want: "-40d"},
		{name: "datetime diff", input: "2026-11-16T18:30:00-06:00 - 2026-11-14T14:30:00-06:00", want: "2d 4h"},
		{name: "mul dur by num", input: "2h * 3", want: "6h"},
		{name: "mul num by dur", input: "3 * 2h", want: "6h"},
		{name: "mul dur by fraction", input: "2h * 1.5", want: "3h"},
		{name: "add durations", input: "1h30m + 45m", want: "2h 15m"},
		{name: "bare duration folded", input: "90m", want: "1h 30m"},
		{name: "fractional hours", input: "1.5h", want: "1h 30m"},
		{name: "time wraps forward past midnight", input: "23:30 + 1h", want: "00:30"},
		{name: "time wraps backward past midnight", input: "00:30 - 1h", want: "23:30"},
		{name: "span shifted forward by duration", input: "2026 + 2d", want: "2026-01-03 → 2027-01-02"},
		{name: "span shifted backward by duration", input: "2026 - 1d", want: "2025-12-31 → 2026-12-30"},
		{name: "today plus today error", input: "today + today", wantErr: "cannot add"},
		{name: "today mul 2 error", input: "today * 2", wantErr: "cannot multiply"},
		{name: "2M mul 1.5 error", input: "2M * 1.5", wantErr: "fractional"},
		{name: "fractional day error", input: "1.5d", wantErr: "fractional"},
		{name: "add business days", input: "2026-11-13 + 3 business days", want: "2026-11-19"},
		{name: "business day duration value", input: "3 business days", want: "3 business days"},
		{name: "business day combine error", input: "2d 1 business day", wantErr: "business days can't be combined with other units"},
		{name: "business day plus day error", input: "1 business day + 2d", wantErr: "business days can't be combined with other units"},
		{name: "fractional business day error", input: "1.5 business days", wantErr: "fractional values aren't supported for business days"},
		{
			name:  "NY DST spring forward add 1d keeps wall clock",
			input: "2026-03-07 12:00 + 1d",
			loc:   ny,
			want:  "2026-03-08T12:00:00-04:00",
		},
		{
			name:  "NY DST spring forward add 24h adds 24 real hours",
			input: "2026-03-07 12:00 + 24h",
			loc:   ny,
			want:  "2026-03-08T13:00:00-04:00",
		},
		{
			name:  "ago and in from ctx.Now not stage",
			input: "2d ago",
			want:  "2026-11-12T14:30:00-06:00",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := ctx
			if tc.loc != nil {
				c.Loc = tc.loc
				c.Now = ctx.Now.In(tc.loc)
			}
			if tc.cfgMod != nil {
				tc.cfgMod(&c.Cfg)
			}
			got, err := Eval(tc.input, c)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("Eval(%q) succeeded (%v), want error containing %q", tc.input, got, tc.wantErr)
				}
				var evalErr *Error
				if !errors.As(err, &evalErr) {
					t.Fatalf("error type = %T, want *Error", err)
				}
				if !strings.Contains(evalErr.Msg, tc.wantErr) {
					t.Errorf("error = %q, want substring %q", evalErr.Msg, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Eval(%q) error: %v", tc.input, err)
			}
			if got.String() != tc.want {
				t.Errorf("Eval(%q) = %q, want %q", tc.input, got.String(), tc.want)
			}
		})
	}
}
