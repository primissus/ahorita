// Package settings defines the configuration keys, validation, and days-off
// parsing shared between the ahorita CLI and WebAssembly front ends.
package settings

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/primissus/ahorita/expr"
)

// Values holds the engine configuration, timezone location, formatting options,
// and calendar paths resolved by a front end.
type Values struct {
	Cfg       expr.Config
	Loc       *time.Location
	Opts      expr.Options // Type, Pattern
	Calendars []string
}

// DefaultValues returns Values populated with expr.DefaultConfig() and loc.
func DefaultValues(loc *time.Location) Values {
	return Values{
		Cfg: expr.DefaultConfig(),
		Loc: loc,
	}
}

// Def describes one named setting across CLI flags, env vars, config files,
// and WASM options.
type Def struct {
	Key    string
	Flags  []string
	Env    string
	IsBool bool
	Usage  string
	Get    func(v *Values) string
	Apply  func(v *Values, value string) error
}

// Table lists all configurable settings in precedence/help order.
var Table = []Def{
	{
		Key:   "type",
		Flags: []string{"t", "type"},
		Env:   "AHORITA_TYPE",
		Usage: "output type (date, datetime, time, range, relative)",
		Get:   func(v *Values) string { return v.Opts.Type },
		Apply: func(v *Values, value string) error {
			val := strings.ToLower(value)
			switch val {
			case "date", "datetime", "time", "range", "relative":
				v.Opts.Type = val
				return nil
			default:
				return fmt.Errorf("invalid type %q: allowed values are date, datetime, time, range, relative", value)
			}
		},
	},
	{
		Key:   "format",
		Flags: []string{"f", "format"},
		Env:   "AHORITA_FORMAT",
		Usage: "output pattern or preset (iso, rfc, rfc3339, unix, human, short)",
		Get:   func(v *Values) string { return v.Opts.Pattern },
		Apply: func(v *Values, value string) error {
			v.Opts.Pattern = value
			return nil
		},
	},
	{
		Key:   "tz",
		Flags: []string{"tz"},
		Env:   "AHORITA_TZ",
		Usage: "default IANA timezone",
		Get: func(v *Values) string {
			if v.Loc == nil {
				return ""
			}
			return v.Loc.String()
		},
		Apply: func(v *Values, value string) error {
			if value == "" || strings.EqualFold(strings.TrimSpace(value), "local") {
				return fmt.Errorf("invalid tz %q: must be an IANA timezone name", value)
			}
			loc, err := time.LoadLocation(value)
			if err != nil {
				var zErr error
				loc, zErr = expr.LookupZone(value, false)
				if zErr != nil {
					return fmt.Errorf("invalid tz %q: %w", value, zErr)
				}
			}
			v.Loc = loc
			return nil
		},
	},
	{
		Key:    "show_timezone",
		Flags:  []string{"z", "show-tz", "show-timezone"},
		Env:    "AHORITA_SHOW_TIMEZONE",
		IsBool: true,
		Usage:  "append timezone to formatted output (true, false)",
		Get:    func(v *Values) string { return strconv.FormatBool(v.Opts.ShowTimezone) },
		Apply: func(v *Values, value string) error {
			switch strings.ToLower(value) {
			case "true":
				v.Opts.ShowTimezone = true
				return nil
			case "false":
				v.Opts.ShowTimezone = false
				return nil
			default:
				return fmt.Errorf("invalid show_timezone %q: allowed values are true, false", value)
			}
		},
	},
	{
		Key:   "lang",
		Flags: []string{"lang"},
		Env:   "AHORITA_LANG",
		Usage: "default language (en, es)",
		Get:   func(v *Values) string { return string(v.Cfg.Lang) },
		Apply: func(v *Values, value string) error {
			switch strings.ToLower(value) {
			case "en":
				v.Cfg.Lang = expr.EN
				return nil
			case "es":
				v.Cfg.Lang = expr.ES
				return nil
			default:
				return fmt.Errorf("invalid lang %q: allowed values are en, es", value)
			}
		},
	},
	{
		Key:    "strict",
		Flags:  []string{"strict"},
		Env:    "AHORITA_STRICT",
		IsBool: true,
		Usage:  "error on ambiguous input or out-of-order stages",
		Get:    func(v *Values) string { return strconv.FormatBool(v.Cfg.Strict) },
		Apply: func(v *Values, value string) error {
			switch strings.ToLower(value) {
			case "true":
				v.Cfg.Strict = true
				return nil
			case "false":
				v.Cfg.Strict = false
				return nil
			default:
				return fmt.Errorf("invalid strict %q: allowed values are true, false", value)
			}
		},
	},
	{
		Key:   "week_start",
		Flags: []string{"day-week-start"},
		Env:   "AHORITA_WEEK_START",
		Usage: "first day of the week (weekday in English or Spanish)",
		Get:   func(v *Values) string { return strings.ToLower(v.Cfg.WeekStart.String()) },
		Apply: func(v *Values, value string) error {
			toks := expr.Normalize(expr.Lex(value), expr.EN)
			if len(toks) == 2 && toks[1].Kind == expr.EOF {
				if wd, ok := weekdayKey(toks[0].Key); ok {
					v.Cfg.WeekStart = wd
					return nil
				}
			}
			return fmt.Errorf("invalid week_start %q: allowed values are a weekday name in English or Spanish", value)
		},
	},
	{
		Key:   "week_one",
		Flags: []string{"week-one"},
		Env:   "AHORITA_WEEK_ONE",
		Usage: "week 1 rule (iso, first-day, full, days)",
		Get:   func(v *Values) string { return v.Cfg.WeekOne },
		Apply: func(v *Values, value string) error {
			val := strings.ToLower(value)
			switch val {
			case "iso", "first-day", "full", "days":
				v.Cfg.WeekOne = val
				return nil
			default:
				return fmt.Errorf("invalid week_one %q: allowed values are iso, first-day, full, days", value)
			}
		},
	},
	{
		Key:   "month_overflow",
		Flags: []string{"month-overflow"},
		Env:   "AHORITA_MONTH_OVERFLOW",
		Usage: "month arithmetic rule (clamp, overflow)",
		Get: func(v *Values) string {
			if v.Cfg.MonthOverflow {
				return "overflow"
			}
			return "clamp"
		},
		Apply: func(v *Values, value string) error {
			switch strings.ToLower(value) {
			case "clamp":
				v.Cfg.MonthOverflow = false
				return nil
			case "overflow":
				v.Cfg.MonthOverflow = true
				return nil
			default:
				return fmt.Errorf("invalid month_overflow %q: allowed values are clamp, overflow", value)
			}
		},
	},
	{
		Key:    "count_inclusive",
		Flags:  []string{"count-inclusive"},
		Env:    "AHORITA_COUNT_INCLUSIVE",
		IsBool: true,
		Usage:  "include end dates when counting (true, false)",
		Get:    func(v *Values) string { return strconv.FormatBool(v.Cfg.CountInclusive) },
		Apply: func(v *Values, value string) error {
			switch strings.ToLower(value) {
			case "true":
				v.Cfg.CountInclusive = true
				return nil
			case "false":
				v.Cfg.CountInclusive = false
				return nil
			default:
				return fmt.Errorf("invalid count_inclusive %q: allowed values are true, false", value)
			}
		},
	},
	{
		Key:   "country",
		Flags: []string{"country"},
		Env:   "AHORITA_COUNTRY",
		Usage: "holiday country code (two-letter code, or none)",
		Get: func(v *Values) string {
			if v.Cfg.Country == "" {
				return "none"
			}
			return v.Cfg.Country
		},
		Apply: func(v *Values, value string) error {
			val := strings.TrimSpace(value)
			if strings.EqualFold(val, "none") {
				v.Cfg.Country = ""
				return nil
			}
			code := strings.ToUpper(val)
			if len(code) != 2 || code[0] < 'A' || code[0] > 'Z' || code[1] < 'A' || code[1] > 'Z' {
				return fmt.Errorf("invalid country %q: allowed values are a two-letter country code or none", value)
			}
			res := expr.Run("holidays "+code+" 2026", expr.Context{Loc: time.UTC}, expr.Options{}, false)
			if res.Error != nil {
				return fmt.Errorf("invalid country %q: %s", value, res.Error.Msg)
			}
			v.Cfg.Country = code
			return nil
		},
	},
	{
		Key:   "weekend",
		Flags: []string{"weekend"},
		Env:   "AHORITA_WEEKEND",
		Usage: "comma-separated weekend weekdays in English or Spanish, or none",
		Get: func(v *Values) string {
			if len(v.Cfg.Weekend) == 0 {
				return "none"
			}
			parts := make([]string, len(v.Cfg.Weekend))
			for i, w := range v.Cfg.Weekend {
				parts[i] = strings.ToLower(w.String())
			}
			return strings.Join(parts, ",")
		},
		Apply: func(v *Values, value string) error {
			val := strings.TrimSpace(value)
			if strings.EqualFold(val, "none") {
				v.Cfg.Weekend = []time.Weekday{}
				return nil
			}
			if val == "" {
				return fmt.Errorf("invalid weekend %q: allowed values are comma-separated weekday names or none", value)
			}
			raw := strings.Split(val, ",")
			days := make([]time.Weekday, 0, len(raw))
			for _, item := range raw {
				part := strings.TrimSpace(item)
				if part == "" {
					return fmt.Errorf("invalid weekend %q: empty weekday name", value)
				}
				toks := expr.Normalize(expr.Lex(part), expr.EN)
				if len(toks) != 2 || toks[1].Kind != expr.EOF {
					return fmt.Errorf("invalid weekend %q: %q is not a weekday name in English or Spanish", value, part)
				}
				wd, ok := weekdayKey(toks[0].Key)
				if !ok {
					return fmt.Errorf("invalid weekend %q: %q is not a weekday name in English or Spanish", value, part)
				}
				if !slices.Contains(days, wd) {
					days = append(days, wd)
				}
			}
			v.Cfg.Weekend = days
			return nil
		},
	},
	{
		Key:   "calendar",
		Flags: []string{"calendar"},
		Env:   "AHORITA_CALENDAR",
		Usage: "days-off file path (repeatable; comma-separated in env/config)",
		Get: func(v *Values) string {
			return strings.Join(v.Calendars, ",")
		},
		Apply: func(v *Values, value string) error {
			val := strings.TrimSpace(value)
			if val == "" {
				return fmt.Errorf("invalid calendar %q: path cannot be empty", value)
			}
			raw := strings.Split(val, ",")
			paths := make([]string, 0, len(raw))
			for _, item := range raw {
				p := strings.TrimSpace(item)
				if p == "" {
					return fmt.Errorf("invalid calendar %q: empty path", value)
				}
				paths = append(paths, p)
			}
			v.Calendars = paths
			return nil
		},
	},
}

// ByKey looks up a setting definition by its canonical key.
func ByKey(key string) (*Def, bool) {
	for i := range Table {
		if Table[i].Key == key {
			return &Table[i], true
		}
	}
	return nil, false
}

// Lookup finds a setting definition by canonical key, flag name, env var name,
// or common alias (case-insensitive, '-' and '_' treated interchangeably).
func Lookup(name string) (*Def, bool) {
	norm := strings.ToLower(strings.TrimSpace(name))
	norm = strings.ReplaceAll(norm, "-", "_")
	norm = strings.TrimPrefix(norm, "ahorita_")
	if norm == "" {
		return nil, false
	}

	switch norm {
	case "day_week_start":
		norm = "week_start"
	case "show_tz":
		norm = "show_timezone"
	case "custom_format":
		norm = "format"
	case "timezone":
		norm = "tz"
	case "language":
		norm = "lang"
	case "calendars":
		norm = "calendar"
	}

	if def, ok := ByKey(norm); ok {
		return def, true
	}
	for i := range Table {
		for _, fl := range Table[i].Flags {
			if strings.ReplaceAll(strings.ToLower(fl), "-", "_") == norm {
				return &Table[i], true
			}
		}
	}
	return nil, false
}

// Reset restores the setting named by key on v to its value in defaults.
func Reset(v *Values, key string, defaults Values) bool {
	if v == nil {
		return false
	}
	def, ok := Lookup(key)
	if !ok {
		return false
	}
	switch def.Key {
	case "type":
		v.Opts.Type = defaults.Opts.Type
	case "format":
		v.Opts.Pattern = defaults.Opts.Pattern
	case "tz":
		v.Loc = defaults.Loc
	case "show_timezone":
		v.Opts.ShowTimezone = defaults.Opts.ShowTimezone
	case "lang":
		v.Cfg.Lang = defaults.Cfg.Lang
	case "strict":
		v.Cfg.Strict = defaults.Cfg.Strict
	case "week_start":
		v.Cfg.WeekStart = defaults.Cfg.WeekStart
	case "week_one":
		v.Cfg.WeekOne = defaults.Cfg.WeekOne
	case "month_overflow":
		v.Cfg.MonthOverflow = defaults.Cfg.MonthOverflow
	case "count_inclusive":
		v.Cfg.CountInclusive = defaults.Cfg.CountInclusive
	case "country":
		v.Cfg.Country = defaults.Cfg.Country
	case "weekend":
		v.Cfg.Weekend = slices.Clone(defaults.Cfg.Weekend)
	case "calendar":
		v.Calendars = slices.Clone(defaults.Calendars)
	default:
		return false
	}
	return true
}

// Apply looks up key and applies value to v.
func Apply(v *Values, key, value string) error {
	def, ok := ByKey(key)
	if !ok {
		return fmt.Errorf("unknown setting %q", key)
	}
	return def.Apply(v, value)
}

// FlagTakesValue reports whether a CLI flag name from Table consumes the next argument.
func FlagTakesValue(name string) bool {
	for _, def := range Table {
		if !def.IsBool && slices.Contains(def.Flags, name) {
			return true
		}
	}
	return false
}

func weekdayKey(key string) (time.Weekday, bool) {
	switch key {
	case "sunday":
		return time.Sunday, true
	case "monday":
		return time.Monday, true
	case "tuesday":
		return time.Tuesday, true
	case "wednesday":
		return time.Wednesday, true
	case "thursday":
		return time.Thursday, true
	case "friday":
		return time.Friday, true
	case "saturday":
		return time.Saturday, true
	default:
		return 0, false
	}
}

// ParseDaysOffICS parses iCalendar (.ics) data into non-business dates/spans
// within [now-1y, now+5y].
func ParseDaysOffICS(data string, ctx expr.Context) ([]expr.Value, error) {
	loc := ctx.Loc
	if loc == nil {
		loc = ctx.Now.Location()
		if loc == nil {
			loc = time.UTC
		}
	}
	now := ctx.Now.In(loc)
	return expr.DaysFromICS(data, now.AddDate(-1, 0, 0), now.AddDate(5, 0, 0))
}

// ParseDaysOffText parses newline-delimited days-off expressions (dates or
// whole-day spans, with optional # comments). On error it returns the 1-based
// failing line number alongside the error.
func ParseDaysOffText(data string, ctx expr.Context) ([]expr.Value, int, error) {
	lines := strings.Split(data, "\n")
	var out []expr.Value
	for i, rawLine := range lines {
		lineNum := i + 1
		line := stripCalendarComment(rawLine)
		if line == "" {
			continue
		}
		val, err := expr.Eval(line, ctx)
		if err != nil {
			return nil, lineNum, err
		}
		if val.Kind != expr.KDate && !(val.Kind == expr.KSpan && val.Days) {
			return nil, lineNum, fmt.Errorf("expected a date or whole-day span, got %s", valueKindLabel(val))
		}
		out = append(out, val)
	}
	return out, 0, nil
}

func stripCalendarComment(line string) string {
	inQuote := false
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '"':
			inQuote = !inQuote
		case '#':
			if !inQuote {
				return strings.TrimSpace(line[:i])
			}
		}
	}
	return strings.TrimSpace(line)
}

func valueKindLabel(v expr.Value) string {
	switch v.Kind {
	case expr.KDateTime:
		return "a date-time"
	case expr.KTime:
		return "a time"
	case expr.KDur:
		return "a duration"
	case expr.KSpan:
		return "a non-whole-day span"
	case expr.KNumber:
		return "a number"
	case expr.KList:
		return "a list"
	case expr.KText:
		return "text"
	case expr.KRecurrence:
		return "a recurrence"
	default:
		return "a value"
	}
}
