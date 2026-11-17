package expr

import (
	"strings"
	"testing"
	"time"
)

func TestBoundaryAcronymsAndOptionalOf(t *testing.T) {
	ctx := refCtx(t)

	tests := []struct {
		input string
		want  string
	}{
		{"EoD", "2026-11-14T23:59:59-06:00"},
		{"eod", "2026-11-14T23:59:59-06:00"},
		{"SoD", "2026-11-14"},
		{"bod", "2026-11-14"},
		{"EoW", "2026-11-15T23:59:59-06:00"},
		{"SoW", "2026-11-09"},
		{"bow", "2026-11-09"},
		{"EoWW", "2026-11-13T23:59:59-06:00"},
		{"SoWW", "2026-11-09"},
		{"boww", "2026-11-09"},
		{"EoM", "2026-11-30T23:59:59-06:00"},
		{"eom", "2026-11-30T23:59:59-06:00"},
		{"SoM", "2026-11-01"},
		{"som", "2026-11-01"},
		{"bom", "2026-11-01"},
		{"EoQ", "2026-12-31T23:59:59-06:00"},
		{"SoQ", "2026-10-01"},
		{"boq", "2026-10-01"},
		{"EoY", "2026-12-31T23:59:59-06:00"},
		{"SoY", "2026-01-01"},
		{"boy", "2026-01-01"},
		{"tomorrow, EoD", "2026-11-15T23:59:59-06:00"},
		{"next month, SoD", "2026-12-01"},
		{"end day", "2026-11-14T23:59:59-06:00"},
		{"start day", "2026-11-14"},
		{"star of day", "2026-11-14"},
		{"star day", "2026-11-14"},
		{"end minute", "2026-11-14T14:30:59-06:00"},
		{"start minute", "2026-11-14T14:30:00-06:00"},
		{"start month", "2026-11-01"},
		{"end month", "2026-11-30T23:59:59-06:00"},
		{"end year", "2026-12-31T23:59:59-06:00"},
		{"end week", "2026-11-15T23:59:59-06:00"},
		{"start hour", "2026-11-14T14:00:00-06:00"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := Eval(tc.input, ctx)
			if err != nil {
				t.Fatalf("Eval(%q) error: %v", tc.input, err)
			}
			if got.String() != tc.want {
				t.Fatalf("Eval(%q) = %q, want %q", tc.input, got.String(), tc.want)
			}
		})
	}
}

func TestUnitConversionsAndCounts(t *testing.T) {
	ctx := refCtx(t)

	tests := []struct {
		input   string
		want    string
		wantErr string
	}{
		// Unit conversions with `in` and `to`
		{input: "yesterday 2pm - now, in hours", want: "-24.5"},
		{input: "yesterday 2pm - now in hours", want: "-24.5"},
		{input: "yesterday 2pm - now, in minutes", want: "-1470"},
		{input: "yesterday 2pm - now, in seconds", want: "-88200"},
		{input: "yesterday 2pm - now, in millis", want: "-88200000"},
		{input: "yesterday 2pm - now, in milliseconds", want: "-88200000"},
		{input: "yesterday 2pm - now, in ms", want: "-88200000"},
		{input: "yesterday 2pm - now, in days", want: "-1.02"},
		{input: "yesterday 2pm - now, in weeks", want: "-0.15"},
		{input: "yesterday 2pm - now, in months", want: "0"},
		{input: "yesterday 2pm - now, in quarters", want: "0"},
		{input: "yesterday 2pm - now, in halfs", want: "0"},
		{input: "yesterday 2pm - now, in years", want: "0"},
		{input: "2026-11-16 .. 2026-11-20 in days", want: "5"},
		{input: "2026-11-16 .. 2026-11-20 in business days", want: "4"},
		{input: "2026 in halves", want: "2"},
		{input: "2026 in halfs", want: "2"},
		{input: "2026 in quarters", want: "4"},

		// Direct duration conversions
		{input: "180s in hours", want: "0.05"},
		{input: "1500ms in seconds", want: "1.5"},
		{input: "1500 millis in milliseconds", want: "1500"},
		{input: "1y in halfs", want: "2"},
		{input: "1y in halves", want: "2"},
		{input: "1y in quarters", want: "4"},
		{input: "1y in months", want: "12"},
		{input: "6M in halfs", want: "1"},
		{input: "3M in years", want: "0.25"},
		{input: "2h * 3 in minutes", want: "360"},
		{input: "180s in halfs", wantErr: "no fixed length"},

		// Shorthand unit counts
		{input: "days to 2027-01-01", want: "48"},
		{input: "days to 2027", want: "48"},
		{input: "how many days to 2027", want: "48"},
		{input: "weeks to 2027", want: "6.86"},
		{input: "hours to tomorrow", want: "9.5"},
		{input: "minutes to 15:00", want: "30"},
		{input: "days from 2026-11-16 to 2026-11-20", want: "4"},
		{input: "days between 2026-11-16 and 2026-11-20", want: "4"},
		{input: "business days to 2026-11-20", want: "4"},
		{input: "mondays from today to 2026-12-31", want: "7"},
		{input: "how many mondays to 2026-12-31", want: "7"},
		{input: "days from 2026 to 2027", want: "365"},

		// today arithmetic promotes to now, while bare today remains a date
		{input: "today", want: "2026-11-14"},
		{input: "today + 3 days", want: "2026-11-17T14:30:00-06:00"},
		{input: "today - 1w", want: "2026-11-07T14:30:00-06:00"},
		{input: "today, EoD", want: "2026-11-14T23:59:59-06:00"},
		{input: "2026-12-24 - today", want: "40d"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
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
				t.Fatalf("Eval(%q) error: %v", tc.input, err)
			}
			if got.String() != tc.want {
				t.Fatalf("Eval(%q) = %q, want %q", tc.input, got.String(), tc.want)
			}
		})
	}
}

func TestConditionalsIf(t *testing.T) {
	ctx := refCtx(t) // Saturday 2026-11-14T14:30:00-06:00

	fridayCtx := ctx
	fridayCtx.Now = time.Date(2026, time.November, 13, 10, 0, 0, 0, ctx.Loc)

	wednesdayCtx := ctx
	wednesdayCtx.Now = time.Date(2026, time.November, 11, 10, 0, 0, 0, ctx.Loc)

	holidayCtx := ctx
	holidayCtx.Now = time.Date(2026, time.November, 16, 10, 0, 0, 0, ctx.Loc) // Revolution Day (MX)

	tests := []struct {
		name    string
		ctx     Context
		input   string
		want    string
		wantErr string
	}{
		{
			name:  "if on Saturday takes else branch",
			ctx:   ctx,
			input: "if today is friday, from today to next friday, else from last friday to this friday",
			want:  "2026-11-13 → 2026-11-13",
		},
		{
			name:  "if on Wednesday takes else branch",
			ctx:   wednesdayCtx,
			input: "if today is friday, from today to next friday, else from last friday to this friday",
			want:  "2026-11-06 → 2026-11-13",
		},
		{
			name:  "if on Friday takes then branch",
			ctx:   fridayCtx,
			input: "if today is friday, from today to next friday, else from last friday to this friday",
			want:  "2026-11-13 → 2026-11-20",
		},
		{
			name:  "is not weekday",
			ctx:   ctx,
			input: "if today is not friday, tomorrow, else today",
			want:  "2026-11-15",
		},
		{
			name:  "isnt weekend on Friday",
			ctx:   fridayCtx,
			input: "if today isnt weekend, today, else next monday",
			want:  "2026-11-13",
		},
		{
			name:  "is weekend on Saturday",
			ctx:   ctx,
			input: "if today is weekend, next monday, else today",
			want:  "2026-11-16",
		},
		{
			name:  "is weekday on Friday",
			ctx:   fridayCtx,
			input: "if today is weekday, today, else next monday",
			want:  "2026-11-13",
		},
		{
			name:  "is business day on holiday",
			ctx:   holidayCtx,
			input: "if today is business day, today, else next business day",
			want:  "2026-11-17",
		},
		{
			name:  "is holiday on MX holiday",
			ctx:   holidayCtx,
			input: "if today is holiday, next business day, else today",
			want:  "2026-11-17",
		},
		{
			name:  "month containment",
			ctx:   ctx,
			input: "if today is november, 1d, else 2d",
			want:  "1d",
		},
		{
			name:  "year containment",
			ctx:   ctx,
			input: "if today is 2026, 1d, else 2d",
			want:  "1d",
		},
		{
			name:  "numeric comparison with count",
			ctx:   ctx,
			input: "if days to 2027 > 30, 1M, else 1w",
			want:  "1M",
		},
		{
			name:  "before and after",
			ctx:   ctx,
			input: "if today before 2027-01-01, today, else 2027-01-01",
			want:  "2026-11-14",
		},
		{
			name:  "then and otherwise without commas",
			ctx:   ctx,
			input: "if now >= today 12:00 then tomorrow otherwise today",
			want:  "2026-11-15",
		},
		{
			name:  "pipe inside then and else",
			ctx:   ctx,
			input: "if today == 2026-11-14 | today | else tomorrow",
			want:  "2026-11-14",
		},
		{
			name:  "Spanish conditional",
			ctx:   ctx,
			input: "si hoy es viernes entonces hoy sino lunes",
			want:  "2026-11-09",
		},
		{
			name:    "error in untaken then branch is caught",
			ctx:     ctx,
			input:   "if today is friday, 2026-13-45, else today",
			wantErr: "invalid date",
		},
		{
			name:    "error in untaken else branch is caught",
			ctx:     ctx,
			input:   "if today is saturday, today, else 2026-13-45",
			wantErr: "invalid date",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Eval(tc.input, tc.ctx)
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
				t.Fatalf("Eval(%q) error: %v", tc.input, err)
			}
			if got.String() != tc.want {
				t.Fatalf("Eval(%q) = %q, want %q", tc.input, got.String(), tc.want)
			}
		})
	}
}

func TestAliasesAndValidation(t *testing.T) {
	ctx := refCtx(t)
	ctx.Cfg.Aliases = map[string]string{
		"oncall week": "if today is friday, from today to next friday, else from last friday to this friday",
		"standup":     "next monday 9am",
	}

	tests := []struct {
		name      string
		input     string
		want      string
		wantAlias string
		wantErr   string
	}{
		{
			name:  "use configured multi-word alias",
			input: "oncall week",
			want:  "2026-11-13 → 2026-11-13",
		},
		{
			name:  "case-insensitive alias with pipeline stage",
			input: "Oncall Week, to days",
			want:  "1",
		},
		{
			name:  "alias with timezone conversion",
			input: "standup to UTC",
			want:  "2026-11-16T15:00:00Z",
		},
		{
			name:      "define alias referencing another alias",
			input:     "standup shift = standup + 2h",
			want:      "2026-11-16T11:00:00-06:00",
			wantAlias: "standup shift",
		},
		{
			name:    "self-referential alias fails",
			input:   "bad loop = bad loop + 1d",
			wantErr: "circular alias",
		},
		{
			name:    "keyword alias rejected",
			input:   "today = tomorrow",
			wantErr: "conflicts with a built-in keyword",
		},
		{
			name:    "boundary acronym alias rejected",
			input:   "EoD = now",
			wantErr: "conflicts with a built-in keyword",
		},
		{
			name:    "timezone alias rejected",
			input:   "Tokyo = now",
			wantErr: "conflicts with a timezone",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := Run(tc.input, ctx, Options{}, false)
			if tc.wantErr != "" {
				if res.Error == nil {
					t.Fatalf("Run(%q) succeeded (%s), want error containing %q", tc.input, res.Text, tc.wantErr)
				}
				if !strings.Contains(res.Error.Msg, tc.wantErr) {
					t.Fatalf("Run(%q) error = %q, want substring %q", tc.input, res.Error.Msg, tc.wantErr)
				}
				return
			}
			if res.Error != nil {
				t.Fatalf("Run(%q) unexpected error: %+v", tc.input, res.Error)
			}
			if res.Text != tc.want {
				t.Fatalf("Run(%q).Text = %q, want %q", tc.input, res.Text, tc.want)
			}
			if res.Alias != tc.wantAlias {
				t.Fatalf("Run(%q).Alias = %q, want %q", tc.input, res.Alias, tc.wantAlias)
			}
		})
	}

	// Mutual recursion cycle check
	cycCtx := refCtx(t)
	cycCtx.Cfg.Aliases = map[string]string{
		"alpha": "beta + 1d",
		"beta":  "alpha + 1d",
	}
	if _, err := Eval("alpha", cycCtx); err == nil || !strings.Contains(err.Error(), "circular alias") {
		t.Fatalf("Eval(alpha) with mutual cycle = %v, want circular alias error", err)
	}
}

func TestMarshalYAMLAndFormatPresets(t *testing.T) {
	ctx := refCtx(t)

	for _, pat := range []string{"json", "yaml", "yml"} {
		res := Run("today", ctx, Options{Pattern: pat}, true)
		if res.Error != nil {
			t.Fatalf("Run(today, %q) error: %+v", pat, res.Error)
		}
		if res.Text != "2026-11-14" {
			t.Fatalf("Run(today, %q).Text = %q, want 2026-11-14", pat, res.Text)
		}
	}

	res := Run("oncall week = today", ctx, Options{}, true)
	yml := MarshalYAML(res)
	for _, wantSub := range []string{
		`input: "oncall week = today"`,
		`alias: "oncall week"`,
		`kind: "date"`,
		`text: "2026-11-14"`,
		`iso: "2026-11-14"`,
		`unix: `,
		`explain:`,
		`- stage: "today"`,
	} {
		if !strings.Contains(yml, wantSub) {
			t.Errorf("MarshalYAML missing %q in:\n%s", wantSub, yml)
		}
	}

	errRes := Run("now +", ctx, Options{}, false)
	errYAML := MarshalYAML(errRes)
	for _, wantSub := range []string{
		`input: "now +"`,
		`error:`,
		`pos: 5`,
		`message: "unexpected end of input"`,
		`expected:`,
	} {
		if !strings.Contains(errYAML, wantSub) {
			t.Errorf("MarshalYAML error output missing %q in:\n%s", wantSub, errYAML)
		}
	}
}
