package expr

import "testing"

func TestPrint(t *testing.T) {
	tests := []struct {
		name string
		expr Expr
		want string
	}{
		{
			name: "literals and unix",
			expr: Expr{Stages: []Node{
				{Op: "date", Text: "2026-06-02"},
				{Op: "datetime", Text: "2026-06-02T14:30:00-06:00"},
				{Op: "clock", Text: "14:30"},
				{Op: "slash", Text: "02/06/2026"},
				{Op: "unix", Text: "1700000000"},
				{Op: "num", Text: "3"},
				{Op: "year", Text: "2026"},
			}},
			want: "2026-06-02 | 2026-06-02T14:30:00-06:00 | 14:30 | 02/06/2026 | @1700000000 | 3 | 2026",
		},
		{
			name: "arithmetic and durations",
			expr: Expr{Stages: []Node{
				{Op: "+", Args: []Node{
					{Op: "now"},
					{Op: "dur", Args: []Node{
						{Op: "hour", Text: "1"},
						{Op: "minute", Text: "30"},
					}},
				}},
				{Op: "*", Args: []Node{
					{Op: "dur", Args: []Node{{Op: "month", Text: "3"}}},
					{Op: "num", Text: "2"},
				}},
				{Op: "+", Args: []Node{
					{Op: "today"},
					{Op: "dur", Args: []Node{{Op: "businessday", Text: "3"}}},
				}},
				{Op: "dur", Args: []Node{{Op: "businessday", Text: "1"}}},
				{Op: "dur", Args: []Node{{Op: "half", Text: "2"}}},
			}},
			want: "now + 1h 30m | 3M * 2 | today + 3 business days | 1 business day | 2 halves",
		},
		{
			name: "ago in rel month period week",
			expr: Expr{Stages: []Node{
				{Op: "ago", Args: []Node{{Op: "dur", Args: []Node{{Op: "day", Text: "2"}}}}},
				{Op: "in", Args: []Node{{Op: "dur", Args: []Node{{Op: "month", Text: "3"}}}}},
				{Op: "rel", Text: "last", Args: []Node{{Op: "unit", Text: "week"}}},
				{Op: "rel", Text: "next", Args: []Node{{Op: "weekday", Text: "monday"}}},
				{Op: "rel", Text: "next", Args: []Node{{Op: "month", Text: "august", Args: []Node{{Op: "num", Text: "20"}}}}},
				{Op: "month", Text: "august", Args: []Node{{Op: "num", Text: "20"}, {Op: "year", Text: "2027"}}},
				{Op: "period", Text: "q3", Args: []Node{{Op: "year", Text: "2027"}}},
				{Op: "week", Text: "2"},
			}},
			want: "2d ago | in 3M | last week | next monday | next august 20 | august 20 2027 | q3 2027 | week 2",
		},
		{
			name: "bound of closest during starting",
			expr: Expr{Stages: []Node{
				{Op: "bound", Text: "end", Args: []Node{{Op: "unit", Text: "workweek"}}},
				{Op: "bound", Text: "start", Args: []Node{{Op: "unit", Text: "businessday"}}},
				{Op: "of", Args: []Node{{Op: "week", Text: "2"}, {Op: "month", Text: "march"}}},
				{Op: "closest", Args: []Node{{Op: "weekday", Text: "monday"}}},
				{Op: "closest", Args: []Node{{Op: "weekday", Text: "monday"}, {Op: "date", Text: "2026-12-01"}}},
				{Op: "during", Args: []Node{{Op: "dur", Args: []Node{{Op: "day", Text: "2"}}}}},
				{Op: "starting", Args: []Node{{Op: "rel", Text: "next", Args: []Node{{Op: "weekday", Text: "monday"}}}}},
			}},
			want: "end of work week | start of business day | week 2 of march | closest monday | closest monday to 2026-12-01 | during 2d | starting next monday",
		},
		{
			name: "span count inclusive to zone interpret inzone at",
			expr: Expr{Stages: []Node{
				{Op: "span", Args: []Node{{Op: "today"}, {Op: "tomorrow"}}},
				{Op: "inclusive", Args: []Node{{Op: "count", Args: []Node{
					{Op: "unit", Text: "day"},
					{Op: "date", Text: "2026-11-16"},
					{Op: "date", Text: "2026-11-20"},
				}}}},
				{Op: "count", Args: []Node{
					{Op: "weekday", Text: "monday"},
					{Op: "today"},
					{Op: "date", Text: "2026-12-31"},
				}},
				{Op: "count", Args: []Node{
					{Op: "unit", Text: "half"},
					{Op: "today"},
					{Op: "tomorrow"},
				}},
				{Op: "to", Args: []Node{{Op: "unit", Text: "hour"}}},
				{Op: "to", Args: []Node{{Op: "unit", Text: "businessday"}}},
				{Op: "to", Args: []Node{{Op: "zone", Text: "New York"}}},
				{Op: "to", Args: []Node{{Op: "key", Text: "cron"}}},
				{Op: "interpret", Args: []Node{{Op: "clock", Text: "14:00"}, {Op: "zone", Text: "Tokyo"}}},
				{Op: "interpret", Args: []Node{{Op: "zone", Text: "Tokyo"}}},
				{Op: "inzone", Args: []Node{
					{Op: "at", Args: []Node{{Op: "yesterday"}, {Op: "clock", Text: "14:30"}}},
					{Op: "zone", Text: "Port of Spain"},
				}},
				{Op: "inzone", Args: []Node{{Op: "zone", Text: "UTC"}}},
				{Op: "orsame", Args: []Node{
					{Op: "rel", Text: "next", Args: []Node{{Op: "unit", Text: "businessday"}}},
					{Op: "date", Text: "2026-11-14"},
				}},
				{Op: "orsame", Args: []Node{
					{Op: "rel", Text: "last", Args: []Node{{Op: "unit", Text: "businessday"}}},
				}},
				{Op: "holidays"},
				{Op: "holidays", Text: "MX", Args: []Node{{Op: "year", Text: "2026"}}},
			}},
			want: `today .. tomorrow | how many days between 2026-11-16 and 2026-11-20 inclusive | how many mondays between today and 2026-12-31 | how many halves between today and tomorrow | to hours | to business days | to "New York" | to cron | 14:00 @ Tokyo | @ Tokyo | yesterday 14:30 in "Port of Spain" | in UTC | next business day on or after 2026-11-14 | last business day on or before | holidays | holidays MX 2026`,
		},
		{
			name: "parentheses grouping",
			expr: Expr{Stages: []Node{
				{Op: "*", Args: []Node{
					{Op: "num", Text: "2"},
					{Op: "+", Args: []Node{
						{Op: "dur", Args: []Node{{Op: "hour", Text: "1"}}},
						{Op: "dur", Args: []Node{{Op: "minute", Text: "30"}}},
					}},
				}},
				{Op: "*", Args: []Node{
					{Op: "-", Args: []Node{
						{Op: "dur", Args: []Node{{Op: "hour", Text: "2"}}},
						{Op: "dur", Args: []Node{{Op: "minute", Text: "30"}}},
					}},
					{Op: "num", Text: "3"},
				}},
				{Op: "-", Args: []Node{
					{Op: "today"},
					{Op: "+", Args: []Node{
						{Op: "dur", Args: []Node{{Op: "day", Text: "1"}}},
						{Op: "dur", Args: []Node{{Op: "hour", Text: "1"}}},
					}},
				}},
				{Op: "-", Args: []Node{
					{Op: "+", Args: []Node{
						{Op: "today"},
						{Op: "dur", Args: []Node{{Op: "day", Text: "1"}}},
					}},
					{Op: "dur", Args: []Node{{Op: "hour", Text: "1"}}},
				}},
				{Op: "at", Args: []Node{
					{Op: "-", Args: []Node{
						{Op: "today"},
						{Op: "dur", Args: []Node{{Op: "day", Text: "1"}}},
					}},
					{Op: "clock", Text: "14:00"},
				}},
				{Op: "interpret", Args: []Node{
					{Op: "+", Args: []Node{
						{Op: "today"},
						{Op: "dur", Args: []Node{{Op: "day", Text: "1"}}},
					}},
					{Op: "zone", Text: "Tokyo"},
				}},
				{Op: "inzone", Args: []Node{
					{Op: "-", Args: []Node{
						{Op: "now"},
						{Op: "dur", Args: []Node{{Op: "hour", Text: "1"}}},
					}},
					{Op: "zone", Text: "Tokyo"},
				}},
				{Op: "inzone", Args: []Node{
					{Op: "*", Args: []Node{
						{Op: "dur", Args: []Node{{Op: "hour", Text: "2"}}},
						{Op: "num", Text: "3"},
					}},
					{Op: "zone", Text: "Tokyo"},
				}},
			}},
			want: `2 * (1h + 30m) | (2h - 30m) * 3 | today - (1d + 1h) | today + 1d - 1h | (today - 1d) 14:00 | (today + 1d) @ Tokyo | (now - 1h) in Tokyo | (2h * 3) in Tokyo`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.expr.String(); got != tc.want {
				t.Errorf("got  %q\nwant %q", got, tc.want)
			}
		})
	}

	errVal := &Error{Pos: 3, Msg: "boom", Expected: []string{"a value"}}
	if errVal.Error() != "boom" {
		t.Errorf("Error() = %q, want %q", errVal.Error(), "boom")
	}
}
