package expr

import (
	"errors"
	"strings"
	"testing"
	"time"
	_ "time/tzdata"
)

func TestHolidays(t *testing.T) {
	mxLoc, err := time.LoadLocation("America/Mexico_City")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}

	t.Run("MX 2026", func(t *testing.T) {
		got, err := holidays("MX", 2026, mxLoc)
		if err != nil {
			t.Fatalf("holidays(MX, 2026): %v", err)
		}
		want := []struct {
			date string
			name string
		}{
			{"2026-01-01", "New Year's Day"},
			{"2026-02-02", "Constitution Day"},
			{"2026-03-16", "Benito Juárez's Birthday"},
			{"2026-05-01", "Labour Day"},
			{"2026-09-16", "Independence Day"},
			{"2026-11-16", "Revolution Day"},
			{"2026-12-25", "Christmas Day"},
		}
		if len(got) != len(want) {
			t.Fatalf("len(MX 2026) = %d, want %d: %+v", len(got), len(want), got)
		}
		for i, w := range want {
			if gotDate := got[i].Date.Format("2006-01-02"); gotDate != w.date || got[i].Name != w.name {
				t.Errorf("MX 2026[%d] = (%s, %q), want (%s, %q)", i, gotDate, got[i].Name, w.date, w.name)
			}
		}
	})

	t.Run("MX inauguration sexennial years", func(t *testing.T) {
		for _, tc := range []struct {
			year    int
			wantLen int
			hasOct1 bool
		}{
			{2018, 7, false},
			{2024, 8, true},
			{2025, 7, false},
			{2030, 8, true},
			{2036, 8, true},
		} {
			got, err := holidays("MX", tc.year, mxLoc)
			if err != nil {
				t.Fatalf("holidays(MX, %d): %v", tc.year, err)
			}
			if len(got) != tc.wantLen {
				t.Errorf("len(MX %d) = %d, want %d", tc.year, len(got), tc.wantLen)
			}
			foundOct1 := false
			for _, h := range got {
				if h.Date.Month() == time.October && h.Date.Day() == 1 {
					foundOct1 = true
				}
			}
			if foundOct1 != tc.hasOct1 {
				t.Errorf("MX %d Oct 1 present = %v, want %v", tc.year, foundOct1, tc.hasOct1)
			}
		}
	})

	t.Run("US 2026 and 2027 cross-year observed New Year", func(t *testing.T) {
		got26, err := holidays("US", 2026, mxLoc)
		if err != nil {
			t.Fatalf("holidays(US, 2026): %v", err)
		}
		if len(got26) != 11 {
			t.Fatalf("len(US 2026) = %d, want 11: %+v", len(got26), got26)
		}
		hasJuly3 := false
		for _, h := range got26 {
			if h.Date.Format("2006-01-02") == "2026-07-03" {
				hasJuly3 = true
			}
		}
		if !hasJuly3 {
			t.Errorf("US 2026 should observe Independence Day on 2026-07-03")
		}

		// 2028-01-01 is Saturday, so US observes 2028 New Year's Day on Friday 2027-12-31.
		got27, err := holidays("us", 2027, mxLoc)
		if err != nil {
			t.Fatalf("holidays(us, 2027): %v", err)
		}
		hasDec31 := false
		for _, h := range got27 {
			if h.Date.Format("2006-01-02") == "2027-12-31" {
				hasDec31 = true
			}
		}
		if !hasDec31 {
			t.Errorf("US 2027 should observe 2028 New Year's Day on 2027-12-31")
		}

		got28, err := holidays("US", 2028, mxLoc)
		if err != nil {
			t.Fatalf("holidays(US, 2028): %v", err)
		}
		for _, h := range got28 {
			if h.Date.Year() != 2028 {
				t.Errorf("US 2028 contains out-of-year holiday: %v", h)
			}
		}
	})

	t.Run("unknown country returns Error", func(t *testing.T) {
		_, err := holidays("XX", 2026, mxLoc)
		if err == nil {
			t.Fatal("expected error for XX")
		}
		var evalErr *Error
		if !errors.As(err, &evalErr) {
			t.Fatalf("err = %T, want *Error", err)
		}
		if !strings.Contains(evalErr.Msg, "XX") {
			t.Errorf("err.Msg = %q, want mention of XX", evalErr.Msg)
		}
	})

	t.Run("all supported countries succeed", func(t *testing.T) {
		for _, code := range supportedCountries() {
			got, err := holidays(code, 2026, mxLoc)
			if err != nil {
				t.Errorf("holidays(%q, 2026) error: %v", code, err)
			}
			for _, h := range got {
				if h.Date.Location() != mxLoc {
					t.Errorf("holidays(%q) returned location %v, want %v", code, h.Date.Location(), mxLoc)
					break
				}
				if h.Date.Year() != 2026 {
					t.Errorf("holidays(%q, 2026) returned year %d", code, h.Date.Year())
				}
			}
		}
	})

	t.Run("isBusinessDay", func(t *testing.T) {
		ctx := testRefContext(t)
		checks := []struct {
			name string
			date time.Time
			mod  func(*Config)
			want bool
		}{
			{
				name: "Saturday weekend false",
				date: time.Date(2026, 11, 14, 0, 0, 0, 0, mxLoc),
				want: false,
			},
			{
				name: "Sunday weekend false",
				date: time.Date(2026, 11, 15, 0, 0, 0, 0, mxLoc),
				want: false,
			},
			{
				name: "MX Revolution Day false",
				date: time.Date(2026, 11, 16, 0, 0, 0, 0, mxLoc),
				want: false,
			},
			{
				name: "no-holiday config makes 2026-11-16 true",
				date: time.Date(2026, 11, 16, 0, 0, 0, 0, mxLoc),
				mod:  func(c *Config) { c.Country = "" },
				want: true,
			},
			{
				name: "weekday true",
				date: time.Date(2026, 11, 17, 0, 0, 0, 0, mxLoc),
				want: true,
			},
			{
				name: "custom weekend Fri/Sat",
				date: time.Date(2026, 11, 13, 0, 0, 0, 0, mxLoc),
				mod:  func(c *Config) { c.Weekend = []time.Weekday{time.Friday, time.Saturday} },
				want: false,
			},
			{
				name: "custom weekend makes Sunday business day",
				date: time.Date(2026, 11, 15, 0, 0, 0, 0, mxLoc),
				mod:  func(c *Config) { c.Weekend = []time.Weekday{time.Friday, time.Saturday} },
				want: true,
			},
			{
				name: "DaysOff single KDate",
				date: time.Date(2026, 11, 17, 0, 0, 0, 0, mxLoc),
				mod: func(c *Config) {
					c.DaysOff = []Value{{Kind: KDate, T: time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)}}
				},
				want: false,
			},
			{
				name: "DaysOff KSpan inclusive start exclusive end",
				date: time.Date(2026, 12, 31, 0, 0, 0, 0, mxLoc),
				mod: func(c *Config) {
					c.DaysOff = []Value{{
						Kind: KSpan,
						T:    time.Date(2026, 12, 24, 0, 0, 0, 0, mxLoc),
						End:  time.Date(2027, 1, 3, 0, 0, 0, 0, mxLoc),
						Days: true,
					}}
				},
				want: false,
			},
			{
				name: "DaysOff KSpan end is exclusive",
				date: time.Date(2027, 1, 4, 0, 0, 0, 0, mxLoc),
				mod: func(c *Config) {
					c.DaysOff = []Value{{
						Kind: KSpan,
						T:    time.Date(2026, 12, 24, 0, 0, 0, 0, mxLoc),
						End:  time.Date(2027, 1, 4, 0, 0, 0, 0, mxLoc),
						Days: true,
					}}
				},
				want: true,
			},
		}
		for _, tc := range checks {
			t.Run(tc.name, func(t *testing.T) {
				c := ctx
				if tc.mod != nil {
					tc.mod(&c.Cfg)
				}
				got, err := isBusinessDay(tc.date, c)
				if err != nil {
					t.Fatalf("isBusinessDay error: %v", err)
				}
				if got != tc.want {
					t.Errorf("isBusinessDay(%s) = %v, want %v", tc.date.Format("2006-01-02"), got, tc.want)
				}
			})
		}
	})
}

func TestBusiness(t *testing.T) {
	ctx := testRefContext(t)

	tests := []struct {
		name    string
		input   string
		cfgMod  func(*Config)
		want    string
		wantErr string
	}{
		// Arithmetic +/-
		{name: "add 3 business days MX", input: "2026-11-13 + 3 business days", want: "2026-11-19"},
		{
			name:   "add 3 business days no holidays",
			input:  "2026-11-13 + 3 business days",
			cfgMod: func(c *Config) { c.Country = "" },
			want:   "2026-11-18",
		},
		{name: "sub 1 business day MX", input: "2026-11-17 - 1 business day", want: "2026-11-13"},
		{
			name:   "sub 1 business day no holidays",
			input:  "2026-11-17 - 1 business day",
			cfgMod: func(c *Config) { c.Country = "" },
			want:   "2026-11-16",
		},
		{name: "add 0 business days stays on date", input: "2026-11-14 + 0 business days", want: "2026-11-14"},
		{name: "datetime + business days keeps clock", input: "2026-11-13 14:30 + 3 business days", want: "2026-11-19T14:30:00-06:00"},
		{name: "in 3 business days", input: "in 3 business days", want: "2026-11-19T14:30:00-06:00"},
		{name: "1 business day ago from Tue", input: "2026-11-17 - 1 business day", want: "2026-11-13"},
		{
			name:  "custom weekend Fri/Sat add 1 business day",
			input: "2026-11-12 + 1 business day",
			cfgMod: func(c *Config) {
				c.Country = ""
				c.Weekend = []time.Weekday{time.Friday, time.Saturday}
			},
			want: "2026-11-15",
		},
		{
			name:  "weekend none counts Sat and Sun",
			input: "2026-11-13 + 3 business days",
			cfgMod: func(c *Config) {
				c.Weekend = []time.Weekday{}
			},
			want: "2026-11-17",
		},
		{
			name:  "DaysOff skipped by addition",
			input: "2026-11-16 + 1 business day",
			cfgMod: func(c *Config) {
				c.DaysOff = []Value{{Kind: KDate, T: time.Date(2026, 11, 17, 0, 0, 0, 0, ctx.Loc)}}
			},
			want: "2026-11-18",
		},
		// Relative next/last/this business day
		{name: "next business day from Sat MX", input: "2026-11-14, next business day", want: "2026-11-17"},
		{
			name:   "next business day from Sat no holidays",
			input:  "2026-11-14, next business day",
			cfgMod: func(c *Config) { c.Country = "" },
			want:   "2026-11-16",
		},
		{name: "last business day from Tue MX", input: "2026-11-17, last business day", want: "2026-11-13"},
		{
			name:   "last business day from Tue no holidays",
			input:  "2026-11-17, last business day",
			cfgMod: func(c *Config) { c.Country = "" },
			want:   "2026-11-16",
		},
		{name: "next business day from holiday MX", input: "2026-11-16, next business day", want: "2026-11-17"},
		{name: "last business day from holiday MX", input: "2026-11-16, last business day", want: "2026-11-13"},
		{name: "this business day error", input: "this business day", wantErr: "next or last"},
		{name: "next business day with clock", input: "2026-11-14, next business day 9:00", want: "2026-11-17T09:00:00-06:00"},
		// Counting & to business days
		{name: "next month to business days MX", input: "next month, to business days", want: "22 business days"},
		{
			name:   "next month to business days no holidays",
			input:  "next month, to business days",
			cfgMod: func(c *Config) { c.Country = "" },
			want:   "23 business days",
		},
		{name: "span to business days MX", input: "2026-11-16 .. 2026-11-20, to business days", want: "4 business days"},
		{
			name:   "span to business days no holidays",
			input:  "2026-11-16 .. 2026-11-20, to business days",
			cfgMod: func(c *Config) { c.Country = "" },
			want:   "5 business days",
		},
		{name: "diff to business days MX", input: "2026-11-20 - 2026-11-16, to business days", want: "4 business days"},
		{name: "diff to business days inclusive MX", input: "2026-11-20 - 2026-11-16, to business days inclusive", want: "4 business days"},
		{
			name:   "diff to business days inclusive no holidays",
			input:  "2026-11-20 - 2026-11-16, to business days inclusive",
			cfgMod: func(c *Config) { c.Country = "" },
			want:   "5 business days",
		},
		{name: "negative diff to business days", input: "2026-11-13 - 2026-11-20, to business days", want: "-4 business days"},
		{name: "how many business days MX", input: "how many business days between 2026-11-13 and 2026-11-20", want: "4 business days"},
		{
			name:   "how many business days no holidays",
			input:  "how many business days between 2026-11-13 and 2026-11-20",
			cfgMod: func(c *Config) { c.Country = "" },
			want:   "5 business days",
		},
		{name: "how many business days inclusive MX", input: "how many business days between 2026-11-13 and 2026-11-20 inclusive", want: "5 business days"},
		{name: "during business days to days", input: "during 3 business days, to days", want: "5 days"},
		{name: "3 business days to days errors", input: "3 business days to days", wantErr: "no fixed length"},
		{name: "2d to business days errors", input: "2d to business days", wantErr: "give a span"},
		// on or after / on or before
		{name: "next business day on or after Sat MX", input: "next business day on or after 2026-11-14", want: "2026-11-17"},
		{name: "next business day on or after holiday MX", input: "next business day on or after 2026-11-16", want: "2026-11-17"},
		{name: "next business day on or after business day", input: "next business day on or after 2026-11-17", want: "2026-11-17"},
		{
			name:   "next business day on or after no holidays Mon",
			input:  "next business day on or after 2026-11-16",
			cfgMod: func(c *Config) { c.Country = "" },
			want:   "2026-11-16",
		},
		{name: "pipeline next business day on or after", input: "2026-11-14, next business day on or after", want: "2026-11-17"},
		{name: "pipeline next business day on or after same day", input: "2026-11-17, next business day on or after", want: "2026-11-17"},
		{name: "next business day on or after with clock", input: "next business day on or after 2026-11-14 9:00", want: "2026-11-17T09:00:00-06:00"},
		{name: "last business day on or before holiday MX", input: "last business day on or before 2026-11-16", want: "2026-11-13"},
		{name: "last business day on or before business day", input: "last business day on or before 2026-11-17", want: "2026-11-17"},
		{name: "pipeline last business day on or before", input: "2026-11-16, last business day on or before", want: "2026-11-13"},
		{name: "mismatched last business day on or after", input: "last business day on or after 2026-11-14", wantErr: "next business day"},
		{name: "mismatched next business day on or before", input: "next business day on or before 2026-11-14", wantErr: "last business day"},
		{name: "mismatched this business day on or after", input: "this business day on or after 2026-11-14", wantErr: "next business day"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := ctx
			if tc.cfgMod != nil {
				tc.cfgMod(&c.Cfg)
			}
			got, err := Eval(tc.input, c)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("Eval(%q) = %s, want error containing %q", tc.input, got.String(), tc.wantErr)
				}
				var evalErr *Error
				if !errors.As(err, &evalErr) {
					t.Fatalf("err = %T, want *Error", err)
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

func TestHolidayList(t *testing.T) {
	ctx := testRefContext(t)

	tests := []struct {
		name      string
		input     string
		cfgMod    func(*Config)
		wantKind  Kind
		want      string
		wantCount int
		wantSub   []string
		wantErr   string
	}{
		{
			name:     "holidays defaults to cfg Country and Now year",
			input:    "holidays",
			wantKind: KList,
			want: strings.Join([]string{
				"2026-01-01  New Year's Day",
				"2026-02-02  Constitution Day",
				"2026-03-16  Benito Juárez's Birthday",
				"2026-05-01  Labour Day",
				"2026-09-16  Independence Day",
				"2026-11-16  Revolution Day",
				"2026-12-25  Christmas Day",
			}, "\n"),
			wantCount: 7,
		},
		{
			name:      "holidays MX 2024 includes inauguration day",
			input:     "holidays MX 2024",
			wantKind:  KList,
			wantCount: 8,
			wantSub:   []string{"2024-10-01  Inauguration Day"},
		},
		{
			name:      "holidays US 2026",
			input:     "holidays US 2026",
			wantKind:  KList,
			wantCount: 11,
			wantSub:   []string{"2026-07-03  Independence Day", "2026-11-26  Thanksgiving Day"},
		},
		{
			name:     "feriados MX 2026 uses Spanish names",
			input:    "feriados MX 2026",
			wantKind: KList,
			want: strings.Join([]string{
				"2026-01-01  Año Nuevo",
				"2026-02-02  Día de la Constitución",
				"2026-03-16  Natalicio de Benito Juárez",
				"2026-05-01  Día del Trabajo",
				"2026-09-16  Día de la Independencia",
				"2026-11-16  Día de la Revolución",
				"2026-12-25  Navidad",
			}, "\n"),
			wantCount: 7,
		},
		{
			name:      "Spanish dias festivos in pipeline",
			input:     "2026, días festivos",
			wantKind:  KList,
			wantCount: 7,
			wantSub:   []string{"2026-11-16  Día de la Revolución"},
		},
		{
			name:      "next month holidays filters inside span",
			input:     "next month, holidays",
			wantKind:  KList,
			wantCount: 1,
			want:      "2026-12-25  Christmas Day",
		},
		{
			name:      "span cross-year holidays",
			input:     "2026-12-20 .. 2027-01-05, holidays",
			wantKind:  KList,
			wantCount: 2,
			want:      "2026-12-25  Christmas Day\n2027-01-01  New Year's Day",
		},
		{
			name:      "holidays DE 2026 does not collide with Spanish de",
			input:     "holidays DE 2026",
			wantKind:  KList,
			wantCount: 9,
			wantSub:   []string{"2026-10-03  Tag der Deutschen Einheit"},
		},
		{
			name:      "holidays AT 2026 does not drop AT filler",
			input:     "holidays AT 2026",
			wantKind:  KList,
			wantCount: 13,
			wantSub:   []string{"2026-10-26  Nationalfeiertag"},
		},
		{
			name:      "holidays HR 2026 does not collide with hour",
			input:     "holidays HR 2026",
			wantKind:  KList,
			wantCount: 14,
		},
		{
			name:      "no holiday config yields empty list",
			input:     "holidays 2026",
			cfgMod:    func(c *Config) { c.Country = "" },
			wantKind:  KList,
			wantCount: 0,
			want:      "",
		},
		{
			name:    "unknown country errors",
			input:   "holidays XX 2026",
			wantErr: "XX",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := ctx
			if tc.cfgMod != nil {
				tc.cfgMod(&c.Cfg)
			}
			got, err := Eval(tc.input, c)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("Eval(%q) = %s, want error containing %q", tc.input, got.String(), tc.wantErr)
				}
				var evalErr *Error
				if !errors.As(err, &evalErr) {
					t.Fatalf("err = %T, want *Error", err)
				}
				if !strings.Contains(evalErr.Msg, tc.wantErr) {
					t.Errorf("error = %q, want substring %q", evalErr.Msg, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Eval(%q) error: %v", tc.input, err)
			}
			if got.Kind != tc.wantKind {
				t.Errorf("Eval(%q).Kind = %v, want %v", tc.input, got.Kind, tc.wantKind)
			}
			if len(got.Items) != tc.wantCount {
				t.Errorf("len(Items) = %d, want %d (%q)", len(got.Items), tc.wantCount, got.String())
			}
			for _, item := range got.Items {
				if item.Kind != KDate {
					t.Errorf("item kind = %v, want KDate", item.Kind)
				}
			}
			if tc.want != "" && got.String() != tc.want {
				t.Errorf("Eval(%q) =\n%s\nwant:\n%s", tc.input, got.String(), tc.want)
			}
			for _, sub := range tc.wantSub {
				if !strings.Contains(got.String(), sub) {
					t.Errorf("Eval(%q) missing %q in:\n%s", tc.input, sub, got.String())
				}
			}
		})
	}
}
