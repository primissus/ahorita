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
		{name: "tz empty", key: "tz", val: "", wantErr: "invalid tz"},
		{name: "tz invalid", key: "tz", val: "Mars/Olympus", wantErr: "invalid tz"},
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
	if FlagTakesValue("strict") || FlagTakesValue("count-inclusive") || FlagTakesValue("unknown") {
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
