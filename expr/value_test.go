package expr

import (
	"testing"
	"time"
	_ "time/tzdata"
)

func TestValueString(t *testing.T) {
	mx, err := time.LoadLocation("America/Mexico_City")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}

	tests := []struct {
		name string
		val  Value
		want string
	}{
		{
			name: "date",
			val: Value{
				Kind: KDate,
				T:    time.Date(2026, 11, 14, 0, 0, 0, 0, mx),
			},
			want: "2026-11-14",
		},
		{
			name: "date-time with offset",
			val: Value{
				Kind: KDateTime,
				T:    time.Date(2026, 11, 14, 14, 30, 0, 0, mx),
			},
			want: "2026-11-14T14:30:00-06:00",
		},
		{
			name: "date-time UTC",
			val: Value{
				Kind: KDateTime,
				T:    time.Date(2026, 11, 14, 20, 30, 0, 0, time.UTC),
			},
			want: "2026-11-14T20:30:00Z",
		},
		{
			name: "time without seconds",
			val: Value{
				Kind: KTime,
				T:    time.Date(2026, 11, 14, 14, 30, 0, 0, mx),
			},
			want: "14:30",
		},
		{
			name: "time with seconds",
			val: Value{
				Kind: KTime,
				T:    time.Date(2026, 11, 14, 14, 30, 5, 0, mx),
			},
			want: "14:30:05",
		},
		{
			name: "duration positive days",
			val: Value{
				Kind: KDur,
				D:    Duration{Days: 40},
			},
			want: "40d",
		},
		{
			name: "duration negative days",
			val: Value{
				Kind: KDur,
				D:    Duration{Days: -40},
			},
			want: "-40d",
		},
		{
			name: "duration folded minutes",
			val: Value{
				Kind: KDur,
				D:    Duration{Nanos: 90 * time.Minute},
			},
			want: "1h 30m",
		},
		{
			name: "duration folded 52 hours",
			val: Value{
				Kind: KDur,
				D:    Duration{Nanos: 52 * time.Hour},
			},
			want: "2d 4h",
		},
		{
			name: "duration folded months into years",
			val: Value{
				Kind: KDur,
				D:    Duration{Months: 14},
			},
			want: "1y 2M",
		},
		{
			name: "duration zero",
			val: Value{
				Kind: KDur,
				D:    Duration{},
			},
			want: "0s",
		},
		{
			name: "duration week stored as days",
			val: Value{
				Kind: KDur,
				D:    Duration{Days: 7},
			},
			want: "7d",
		},
		{
			name: "duration single business day",
			val: Value{
				Kind: KDur,
				D:    Duration{BizDays: 1},
			},
			want: "1 business day",
		},
		{
			name: "duration negative business days",
			val: Value{
				Kind: KDur,
				D:    Duration{BizDays: -3},
			},
			want: "-3 business days",
		},
		{
			name: "duration negative compound",
			val: Value{
				Kind: KDur,
				D:    Duration{Days: -1, Nanos: -2 * time.Hour},
			},
			want: "-1d 2h",
		},
		{
			name: "span of whole days",
			val: Value{
				Kind: KSpan,
				T:    time.Date(2026, 11, 2, 0, 0, 0, 0, mx),
				End:  time.Date(2026, 11, 9, 0, 0, 0, 0, mx),
				Days: true,
			},
			want: "2026-11-02 → 2026-11-08",
		},
		{
			name: "other span",
			val: Value{
				Kind: KSpan,
				T:    time.Date(2026, 11, 14, 14, 30, 0, 0, mx),
				End:  time.Date(2026, 11, 14, 16, 30, 0, 0, mx),
				Days: false,
			},
			want: "2026-11-14T14:30:00-06:00 → 2026-11-14T16:30:00-06:00",
		},
		{
			name: "number whole",
			val: Value{
				Kind: KNumber,
				N:    4,
			},
			want: "4",
		},
		{
			name: "number fractional",
			val: Value{
				Kind: KNumber,
				N:    1.5,
			},
			want: "1.5",
		},
		{
			name: "number singular day",
			val: Value{
				Kind: KNumber,
				N:    1,
				Unit: "day",
			},
			want: "1 day",
		},
		{
			name: "number negative singular day",
			val: Value{
				Kind: KNumber,
				N:    -1,
				Unit: "day",
			},
			want: "-1 day",
		},
		{
			name: "number plural days",
			val: Value{
				Kind: KNumber,
				N:    4,
				Unit: "day",
			},
			want: "4 days",
		},
		{
			name: "number fractional weeks",
			val: Value{
				Kind: KNumber,
				N:    1.2857,
				Unit: "week",
			},
			want: "1.29 weeks",
		},
		{
			name: "number precision 2 uses plural",
			val: Value{
				Kind:      KNumber,
				N:         1,
				Unit:      "week",
				Precision: 3,
			},
			want: "1.00 weeks",
		},
		{
			name: "number singular half",
			val: Value{
				Kind: KNumber,
				N:    1,
				Unit: "half",
			},
			want: "1 half",
		},
		{
			name: "number plural halves",
			val: Value{
				Kind: KNumber,
				N:    2,
				Unit: "half",
			},
			want: "2 halves",
		},
		{
			name: "number singular business day",
			val: Value{
				Kind: KNumber,
				N:    1,
				Unit: "businessday",
			},
			want: "1 business day",
		},
		{
			name: "number plural business days",
			val: Value{
				Kind: KNumber,
				N:    4,
				Unit: "businessday",
			},
			want: "4 business days",
		},
		{
			name: "number zero months",
			val: Value{
				Kind: KNumber,
				N:    0,
				Unit: "month",
			},
			want: "0 months",
		},
		{
			name: "list with labels",
			val: Value{
				Kind: KList,
				Items: []Value{
					{Kind: KDate, T: time.Date(2026, 11, 16, 0, 0, 0, 0, mx), Str: "Revolution Day"},
					{Kind: KDate, T: time.Date(2026, 12, 25, 0, 0, 0, 0, mx)},
				},
			},
			want: "2026-11-16  Revolution Day\n2026-12-25",
		},
		{
			name: "text",
			val: Value{
				Kind: KText,
				Str:  "0 0 * * 1",
			},
			want: "0 0 * * 1",
		},
		{
			name: "recurrence",
			val: Value{
				Kind: KRecurrence,
				T:    time.Date(2026, 11, 14, 14, 30, 0, 0, mx),
				Days: true,
				Str:  "DTSTART;TZID=America/Mexico_City:20261114T000000\nRRULE:FREQ=WEEKLY;BYDAY=MO",
			},
			want: "2026-11-16\n2026-11-23\n2026-11-30\n2026-12-07\n2026-12-14",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.val.String(); got != tc.want {
				t.Errorf("Value.String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatNumberWithUnit(t *testing.T) {
	tests := []struct {
		name  string
		val   Value
		style string
		want  string
	}{
		{name: "unitless long", val: Value{Kind: KNumber, N: 5}, style: "long", want: "5"},
		{name: "unitless short", val: Value{Kind: KNumber, N: 5}, style: "short", want: "5"},
		{name: "unitless none", val: Value{Kind: KNumber, N: 5}, style: "none", want: "5"},
		{name: "ms long", val: Value{Kind: KNumber, N: 1500, Unit: "millisecond"}, style: "long", want: "1500 milliseconds"},
		{name: "ms short", val: Value{Kind: KNumber, N: 1500, Unit: "millisecond"}, style: "short", want: "1500ms"},
		{name: "ms none", val: Value{Kind: KNumber, N: 1500, Unit: "millisecond"}, style: "none", want: "1500"},
		{name: "second short", val: Value{Kind: KNumber, N: 1.5, Unit: "second"}, style: "short", want: "1.5s"},
		{name: "minute short", val: Value{Kind: KNumber, N: 30, Unit: "minute"}, style: "short", want: "30m"},
		{name: "hour short", val: Value{Kind: KNumber, N: -24.5, Unit: "hour"}, style: "short", want: "-24.5h"},
		{name: "day short", val: Value{Kind: KNumber, N: 1, Unit: "day"}, style: "short", want: "1d"},
		{name: "week short", val: Value{Kind: KNumber, N: 1.2857, Unit: "week"}, style: "short", want: "1.29w"},
		{name: "month short", val: Value{Kind: KNumber, N: 12, Unit: "month"}, style: "short", want: "12M"},
		{name: "quarter short", val: Value{Kind: KNumber, N: 4, Unit: "quarter"}, style: "short", want: "4q"},
		{name: "half short singular", val: Value{Kind: KNumber, N: 1, Unit: "half"}, style: "short", want: "1 half"},
		{name: "half short plural", val: Value{Kind: KNumber, N: 2, Unit: "half"}, style: "short", want: "2 halves"},
		{name: "year short", val: Value{Kind: KNumber, N: 1.5, Unit: "year"}, style: "short", want: "1.5y"},
		{name: "businessday short singular", val: Value{Kind: KNumber, N: 1, Unit: "businessday"}, style: "short", want: "1 business day"},
		{name: "businessday short plural", val: Value{Kind: KNumber, N: 4, Unit: "businessday"}, style: "short", want: "4 business days"},
		{name: "workweek long", val: Value{Kind: KNumber, N: 2, Unit: "workweek"}, style: "long", want: "2 work weeks"},
		{name: "workweek short singular", val: Value{Kind: KNumber, N: 1, Unit: "workweek"}, style: "short", want: "1 work week"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatNumberWithUnit(tc.val, tc.style); got != tc.want {
				t.Errorf("formatNumberWithUnit(%+v, %q) = %q, want %q", tc.val, tc.style, got, tc.want)
			}
		})
	}
}
