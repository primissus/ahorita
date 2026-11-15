package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/primissus/ahorita/expr"
)

func refNow(t *testing.T) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("America/Mexico_City")
	if err != nil {
		t.Fatalf("load reference location: %v", err)
	}
	return time.Date(2026, time.November, 14, 14, 30, 0, 0, loc)
}

func testEnv(t *testing.T, terminal bool) env {
	t.Helper()
	fixedNow := refNow(t)
	return env{
		getenv:          envMap(nil),
		now:             func() time.Time { return fixedNow },
		readFile:        filesMap(nil),
		stdinIsTerminal: terminal,
		writeFile:       os.WriteFile,
	}
}

func withTZ(args ...string) []string {
	return append([]string{"--tz", "America/Mexico_City"}, args...)
}

func TestRun(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "out.txt")

	tests := []struct {
		name       string
		args       []string
		stdin      string
		terminal   bool
		wantCode   int
		wantStdout string
		checkOut   func(t *testing.T, stdout, stderr string)
	}{
		{
			name:       "bare terminal prints now",
			args:       withTZ(),
			terminal:   true,
			wantCode:   0,
			wantStdout: "2026-11-14T14:30:00-06:00\n",
		},
		{
			name:       "now + 2days",
			args:       withTZ("now", "+", "2days"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "2026-11-16T14:30:00-06:00\n",
		},
		{
			name:       "-t date now",
			args:       withTZ("-t", "date", "now"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "2026-11-14\n",
		},
		{
			name:       "-t datetime now",
			args:       withTZ("-t", "datetime", "now"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "2026-11-14T14:30:00-06:00\n",
		},
		{
			name:       "-t time now",
			args:       withTZ("-t", "time", "now"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "14:30\n",
		},
		{
			name:       "-t range now",
			args:       withTZ("-t", "range", "now"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "0s\n",
		},
		{
			name:       "-t relative now",
			args:       withTZ("-t", "relative", "now"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "now\n",
		},
		{
			name:       "-t relative today + 2d",
			args:       withTZ("-t", "relative", "today + 2d"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "in 2 days\n",
		},
		{
			name:       "-f dd-MM-yyyy 2026-06-02",
			args:       withTZ("-f", "dd-MM-yyyy", "2026-06-02"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "02-06-2026\n",
		},
		{
			name:       "-f d/M/yy 2026-06-02",
			args:       withTZ("-f", "d/M/yy", "2026-06-02"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "2/6/26\n",
		},
		{
			name:       "-n today",
			args:       withTZ("-n", "today"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "2026-11-14",
		},
		{
			name:       "today - 1w, monday",
			args:       withTZ("today", "-", "1w,", "monday"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "2026-11-02\n",
		},
		{
			name:       "--explain today - 1w, monday",
			args:       withTZ("--explain", "today", "-", "1w,", "monday"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "2026-11-02\n",
			checkOut: func(t *testing.T, stdout, stderr string) {
				lines := strings.Split(strings.TrimSuffix(stderr, "\n"), "\n")
				if len(lines) != 2 {
					t.Fatalf("stderr lines = %d (%q), want 2", len(lines), stderr)
				}
				if !strings.Contains(lines[0], "today - 1w") || !strings.Contains(lines[0], "2026-11-07") {
					t.Errorf("first explain line = %q, want today - 1w and 2026-11-07", lines[0])
				}
			},
		},
		{
			name:       "--strict end of day, today",
			args:       withTZ("--strict", "end", "of", "day,", "today"),
			terminal:   true,
			wantCode:   1,
			wantStdout: "",
			checkOut: func(t *testing.T, stdout, stderr string) {
				if !strings.Contains(stderr, "out of order") {
					t.Errorf("stderr = %q, want out of order", stderr)
				}
			},
		},
		{
			name:       "now + 2x caret alignment",
			args:       withTZ("now", "+", "2x"),
			terminal:   true,
			wantCode:   1,
			wantStdout: "",
			checkOut: func(t *testing.T, stdout, stderr string) {
				if !strings.Contains(stderr, "unknown unit") {
					t.Errorf("stderr = %q, want unknown unit", stderr)
				}
				lines := strings.Split(strings.TrimSuffix(stderr, "\n"), "\n")
				if len(lines) != 3 {
					t.Fatalf("stderr lines = %d (%q), want 3", len(lines), stderr)
				}
				if lines[2] != "        ^" {
					t.Errorf("caret line = %q, want 8 spaces before ^", lines[2])
				}
			},
		},
		{
			name:       "now + includes expected line",
			args:       withTZ("now", "+"),
			terminal:   true,
			wantCode:   1,
			wantStdout: "",
			checkOut: func(t *testing.T, stdout, stderr string) {
				if !strings.Contains(stderr, "  expected: a value") {
					t.Errorf("stderr = %q, want expected line", stderr)
				}
			},
		},
		{
			name:       "hace 2 díaz error",
			args:       withTZ("hace", "2", "díaz"),
			terminal:   true,
			wantCode:   1,
			wantStdout: "",
			checkOut: func(t *testing.T, stdout, stderr string) {
				lines := strings.Split(strings.TrimSuffix(stderr, "\n"), "\n")
				if len(lines) < 3 {
					t.Fatalf("stderr = %q, want caret output", stderr)
				}
				if lines[2] != "       ^" {
					t.Errorf("caret line = %q, want 7 spaces before ^", lines[2])
				}
			},
		},
		{
			name:       "rune-counted caret after multi-byte í in 2 días + 2x",
			args:       withTZ("2", "días", "+", "2x"),
			terminal:   true,
			wantCode:   1,
			wantStdout: "",
			checkOut: func(t *testing.T, stdout, stderr string) {
				lines := strings.Split(strings.TrimSuffix(stderr, "\n"), "\n")
				if len(lines) != 3 {
					t.Fatalf("stderr lines = %d (%q), want 3", len(lines), stderr)
				}
				// "  2 días + 2x": 2 + 9 runes before "2x" = 11 spaces (byte offset would be 12)
				if lines[2] != "           ^" {
					t.Errorf("caret line = %q, want 11 spaces before ^ (rune count)", lines[2])
				}
			},
		},
		{
			name:     "--json now + 2d",
			args:     withTZ("--json", "now", "+", "2d"),
			terminal: true,
			wantCode: 0,
			checkOut: func(t *testing.T, stdout, stderr string) {
				if stderr != "" {
					t.Errorf("unexpected stderr: %q", stderr)
				}
				var res expr.Result
				if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &res); err != nil {
					t.Fatalf("invalid json %q: %v", stdout, err)
				}
				if res.Kind != "datetime" {
					t.Errorf("kind = %q, want datetime", res.Kind)
				}
			},
		},
		{
			name:     "--json now +",
			args:     withTZ("--json", "now", "+"),
			terminal: true,
			wantCode: 1,
			checkOut: func(t *testing.T, stdout, stderr string) {
				var res expr.Result
				if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &res); err != nil {
					t.Fatalf("invalid json %q: %v", stdout, err)
				}
				if res.Error == nil || res.Error.Pos != 5 {
					t.Errorf("error = %+v, want pos 5", res.Error)
				}
			},
		},
		{
			name:       "--json --explain sends steps inside JSON only",
			args:       withTZ("--json", "--explain", "today", "-", "1w,", "monday"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "",
			checkOut: func(t *testing.T, stdout, stderr string) {
				if stderr != "" {
					t.Errorf("unexpected stderr in --json --explain: %q", stderr)
				}
				var res expr.Result
				if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &res); err != nil {
					t.Fatalf("invalid json %q: %v", stdout, err)
				}
				if len(res.Explain) != 2 {
					t.Errorf("explain = %+v, want 2 steps", res.Explain)
				}
			},
		},
		{
			name:       "--nope bad flag",
			args:       withTZ("--nope"),
			terminal:   true,
			wantCode:   2,
			wantStdout: "",
			checkOut: func(t *testing.T, stdout, stderr string) {
				if !strings.Contains(stderr, "ahorita:") {
					t.Errorf("stderr = %q, want ahorita:", stderr)
				}
			},
		},
		{
			name:       "stdin lines when not a terminal",
			args:       withTZ(),
			stdin:      "today\n\n  \ntomorrow\n",
			terminal:   false,
			wantCode:   0,
			wantStdout: "2026-11-14\n2026-11-15\n",
		},
		{
			name:       "-o writes file and leaves stdout empty",
			args:       withTZ("-o", outPath, "today"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "",
			checkOut: func(t *testing.T, stdout, stderr string) {
				data, err := os.ReadFile(outPath)
				if err != nil {
					t.Fatalf("ReadFile(%q): %v", outPath, err)
				}
				if string(data) != "2026-11-14\n" {
					t.Errorf("file = %q, want 2026-11-14\\n", string(data))
				}
			},
		},
		{
			name:       "--lang es -t relative on wordless date",
			args:       withTZ("--lang", "es", "-t", "relative", "2026-11-16"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "en 2 días\n",
		},
		{
			name:       "Spanish input hoy + 2d -t relative",
			args:       withTZ("-t", "relative", "hoy", "+", "2d"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "en 2 días\n",
		},
		{
			name:       "--lang es -t relative today + 2d follows F.4 input language",
			args:       withTZ("--lang", "es", "-t", "relative", "today", "+", "2d"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "in 2 days\n",
		},
		{
			name:       "--lang es disambiguates slash date",
			args:       withTZ("--lang", "es", "02/06/2026"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "2026-06-02\n",
		},
		{
			name:       "double-dash stops flag parsing for leading minus",
			args:       withTZ("--", "-05:00"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "2026-11-14T14:30:00-05:00\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			e := testEnv(t, tc.terminal)
			code := run(tc.args, strings.NewReader(tc.stdin), &stdout, &stderr, e)
			if code != tc.wantCode {
				t.Errorf("exit = %d, want %d (stdout=%q stderr=%q)", code, tc.wantCode, stdout.String(), stderr.String())
			}
			if tc.wantStdout != "" && stdout.String() != tc.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout.String(), tc.wantStdout)
			}
			if tc.wantStdout == "" && tc.checkOut == nil && stdout.String() != "" {
				t.Errorf("stdout = %q, want empty", stdout.String())
			}
			if tc.checkOut != nil {
				tc.checkOut(t, stdout.String(), stderr.String())
			}
		})
	}
}

func TestClipboard(t *testing.T) {
	t.Run("copies output without trailing newline", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		var gotName string
		var gotArgs []string
		var gotInput string

		e := testEnv(t, true)
		e.goos = "darwin"
		e.lookPath = func(name string) (string, error) {
			if name == "pbcopy" {
				return "/usr/bin/pbcopy", nil
			}
			return "", exec.ErrNotFound
		}
		e.runCmd = func(name string, args []string, input string) error {
			gotName = name
			gotArgs = args
			gotInput = input
			return nil
		}

		code := run(withTZ("-c", "today"), nil, &stdout, &stderr, e)
		if code != 0 {
			t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
		}
		if stdout.String() != "2026-11-14\n" {
			t.Errorf("stdout = %q, want 2026-11-14\\n", stdout.String())
		}
		if gotName != "pbcopy" || len(gotArgs) != 0 {
			t.Errorf("cmd = %q %v, want pbcopy []", gotName, gotArgs)
		}
		if gotInput != "2026-11-14" {
			t.Errorf("clipboard input = %q, want 2026-11-14", gotInput)
		}
	})

	t.Run("linux falls back to xclip when wl-copy missing", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		var gotName string
		var gotArgs []string
		var gotInput string

		e := testEnv(t, true)
		e.goos = "linux"
		e.lookPath = func(name string) (string, error) {
			if name == "xclip" {
				return "/usr/bin/xclip", nil
			}
			return "", exec.ErrNotFound
		}
		e.runCmd = func(name string, args []string, input string) error {
			gotName = name
			gotArgs = args
			gotInput = input
			return nil
		}

		code := run(withTZ("-c", "today"), nil, &stdout, &stderr, e)
		if code != 0 {
			t.Fatalf("exit = %d, want 0", code)
		}
		if gotName != "xclip" || strings.Join(gotArgs, " ") != "-selection clipboard" {
			t.Errorf("cmd = %q %v, want xclip [-selection clipboard]", gotName, gotArgs)
		}
		if gotInput != "2026-11-14" {
			t.Errorf("clipboard input = %q, want 2026-11-14", gotInput)
		}
	})

	t.Run("no clipboard available prints output and returns 1", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		e := testEnv(t, true)
		e.goos = "linux"
		e.lookPath = func(string) (string, error) {
			return "", exec.ErrNotFound
		}

		code := run(withTZ("-c", "today"), nil, &stdout, &stderr, e)
		if code != 1 {
			t.Fatalf("exit = %d, want 1", code)
		}
		if stdout.String() != "2026-11-14\n" {
			t.Errorf("stdout = %q, want 2026-11-14\\n", stdout.String())
		}
		if !strings.Contains(stderr.String(), "ahorita: can't copy:") {
			t.Errorf("stderr = %q, want ahorita: can't copy:", stderr.String())
		}
	})

	t.Run("clipboard failure prints output and returns 1", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		e := testEnv(t, true)
		e.goos = "windows"
		e.lookPath = func(name string) (string, error) {
			if name == "clip.exe" {
				return "C:\\Windows\\System32\\clip.exe", nil
			}
			return "", exec.ErrNotFound
		}
		e.runCmd = func(string, []string, string) error {
			return errors.New("clipboard locked")
		}

		code := run(withTZ("-c", "today"), nil, &stdout, &stderr, e)
		if code != 1 {
			t.Fatalf("exit = %d, want 1", code)
		}
		if stdout.String() != "2026-11-14\n" {
			t.Errorf("stdout = %q, want 2026-11-14\\n", stdout.String())
		}
		if !strings.Contains(stderr.String(), "ahorita: can't copy: clipboard locked") {
			t.Errorf("stderr = %q, want clipboard error", stderr.String())
		}
	})

	t.Run("-o and -c together write file and copy", func(t *testing.T) {
		outPath := filepath.Join(t.TempDir(), "copied.txt")
		var stdout, stderr bytes.Buffer
		var gotInput string

		e := testEnv(t, true)
		e.goos = "darwin"
		e.lookPath = func(name string) (string, error) { return "/usr/bin/" + name, nil }
		e.runCmd = func(_ string, _ []string, input string) error {
			gotInput = input
			return nil
		}

		code := run(withTZ("-o", outPath, "-c", "today"), nil, &stdout, &stderr, e)
		if code != 0 {
			t.Fatalf("exit = %d, want 0 (stderr=%q)", code, stderr.String())
		}
		if stdout.String() != "" {
			t.Errorf("stdout = %q, want empty", stdout.String())
		}
		data, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if string(data) != "2026-11-14\n" {
			t.Errorf("file = %q, want 2026-11-14\\n", string(data))
		}
		if gotInput != "2026-11-14" {
			t.Errorf("clipboard input = %q, want 2026-11-14", gotInput)
		}
	})
}

func TestErrorMessages(t *testing.T) {
	disallowed := []string{
		"runtime error",
		"%!",
		"panic",
		"goroutine",
		"expr.Value",
		"expr.Error",
		"{Kind:",
	}

	cases := []struct {
		name     string
		args     []string
		wantExit int
		wantErr  string
	}{
		{
			name:     "now_plus",
			args:     withTZ("now +"),
			wantExit: 1,
			wantErr:  "ahorita: unexpected end of input\n  now +\n       ^\n  expected: a value\n",
		},
		{
			name:     "now_plus_2x",
			args:     withTZ("now + 2x"),
			wantExit: 1,
			wantErr:  "ahorita: unknown unit \"x\"\n  now + 2x\n        ^\n",
		},
		{
			name:     "today_plus_today",
			args:     withTZ("today + today"),
			wantExit: 1,
			wantErr:  "ahorita: cannot add a date to a date\n  today + today\n  ^\n",
		},
		{
			name:     "3_months_to_days",
			args:     withTZ("3 months to days"),
			wantExit: 1,
			wantErr:  "ahorita: a month has no fixed length; give a start: during 3 months, to days\n  3 months to days\n           ^\n",
		},
		{
			name:     "now_to_Atlantis",
			args:     withTZ("now to Atlantis"),
			wantExit: 1,
			wantErr:  "ahorita: unknown timezone or word \"Atlantis\"\n  now to Atlantis\n         ^\n",
		},
		{
			name:     "2026-13-45",
			args:     withTZ("2026-13-45"),
			wantExit: 1,
			wantErr:  "ahorita: invalid date \"2026-13-45\"\n  2026-13-45\n  ^\n",
		},
		{
			name:     "week_54",
			args:     withTZ("week 54"),
			wantExit: 1,
			wantErr:  "ahorita: week 54 starts after the period ends\n  week 54\n  ^\n",
		},
		{
			name:     "type_date_duration",
			args:     withTZ("-t", "date", "2h"),
			wantExit: 1,
			wantErr:  "ahorita: -t date doesn't apply to a duration\n",
		},
		{
			name:     "format_bad_token",
			args:     withTZ("-f", "dd-QQ-yyyy", "today"),
			wantExit: 1,
			wantErr:  "ahorita: unsupported pattern token \"QQ\" (letter \"Q\"); supported tokens: yyyy, yy, MMMM, MMM, MM, M, dd, d, EEEE, EEE, HH, H, hh, h, mm, m, ss, s, a, XXX, 'text'\n",
		},
		{
			name:     "parentheses",
			args:     withTZ("(today)"),
			wantExit: 1,
			wantErr:  "ahorita: parentheses aren't supported\n  (today)\n  ^\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out, errBuf bytes.Buffer
			code := run(tc.args, strings.NewReader(""), &out, &errBuf, testEnv(t, true))
			if code != tc.wantExit {
				t.Fatalf("exit code = %d, want %d; stderr=%q", code, tc.wantExit, errBuf.String())
			}
			if out.Len() != 0 {
				t.Errorf("stdout = %q, want empty", out.String())
			}
			got := errBuf.String()
			if got != tc.wantErr {
				t.Errorf("stderr =\n%q\nwant:\n%q", got, tc.wantErr)
			}
			for _, bad := range disallowed {
				if strings.Contains(got, bad) {
					t.Errorf("stderr leaked Go internal token %q in %q", bad, got)
				}
			}
		})
	}
}

func TestBusinessCLI(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		files    map[string]string
		wantExit int
		wantOut  string
		wantSub  string
		wantErr  string
	}{
		{
			name:    "country none skips MX holiday",
			args:    withTZ("--country", "none", "2026-11-14, next business day"),
			wantOut: "2026-11-16\n",
		},
		{
			name:    "country none add business days across Nov 16",
			args:    withTZ("--country", "none", "2026-11-13 + 3 business days"),
			wantOut: "2026-11-18\n",
		},
		{
			name:    "default MX skips Nov 16 Revolution Day",
			args:    withTZ("2026-11-13 + 3 business days"),
			wantOut: "2026-11-19\n",
		},
		{
			name: "calendar whole-day span with inline comment",
			args: withTZ("--calendar", "/trip.txt", "2026-11-14, next business day"),
			files: map[string]string{
				"/trip.txt": "# vacation\n\n2026-11-17 .. 2027-01-02 # trip\n",
			},
			wantOut: "2027-01-04\n",
		},
		{
			name: "calendar expression uses reference now",
			args: withTZ("--calendar", "/off.txt", "2026-11-19 + 1 business day"),
			files: map[string]string{
				"/off.txt": "# comment\n\nnext friday # 2026-11-20 from refNow 2026-11-14\n",
			},
			wantOut: "2026-11-23\n",
		},
		{
			name: "repeatable calendar flags combine days off",
			args: withTZ("--calendar", "/f1.txt", "--calendar", "/f2.txt", "2026-11-14, next business day"),
			files: map[string]string{
				"/f1.txt": "2026-11-17\n",
				"/f2.txt": "2026-11-18\n",
			},
			wantOut: "2026-11-19\n",
		},
		{
			name:     "calendar invalid value kind duration exits 2",
			args:     withTZ("--calendar", "/bad.txt", "today"),
			files:    map[string]string{"/bad.txt": "2h\n"},
			wantExit: 2,
			wantErr:  "/bad.txt: line 1:",
		},
		{
			name:     "calendar eval error on line 2 exits 2",
			args:     withTZ("--calendar", "/bad.txt", "today"),
			files:    map[string]string{"/bad.txt": "# header\n2026-13-45\n"},
			wantExit: 2,
			wantErr:  "/bad.txt: line 2:",
		},
		{
			name:     "missing calendar file exits 2",
			args:     withTZ("--calendar", "/missing.txt", "today"),
			wantExit: 2,
			wantErr:  "/missing.txt",
		},
		{
			name:    "weekend sunday makes saturday a business day",
			args:    withTZ("--country", "none", "--weekend", "sunday", "2026-11-13, next business day"),
			wantOut: "2026-11-14\n",
		},
		{
			name:    "weekend none counts every calendar day",
			args:    withTZ("--country", "none", "--weekend", "none", "2026-11-13 + 3 business days"),
			wantOut: "2026-11-16\n",
		},
		{
			name:     "invalid country exits 2",
			args:     withTZ("--country", "XX", "today"),
			wantExit: 2,
			wantErr:  "country",
		},
		{
			name:    "holidays MX 2026 English",
			args:    withTZ("holidays MX 2026"),
			wantSub: "2026-01-01  New Year's Day",
		},
		{
			name:    "feriados MX 2026 Spanish",
			args:    withTZ("feriados MX 2026"),
			wantSub: "2026-01-01  Año Nuevo",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := testEnv(t, true)
			if tc.files != nil {
				e.readFile = filesMap(tc.files)
			}
			var out, errBuf bytes.Buffer
			code := run(tc.args, strings.NewReader(""), &out, &errBuf, e)
			if code != tc.wantExit {
				t.Fatalf("exit code = %d, want %d; stdout=%q stderr=%q", code, tc.wantExit, out.String(), errBuf.String())
			}
			if tc.wantOut != "" && out.String() != tc.wantOut {
				t.Errorf("stdout = %q, want %q", out.String(), tc.wantOut)
			}
			if tc.wantSub != "" && !strings.Contains(out.String(), tc.wantSub) {
				t.Errorf("stdout = %q, want substring %q", out.String(), tc.wantSub)
			}
			if tc.wantErr != "" && !strings.Contains(errBuf.String(), tc.wantErr) {
				t.Errorf("stderr = %q, want substring %q", errBuf.String(), tc.wantErr)
			}
		})
	}
}
