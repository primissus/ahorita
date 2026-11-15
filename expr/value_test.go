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
