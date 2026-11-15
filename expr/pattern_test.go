package expr

import (
	"strings"
	"testing"
	"time"
	_ "time/tzdata"
)

func TestPattern(t *testing.T) {
	ctx := testRefContext(t)

	tests := []struct {
		name    string
		input   string
		val     *Value
		pattern string
		lang    Lang
		want    string
		wantErr []string
	}{
		// Plan 5 Step 2 verification table
		{name: "dd-MM-yyyy", input: "2026-06-02", pattern: "dd-MM-yyyy", lang: EN, want: "02-06-2026"},
		{name: "d/M/yy", input: "2026-06-02", pattern: "d/M/yy", lang: EN, want: "2/6/26"},
		{name: "d/M/yyyy", input: "2026-06-02", pattern: "d/M/yyyy", lang: EN, want: "2/6/2026"},
		{name: "Spanish full weekday and month", input: "2026-06-02", pattern: "EEEE d 'de' MMMM", lang: ES, want: "martes 2 de junio"},
		{name: "English short weekday and month", input: "2026-06-02", pattern: "EEE, MMM d", lang: EN, want: "Tue, Jun 2"},
		{name: "now HH:mm", input: "now", pattern: "HH:mm", lang: EN, want: "14:30"},
		{name: "now h:mm a", input: "now", pattern: "h:mm a", lang: EN, want: "2:30 PM"},
		{name: "now ISO with quoted T and XXX", input: "now", pattern: "yyyy-MM-dd'T'HH:mm:ssXXX", lang: EN, want: "2026-11-14T14:30:00-06:00"},
		{name: "now to UTC XXX", input: "now to UTC", pattern: "HH:mm XXX", lang: EN, want: "20:30 Z"},
		{name: "span next month dd/MM", input: "next month", pattern: "dd/MM", lang: EN, want: "01/12 → 31/12"},
		{name: "unsupported letter Q", input: "2026-06-02", pattern: "dd-QQ-yyyy", lang: EN, wantErr: []string{"Q", "yyyy"}},

		// Additional token and edge-case coverage
		{name: "date with time tokens uses 00:00:00", input: "2026-06-02", pattern: "HH:mm:ss", lang: EN, want: "00:00:00"},
		{name: "midnight 12h AM English", input: "2026-06-02", pattern: "hh:mm:ss a", lang: EN, want: "12:00:00 AM"},
		{name: "midnight 24h unpadded", input: "2026-06-02", pattern: "H:m:s", lang: EN, want: "0:0:0"},
		{name: "noon 12h PM Spanish", input: "12:05:09", pattern: "h:mm:s a", lang: ES, want: "12:05:9 p. m."},
		{name: "Spanish AM", input: "09:05", pattern: "h:mm a", lang: ES, want: "9:05 a. m."},
		{name: "Spanish short Saturday accent", input: "hoy", pattern: "EEE d MMM", lang: ES, want: "sáb 14 nov"},
		{name: "English input overrides ES config per F.4", input: "today", pattern: "EEE d MMM", lang: ES, want: "Sat 14 Nov"},
		{name: "Spanish short Wednesday accent", input: "2026-11-18", pattern: "EEE EEEE", lang: ES, want: "mié miércoles"},
		{name: "escaped single quote alone and inside", input: "2026-06-02", pattern: "''yy'' 'o''clock'", lang: EN, want: "'26' o'clock"},
		{name: "duration ignores pattern", input: "90m", pattern: "dd-MM-yyyy", lang: EN, want: "1h 30m"},
		{name: "number ignores pattern", input: "42", pattern: "dd-MM-yyyy", lang: EN, want: "42"},
		{name: "non-whole-day span", input: "now .. now + 2h", pattern: "HH:mm", lang: EN, want: "14:30 → 16:30"},
		{
			name:    "positive half-hour offset XXX",
			val:     &Value{Kind: KDateTime, T: time.Date(2026, 6, 2, 10, 0, 0, 0, time.FixedZone("IST", 5*3600+30*60))},
			pattern: "HH:mm XXX",
			lang:    EN,
			want:    "10:00 +05:30",
		},
		{name: "unterminated quote error", input: "2026-06-02", pattern: "yyyy 'de", lang: ES, wantErr: []string{"unterminated"}},
		{name: "unsupported token length XXX vs XX", input: "2026-06-02", pattern: "yyyy XX", lang: EN, wantErr: []string{"X", "XXX"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := ctx
			c.Cfg.Lang = tc.lang
			var v Value
			if tc.val != nil {
				v = *tc.val
			} else {
				var err error
				v, err = Eval(tc.input, c)
				if err != nil {
					t.Fatalf("Eval(%q) unexpected error: %v", tc.input, err)
				}
			}

			got, err := Format(v, c, Options{Pattern: tc.pattern})
			if len(tc.wantErr) > 0 {
				if err == nil {
					t.Fatalf("Format(%q, pattern=%q) = %q, want error", tc.input, tc.pattern, got)
				}
				for _, sub := range tc.wantErr {
					if !strings.Contains(err.Error(), sub) {
						t.Errorf("error %q missing substring %q", err.Error(), sub)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("Format(%q, pattern=%q) unexpected error: %v", tc.input, tc.pattern, err)
			}
			if got != tc.want {
				t.Errorf("Format(%q, pattern=%q) = %q, want %q", tc.input, tc.pattern, got, tc.want)
			}
		})
	}

	t.Run("renderPattern all months and weekdays", func(t *testing.T) {
		for m := time.January; m <= time.December; m++ {
			tm := time.Date(2026, m, 1, 0, 0, 0, 0, ctx.Loc)
			enGot, err := renderPattern(tm, "MMMM MMM", EN)
			if err != nil {
				t.Fatalf("renderPattern EN month %v: %v", m, err)
			}
			if enGot != enNames.monthsFull[m-1]+" "+enNames.monthsShort[m-1] {
				t.Errorf("EN month %v = %q", m, enGot)
			}
			esGot, err := renderPattern(tm, "MMMM MMM", ES)
			if err != nil {
				t.Fatalf("renderPattern ES month %v: %v", m, err)
			}
			if esGot != esNames.monthsFull[m-1]+" "+esNames.monthsShort[m-1] {
				t.Errorf("ES month %v = %q", m, esGot)
			}
		}

		wantESWeekdays := []string{
			"domingo dom", "lunes lun", "martes mar", "miércoles mié",
			"jueves jue", "viernes vie", "sábado sáb",
		}
		for d := 0; d < 7; d++ {
			// 2026-11-15 is Sunday (Weekday 0)
			tm := time.Date(2026, time.November, 15+d, 0, 0, 0, 0, ctx.Loc)
			esGot, err := renderPattern(tm, "EEEE EEE", ES)
			if err != nil {
				t.Fatalf("renderPattern ES weekday %d: %v", d, err)
			}
			if esGot != wantESWeekdays[d] {
				t.Errorf("ES weekday %d = %q, want %q", d, esGot, wantESWeekdays[d])
			}
		}
	})
}
