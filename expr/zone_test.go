package expr

import (
	"errors"
	"sort"
	"strings"
	"testing"
	"time"
	_ "time/tzdata"
)

func TestLookupZone(t *testing.T) {
	nov2026 := time.Date(2026, time.November, 14, 12, 0, 0, 0, time.UTC)
	jul2026 := time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		input       string
		wantLocName string
		checkTime   time.Time
		wantOffset  int // checked when checkOffset is true
		checkOffset bool
		wantErrAll  []string
		wantErrNone []string
	}{
		// Step 3 verify table
		{name: "UTC upper", input: "UTC", wantLocName: "UTC"},
		{name: "utc lower", input: "utc", wantLocName: "UTC"},
		{name: "Z", input: "Z", wantLocName: "UTC"},
		{name: "GMT", input: "GMT", wantLocName: "UTC"},
		{name: "Asia/Tokyo exact", input: "Asia/Tokyo", wantLocName: "Asia/Tokyo"},
		{name: "asia/tokyo lower", input: "asia/tokyo", wantLocName: "Asia/Tokyo"},
		{name: "Tokyo", input: "Tokyo", wantLocName: "Asia/Tokyo"},
		{name: "tokyo", input: "tokyo", wantLocName: "Asia/Tokyo"},
		{name: "Tokio alias", input: "Tokio", wantLocName: "Asia/Tokyo"},
		{name: "New York", input: "New York", wantLocName: "America/New_York"},
		{name: "new_york", input: "new_york", wantLocName: "America/New_York"},
		{name: "NYC", input: "NYC", wantLocName: "America/New_York"},
		{name: "Nueva York", input: "Nueva York", wantLocName: "America/New_York"},
		{name: "Mexico City", input: "Mexico City", wantLocName: "America/Mexico_City"},
		{name: "CDMX", input: "CDMX", wantLocName: "America/Mexico_City"},
		{name: "São Paulo", input: "São Paulo", wantLocName: "America/Sao_Paulo"},
		{name: "sao paulo", input: "sao paulo", wantLocName: "America/Sao_Paulo"},
		{
			name:        "Buenos Aires Nov 2026 offset -3h",
			input:       "Buenos Aires",
			wantLocName: "America/Argentina/Buenos_Aires",
			checkTime:   nov2026,
			wantOffset:  -3 * 3600,
			checkOffset: true,
		},
		{
			name:        "+09:00",
			input:       "+09:00",
			wantLocName: "+09:00",
			checkTime:   nov2026,
			wantOffset:  9 * 3600,
			checkOffset: true,
		},
		{
			name:        "UTC+9",
			input:       "UTC+9",
			wantLocName: "+09:00",
			checkTime:   nov2026,
			wantOffset:  9 * 3600,
			checkOffset: true,
		},
		{
			name:        "-06:00",
			input:       "-06:00",
			wantLocName: "-06:00",
			checkTime:   nov2026,
			wantOffset:  -6 * 3600,
			checkOffset: true,
		},
		{
			name:        "JST",
			input:       "JST",
			wantLocName: "JST",
			checkTime:   nov2026,
			wantOffset:  9 * 3600,
			checkOffset: true,
		},
		{
			name:        "EST",
			input:       "EST",
			wantLocName: "EST",
			checkTime:   nov2026,
			wantOffset:  -5 * 3600,
			checkOffset: true,
		},
		{
			name:       "CST ambiguous",
			input:      "CST",
			wantErrAll: []string{`"CST" is ambiguous`, "America/Chicago", "Asia/Shanghai", "America/Havana"},
		},
		{
			name:       "IST ambiguous",
			input:      "IST",
			wantErrAll: []string{`"IST" is ambiguous`, "Asia/Kolkata", "Europe/Dublin", "Asia/Jerusalem"},
		},
		{
			name:        "Atlantis unknown",
			input:       "Atlantis",
			wantErrAll:  []string{"Atlantis"},
			wantErrNone: []string{"did you mean"},
		},
		{
			name:       "Tok did you mean Tokyo",
			input:      "Tok",
			wantErrAll: []string{"did you mean", "Tokyo"},
		},

		// Additional coverage: full IANA, offsets, aliases, abbreviations, errors
		{name: "america/new_york lower", input: "america/new_york", wantLocName: "America/New_York"},
		{name: "America/New York space", input: "America/New York", wantLocName: "America/New_York"},
		{name: "Port of Spain", input: "Port of Spain", wantLocName: "America/Port_of_Spain"},
		{name: "Bogotá", input: "Bogotá", wantLocName: "America/Bogota"},
		{name: "Zürich", input: "Zürich", wantLocName: "Europe/Zurich"},
		{name: "US/Eastern full IANA", input: "US/Eastern", wantLocName: "US/Eastern"},
		{
			name:        "Etc/GMT+5 is inverted UTC-5",
			input:       "Etc/GMT+5",
			wantLocName: "Etc/GMT+5",
			checkTime:   nov2026,
			wantOffset:  -5 * 3600,
			checkOffset: true,
		},
		{
			name:        "GMT+5 is fixed +5h",
			input:       "GMT+5",
			wantLocName: "+05:00",
			checkTime:   nov2026,
			wantOffset:  5 * 3600,
			checkOffset: true,
		},
		{
			name:        "GMT-5 is fixed -5h",
			input:       "GMT-5",
			wantLocName: "-05:00",
			checkTime:   nov2026,
			wantOffset:  -5 * 3600,
			checkOffset: true,
		},
		{
			name:        "+0900",
			input:       "+0900",
			wantLocName: "+09:00",
			checkTime:   nov2026,
			wantOffset:  9 * 3600,
			checkOffset: true,
		},
		{
			name:        "-6",
			input:       "-6",
			wantLocName: "-06:00",
			checkTime:   nov2026,
			wantOffset:  -6 * 3600,
			checkOffset: true,
		},
		{
			name:        "utc+2",
			input:       "utc+2",
			wantLocName: "+02:00",
			checkTime:   nov2026,
			wantOffset:  2 * 3600,
			checkOffset: true,
		},
		{
			name:        "+05:30",
			input:       "+05:30",
			wantLocName: "+05:30",
			checkTime:   nov2026,
			wantOffset:  5*3600 + 30*60,
			checkOffset: true,
		},
		{
			name:        "UTC+0",
			input:       "UTC+0",
			wantLocName: "+00:00",
			checkTime:   nov2026,
			wantOffset:  0,
			checkOffset: true,
		},
		// All aliases
		{name: "DC", input: "DC", wantLocName: "America/New_York"},
		{name: "SF", input: "SF", wantLocName: "America/Los_Angeles"},
		{name: "LA", input: "LA", wantLocName: "America/Los_Angeles"},
		{name: "México", input: "México", wantLocName: "America/Mexico_City"},
		{name: "Ciudad de México", input: "Ciudad de México", wantLocName: "America/Mexico_City"},
		{name: "Londres", input: "Londres", wantLocName: "Europe/London"},
		{name: "Moscú", input: "Moscú", wantLocName: "Europe/Moscow"},
		{name: "Pekín", input: "Pekín", wantLocName: "Asia/Shanghai"},
		{name: "Beijing", input: "Beijing", wantLocName: "Asia/Shanghai"},
		{name: "Nueva Delhi", input: "Nueva Delhi", wantLocName: "Asia/Kolkata"},
		{name: "Delhi", input: "Delhi", wantLocName: "Asia/Kolkata"},
		{name: "Mumbai", input: "Mumbai", wantLocName: "Asia/Kolkata"},
		// Fixed-offset abbreviations stay fixed in July (no DST shift)
		{
			name:        "EST in July stays -5",
			input:       "EST",
			wantLocName: "EST",
			checkTime:   jul2026,
			wantOffset:  -5 * 3600,
			checkOffset: true,
		},
		{
			name:        "CET in July stays +1",
			input:       "CET",
			wantLocName: "CET",
			checkTime:   jul2026,
			wantOffset:  1 * 3600,
			checkOffset: true,
		},
		{
			name:        "EET in July stays +2",
			input:       "EET",
			wantLocName: "EET",
			checkTime:   jul2026,
			wantOffset:  2 * 3600,
			checkOffset: true,
		},
		{
			name:        "WET in July stays 0",
			input:       "WET",
			wantLocName: "WET",
			checkTime:   jul2026,
			wantOffset:  0,
			checkOffset: true,
		},
		{
			name:        "lowercase jst keeps JST",
			input:       "jst",
			wantLocName: "JST",
			checkTime:   nov2026,
			wantOffset:  9 * 3600,
			checkOffset: true,
		},
		{
			name:        "NZDT",
			input:       "NZDT",
			wantLocName: "NZDT",
			checkTime:   nov2026,
			wantOffset:  13 * 3600,
			checkOffset: true,
		},
		// Remaining ambiguous abbreviations
		{
			name:       "CDT ambiguous",
			input:      "CDT",
			wantErrAll: []string{`"CDT" is ambiguous: America/Chicago or America/Havana`},
		},
		{
			name:       "BST ambiguous",
			input:      "BST",
			wantErrAll: []string{`"BST" is ambiguous: Europe/London or Asia/Dhaka`},
		},
		{
			name:       "AST ambiguous",
			input:      "AST",
			wantErrAll: []string{`"AST" is ambiguous: America/Halifax or Asia/Riyadh`},
		},
		// Duplicate cities: first in sorted order
		{name: "Istanbul picks Asia/Istanbul", input: "Istanbul", wantLocName: "Asia/Istanbul"},
		{name: "Cordoba picks America/Argentina/Cordoba", input: "Cordoba", wantLocName: "America/Argentina/Cordoba"},
		// Non-continent city is excluded from bare city lookup
		{name: "Longyearbyen not matched as city", input: "Longyearbyen", wantErrAll: []string{"Longyearbyen"}},
		{name: "Arctic/Longyearbyen full IANA works", input: "Arctic/Longyearbyen", wantLocName: "Arctic/Longyearbyen"},
		// Errors & suggestion thresholds
		{name: "empty rejected", input: "", wantErrAll: []string{"unknown timezone"}},
		{name: "whitespace rejected", input: "   ", wantErrAll: []string{"unknown timezone"}},
		{name: "Local rejected", input: "Local", wantErrAll: []string{"Local"}},
		{name: "invalid offset +25:00", input: "+25:00", wantErrAll: []string{"+25:00"}},
		{name: "invalid offset +09:60", input: "+09:60", wantErrAll: []string{"+09:60"}},
		{name: "invalid offset +09:00:00", input: "+09:00:00", wantErrAll: []string{"+09:00:00"}},
		{name: "invalid offset +9.5", input: "+9.5", wantErrAll: []string{"+9.5"}},
		{name: "too many suggestions suppressed", input: "a", wantErrAll: []string{"unknown timezone"}, wantErrNone: []string{"did you mean"}},
		{name: "single fuzzy york resolves", input: "york", wantLocName: "America/New_York"},

		// Expanded dictionary: states/regions, countries, non-IANA cities, exonyms
		{name: "california", input: "california", wantLocName: "America/Los_Angeles"},
		{name: "texas", input: "texas", wantLocName: "America/Chicago"},
		{name: "florida", input: "florida", wantLocName: "America/New_York"},
		{name: "arizona", input: "arizona", wantLocName: "America/Phoenix"},
		{name: "jalisco", input: "jalisco", wantLocName: "America/Mexico_City"},
		{name: "nuevo leon", input: "nuevo leon", wantLocName: "America/Monterrey"},
		{name: "baja california", input: "baja california", wantLocName: "America/Tijuana"},
		{name: "quintana roo", input: "quintana roo", wantLocName: "America/Cancun"},
		{name: "ontario", input: "ontario", wantLocName: "America/Toronto"},
		{name: "british columbia", input: "british columbia", wantLocName: "America/Vancouver"},
		{name: "pacific time", input: "pacific time", wantLocName: "America/Los_Angeles"},
		{name: "seattle", input: "seattle", wantLocName: "America/Los_Angeles"},
		{name: "guadalajara", input: "guadalajara", wantLocName: "America/Mexico_City"},
		{name: "barcelona", input: "barcelona", wantLocName: "Europe/Madrid"},
		{name: "spain", input: "spain", wantLocName: "Europe/Madrid"},
		{name: "España", input: "España", wantLocName: "Europe/Madrid"},
		{name: "japan", input: "japan", wantLocName: "Asia/Tokyo"},
		{name: "Japón", input: "Japón", wantLocName: "Asia/Tokyo"},
		{name: "uk", input: "uk", wantLocName: "Europe/London"},
		{name: "brazil", input: "brazil", wantLocName: "America/Sao_Paulo"},
		{name: "new zealand", input: "new zealand", wantLocName: "Pacific/Auckland"},

		// Tiered fuzzy prefix/substring
		{name: "fuzzy word suffix angeles", input: "angeles", wantLocName: "America/Los_Angeles"},
		{name: "fuzzy multi-word prefix los_ang", input: "los_ang", wantLocName: "America/Los_Angeles"},
		{name: "fuzzy multi-word prefix mex_city", input: "mex_city", wantLocName: "America/Mexico_City"},
		{name: "fuzzy prefix toky", input: "toky", wantLocName: "Asia/Tokyo"},

		// Tier 3 Damerau-Levenshtein typo tolerance
		{name: "typo los_angels", input: "los_angels", wantLocName: "America/Los_Angeles"},
		{name: "typo los_angls", input: "los_angls", wantLocName: "America/Los_Angeles"},
		{name: "typo sao_paolo", input: "sao_paolo", wantLocName: "America/Sao_Paulo"},
		{name: "typo tokoy", input: "tokoy", wantLocName: "Asia/Tokyo"},
		{name: "typo amsterdm", input: "amsterdm", wantLocName: "Europe/Amsterdam"},
		{name: "slashed unknown skipped by fuzzy", input: "Mars/Olympus", wantErrAll: []string{"Mars/Olympus"}, wantErrNone: []string{"did you mean"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			loc, err := lookupZone(tc.input)
			if len(tc.wantErrAll) > 0 {
				if err == nil {
					t.Fatalf("lookupZone(%q) = %v, want error", tc.input, loc)
				}
				var exprErr *Error
				if !errors.As(err, &exprErr) {
					t.Fatalf("error type = %T, want *Error", err)
				}
				msg := err.Error()
				for _, sub := range tc.wantErrAll {
					if !strings.Contains(msg, sub) {
						t.Errorf("error %q missing substring %q", msg, sub)
					}
				}
				for _, sub := range tc.wantErrNone {
					if strings.Contains(msg, sub) {
						t.Errorf("error %q unexpectedly contains %q", msg, sub)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("lookupZone(%q) unexpected error: %v", tc.input, err)
			}
			if tc.wantLocName != "" && loc.String() != tc.wantLocName {
				t.Errorf("lookupZone(%q).String() = %q, want %q", tc.input, loc.String(), tc.wantLocName)
			}
			if tc.checkOffset {
				_, off := tc.checkTime.In(loc).Zone()
				if off != tc.wantOffset {
					t.Errorf("lookupZone(%q) offset at %s = %d, want %d", tc.input, tc.checkTime.Format(time.RFC3339), off, tc.wantOffset)
				}
			}
		})
	}
}

func TestZoneEndToEnd(t *testing.T) {
	ctx := refCtx(t)

	tests := []struct {
		name    string
		input   string
		lang    Lang
		want    string
		wantErr string
		wantPos int
	}{
		// Step 5 required 12 rows (with C.3 = A)
		{name: "2026-11-14 14:30 in Tokyo", input: "2026-11-14 14:30 in Tokyo", want: "2026-11-14T14:30:00+09:00"},
		{name: "now in Tokyo", input: "now in Tokyo", want: "2026-11-15T05:30:00+09:00"},
		{name: "2pm @ Tokyo", input: "2pm @ Tokyo", want: "2026-11-14T14:00:00+09:00"},
		{name: "now to New York", input: "now to New York", want: "2026-11-14T15:30:00-05:00"},
		{name: "now to CDMX", input: "now to CDMX", want: "2026-11-14T14:30:00-06:00"},
		{name: "now to +09:00", input: "now to +09:00", want: "2026-11-15T05:30:00+09:00"},
		{name: "now to JST", input: "now to JST", want: "2026-11-15T05:30:00+09:00"},
		{name: "9am @ NYC, to Tokyo", input: "9am @ NYC, to Tokyo", want: "2026-11-14T23:00:00+09:00"},
		{name: "now to Ciudad de México", input: `now to "Ciudad de México"`, want: "2026-11-14T14:30:00-06:00"},
		{name: "now to IST ambiguous", input: "now to IST", wantErr: "ambiguous", wantPos: 7},
		{name: "now to CST is the local zone", input: "now to CST", want: "2026-11-14T14:30:00-06:00"},
		{name: "summer NY DST to UTC", input: "2026-07-01 12:00 @ New York, to UTC", want: "2026-07-01T16:00:00Z"},
		{name: "winter NY std to UTC", input: "2026-12-01 12:00 @ New York, to UTC", want: "2026-12-01T17:00:00Z"},

		// Additional end-to-end coverage
		{name: "now to UTC+2", input: "now to UTC+2", want: "2026-11-14T22:30:00+02:00"},
		{name: "now to GMT-5", input: "now to GMT-5", want: "2026-11-14T15:30:00-05:00"},
		{name: "now to Etc/GMT+5 quoted inverted", input: `now to "Etc/GMT+5"`, want: "2026-11-14T15:30:00-05:00"},
		{name: "2pm @ -06:00", input: "2pm @ -06:00", want: "2026-11-14T14:00:00-06:00"},
		{name: "2026-11-14 14:30, +09:00", input: "2026-11-14 14:30, +09:00", want: "2026-11-14T14:30:00+09:00"},
		{name: "2026-11-14 14:30 in +09:00", input: "2026-11-14 14:30 in +09:00", want: "2026-11-14T14:30:00+09:00"},
		{name: "now | in +09:00", input: "now | in +09:00", want: "2026-11-15T05:30:00+09:00"},
		{name: "now | @ +09:00", input: "now | @ +09:00", want: "2026-11-14T14:30:00+09:00"},
		{name: "EST stays -5 in July", input: "2026-07-01 12:00 @ EST, to UTC", want: "2026-07-01T17:00:00Z"},
		{name: "CET stays +1 in July", input: "2026-07-01 12:00 @ CET, to UTC", want: "2026-07-01T11:00:00Z"},
		{name: "Port of Spain quoted", input: `now to "Port of Spain"`, want: "2026-11-14T16:30:00-04:00"},
		{name: "Sao Paulo accented", input: `now to "São Paulo"`, want: "2026-11-14T17:30:00-03:00"},
		{name: "Buenos Aires", input: "now to Buenos Aires", want: "2026-11-14T17:30:00-03:00"},
		{name: "Spanish ahora a Tokio", input: "ahora a Tokio", lang: ES, want: "2026-11-15T05:30:00+09:00"},
		{name: "Spanish date en Nueva York", input: "2026-11-14 14:30 en Nueva York", lang: ES, want: "2026-11-14T14:30:00-05:00"},
		{name: "unknown Tok suggests Tokyo with pos", input: "now to Tok", wantErr: "did you mean", wantPos: 7},
		{name: "fuzzy angeles", input: "now to angeles", want: "2026-11-14T12:30:00-08:00"},
		{name: "fuzzy typo los_angels", input: "now to los_angels", want: "2026-11-14T12:30:00-08:00"},
		{name: "alias california", input: "now to california", want: "2026-11-14T12:30:00-08:00"},
		{name: "alias guadalajara", input: "now to guadalajara", want: "2026-11-14T14:30:00-06:00"},
		{name: "alias spain", input: "now to spain", want: "2026-11-14T21:30:00+01:00"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := ctx
			if tc.lang != "" {
				c.Cfg.Lang = tc.lang
			}
			got, err := Eval(tc.input, c)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("Eval(%q) = %s, want error containing %q", tc.input, got.String(), tc.wantErr)
				}
				var exprErr *Error
				if !errors.As(err, &exprErr) {
					t.Fatalf("error type = %T, want *Error", err)
				}
				if exprErr.Pos != tc.wantPos {
					t.Errorf("Pos = %d, want %d (msg: %q)", exprErr.Pos, tc.wantPos, exprErr.Msg)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("error = %q, want substring %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Eval(%q) unexpected error: %v", tc.input, err)
			}
			if got.String() != tc.want {
				t.Errorf("Eval(%q) = %q, want %q", tc.input, got.String(), tc.want)
			}
		})
	}
}

func TestCityIndexDeterministicFirstCandidate(t *testing.T) {
	byCity := make(map[string][]string)
	for _, name := range zoneNames {
		if !hasContinentPrefix(name) {
			continue
		}
		slash := strings.LastIndexByte(name, '/')
		key := foldZone(name[slash+1:])
		byCity[key] = append(byCity[key], name)
	}
	var multi []string
	for k, list := range byCity {
		if len(list) > 1 {
			sort.Strings(list)
			if zones.cityByFolded[k] != list[0] {
				t.Errorf("city %q mapped to %q, want first sorted %q", k, zones.cityByFolded[k], list[0])
			}
			multi = append(multi, k)
		}
	}
	if len(multi) == 0 {
		t.Fatal("expected at least one duplicate city across continent prefixes")
	}
}

func TestZoneAliasesValid(t *testing.T) {
	if len(zoneAliases) == 0 {
		t.Fatal("zoneAliases is empty")
	}
	for k, target := range zoneAliases {
		if folded := foldZone(k); folded != k {
			t.Errorf("zoneAliases key %q is not folded (foldZone=%q)", k, folded)
		}
		if _, err := time.LoadLocation(target); err != nil {
			t.Errorf("zoneAliases[%q] = %q failed time.LoadLocation: %v", k, target, err)
		}
	}
}

func TestLookupZoneStrict(t *testing.T) {
	strictCases := []struct {
		name    string
		input   string
		wantSub string
	}{
		{name: "york suggests New York", input: "york", wantSub: "did you mean: New York"},
		{name: "angeles suggests Los Angeles", input: "angeles", wantSub: "did you mean: Los Angeles"},
		{name: "los_angels suggests Los Angeles", input: "los_angels", wantSub: "did you mean: Los Angeles"},
		{name: "tokoy suggests Tokyo", input: "tokoy", wantSub: "did you mean: Tokyo"},
		{name: "sao_paolo suggests Sao Paulo", input: "sao_paolo", wantSub: "did you mean: Sao Paulo"},
	}
	for _, tc := range strictCases {
		t.Run(tc.name, func(t *testing.T) {
			loc, err := LookupZone(tc.input, true)
			if err == nil {
				t.Fatalf("LookupZone(%q, true) = %v, want error", tc.input, loc)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("LookupZone(%q, true) err = %q, want substring %q", tc.input, err.Error(), tc.wantSub)
			}
		})
	}

	// Exact alias and exact city still work under strict=true.
	for _, exact := range []string{"california", "seattle", "Tokio", "CDMX", "Tokyo", "America/Los_Angeles"} {
		if _, err := LookupZone(exact, true); err != nil {
			t.Errorf("LookupZone(%q, true) unexpected error: %v", exact, err)
		}
	}

	// End-to-end strict rejects fuzzy timezone with position.
	ctx := refCtx(t)
	ctx.Cfg.Strict = true
	if _, err := Eval("now to angeles", ctx); err == nil {
		t.Fatal("Eval(\"now to angeles\", strict) succeeded, want error")
	} else {
		var exprErr *Error
		if !errors.As(err, &exprErr) {
			t.Fatalf("error type = %T, want *Error", err)
		}
		if exprErr.Pos != 7 {
			t.Errorf("Pos = %d, want 7 (msg: %q)", exprErr.Pos, exprErr.Msg)
		}
		if !strings.Contains(exprErr.Msg, "did you mean: Los Angeles") {
			t.Errorf("Msg = %q, want substring %q", exprErr.Msg, "did you mean: Los Angeles")
		}
	}
}
