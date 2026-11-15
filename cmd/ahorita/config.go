package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/primissus/ahorita/expr"
)

type settings struct {
	cfg       expr.Config
	loc       *time.Location
	opts      expr.Options // Type, Pattern
	calendars []string
	// output switches, flags only:
	noNewline, copy, json, explain bool
	output                         string
}

type settingDef struct {
	key    string
	flags  []string
	env    string
	isBool bool
	usage  string
	get    func(s *settings) string
	apply  func(s *settings, value string) error
}

var settingsTable = []settingDef{
	{
		key:   "type",
		flags: []string{"t", "type"},
		env:   "AHORITA_TYPE",
		usage: "output type (date, datetime, time, range, relative)",
		get:   func(s *settings) string { return s.opts.Type },
		apply: func(s *settings, value string) error {
			v := strings.ToLower(value)
			switch v {
			case "date", "datetime", "time", "range", "relative":
				s.opts.Type = v
				return nil
			default:
				return fmt.Errorf("invalid type %q: allowed values are date, datetime, time, range, relative", value)
			}
		},
	},
	{
		key:   "format",
		flags: []string{"f", "format"},
		env:   "AHORITA_FORMAT",
		usage: "output pattern or preset (iso, rfc3339, unix, human, short)",
		get:   func(s *settings) string { return s.opts.Pattern },
		apply: func(s *settings, value string) error {
			s.opts.Pattern = value
			return nil
		},
	},
	{
		key:   "tz",
		flags: []string{"tz"},
		env:   "AHORITA_TZ",
		usage: "default IANA timezone",
		get: func(s *settings) string {
			if s.loc == nil {
				return ""
			}
			return s.loc.String()
		},
		apply: func(s *settings, value string) error {
			if value == "" {
				return fmt.Errorf("invalid tz %q: must be an IANA timezone name", value)
			}
			loc, err := time.LoadLocation(value)
			if err != nil {
				return fmt.Errorf("invalid tz %q: must be an IANA timezone name (%w)", value, err)
			}
			s.loc = loc
			return nil
		},
	},
	{
		key:   "lang",
		flags: []string{"lang"},
		env:   "AHORITA_LANG",
		usage: "default language (en, es)",
		get:   func(s *settings) string { return string(s.cfg.Lang) },
		apply: func(s *settings, value string) error {
			switch strings.ToLower(value) {
			case "en":
				s.cfg.Lang = expr.EN
				return nil
			case "es":
				s.cfg.Lang = expr.ES
				return nil
			default:
				return fmt.Errorf("invalid lang %q: allowed values are en, es", value)
			}
		},
	},
	{
		key:    "strict",
		flags:  []string{"strict"},
		env:    "AHORITA_STRICT",
		isBool: true,
		usage:  "error on ambiguous input or out-of-order stages",
		get:    func(s *settings) string { return strconv.FormatBool(s.cfg.Strict) },
		apply: func(s *settings, value string) error {
			switch strings.ToLower(value) {
			case "true":
				s.cfg.Strict = true
				return nil
			case "false":
				s.cfg.Strict = false
				return nil
			default:
				return fmt.Errorf("invalid strict %q: allowed values are true, false", value)
			}
		},
	},
	{
		key:   "week_start",
		flags: []string{"day-week-start"},
		env:   "AHORITA_WEEK_START",
		usage: "first day of the week (weekday in English or Spanish)",
		get:   func(s *settings) string { return strings.ToLower(s.cfg.WeekStart.String()) },
		apply: func(s *settings, value string) error {
			toks := expr.Normalize(expr.Lex(value), expr.EN)
			if len(toks) == 2 && toks[1].Kind == expr.EOF {
				if wd, ok := weekdayKey(toks[0].Key); ok {
					s.cfg.WeekStart = wd
					return nil
				}
			}
			return fmt.Errorf("invalid week_start %q: allowed values are a weekday name in English or Spanish", value)
		},
	},
	{
		key:   "week_one",
		flags: []string{"week-one"},
		env:   "AHORITA_WEEK_ONE",
		usage: "week 1 rule (iso, first-day, full, days)",
		get:   func(s *settings) string { return s.cfg.WeekOne },
		apply: func(s *settings, value string) error {
			v := strings.ToLower(value)
			switch v {
			case "iso", "first-day", "full", "days":
				s.cfg.WeekOne = v
				return nil
			default:
				return fmt.Errorf("invalid week_one %q: allowed values are iso, first-day, full, days", value)
			}
		},
	},
	{
		key:   "month_overflow",
		flags: []string{"month-overflow"},
		env:   "AHORITA_MONTH_OVERFLOW",
		usage: "month arithmetic rule (clamp, overflow)",
		get: func(s *settings) string {
			if s.cfg.MonthOverflow {
				return "overflow"
			}
			return "clamp"
		},
		apply: func(s *settings, value string) error {
			switch strings.ToLower(value) {
			case "clamp":
				s.cfg.MonthOverflow = false
				return nil
			case "overflow":
				s.cfg.MonthOverflow = true
				return nil
			default:
				return fmt.Errorf("invalid month_overflow %q: allowed values are clamp, overflow", value)
			}
		},
	},
	{
		key:    "count_inclusive",
		flags:  []string{"count-inclusive"},
		env:    "AHORITA_COUNT_INCLUSIVE",
		isBool: true,
		usage:  "include end dates when counting (true, false)",
		get:    func(s *settings) string { return strconv.FormatBool(s.cfg.CountInclusive) },
		apply: func(s *settings, value string) error {
			switch strings.ToLower(value) {
			case "true":
				s.cfg.CountInclusive = true
				return nil
			case "false":
				s.cfg.CountInclusive = false
				return nil
			default:
				return fmt.Errorf("invalid count_inclusive %q: allowed values are true, false", value)
			}
		},
	},
	{
		key:   "country",
		flags: []string{"country"},
		env:   "AHORITA_COUNTRY",
		usage: "holiday country code (two-letter code, or none)",
		get: func(s *settings) string {
			if s.cfg.Country == "" {
				return "none"
			}
			return s.cfg.Country
		},
		apply: func(s *settings, value string) error {
			v := strings.TrimSpace(value)
			if strings.EqualFold(v, "none") {
				s.cfg.Country = ""
				return nil
			}
			code := strings.ToUpper(v)
			if len(code) != 2 || code[0] < 'A' || code[0] > 'Z' || code[1] < 'A' || code[1] > 'Z' {
				return fmt.Errorf("invalid country %q: allowed values are a two-letter country code or none", value)
			}
			res := expr.Run("holidays "+code+" 2026", expr.Context{Loc: time.UTC}, expr.Options{}, false)
			if res.Error != nil {
				return fmt.Errorf("invalid country %q: %s", value, res.Error.Msg)
			}
			s.cfg.Country = code
			return nil
		},
	},
	{
		key:   "weekend",
		flags: []string{"weekend"},
		env:   "AHORITA_WEEKEND",
		usage: "comma-separated weekend weekdays in English or Spanish, or none",
		get: func(s *settings) string {
			if len(s.cfg.Weekend) == 0 {
				return "none"
			}
			parts := make([]string, len(s.cfg.Weekend))
			for i, w := range s.cfg.Weekend {
				parts[i] = strings.ToLower(w.String())
			}
			return strings.Join(parts, ",")
		},
		apply: func(s *settings, value string) error {
			v := strings.TrimSpace(value)
			if strings.EqualFold(v, "none") {
				s.cfg.Weekend = []time.Weekday{}
				return nil
			}
			if v == "" {
				return fmt.Errorf("invalid weekend %q: allowed values are comma-separated weekday names or none", value)
			}
			raw := strings.Split(v, ",")
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
			s.cfg.Weekend = days
			return nil
		},
	},
	{
		key:   "calendar",
		flags: []string{"calendar"},
		env:   "AHORITA_CALENDAR",
		usage: "days-off file path (repeatable; comma-separated in env/config)",
		get: func(s *settings) string {
			return strings.Join(s.calendars, ",")
		},
		apply: func(s *settings, value string) error {
			v := strings.TrimSpace(value)
			if v == "" {
				return fmt.Errorf("invalid calendar %q: path cannot be empty", value)
			}
			raw := strings.Split(v, ",")
			paths := make([]string, 0, len(raw))
			for _, item := range raw {
				p := strings.TrimSpace(item)
				if p == "" {
					return fmt.Errorf("invalid calendar %q: empty path", value)
				}
				paths = append(paths, p)
			}
			s.calendars = paths
			return nil
		},
	},
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

type settingValue struct {
	def *settingDef
	s   *settings
}

func (v *settingValue) String() string {
	if v.s == nil || v.def == nil || v.def.get == nil {
		return ""
	}
	return v.def.get(v.s)
}

func (v *settingValue) Set(val string) error {
	return v.def.apply(v.s, val)
}

type boolSettingValue struct {
	settingValue
}

func (v *boolSettingValue) IsBoolFlag() bool {
	return true
}

type calendarFlagValue struct {
	s    *settings
	seen bool
}

func (v *calendarFlagValue) String() string {
	if v.s == nil {
		return ""
	}
	return strings.Join(v.s.calendars, ",")
}

func (v *calendarFlagValue) Set(val string) error {
	p := strings.TrimSpace(val)
	if p == "" {
		return fmt.Errorf("invalid calendar %q: path cannot be empty", val)
	}
	if !v.seen {
		v.seen = true
		v.s.calendars = nil
	}
	v.s.calendars = append(v.s.calendars, p)
	return nil
}

type helpError struct {
	text string
}

func (e helpError) Error() string {
	return e.text
}

func load(args []string, getenv func(string) string, readFile func(string) ([]byte, error)) (settings, []string, error) {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	if readFile == nil {
		readFile = func(string) ([]byte, error) { return nil, os.ErrNotExist }
	}

	s := settings{
		cfg: expr.DefaultConfig(),
		loc: time.Local,
	}
	lc := getenv("LC_ALL")
	if lc == "" {
		lc = getenv("LANG")
	}
	if strings.HasPrefix(strings.ToLower(lc), "es") {
		s.cfg.Lang = expr.ES
	}

	if err := loadConfigFile(args, getenv, readFile, &s); err != nil {
		return settings{}, nil, err
	}

	for i := range settingsTable {
		def := &settingsTable[i]
		if v := getenv(def.env); v != "" {
			if err := def.apply(&s, v); err != nil {
				return settings{}, nil, err
			}
		}
	}

	fs := flag.NewFlagSet("ahorita", flag.ContinueOnError)
	var flagBuf bytes.Buffer
	fs.SetOutput(&flagBuf)

	for i := range settingsTable {
		def := &settingsTable[i]
		var fv flag.Value
		switch {
		case def.key == "calendar":
			fv = &calendarFlagValue{s: &s}
		case def.isBool:
			fv = &boolSettingValue{settingValue: settingValue{def: def, s: &s}}
		default:
			sv := settingValue{def: def, s: &s}
			fv = &sv
		}
		for _, name := range def.flags {
			fs.Var(fv, name, def.usage)
		}
	}

	var configFile string
	fs.BoolVar(&s.noNewline, "n", false, "do not print a trailing newline")
	fs.BoolVar(&s.copy, "c", false, "copy output to clipboard")
	fs.BoolVar(&s.json, "json", false, "print structured JSON")
	fs.BoolVar(&s.explain, "explain", false, "print evaluation steps")
	fs.StringVar(&configFile, "config-file", "", "path to config file")
	fs.StringVar(&s.output, "o", "", "write output to file")
	fs.StringVar(&s.output, "output", "", "write output to file")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return settings{}, nil, helpError{text: strings.TrimSpace(flagBuf.String())}
		}
		return settings{}, nil, err
	}

	return s, fs.Args(), nil
}

func loadConfigFile(args []string, getenv func(string) string, readFile func(string) ([]byte, error), s *settings) error {
	path, explicit, err := resolveConfigPath(args, getenv)
	if err != nil {
		return err
	}
	if path == "" {
		return nil
	}

	data, err := readFile(path)
	if err != nil {
		if !explicit && errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read config %q: %w", path, err)
	}

	return parseConfig(path, data, s)
}

func resolveConfigPath(args []string, getenv func(string) string) (string, bool, error) {
	if p, ok := findConfigFlag(args); ok {
		if p == "" {
			return "", true, fmt.Errorf("empty --config-file path")
		}
		return p, true, nil
	}
	if p := getenv("AHORITA_CONFIG"); p != "" {
		return p, true, nil
	}
	if xdg := getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "ahorita", "config.toml"), false, nil
	}
	home := getenv("HOME")
	if home == "" {
		if h, err := os.UserHomeDir(); err == nil {
			home = h
		}
	}
	if home == "" {
		return "", false, nil
	}
	return filepath.Join(home, ".config", "ahorita", "config.toml"), false, nil
}

func findConfigFlag(args []string) (string, bool) {
	var found string
	var has bool
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" || !strings.HasPrefix(arg, "-") || arg == "-" {
			break
		}
		name := strings.TrimPrefix(arg, "--")
		if name == arg {
			name = strings.TrimPrefix(arg, "-")
		}
		if eq := strings.IndexByte(name, '='); eq >= 0 {
			if name[:eq] == "config-file" {
				found = name[eq+1:]
				has = true
			}
			continue
		}
		if name == "config-file" {
			if i+1 < len(args) {
				found = args[i+1]
				has = true
				i++
			}
			continue
		}
		if flagTakesValue(name) {
			i++
		}
	}
	return found, has
}

func flagTakesValue(name string) bool {
	if name == "o" || name == "output" || name == "config-file" {
		return true
	}
	for _, def := range settingsTable {
		if !def.isBool && slices.Contains(def.flags, name) {
			return true
		}
	}
	return false
}

func parseConfig(path string, data []byte, s *settings) error {
	byKey := make(map[string]*settingDef, len(settingsTable))
	for i := range settingsTable {
		byKey[settingsTable[i].key] = &settingsTable[i]
	}

	lines := strings.Split(string(data), "\n")
	for i, rawLine := range lines {
		lineNum := i + 1
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			return fmt.Errorf("%s:%d: expected key = value", path, lineNum)
		}
		key := strings.TrimSpace(line[:eq])
		if key == "" {
			return fmt.Errorf("%s:%d: expected key = value", path, lineNum)
		}
		rawVal := strings.TrimSpace(line[eq+1:])
		val, err := parseConfigValue(rawVal)
		if err != nil {
			return fmt.Errorf("%s:%d: %w", path, lineNum, err)
		}
		def, ok := byKey[key]
		if !ok {
			return fmt.Errorf("%s:%d: unknown key %q", path, lineNum, key)
		}
		if err := def.apply(s, val); err != nil {
			return fmt.Errorf("%s:%d: %w", path, lineNum, err)
		}
	}
	return nil
}

func parseConfigValue(raw string) (string, error) {
	if strings.HasPrefix(raw, `"`) {
		end := strings.IndexByte(raw[1:], '"')
		if end < 0 {
			return "", fmt.Errorf("unterminated quote")
		}
		val := raw[1 : 1+end]
		rest := strings.TrimSpace(raw[2+end:])
		if rest != "" && !strings.HasPrefix(rest, "#") {
			return "", fmt.Errorf("unexpected trailing text %q", rest)
		}
		return val, nil
	}
	if strings.HasPrefix(raw, "#") {
		return "", nil
	}
	for i := 1; i < len(raw); i++ {
		if raw[i] == '#' && (raw[i-1] == ' ' || raw[i-1] == '\t') {
			return strings.TrimSpace(raw[:i]), nil
		}
	}
	return raw, nil
}

func loadDaysOff(paths []string, readFile func(string) ([]byte, error), ctx *expr.Context) error {
	if len(paths) == 0 {
		return nil
	}
	if readFile == nil {
		readFile = func(string) ([]byte, error) { return nil, os.ErrNotExist }
	}
	for _, path := range paths {
		data, err := readFile(path)
		if err != nil {
			return fmt.Errorf("read calendar %q: %w", path, err)
		}
		if strings.EqualFold(filepath.Ext(path), ".ics") {
			loc := ctx.Loc
			if loc == nil {
				loc = ctx.Now.Location()
				if loc == nil {
					loc = time.UTC
				}
			}
			now := ctx.Now.In(loc)
			vals, err := expr.DaysFromICS(string(data), now.AddDate(-1, 0, 0), now.AddDate(5, 0, 0))
			if err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			ctx.Cfg.DaysOff = append(ctx.Cfg.DaysOff, vals...)
			continue
		}
		lines := strings.Split(string(data), "\n")
		for i, rawLine := range lines {
			lineNum := i + 1
			line := stripCalendarComment(rawLine)
			if line == "" {
				continue
			}
			val, err := expr.Eval(line, *ctx)
			if err != nil {
				return fmt.Errorf("%s: line %d: %w", path, lineNum, err)
			}
			if val.Kind != expr.KDate && !(val.Kind == expr.KSpan && val.Days) {
				return fmt.Errorf("%s: line %d: expected a date or whole-day span, got %s", path, lineNum, valueKindLabel(val))
			}
			ctx.Cfg.DaysOff = append(ctx.Cfg.DaysOff, val)
		}
	}
	return nil
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
