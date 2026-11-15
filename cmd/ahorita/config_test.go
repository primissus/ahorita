package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/primissus/ahorita/expr"
)

func envMap(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func filesMap(m map[string]string) func(string) ([]byte, error) {
	return func(path string) ([]byte, error) {
		if body, ok := m[path]; ok {
			return []byte(body), nil
		}
		return nil, os.ErrNotExist
	}
}

func TestLoad(t *testing.T) {
	noEnv := envMap(nil)
	noFiles := filesMap(nil)

	tests := []struct {
		name     string
		args     []string
		env      map[string]string
		wantArgs []string
		wantErr  string
		check    func(t *testing.T, s settings)
	}{
		{
			name: "no input defaults",
			check: func(t *testing.T, s settings) {
				if !reflect.DeepEqual(s.cfg, expr.DefaultConfig()) {
					t.Errorf("cfg = %+v, want %+v", s.cfg, expr.DefaultConfig())
				}
				if s.loc != time.Local {
					t.Errorf("loc = %v, want time.Local", s.loc)
				}
				if s.opts != (expr.Options{}) {
					t.Errorf("opts = %+v, want empty", s.opts)
				}
			},
		},
		{
			name: "AHORITA_TYPE=date",
			env:  map[string]string{"AHORITA_TYPE": "date"},
			check: func(t *testing.T, s settings) {
				if s.opts.Type != "date" {
					t.Errorf("opts.Type = %q, want date", s.opts.Type)
				}
			},
		},
		{
			name: "AHORITA_TYPE=date plus -t time flag wins",
			args: []string{"-t", "time"},
			env:  map[string]string{"AHORITA_TYPE": "date"},
			check: func(t *testing.T, s settings) {
				if s.opts.Type != "time" {
					t.Errorf("opts.Type = %q, want time", s.opts.Type)
				}
			},
		},
		{
			name: "day-week-start domingo",
			args: []string{"--day-week-start", "domingo"},
			check: func(t *testing.T, s settings) {
				if s.cfg.WeekStart != time.Sunday {
					t.Errorf("WeekStart = %v, want Sunday", s.cfg.WeekStart)
				}
			},
		},
		{
			name: "day-week-start miércoles accented",
			args: []string{"--day-week-start", "miércoles"},
			check: func(t *testing.T, s settings) {
				if s.cfg.WeekStart != time.Wednesday {
					t.Errorf("WeekStart = %v, want Wednesday", s.cfg.WeekStart)
				}
			},
		},
		{
			name:    "day-week-start xyz error",
			args:    []string{"--day-week-start", "xyz"},
			wantErr: "week_start",
		},
		{
			name:    "-t week error",
			args:    []string{"-t", "week"},
			wantErr: "type",
		},
		{
			name: "LANG=es_MX.UTF-8",
			env:  map[string]string{"LANG": "es_MX.UTF-8"},
			check: func(t *testing.T, s settings) {
				if s.cfg.Lang != expr.ES {
					t.Errorf("Lang = %q, want es", s.cfg.Lang)
				}
			},
		},
		{
			name: "LC_ALL overrides LANG",
			env:  map[string]string{"LC_ALL": "en_US.UTF-8", "LANG": "es_MX.UTF-8"},
			check: func(t *testing.T, s settings) {
				if s.cfg.Lang != expr.EN {
					t.Errorf("Lang = %q, want en", s.cfg.Lang)
				}
			},
		},
		{
			name: "LANG=es_MX.UTF-8 plus --lang en flag wins",
			args: []string{"--lang", "en"},
			env:  map[string]string{"LANG": "es_MX.UTF-8"},
			check: func(t *testing.T, s settings) {
				if s.cfg.Lang != expr.EN {
					t.Errorf("Lang = %q, want en", s.cfg.Lang)
				}
			},
		},
		{
			name: "--tz Asia/Tokyo",
			args: []string{"--tz", "Asia/Tokyo"},
			check: func(t *testing.T, s settings) {
				if s.loc.String() != "Asia/Tokyo" {
					t.Errorf("loc = %q, want Asia/Tokyo", s.loc.String())
				}
			},
		},
		{
			name:    "--tz Nowhere error",
			args:    []string{"--tz", "Nowhere"},
			wantErr: "tz",
		},
		{
			name:     "-t date now + 2d remaining args",
			args:     []string{"-t", "date", "now", "+", "2d"},
			wantArgs: []string{"now", "+", "2d"},
			check: func(t *testing.T, s settings) {
				if s.opts.Type != "date" {
					t.Errorf("opts.Type = %q, want date", s.opts.Type)
				}
			},
		},
		{
			name:     "--strict does not eat expression",
			args:     []string{"--strict", "today"},
			wantArgs: []string{"today"},
			check: func(t *testing.T, s settings) {
				if !s.cfg.Strict {
					t.Error("Strict = false, want true")
				}
			},
		},
		{
			name: "AHORITA_STRICT=true overridden by --strict=false",
			args: []string{"--strict=false"},
			env:  map[string]string{"AHORITA_STRICT": "true"},
			check: func(t *testing.T, s settings) {
				if s.cfg.Strict {
					t.Error("Strict = true, want false")
				}
			},
		},
		{
			name:     "all remaining settings flags and output switches",
			args:     []string{"-n", "-c", "--json", "--explain", "-o", "/tmp/out.txt", "-f", "dd-MM-yyyy", "--week-one", "full", "--month-overflow", "overflow", "--count-inclusive", "today"},
			wantArgs: []string{"today"},
			check: func(t *testing.T, s settings) {
				if !s.noNewline || !s.copy || !s.json || !s.explain {
					t.Errorf("flags not set: %+v", s)
				}
				if s.output != "/tmp/out.txt" {
					t.Errorf("output = %q, want /tmp/out.txt", s.output)
				}
				if s.opts.Pattern != "dd-MM-yyyy" {
					t.Errorf("Pattern = %q, want dd-MM-yyyy", s.opts.Pattern)
				}
				if s.cfg.WeekOne != "full" {
					t.Errorf("WeekOne = %q, want full", s.cfg.WeekOne)
				}
				if !s.cfg.MonthOverflow {
					t.Error("MonthOverflow = false, want true")
				}
				if !s.cfg.CountInclusive {
					t.Error("CountInclusive = false, want true")
				}
			},
		},
		{
			name: "env vars for all config settings",
			env: map[string]string{
				"AHORITA_FORMAT":          "human",
				"AHORITA_TZ":              "UTC",
				"AHORITA_LANG":            "es",
				"AHORITA_STRICT":          "true",
				"AHORITA_WEEK_START":      "sunday",
				"AHORITA_WEEK_ONE":        "days",
				"AHORITA_MONTH_OVERFLOW":  "overflow",
				"AHORITA_COUNT_INCLUSIVE": "true",
			},
			check: func(t *testing.T, s settings) {
				if s.opts.Pattern != "human" || s.loc.String() != "UTC" || s.cfg.Lang != expr.ES ||
					!s.cfg.Strict || s.cfg.WeekStart != time.Sunday || s.cfg.WeekOne != "days" ||
					!s.cfg.MonthOverflow || !s.cfg.CountInclusive {
					t.Errorf("unexpected env settings: %+v", s)
				}
			},
		},
		{
			name:    "invalid week_one",
			args:    []string{"--week-one", "bad"},
			wantErr: "week_one",
		},
		{
			name:    "invalid month_overflow",
			args:    []string{"--month-overflow", "bad"},
			wantErr: "month_overflow",
		},
		{
			name:    "invalid lang",
			args:    []string{"--lang", "fr"},
			wantErr: "lang",
		},
		{
			name:    "invalid count_inclusive env",
			env:     map[string]string{"AHORITA_COUNT_INCLUSIVE": "yes"},
			wantErr: "count_inclusive",
		},
		{
			name: "country us flag",
			args: []string{"--country", "us"},
			check: func(t *testing.T, s settings) {
				if s.cfg.Country != "US" {
					t.Errorf("Country = %q, want US", s.cfg.Country)
				}
			},
		},
		{
			name: "country none flag",
			args: []string{"--country", "none"},
			check: func(t *testing.T, s settings) {
				if s.cfg.Country != "" {
					t.Errorf("Country = %q, want empty", s.cfg.Country)
				}
			},
		},
		{
			name:    "invalid country flag",
			args:    []string{"--country", "xx"},
			wantErr: "country",
		},
		{
			name: "AHORITA_COUNTRY env",
			env:  map[string]string{"AHORITA_COUNTRY": "es"},
			check: func(t *testing.T, s settings) {
				if s.cfg.Country != "ES" {
					t.Errorf("Country = %q, want ES", s.cfg.Country)
				}
			},
		},
		{
			name: "weekend sunday flag",
			args: []string{"--weekend", "sunday"},
			check: func(t *testing.T, s settings) {
				want := []time.Weekday{time.Sunday}
				if !reflect.DeepEqual(s.cfg.Weekend, want) {
					t.Errorf("Weekend = %v, want %v", s.cfg.Weekend, want)
				}
			},
		},
		{
			name: "weekend spanish deduplicated",
			args: []string{"--weekend", "viernes,sabado,viernes"},
			check: func(t *testing.T, s settings) {
				want := []time.Weekday{time.Friday, time.Saturday}
				if !reflect.DeepEqual(s.cfg.Weekend, want) {
					t.Errorf("Weekend = %v, want %v", s.cfg.Weekend, want)
				}
			},
		},
		{
			name: "weekend none flag",
			args: []string{"--weekend", "none"},
			check: func(t *testing.T, s settings) {
				if len(s.cfg.Weekend) != 0 {
					t.Errorf("Weekend = %v, want empty", s.cfg.Weekend)
				}
			},
		},
		{
			name:    "invalid weekend flag",
			args:    []string{"--weekend", "funday"},
			wantErr: "weekend",
		},
		{
			name: "AHORITA_WEEKEND env",
			env:  map[string]string{"AHORITA_WEEKEND": "friday,saturday"},
			check: func(t *testing.T, s settings) {
				want := []time.Weekday{time.Friday, time.Saturday}
				if !reflect.DeepEqual(s.cfg.Weekend, want) {
					t.Errorf("Weekend = %v, want %v", s.cfg.Weekend, want)
				}
			},
		},
		{
			name: "repeatable calendar flags override env",
			args: []string{"--calendar", "/tmp/a.txt", "--calendar", "/tmp/b.txt"},
			env:  map[string]string{"AHORITA_CALENDAR": "/tmp/env.txt"},
			check: func(t *testing.T, s settings) {
				want := []string{"/tmp/a.txt", "/tmp/b.txt"}
				if !reflect.DeepEqual(s.calendars, want) {
					t.Errorf("calendars = %v, want %v", s.calendars, want)
				}
			},
		},
		{
			name: "AHORITA_CALENDAR comma separated",
			env:  map[string]string{"AHORITA_CALENDAR": "/tmp/one.txt, /tmp/two.txt"},
			check: func(t *testing.T, s settings) {
				want := []string{"/tmp/one.txt", "/tmp/two.txt"}
				if !reflect.DeepEqual(s.calendars, want) {
					t.Errorf("calendars = %v, want %v", s.calendars, want)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := noEnv
			if tc.env != nil {
				env = envMap(tc.env)
			}
			s, rest, err := load(tc.args, env, noFiles)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("load(%v) succeeded, want error containing %q", tc.args, tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("err = %q, want substring %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("load(%v) unexpected error: %v", tc.args, err)
			}
			if len(tc.wantArgs) == 0 && len(rest) == 0 {
				// ok
			} else if !reflect.DeepEqual(rest, tc.wantArgs) {
				t.Errorf("rest = %v, want %v", rest, tc.wantArgs)
			}
			if tc.check != nil {
				tc.check(t, s)
			}
		})
	}
}

func TestConfigFile(t *testing.T) {
	xdgPath := filepath.Join("/xdg", "ahorita", "config.toml")
	homePath := filepath.Join("/home/user", ".config", "ahorita", "config.toml")

	tests := []struct {
		name    string
		args    []string
		env     map[string]string
		files   map[string]string
		wantErr []string
		check   func(t *testing.T, s settings)
	}{
		{
			name: "file setting lang and week_start",
			env:  map[string]string{"XDG_CONFIG_HOME": "/xdg"},
			files: map[string]string{
				xdgPath: "# comment\nlang = \"es\" # spanish\nweek_start = sunday\n",
			},
			check: func(t *testing.T, s settings) {
				if s.cfg.Lang != expr.ES {
					t.Errorf("Lang = %q, want es", s.cfg.Lang)
				}
				if s.cfg.WeekStart != time.Sunday {
					t.Errorf("WeekStart = %v, want Sunday", s.cfg.WeekStart)
				}
			},
		},
		{
			name: "env overrides config file",
			env:  map[string]string{"XDG_CONFIG_HOME": "/xdg", "AHORITA_LANG": "en"},
			files: map[string]string{
				xdgPath: "lang = \"es\"\nweek_start = sunday\n",
			},
			check: func(t *testing.T, s settings) {
				if s.cfg.Lang != expr.EN {
					t.Errorf("Lang = %q, want en", s.cfg.Lang)
				}
				if s.cfg.WeekStart != time.Sunday {
					t.Errorf("WeekStart = %v, want Sunday", s.cfg.WeekStart)
				}
			},
		},
		{
			name: "flag overrides env and config file",
			args: []string{"--lang", "es"},
			env:  map[string]string{"XDG_CONFIG_HOME": "/xdg", "AHORITA_LANG": "en"},
			files: map[string]string{
				xdgPath: "lang = \"es\"\nweek_start = sunday\n",
			},
			check: func(t *testing.T, s settings) {
				if s.cfg.Lang != expr.ES {
					t.Errorf("Lang = %q, want es", s.cfg.Lang)
				}
			},
		},
		{
			name:    "unknown key error mentions line 1",
			args:    []string{"--config-file", "/tmp/bad.toml"},
			files:   map[string]string{"/tmp/bad.toml": "nope = 1\n"},
			wantErr: []string{"/tmp/bad.toml", "1", "nope"},
		},
		{
			name:    "missing equals error mentions line 2",
			args:    []string{"--config-file", "/tmp/bad2.toml"},
			files:   map[string]string{"/tmp/bad2.toml": "# ok\nnot a kv line\n"},
			wantErr: []string{"/tmp/bad2.toml", "2"},
		},
		{
			name:    "explicit --config-file missing is an error",
			args:    []string{"--config-file", "/tmp/missing.toml"},
			wantErr: []string{"/tmp/missing.toml"},
		},
		{
			name:    "explicit AHORITA_CONFIG missing is an error",
			env:     map[string]string{"AHORITA_CONFIG": "/tmp/env-missing.toml"},
			wantErr: []string{"/tmp/env-missing.toml"},
		},
		{
			name: "missing file at default path is ok",
			env:  map[string]string{"HOME": "/home/user"},
			check: func(t *testing.T, s settings) {
				if !reflect.DeepEqual(s.cfg, expr.DefaultConfig()) {
					t.Errorf("cfg = %+v, want default", s.cfg)
				}
			},
		},
		{
			name: "home fallback and quoted hash value",
			args: []string{"-t", "date", "--config-file", "/tmp/custom.toml"},
			files: map[string]string{
				"/tmp/custom.toml": "format = \"yyyy # MM\"\n",
			},
			check: func(t *testing.T, s settings) {
				if s.opts.Type != "date" {
					t.Errorf("opts.Type = %q, want date", s.opts.Type)
				}
				if s.opts.Pattern != "yyyy # MM" {
					t.Errorf("opts.Pattern = %q, want %q", s.opts.Pattern, "yyyy # MM")
				}
			},
		},
		{
			name: "default HOME config path",
			env:  map[string]string{"HOME": "/home/user"},
			files: map[string]string{
				homePath: "tz = \"UTC\"\n",
			},
			check: func(t *testing.T, s settings) {
				if s.loc.String() != "UTC" {
					t.Errorf("loc = %q, want UTC", s.loc.String())
				}
			},
		},
		{
			name: "file setting country weekend and calendar overridden by flag",
			args: []string{"--calendar", "/cli.txt"},
			env:  map[string]string{"XDG_CONFIG_HOME": "/xdg"},
			files: map[string]string{
				xdgPath: "country = \"US\"\nweekend = \"sunday\"\ncalendar = \"/cfg1.txt, /cfg2.txt\"\n",
			},
			check: func(t *testing.T, s settings) {
				if s.cfg.Country != "US" {
					t.Errorf("Country = %q, want US", s.cfg.Country)
				}
				if !reflect.DeepEqual(s.cfg.Weekend, []time.Weekday{time.Sunday}) {
					t.Errorf("Weekend = %v, want [Sunday]", s.cfg.Weekend)
				}
				if !reflect.DeepEqual(s.calendars, []string{"/cli.txt"}) {
					t.Errorf("calendars = %v, want [/cli.txt]", s.calendars)
				}
			},
		},
		{
			name: "file setting calendar without override",
			env:  map[string]string{"XDG_CONFIG_HOME": "/xdg"},
			files: map[string]string{
				xdgPath: "calendar = \"/cfg1.txt, /cfg2.txt\"\n",
			},
			check: func(t *testing.T, s settings) {
				want := []string{"/cfg1.txt", "/cfg2.txt"}
				if !reflect.DeepEqual(s.calendars, want) {
					t.Errorf("calendars = %v, want %v", s.calendars, want)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, _, err := load(tc.args, envMap(tc.env), filesMap(tc.files))
			if len(tc.wantErr) > 0 {
				if err == nil {
					t.Fatalf("load(%v) succeeded, want error", tc.args)
				}
				for _, sub := range tc.wantErr {
					if !strings.Contains(err.Error(), sub) {
						t.Errorf("err = %q, want substring %q", err.Error(), sub)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("load(%v) unexpected error: %v", tc.args, err)
			}
			if tc.check != nil {
				tc.check(t, s)
			}
		})
	}
}
