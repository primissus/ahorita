package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/primissus/ahorita/expr"
	internalsettings "github.com/primissus/ahorita/internal/settings"
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

type settingValue struct {
	def *internalsettings.Def
	v   *internalsettings.Values
}

func (sv *settingValue) String() string {
	if sv.v == nil || sv.def == nil || sv.def.Get == nil {
		return ""
	}
	return sv.def.Get(sv.v)
}

func (sv *settingValue) Set(val string) error {
	return sv.def.Apply(sv.v, val)
}

type boolSettingValue struct {
	settingValue
}

func (bv *boolSettingValue) IsBoolFlag() bool {
	return true
}

type calendarFlagValue struct {
	v    *internalsettings.Values
	seen bool
}

func (cv *calendarFlagValue) String() string {
	if cv.v == nil {
		return ""
	}
	return strings.Join(cv.v.Calendars, ",")
}

func (cv *calendarFlagValue) Set(val string) error {
	p := strings.TrimSpace(val)
	if p == "" {
		return fmt.Errorf("invalid calendar %q: path cannot be empty", val)
	}
	if !cv.seen {
		cv.seen = true
		cv.v.Calendars = nil
	}
	cv.v.Calendars = append(cv.v.Calendars, p)
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

	vals := internalsettings.DefaultValues(time.Local)
	lc := getenv("LC_ALL")
	if lc == "" {
		lc = getenv("LANG")
	}
	if strings.HasPrefix(strings.ToLower(lc), "es") {
		vals.Cfg.Lang = expr.ES
	}

	if err := loadConfigFile(args, getenv, readFile, &vals); err != nil {
		return settings{}, nil, err
	}

	for i := range internalsettings.Table {
		def := &internalsettings.Table[i]
		if val := getenv(def.Env); val != "" {
			if err := def.Apply(&vals, val); err != nil {
				return settings{}, nil, err
			}
		}
	}

	fs := flag.NewFlagSet("ahorita", flag.ContinueOnError)
	var flagBuf bytes.Buffer
	fs.SetOutput(&flagBuf)

	for i := range internalsettings.Table {
		def := &internalsettings.Table[i]
		var fv flag.Value
		switch {
		case def.Key == "calendar":
			fv = &calendarFlagValue{v: &vals}
		case def.IsBool:
			fv = &boolSettingValue{settingValue: settingValue{def: def, v: &vals}}
		default:
			sv := settingValue{def: def, v: &vals}
			fv = &sv
		}
		for _, name := range def.Flags {
			fs.Var(fv, name, def.Usage)
		}
	}

	var s settings
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

	s.cfg = vals.Cfg
	s.loc = vals.Loc
	s.opts = vals.Opts
	s.calendars = vals.Calendars
	return s, fs.Args(), nil
}

func loadConfigFile(args []string, getenv func(string) string, readFile func(string) ([]byte, error), vals *internalsettings.Values) error {
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

	return parseConfig(path, data, vals)
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
	return internalsettings.FlagTakesValue(name)
}

func parseConfig(path string, data []byte, vals *internalsettings.Values) error {
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
		def, ok := internalsettings.ByKey(key)
		if !ok {
			return fmt.Errorf("%s:%d: unknown key %q", path, lineNum, key)
		}
		if err := def.Apply(vals, val); err != nil {
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
			vals, err := internalsettings.ParseDaysOffICS(string(data), *ctx)
			if err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			ctx.Cfg.DaysOff = append(ctx.Cfg.DaysOff, vals...)
			continue
		}
		vals, lineNum, err := internalsettings.ParseDaysOffText(string(data), *ctx)
		if err != nil {
			return fmt.Errorf("%s: line %d: %w", path, lineNum, err)
		}
		ctx.Cfg.DaysOff = append(ctx.Cfg.DaysOff, vals...)
	}
	return nil
}
