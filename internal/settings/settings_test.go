package settings

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/primissus/ahorita/expr"
)

func TestApplyAndGet(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		val     string
		wantGet string
		wantErr string
	}{
		{name: "type date", key: "type", val: "DATE", wantGet: "date"},
		{name: "type bad", key: "type", val: "week", wantErr: "invalid type"},
		{name: "format", key: "format", val: "human", wantGet: "human"},
		{name: "tz valid", key: "tz", val: "Asia/Tokyo", wantGet: "Asia/Tokyo"},
		{name: "tz fuzzy place", key: "tz", val: "los_angeles", wantGet: "America/Los_Angeles"},
		{name: "tz alias", key: "tz", val: "CDMX", wantGet: "America/Mexico_City"},
		{name: "tz empty", key: "tz", val: "", wantErr: "invalid tz"},
		{name: "tz local rejected", key: "tz", val: "Local", wantErr: "invalid tz"},
		{name: "tz invalid", key: "tz", val: "Mars/Olympus", wantErr: "invalid tz"},
		{name: "show_timezone true", key: "show_timezone", val: "TRUE", wantGet: "true"},
		{name: "show_timezone false", key: "show_timezone", val: "false", wantGet: "false"},
		{name: "show_timezone bad", key: "show_timezone", val: "yes", wantErr: "invalid show_timezone"},
		{name: "lang es", key: "lang", val: "ES", wantGet: "es"},
		{name: "lang bad", key: "lang", val: "fr", wantErr: "invalid lang"},
		{name: "strict true", key: "strict", val: "true", wantGet: "true"},
		{name: "strict bad", key: "strict", val: "yes", wantErr: "invalid strict"},
		{name: "week_start es", key: "week_start", val: "miércoles", wantGet: "wednesday"},
		{name: "week_start bad", key: "week_start", val: "funday", wantErr: "invalid week_start"},
		{name: "week_one full", key: "week_one", val: "full", wantGet: "full"},
		{name: "week_one bad", key: "week_one", val: "nope", wantErr: "invalid week_one"},
		{name: "month_overflow", key: "month_overflow", val: "overflow", wantGet: "overflow"},
		{name: "month_overflow bad", key: "month_overflow", val: "bad", wantErr: "invalid month_overflow"},
		{name: "count_inclusive", key: "count_inclusive", val: "true", wantGet: "true"},
		{name: "count_inclusive bad", key: "count_inclusive", val: "1", wantErr: "invalid count_inclusive"},
		{name: "country us", key: "country", val: "us", wantGet: "US"},
		{name: "country none", key: "country", val: "none", wantGet: "none"},
		{name: "country unknown", key: "country", val: "xx", wantErr: "invalid country"},
		{name: "weekend es dedup", key: "weekend", val: "viernes,sabado,viernes", wantGet: "friday,saturday"},
		{name: "weekend none", key: "weekend", val: "none", wantGet: "none"},
		{name: "weekend bad", key: "weekend", val: "funday", wantErr: "invalid weekend"},
		{name: "unknown key", key: "nope", val: "1", wantErr: "unknown setting"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := DefaultValues(time.UTC)
			err := Apply(&v, tc.key, tc.val)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Apply(%q, %q) err = %v, want substring %q", tc.key, tc.val, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Apply(%q, %q) unexpected error: %v", tc.key, tc.val, err)
			}
			def, ok := ByKey(tc.key)
			if !ok {
				t.Fatalf("ByKey(%q) returned false", tc.key)
			}
			if got := def.Get(&v); got != tc.wantGet {
				t.Errorf("Get(%q) = %q, want %q", tc.key, got, tc.wantGet)
			}
		})
	}
}

func TestFlagTakesValue(t *testing.T) {
	if !FlagTakesValue("tz") || !FlagTakesValue("t") || !FlagTakesValue("day-week-start") {
		t.Error("expected value flags to return true")
	}
	if FlagTakesValue("strict") || FlagTakesValue("count-inclusive") || FlagTakesValue("z") || FlagTakesValue("show-tz") || FlagTakesValue("show-timezone") || FlagTakesValue("unknown") {
		t.Error("expected bool/unknown flags to return false")
	}
}

func TestParseDaysOff(t *testing.T) {
	loc, err := time.LoadLocation("America/Mexico_City")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	ctx := expr.Context{
		Now: time.Date(2026, 11, 14, 14, 30, 0, 0, loc),
		Loc: loc,
		Cfg: expr.DefaultConfig(),
	}

	vals, line, err := ParseDaysOffText("# header\n2026-11-17 # personal\n2026-12-24 .. 2026-12-26\n", ctx)
	if err != nil {
		t.Fatalf("ParseDaysOffText unexpected error at line %d: %v", line, err)
	}
	if len(vals) != 2 || vals[0].Kind != expr.KDate || vals[1].Kind != expr.KSpan {
		t.Fatalf("unexpected vals: %+v", vals)
	}

	_, line, err = ParseDaysOffText("2026-11-17\n2h\n", ctx)
	if err == nil || line != 2 || !strings.Contains(err.Error(), "duration") {
		t.Fatalf("want line 2 duration error, got line=%d err=%v", line, err)
	}

	ics := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nDTSTART;VALUE=DATE:20261117\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
	icsVals, err := ParseDaysOffICS(ics, ctx)
	if err != nil {
		t.Fatalf("ParseDaysOffICS unexpected error: %v", err)
	}
	if len(icsVals) != 1 || !reflect.DeepEqual(icsVals[0].T.Format("2006-01-02"), "2026-11-17") {
		t.Fatalf("icsVals = %+v, want 2026-11-17", icsVals)
	}
}

func TestLookupAndReset(t *testing.T) {
	lookupCases := []struct {
		input   string
		wantKey string
		wantOK  bool
	}{
		{input: "format", wantKey: "format", wantOK: true},
		{input: "FORMAT", wantKey: "format", wantOK: true},
		{input: "TZ", wantKey: "tz", wantOK: true},
		{input: "WEEK_START", wantKey: "week_start", wantOK: true},
		{input: "AHORITA_FORMAT", wantKey: "format", wantOK: true},
		{input: "AHORITA_WEEK_START", wantKey: "week_start", wantOK: true},
		{input: "day-week-start", wantKey: "week_start", wantOK: true},
		{input: "show-tz", wantKey: "show_timezone", wantOK: true},
		{input: "custom-format", wantKey: "format", wantOK: true},
		{input: "timezone", wantKey: "tz", wantOK: true},
		{input: "language", wantKey: "lang", wantOK: true},
		{input: "calendars", wantKey: "calendar", wantOK: true},
		{input: "f", wantKey: "format", wantOK: true},
		{input: "t", wantKey: "type", wantOK: true},
		{input: "z", wantKey: "show_timezone", wantOK: true},
		{input: "", wantOK: false},
		{input: "ahorita_", wantOK: false},
		{input: "unknown", wantOK: false},
	}

	for _, tc := range lookupCases {
		def, ok := Lookup(tc.input)
		if ok != tc.wantOK {
			t.Errorf("Lookup(%q) ok = %v, want %v", tc.input, ok, tc.wantOK)
			continue
		}
		if ok && def.Key != tc.wantKey {
			t.Errorf("Lookup(%q).Key = %q, want %q", tc.input, def.Key, tc.wantKey)
		}
	}

	defaults := DefaultValues(time.UTC)
	v := DefaultValues(time.UTC)
	changes := map[string]string{
		"type":            "date",
		"format":          "human",
		"tz":              "America/Mexico_City",
		"show_timezone":   "true",
		"lang":            "es",
		"strict":          "true",
		"week_start":      "sunday",
		"week_one":        "full",
		"month_overflow":  "overflow",
		"count_inclusive": "true",
		"country":         "US",
		"weekend":         "friday,saturday",
		"calendar":        "/tmp/a.txt,/tmp/b.txt",
	}
	for k, val := range changes {
		if err := Apply(&v, k, val); err != nil {
			t.Fatalf("Apply(%q, %q): %v", k, val, err)
		}
	}

	if Reset(&v, "unknown", defaults) {
		t.Error("Reset(unknown) = true, want false")
	}

	for k := range changes {
		if !Reset(&v, strings.ToUpper(k), defaults) {
			t.Fatalf("Reset(%q) = false, want true", k)
		}
	}
	if !reflect.DeepEqual(v, defaults) {
		t.Errorf("after resetting all keys, v = %+v, want %+v", v, defaults)
	}
}
