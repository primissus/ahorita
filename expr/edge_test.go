package expr

import (
	"testing"
	"time"
)

func mustLoc(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("LoadLocation(%q): %v", name, err)
	}
	return loc
}

func TestEdge(t *testing.T) {
	ny := mustLoc(t, "America/New_York")
	mx := mustLoc(t, "America/Mexico_City")

	type edgeCase struct {
		name    string
		input   string
		ctx     Context
		fmtOpts *Options
		want    string
		wantErr bool
		skipBug string
	}

	// Independent calendar calculations for 2026-2029:
	// - 2026 is a common year (Jan 1 = Thu, Mar 1 = Sun, May 1 = Fri, Jul 1 = Wed, Nov 1 = Sun, Dec 31 = Thu).
	// - America/New_York DST 2026:
	//   * Starts 2nd Sunday of March = 2026-03-08 (02:00 EST -> 03:00 EDT; offset jumps -05:00 -> -04:00).
	//   * Ends 1st Sunday of November = 2026-11-01 (02:00 EDT -> 01:00 EST; offset falls back -04:00 -> -05:00).
	// - Leap years: 2028 is leap (Feb 29 exists); 2026, 2027, 2029 are common (Feb has 28 days).

	nyMar7 := Context{
		Now: time.Date(2026, 3, 7, 12, 0, 0, 0, ny),
		Loc: ny,
		Cfg: DefaultConfig(),
	}
	nyMar8 := Context{
		Now: time.Date(2026, 3, 8, 12, 0, 0, 0, ny),
		Loc: ny,
		Cfg: DefaultConfig(),
	}
	nyMar8Strict := Context{
		Now: nyMar8.Now,
		Loc: ny,
		Cfg: func() Config { c := DefaultConfig(); c.Strict = true; return c }(),
	}
	nyNov1 := Context{
		Now: time.Date(2026, 11, 1, 12, 0, 0, 0, ny),
		Loc: ny,
		Cfg: DefaultConfig(),
	}
	nyNov1Strict := Context{
		Now: nyNov1.Now,
		Loc: ny,
		Cfg: func() Config { c := DefaultConfig(); c.Strict = true; return c }(),
	}
	ref := refCtx(t)
	overflowCfg := func() Config { c := DefaultConfig(); c.MonthOverflow = true; return c }()
	overflowCtx := Context{Now: ref.Now, Loc: ref.Loc, Cfg: overflowCfg}

	yearEndCtx := Context{
		Now: time.Date(2026, 12, 31, 23, 30, 0, 0, mx),
		Loc: mx,
		Cfg: DefaultConfig(),
	}
	dayEndCtx := Context{
		Now: time.Date(2026, 11, 14, 23, 30, 0, 0, mx),
		Loc: mx,
		Cfg: DefaultConfig(),
	}

	sundayCfg := func() Config { c := DefaultConfig(); c.WeekStart = time.Sunday; return c }()
	sundayCtx := Context{Now: ref.Now, Loc: ref.Loc, Cfg: sundayCfg}
	inclusiveCfg := func() Config { c := DefaultConfig(); c.CountInclusive = true; return c }()
	inclusiveCtx := Context{Now: ref.Now, Loc: ref.Loc, Cfg: inclusiveCfg}
	weekOneCtx := func(mode string) Context {
		c := DefaultConfig()
		c.WeekOne = mode
		return Context{Now: ref.Now, Loc: ref.Loc, Cfg: c}
	}

	cases := []edgeCase{
		// --- DST start (America/New_York, 2026-03-08 02:00 -> 03:00) ---
		{
			name:  "dst_start/calendar_add_1d",
			input: "2026-03-07 12:00 + 1d",
			ctx:   nyMar7,
			// Calendar +1d keeps wall clock 12:00 on 2026-03-08 (now EDT -04:00).
			want: "2026-03-08T12:00:00-04:00",
		},
		{
			name:  "dst_start/elapsed_24h",
			input: "2026-03-07 12:00 + 24h",
			ctx:   nyMar7,
			// 12:00-05:00 = 17:00Z; +24h = 2026-03-08T17:00:00Z = 13:00-04:00 across the 23h day.
			want: "2026-03-08T13:00:00-04:00",
		},
		{
			name:  "dst_start/gap_non_strict",
			input: "2026-03-08 02:30",
			ctx:   nyMar8,
			// 02:30 doesn't exist; Go time.Date resolves the DST gap to 01:30-05:00.
			want: "2026-03-08T01:30:00-05:00",
		},
		{
			name:    "dst_start/gap_strict_errors",
			input:   "2026-03-08 02:30",
			ctx:     nyMar8Strict,
			wantErr: true,
		},
		{
			name:  "dst_start/end_of_day",
			input: "end of day",
			ctx:   nyMar8,
			// Last second of 2026-03-08 in NY is 23:59:59 EDT (-04:00).
			want: "2026-03-08T23:59:59-04:00",
		},
		{
			name:  "dst_start/date_diff",
			input: "2026-03-09 - 2026-03-08",
			ctx:   nyMar8,
			// Date difference counts whole calendar days: 1d (elapsed midnight-to-midnight is 23h).
			want: "1d",
		},
		{
			name:  "dst_start/hours_between",
			input: "how many hours between 2026-03-08 and 2026-03-09",
			ctx:   nyMar8,
			// 2026-03-08T00:00-05:00 (05:00Z) to 2026-03-09T00:00-04:00 (04:00Z) is 23 real hours.
			want: "23 hours",
		},

		// --- DST end (America/New_York, 2026-11-01 02:00 -> 01:00) ---
		{
			name:  "dst_end/calendar_add_1d",
			input: "2026-10-31 12:00 + 1d",
			ctx:   nyNov1,
			// Calendar +1d keeps wall clock 12:00 on 2026-11-01 (now EST -05:00).
			want: "2026-11-01T12:00:00-05:00",
		},
		{
			name:  "dst_end/elapsed_24h",
			input: "2026-10-31 12:00 + 24h",
			ctx:   nyNov1,
			// 12:00-04:00 = 16:00Z; +24h = 2026-11-01T16:00:00Z = 11:00-05:00 across the 25h day.
			want: "2026-11-01T11:00:00-05:00",
		},
		{
			name:    "dst_end/overlap_strict_errors",
			input:   "2026-11-01 01:30",
			ctx:     nyNov1Strict,
			wantErr: true,
		},
		{
			name:  "dst_end/hours_between",
			input: "how many hours between 2026-11-01 and 2026-11-02",
			ctx:   nyNov1,
			// 2026-11-01T00:00-04:00 (04:00Z) to 2026-11-02T00:00-05:00 (05:00Z) is 25 real hours.
			want: "25 hours",
		},

		// --- Leap years ---
		{
			name:  "leap/2028-02-29_plus_1y",
			input: "2028-02-29 + 1y",
			ctx:   ref,
			// 2029 is common; day 29 clamps to Feb 28.
			want: "2029-02-28",
		},
		{
			name:  "leap/2028-02-29_minus_1y",
			input: "2028-02-29 - 1y",
			ctx:   ref,
			// 2027 is common; day 29 clamps to Feb 28.
			want: "2027-02-28",
		},
		{
			name:  "leap/2027-02-28_plus_1y",
			input: "2027-02-28 + 1y",
			ctx:   ref,
			// Same day of month (28) is valid in leap year 2028.
			want: "2028-02-28",
		},
		{
			name:  "leap/2028-02-28_plus_1d",
			input: "2028-02-28 + 1d",
			ctx:   ref,
			// 2028 has Feb 29.
			want: "2028-02-29",
		},
		{
			name:  "leap/2026-02-28_plus_1d",
			input: "2026-02-28 + 1d",
			ctx:   ref,
			// 2026 is common; next day is Mar 1.
			want: "2026-03-01",
		},
		{
			name:    "leap/2026-02-29_invalid",
			input:   "2026-02-29",
			ctx:     ref,
			wantErr: true,
		},

		// --- Month ends ---
		{
			name:  "month_end/jan31_plus_1M_clamp",
			input: "2026-01-31 + 1M",
			ctx:   ref,
			// Feb 2026 has 28 days; MonthOverflow:false clamps 31 -> 28.
			want: "2026-02-28",
		},
		{
			name:  "month_end/jan31_plus_2M_clamp",
			input: "2026-01-31 + 2M",
			ctx:   ref,
			// Single +2M reaches Mar 2026, which has 31 days.
			want: "2026-03-31",
		},
		{
			name:  "month_end/mar31_minus_1M_clamp",
			input: "2026-03-31 - 1M",
			ctx:   ref,
			// Feb 2026 clamps 31 -> 28.
			want: "2026-02-28",
		},
		{
			name:  "month_end/aug31_plus_6M_clamp",
			input: "2026-08-31 + 6M",
			ctx:   ref,
			// Aug 2026 + 6M = Feb 2027 (common year, 28 days).
			want: "2027-02-28",
		},
		{
			name:  "month_end/may31_plus_1M_clamp",
			input: "2026-05-31 + 1M",
			ctx:   ref,
			// June has 30 days.
			want: "2026-06-30",
		},
		{
			name:  "month_end/jan31_plus_1M_overflow",
			input: "2026-01-31 + 1M",
			ctx:   overflowCtx,
			// With MonthOverflow:true, Jan 31 + 1M rolls 3 days past Feb 28 to Mar 3.
			want: "2026-03-03",
		},

		// --- Year boundary (now = 2026-12-31T23:30:00-06:00, Thursday) ---
		{
			name:  "year_boundary/tomorrow",
			input: "tomorrow",
			ctx:   yearEndCtx,
			want:  "2027-01-01",
		},
		{
			name:  "year_boundary/next_week",
			input: "next week",
			ctx:   yearEndCtx,
			// Dec 31 is Thu of Mon Dec 28..Sun Jan 3, so next week is Mon Jan 4..Sun Jan 10.
			want: "2027-01-04 → 2027-01-10",
		},
		{
			name:  "year_boundary/next_month",
			input: "next month",
			ctx:   yearEndCtx,
			want:  "2027-01-01 → 2027-01-31",
		},
		{
			name:  "year_boundary/next_year",
			input: "next year",
			ctx:   yearEndCtx,
			want:  "2027-01-01 → 2027-12-31",
		},
		{
			name:  "year_boundary/week_1",
			input: "week 1",
			ctx:   yearEndCtx,
			// 2026-01-01 is Thu (ISO week 1 of 2026 runs Mon 2025-12-29 .. Sun 2026-01-04).
			want: "2025-12-29 → 2026-01-04",
		},
		{
			name:  "year_boundary/week_53",
			input: "week 53",
			ctx:   yearEndCtx,
			// 2026 starts and ends on Thu, giving it 53 ISO weeks; week 53 is Mon Dec 28 .. Sun Jan 3.
			want: "2026-12-28 → 2027-01-03",
		},
		{
			name:  "year_boundary/in_Q1",
			input: "in Q1",
			ctx:   yearEndCtx,
			// "in Q1" targets Q1 of the current year (2026).
			want: "2026-01-01 → 2026-03-31",
		},

		// --- Day boundary across zones (now = 2026-11-14T23:30:00-06:00) ---
		{
			name:  "zone_day_boundary/today",
			input: "today",
			ctx:   dayEndCtx,
			want:  "2026-11-14",
		},
		{
			name:  "zone_day_boundary/now_to_UTC",
			input: "now to UTC",
			ctx:   dayEndCtx,
			// 23:30-06:00 + 6h = 05:30Z on the next UTC day (Nov 15).
			want: "2026-11-15T05:30:00Z",
		},
		{
			name:    "zone_day_boundary/now_to_UTC_as_date",
			input:   "now to UTC",
			ctx:     dayEndCtx,
			fmtOpts: &Options{Type: "date"},
			// -t date prints the instant's calendar date in its own zone (UTC -> Nov 15).
			want: "2026-11-15",
		},

		// --- Settings ---
		{
			name:  "settings/week_start_sunday_last_week",
			input: "last week",
			ctx:   sundayCtx,
			// Now is Sat 2026-11-14; with Sunday start, this week is Nov 8..14 -> last week is Nov 1..7.
			want: "2026-11-01 → 2026-11-07",
		},
		{
			name:  "settings/week_start_sunday_end_of_week",
			input: "end of week",
			ctx:   sundayCtx,
			// Week Sun Nov 8..Sat Nov 14 ends on Sat 2026-11-14 at 23:59:59.
			want: "2026-11-14T23:59:59-06:00",
		},
		{
			name:  "settings/count_inclusive",
			input: "how many days between 2026-11-16 and 2026-11-20",
			ctx:   inclusiveCtx,
			// Nov 16..20 has 4 steps + 1 inclusive = 5 days.
			want: "5 days",
		},
		{
			name:  "settings/week_one_iso_h2",
			input: "2nd week, h2",
			ctx:   weekOneCtx("iso"),
			// H2 starts Wed Jul 1; Mon Jun 29..Sun Jul 5 has 5 days in H2 (>=4), so it's week 1 -> week 2 is Jul 6..12.
			want: "2026-07-06 → 2026-07-12",
		},
		{
			name:  "settings/week_one_first_day_h2",
			input: "2nd week, h2",
			ctx:   weekOneCtx("first-day"),
			// Week containing Jul 1 (Jun 29..Jul 5) is week 1 -> week 2 is Jul 6..12.
			want: "2026-07-06 → 2026-07-12",
		},
		{
			name:  "settings/week_one_full_h2",
			input: "2nd week, h2",
			ctx:   weekOneCtx("full"),
			// First full Mon..Sun week inside H2 is Jul 6..12 -> week 2 is Jul 13..19.
			want: "2026-07-13 → 2026-07-19",
		},
		{
			name:  "settings/week_one_days_h2",
			input: "2nd week, h2",
			ctx:   weekOneCtx("days"),
			// 7-day slices from Jul 1: week 1 is Jul 1..7, week 2 is Jul 8..14.
			want: "2026-07-08 → 2026-07-14",
		},
		{
			name:  "settings/week_one_iso_vs_first_day_may",
			input: "2nd week of may",
			ctx:   weekOneCtx("iso"),
			// May 1, 2026 is Fri (Mon Apr 27..Sun May 3 has only 3 days in May): ISO skips to May 4..10 as week 1 -> week 2 is May 11..17.
			want: "2026-05-11 → 2026-05-17",
		},
		{
			name:  "settings/week_one_first_day_may",
			input: "2nd week of may",
			ctx:   weekOneCtx("first-day"),
			// first-day counts Apr 27..May 3 as week 1 -> week 2 is May 4..10.
			want: "2026-05-04 → 2026-05-10",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipBug != "" {
				t.Skip(tc.skipBug)
			}
			v, err := Eval(tc.input, tc.ctx)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Eval(%q) = %s, want error", tc.input, v)
				}
				return
			}
			if err != nil {
				t.Fatalf("Eval(%q) unexpected error: %v", tc.input, err)
			}
			var got string
			if tc.fmtOpts != nil {
				got, err = Format(v, tc.ctx, *tc.fmtOpts)
				if err != nil {
					t.Fatalf("Format(%q, %+v) error: %v", tc.input, *tc.fmtOpts, err)
				}
			} else {
				got = v.String()
			}
			if got != tc.want {
				t.Fatalf("Eval(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
