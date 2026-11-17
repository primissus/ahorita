package expr

import (
	"strings"
	"testing"
	"time"
	_ "time/tzdata"
)

func TestFormatType(t *testing.T) {
	ctx := testRefContext(t)

	tests := []struct {
		name    string
		input   string
		val     *Value
		typ     string
		want    string
		wantErr []string
	}{
		// Plan 5 Step 1 verification table
		{name: "now date", input: "now", typ: "date", want: "2026-11-14"},
		{name: "now datetime", input: "now", typ: "datetime", want: "2026-11-14T14:30:00-06:00"},
		{name: "now time", input: "now", typ: "time", want: "14:30"},
		{name: "now range", input: "now", typ: "range", want: "0s"},
		{name: "now + 52h range", input: "now + 52h", typ: "range", want: "2d 4h"},
		{name: "yesterday datetime", input: "yesterday", typ: "datetime", want: "2026-11-13T00:00:00-06:00"},
		{name: "2d ago date", input: "2d ago", typ: "date", want: "2026-11-12"},
		{name: "next month date", input: "next month", typ: "date", want: "2026-12-01"},
		{name: "next month range", input: "next month", typ: "range", want: "2026-12-01 → 2026-12-31"},
		{name: "next month empty", input: "next month", typ: "", want: "2026-12-01 → 2026-12-31"},
		{name: "date diff empty", input: "2026-12-24 - today", typ: "", want: "40d"},
		{name: "date diff range", input: "2026-12-24 - today", typ: "range", want: "2026-11-14 → 2026-12-24"},
		{name: "2h date error", input: "2h", typ: "date", wantErr: []string{"date", "duration"}},
		{name: "today unknown type error", input: "today", typ: "week", wantErr: []string{"date", "datetime", "time", "range", "relative"}},

		// Additional coverage across kinds and edge cases
		{name: "datetime in Tokyo keeps own zone date", input: "now to Asia/Tokyo", typ: "date", want: "2026-11-15"},
		{name: "today time", input: "today", typ: "time", want: "00:00"},
		{name: "today range", input: "today", typ: "range", want: "0s"},
		{name: "tomorrow range", input: "tomorrow", typ: "range", want: "1d"},
		{name: "yesterday range", input: "yesterday", typ: "range", want: "-1d"},
		{name: "clock date", input: "2:30pm", typ: "date", want: "2026-11-14"},
		{name: "clock datetime", input: "2:30pm", typ: "datetime", want: "2026-11-14T14:30:00-06:00"},
		{name: "clock time", input: "14:30:05", typ: "time", want: "14:30:05"},
		{name: "clock range", input: "16:00", typ: "range", want: "1h 30m"},
		{name: "span datetime", input: "next month", typ: "datetime", want: "2026-12-01T00:00:00-06:00"},
		{name: "span time", input: "next month", typ: "time", want: "00:00"},
		{name: "reversed date diff range", input: "today - 2026-12-24", typ: "range", want: "2026-11-14 → 2026-12-24"},
		{name: "datetime diff range", input: "now + 2h - now", typ: "range", want: "2026-11-14T14:30:00-06:00 → 2026-11-14T16:30:00-06:00"},
		{name: "bare duration range", input: "90m", typ: "range", want: "1h 30m"},
		{name: "date diff date error", input: "2026-12-24 - today", typ: "date", wantErr: []string{"date", "duration"}},
		{name: "number range error", input: "4", typ: "range", wantErr: []string{"range", "number"}},
		{
			name:    "list date error",
			val:     &Value{Kind: KList, Items: []Value{{Kind: KDate, T: time.Date(2026, 11, 16, 0, 0, 0, 0, ctx.Loc)}}},
			typ:     "date",
			wantErr: []string{"date", "list"},
		},
		{
			name:    "text datetime error",
			val:     &Value{Kind: KText, Str: "0 0 * * 1"},
			typ:     "datetime",
			wantErr: []string{"datetime", "text"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var v Value
			if tc.val != nil {
				v = *tc.val
			} else {
				var err error
				v, err = Eval(tc.input, ctx)
				if err != nil {
					t.Fatalf("Eval(%q) unexpected error: %v", tc.input, err)
				}
			}

			got, err := Format(v, ctx, Options{Type: tc.typ})
			if len(tc.wantErr) > 0 {
				if err == nil {
					t.Fatalf("Format(%q, type=%q) = %q, want error", tc.input, tc.typ, got)
				}
				for _, sub := range tc.wantErr {
					if !strings.Contains(err.Error(), sub) {
						t.Errorf("error %q missing substring %q", err.Error(), sub)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("Format(%q, type=%q) unexpected error: %v", tc.input, tc.typ, err)
			}
			if got != tc.want {
				t.Errorf("Format(%q, type=%q) = %q, want %q", tc.input, tc.typ, got, tc.want)
			}
		})
	}

	t.Run("empty options match Value.String byte for byte", func(t *testing.T) {
		vals := []Value{
			{Kind: KDate, T: time.Date(2026, 11, 14, 0, 0, 0, 0, ctx.Loc)},
			{Kind: KDateTime, T: time.Date(2026, 11, 14, 14, 30, 0, 0, ctx.Loc)},
			{Kind: KTime, T: time.Date(2026, 11, 14, 14, 30, 5, 0, ctx.Loc)},
			{Kind: KDur, D: Duration{Days: 2, Nanos: 4 * time.Hour}},
			{Kind: KSpan, T: time.Date(2026, 12, 1, 0, 0, 0, 0, ctx.Loc), End: time.Date(2027, 1, 1, 0, 0, 0, 0, ctx.Loc), Days: true},
			{Kind: KNumber, N: 4.5},
			{Kind: KText, Str: "0 0 * * 1"},
			{Kind: KRecurrence, Str: "FREQ=WEEKLY;BYDAY=MO"},
			{Kind: KList, Items: []Value{{Kind: KDate, T: time.Date(2026, 11, 16, 0, 0, 0, 0, ctx.Loc), Str: "Revolution Day"}}},
		}
		for _, v := range vals {
			got, err := Format(v, ctx, Options{})
			if err != nil {
				t.Fatalf("Format(%v, empty) error: %v", v.Kind, err)
			}
			if want := v.String(); got != want {
				t.Errorf("Format(%v, empty) = %q, want %q", v.Kind, got, want)
			}
		}
	})
}

func TestPreset(t *testing.T) {
	ctx := testRefContext(t)

	tests := []struct {
		name    string
		input   string
		val     *Value
		cfgLang Lang
		typ     string
		preset  string
		want    string
		wantErr []string
	}{
		// Plan 5 Step 3 verification table
		{name: "now human en", input: "now", cfgLang: EN, preset: "human", want: "Sat, Nov 14, 2026, 2:30 PM"},
		{name: "today human en", input: "today", cfgLang: EN, preset: "human", want: "Sat, Nov 14, 2026"},
		{name: "hoy human from input", input: "hoy", cfgLang: EN, preset: "human", want: "sáb 14 nov 2026"},
		{name: "now short en", input: "now", cfgLang: EN, preset: "short", want: "11-14-2026 14:30"},
		{name: "hoy short es", input: "hoy", cfgLang: EN, preset: "short", want: "14-11-2026"},
		{
			name:    "today value without Lang falls back to cfg.Lang es",
			val:     &Value{Kind: KDate, T: time.Date(2026, 11, 14, 0, 0, 0, 0, ctx.Loc)},
			cfgLang: ES,
			preset:  "short",
			want:    "14-11-2026",
		},
		{name: "now unix en", input: "now", cfgLang: EN, preset: "unix", want: "1794688200"},
		{name: "today rfc3339 en", input: "today", cfgLang: EN, preset: "rfc3339", want: "2026-11-14T00:00:00-06:00"},
		{name: "today rfc en", input: "today", cfgLang: EN, preset: "rfc", want: "2026-11-14T00:00:00-06:00"},
		{name: "now iso en", input: "now", cfgLang: EN, preset: "iso", want: "2026-11-14T14:30:00-06:00"},

		// Additional preset coverage
		{name: "ahora human es", input: "ahora", cfgLang: EN, preset: "human", want: "sáb 14 nov 2026, 14:30"},
		{name: "ahora short es", input: "ahora", cfgLang: EN, preset: "short", want: "14-11-2026 14:30"},
		{name: "time human en", input: "14:30", cfgLang: EN, preset: "human", want: "2:30 PM"},
		{name: "time human es", input: "14:30", cfgLang: ES, preset: "human", want: "14:30"},
		{name: "time short en", input: "14:30", cfgLang: EN, preset: "short", want: "14:30"},
		{name: "time rfc3339 error", input: "14:30", cfgLang: EN, preset: "rfc3339", wantErr: []string{"rfc3339", "time"}},
		{name: "time unix error", input: "14:30", cfgLang: EN, preset: "unix", wantErr: []string{"unix", "time"}},
		{name: "time converted to datetime then unix", input: "14:30", cfgLang: EN, typ: "datetime", preset: "unix", want: "1794688200"},
		{name: "today unix midnight", input: "today", cfgLang: EN, preset: "unix", want: "1794636000"},
		{name: "span human en", input: "next month", cfgLang: EN, preset: "human", want: "Tue, Dec 1, 2026 → Thu, Dec 31, 2026"},
		{name: "span short es", input: "2026-12-01 .. 2026-12-31", cfgLang: ES, preset: "short", want: "01-12-2026 → 31-12-2026"},
		{name: "span non-whole-day short en", input: "now .. now + 2h", cfgLang: EN, preset: "short", want: "11-14-2026 14:30 → 11-14-2026 16:30"},
		{name: "duration ignores human preset", input: "90m", cfgLang: EN, preset: "human", want: "1h 30m"},
		{name: "duration errors on unix preset", input: "90m", cfgLang: EN, preset: "unix", wantErr: []string{"unix", "duration"}},
		{name: "number iso preset", input: "42", cfgLang: EN, preset: "iso", want: "42"},
		{name: "type date with human preset", input: "now", cfgLang: EN, typ: "date", preset: "human", want: "Sat, Nov 14, 2026"},
		{name: "type date with custom pattern", input: "now", cfgLang: EN, typ: "date", preset: "dd-MM-yyyy", want: "14-11-2026"},
		{name: "date diff type range with pattern", input: "2026-12-24 - today", cfgLang: EN, typ: "range", preset: "dd/MM", want: "14/11 → 24/12"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := ctx
			if tc.cfgLang != "" {
				c.Cfg.Lang = tc.cfgLang
			}
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

			got, err := Format(v, c, Options{Type: tc.typ, Pattern: tc.preset})
			if len(tc.wantErr) > 0 {
				if err == nil {
					t.Fatalf("Format(%q, preset=%q) = %q, want error", tc.input, tc.preset, got)
				}
				for _, sub := range tc.wantErr {
					if !strings.Contains(err.Error(), sub) {
						t.Errorf("error %q missing substring %q", err.Error(), sub)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("Format(%q, preset=%q) unexpected error: %v", tc.input, tc.preset, err)
			}
			if got != tc.want {
				t.Errorf("Format(%q, preset=%q) = %q, want %q", tc.input, tc.preset, got, tc.want)
			}
		})
	}
}

func TestRelative(t *testing.T) {
	ctx := testRefContext(t)

	tests := []struct {
		name    string
		input   string
		val     *Value
		cfgLang Lang
		want    string
		wantErr []string
	}{
		// Plan 5 Step 4 verification table
		{name: "now en", input: "now", cfgLang: EN, want: "now"},
		{name: "today + 2d en", input: "today + 2d", cfgLang: EN, want: "in 2 days"},
		{name: "tomorrow en", input: "tomorrow", cfgLang: EN, want: "tomorrow"},
		{name: "today - 21d en", input: "today - 21d", cfgLang: EN, want: "3 weeks ago"},
		{name: "today + 45d en", input: "today + 45d", cfgLang: EN, want: "in 1 month"},
		{name: "now + 90m en", input: "now + 90m", cfgLang: EN, want: "in 1 hour"},
		{name: "now - 5m en", input: "now - 5m", cfgLang: EN, want: "5 minutes ago"},
		{name: "next month en", input: "next month", cfgLang: EN, want: "in 2 weeks"},
		{
			name:    "2d ago with Lang ES",
			val:     &Value{Kind: KDateTime, T: ctx.Now.Add(-48 * time.Hour), Lang: ES},
			cfgLang: EN,
			want:    "hace 2 días",
		},
		{name: "hace 2 dias evaluated es", input: "hace 2 días", cfgLang: EN, want: "hace 2 días"},
		{
			name:    "today + 1d with Lang ES",
			val:     &Value{Kind: KDate, T: time.Date(2026, 11, 15, 0, 0, 0, 0, ctx.Loc), Lang: ES},
			cfgLang: EN,
			want:    "mañana",
		},
		{name: "hoy + 1 dia evaluated es", input: "hoy + 1 día", cfgLang: EN, want: "en 1 día"},
		{name: "2h duration error", input: "2h", cfgLang: EN, wantErr: []string{"relative", "duration"}},

		// Additional coverage for units, boundaries, singulars/plurals, kinds
		{name: "today en", input: "today", cfgLang: EN, want: "today"},
		{name: "hoy es", input: "hoy", cfgLang: EN, want: "hoy"},
		{name: "yesterday en", input: "yesterday", cfgLang: EN, want: "yesterday"},
		{name: "ayer es", input: "ayer", cfgLang: EN, want: "ayer"},
		{name: "ahora es", input: "ahora", cfgLang: EN, want: "ahora"},
		{name: "now + 30s en", input: "now + 30s", cfgLang: EN, want: "now"},
		{name: "ahora - 30s es", input: "ahora - 30s", cfgLang: EN, want: "ahora"},
		{name: "now + 1m en", input: "now + 1m", cfgLang: EN, want: "in 1 minute"},
		{name: "ahora - 1m es", input: "ahora - 1m", cfgLang: EN, want: "hace 1 minuto"},
		{name: "ahora + 2h es", input: "ahora + 2h", cfgLang: EN, want: "en 2 horas"},
		{name: "now - 3h en", input: "now - 3h", cfgLang: EN, want: "3 hours ago"},
		{name: "now + 25h en uses 1 day", input: "now + 25h", cfgLang: EN, want: "in 1 day"},
		{name: "ahora - 25h es uses 1 dia", input: "ahora - 25h", cfgLang: EN, want: "hace 1 día"},
		{name: "today + 7d en", input: "today + 7d", cfgLang: EN, want: "in 1 week"},
		{name: "hoy + 14d es", input: "hoy + 14d", cfgLang: EN, want: "en 2 semanas"},
		{name: "hoy + 60d es", input: "hoy + 60d", cfgLang: EN, want: "en 2 meses"},
		{name: "today + 365d en", input: "today + 365d", cfgLang: EN, want: "in 1 year"},
		{name: "hoy - 730d es", input: "hoy - 730d", cfgLang: EN, want: "hace 2 años"},
		{name: "time on today now", input: "14:30", cfgLang: EN, want: "now"},
		{name: "time later today", input: "16:00", cfgLang: EN, want: "in 1 hour"},
		{name: "non-whole-day span uses start instant", input: "now + 2h .. now + 5h", cfgLang: EN, want: "in 2 hours"},
		{name: "whole-day span starting tomorrow", input: "2026-11-15 .. 2026-11-20", cfgLang: EN, want: "tomorrow"},
		{name: "number error", input: "4", cfgLang: EN, wantErr: []string{"relative", "number"}},
		{name: "date diff error", input: "2026-12-24 - today", cfgLang: EN, wantErr: []string{"relative", "duration"}},
		{
			name:    "list error",
			val:     &Value{Kind: KList},
			cfgLang: EN,
			wantErr: []string{"relative", "list"},
		},
		{
			name:    "text error",
			val:     &Value{Kind: KText, Str: "hello"},
			cfgLang: EN,
			wantErr: []string{"relative", "text"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := ctx
			c.Cfg.Lang = tc.cfgLang
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

			got, err := Format(v, c, Options{Type: "relative"})
			if len(tc.wantErr) > 0 {
				if err == nil {
					t.Fatalf("Format(%q, relative) = %q, want error", tc.input, got)
				}
				for _, sub := range tc.wantErr {
					if !strings.Contains(err.Error(), sub) {
						t.Errorf("error %q missing substring %q", err.Error(), sub)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("Format(%q, relative) unexpected error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("Format(%q, relative) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestFormatShowTimezone(t *testing.T) {
	ctx := testRefContext(t)
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatalf("load Asia/Tokyo: %v", err)
	}

	tests := []struct {
		name  string
		input string
		val   *Value
		opts  Options
		want  string
	}{
		{
			name:  "iso datetime",
			input: "now",
			opts:  Options{ShowTimezone: true},
			want:  "2026-11-14T14:30:00-06:00 America/Mexico_City",
		},
		{
			name:  "iso date",
			input: "today",
			opts:  Options{ShowTimezone: true},
			want:  "2026-11-14 America/Mexico_City",
		},
		{
			name:  "iso time",
			input: "now",
			opts:  Options{Type: "time", ShowTimezone: true},
			want:  "14:30 America/Mexico_City",
		},
		{
			name:  "human datetime",
			input: "now",
			opts:  Options{Pattern: "human", ShowTimezone: true},
			want:  "Sat, Nov 14, 2026, 2:30 PM America/Mexico_City",
		},
		{
			name:  "short datetime",
			input: "now",
			opts:  Options{Pattern: "short", ShowTimezone: true},
			want:  "11-14-2026 14:30 America/Mexico_City",
		},
		{
			name:  "rfc3339 datetime",
			input: "now",
			opts:  Options{Pattern: "rfc3339", ShowTimezone: true},
			want:  "2026-11-14T14:30:00-06:00 America/Mexico_City",
		},
		{
			name:  "custom pattern",
			input: "now",
			opts:  Options{Pattern: "yyyy/MM/dd HH:mm", ShowTimezone: true},
			want:  "2026/11/14 14:30 America/Mexico_City",
		},
		{
			name:  "converted timezone",
			input: "now to Asia/Tokyo",
			opts:  Options{Pattern: "human", ShowTimezone: true},
			want:  "Sun, Nov 15, 2026, 5:30 AM Asia/Tokyo",
		},
		{
			name:  "fixed offset zone",
			input: "now to +05:30",
			opts:  Options{ShowTimezone: true},
			want:  "2026-11-15T02:00:00+05:30 +05:30",
		},
		{
			name: "span same zone",
			val:  &Value{Kind: KSpan, T: time.Date(2026, 11, 14, 9, 0, 0, 0, ctx.Loc), End: time.Date(2026, 11, 14, 17, 0, 0, 0, ctx.Loc)},
			opts: Options{ShowTimezone: true},
			want: "2026-11-14T09:00:00-06:00 → 2026-11-14T17:00:00-06:00 America/Mexico_City",
		},
		{
			name: "span different zones",
			val:  &Value{Kind: KSpan, T: time.Date(2026, 11, 14, 9, 0, 0, 0, ctx.Loc), End: time.Date(2026, 11, 15, 0, 0, 0, 0, tokyo)},
			opts: Options{ShowTimezone: true},
			want: "2026-11-14T09:00:00-06:00 America/Mexico_City → 2026-11-15T00:00:00+09:00 Asia/Tokyo",
		},
		{
			name: "list with labels",
			val:  &Value{Kind: KList, Items: []Value{{Kind: KDate, T: time.Date(2026, 11, 16, 0, 0, 0, 0, ctx.Loc), Str: "Revolution Day"}}},
			opts: Options{ShowTimezone: true},
			want: "2026-11-16 America/Mexico_City  Revolution Day",
		},
		{
			name:  "unix unchanged",
			input: "now",
			opts:  Options{Pattern: "unix", ShowTimezone: true},
			want:  "1794688200",
		},
		{
			name:  "relative unchanged",
			input: "now + 2h",
			opts:  Options{Type: "relative", ShowTimezone: true},
			want:  "in 2 hours",
		},
		{
			name:  "duration unchanged",
			input: "2h 30m",
			opts:  Options{ShowTimezone: true},
			want:  "2h 30m",
		},
		{
			name: "number unchanged",
			val:  &Value{Kind: KNumber, N: 2},
			opts: Options{ShowTimezone: true},
			want: "2",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var v Value
			if tc.val != nil {
				v = *tc.val
			} else {
				var err error
				v, err = Eval(tc.input, ctx)
				if err != nil {
					t.Fatalf("Eval(%q) unexpected error: %v", tc.input, err)
				}
			}
			got, err := Format(v, ctx, tc.opts)
			if err != nil {
				t.Fatalf("Format(%q) unexpected error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("Format(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
