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
			name:       "-F dd-MM-yyyy today",
			args:       withTZ("-F", "dd-MM-yyyy", "today"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "14-11-2026\n",
		},
		{
			name:       "--custom-format yyyy/MM/dd HH:mm now",
			args:       withTZ("--custom-format", "yyyy/MM/dd HH:mm", "now"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "2026/11/14 14:30\n",
		},
		{
			name:       "-z now",
			args:       withTZ("-z", "now"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "2026-11-14T14:30:00-06:00 America/Mexico_City\n",
		},
		{
			name:       "--show-tz -f human now",
			args:       withTZ("--show-tz", "-f", "human", "now"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "Sat, Nov 14, 2026, 2:30 PM America/Mexico_City\n",
		},
		{
			name:       "--show-timezone now to Asia/Tokyo",
			args:       withTZ("--show-timezone", "now", "to", "Asia/Tokyo"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "2026-11-15T05:30:00+09:00 Asia/Tokyo\n",
		},
		{
			name:     "-F help",
			args:     []string{"-F", "help"},
			terminal: true,
			wantCode: 2,
			checkOut: func(t *testing.T, stdout, stderr string) {
				if stdout != "" {
					t.Errorf("stdout = %q, want empty", stdout)
				}
				if !strings.HasPrefix(stderr, "Custom format patterns (-F / --custom-format):") || !strings.Contains(stderr, "dd-MM-yyyy") {
					t.Errorf("stderr = %q, want custom format help", stderr)
				}
			},
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
		{
			name:       "parenthesized business day to los_angeles human",
			args:       withTZ("-f", "human", "(now - 1 business day) to los_angeles"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "Fri, Nov 13, 2026, 12:30 PM\n",
		},
		{
			name:       "now to Los Angeles",
			args:       withTZ("now", "to", "Los", "Angeles"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "2026-11-14T12:30:00-08:00\n",
		},
		{
			name:       "now to LA",
			args:       withTZ("now", "to", "LA"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "2026-11-14T12:30:00-08:00\n",
		},
		{
			name:       "now to angeles",
			args:       withTZ("now", "to", "angeles"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "2026-11-14T12:30:00-08:00\n",
		},
		{
			name:       "now to los_angels",
			args:       withTZ("now", "to", "los_angels"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "2026-11-14T12:30:00-08:00\n",
		},
		{
			name:       "--tz los_angeles now",
			args:       []string{"--tz", "los_angeles", "now"},
			terminal:   true,
			wantCode:   0,
			wantStdout: "2026-11-14T12:30:00-08:00\n",
		},
		{
			name:       "--is-valid valid expression",
			args:       withTZ("--is-valid", "now", "+", "2d"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "true\n",
			checkOut: func(t *testing.T, stdout, stderr string) {
				if stderr != "" {
					t.Errorf("stderr = %q, want empty", stderr)
				}
			},
		},
		{
			name:       "--is-valid invalid expression",
			args:       withTZ("--is-valid", "now", "+", "2x"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "false\n",
			checkOut: func(t *testing.T, stdout, stderr string) {
				if stderr != "" {
					t.Errorf("stderr = %q, want empty", stderr)
				}
			},
		},
		{
			name:       "-n --is-valid invalid expression omits newline",
			args:       withTZ("-n", "--is-valid", "now", "+", "2x"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "false",
			checkOut: func(t *testing.T, stdout, stderr string) {
				if stderr != "" {
					t.Errorf("stderr = %q, want empty", stderr)
				}
			},
		},
		{
			name:       "--strict --is-valid ambiguous expression returns false",
			args:       withTZ("--strict", "--is-valid", "end", "of", "day,", "today"),
			terminal:   true,
			wantCode:   0,
			wantStdout: "false\n",
			checkOut: func(t *testing.T, stdout, stderr string) {
				if stderr != "" {
					t.Errorf("stderr = %q, want empty", stderr)
				}
			},
		},
		{
			name:       "--is-valid with stdin lines",
			args:       withTZ("--is-valid"),
			stdin:      "today\nnow + 2x\ntomorrow\n",
			terminal:   false,
			wantCode:   0,
			wantStdout: "true\nfalse\ntrue\n",
			checkOut: func(t *testing.T, stdout, stderr string) {
				if stderr != "" {
					t.Errorf("stderr = %q, want empty", stderr)
				}
			},
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
			args:     withTZ("(today"),
			wantExit: 1,
			wantErr:  "ahorita: unexpected end of input, expected \")\"\n  (today\n        ^\n  expected: )\n",
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
		{
			name:    "cron zone note prints to stderr",
			args:    withTZ("every monday 8pm, to UTC, to cron"),
			wantOut: "0 2 * * 2\n",
			wantErr: "weekday shifts from Monday to Tuesday in UTC",
		},
		{
			name:    "cron zone note in json",
			args:    withTZ("--json", "every monday 8pm, to UTC, to cron"),
			wantSub: `"note":"weekday shifts from Monday to Tuesday in UTC"`,
		},
		{
			name: "calendar ics skips days off and recurring holidays",
			args: withTZ("--country", "none", "--calendar", "/team.ics", "2026-11-13 + 2 business days"),
			files: map[string]string{
				"/team.ics": "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nDTSTART;VALUE=DATE:20261116\r\nEND:VEVENT\r\nBEGIN:VEVENT\r\nDTSTART;TZID=America/Mexico_City:20261117T090000\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n",
			},
			wantOut: "2026-11-18\n",
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

func TestCasesCLI(t *testing.T) {
	cases := []struct {
		id         string
		input      string
		flagType   string
		flagFormat string
		noHolidays bool
		want       string
	}{
		{id: "1", input: "today", want: "2026-11-14"},
		{id: "2", input: "now - 1d", want: "2026-11-13T14:30:00-06:00"},
		{id: "3", input: "2026-06-02 + 2d", want: "2026-06-04"},
		{id: "4", input: "2026-06-02 - 1h", want: "2026-06-01T23:00:00-06:00"},
		{id: "5", input: "yesterday", want: "2026-11-13"},
		{id: "6", input: "2d ago", want: "2026-11-12T14:30:00-06:00"},
		{id: "7", input: "last week", want: "2026-11-02 → 2026-11-08"},
		{id: "8", input: "today - 1w", want: "2026-11-07T14:30:00-06:00"},
		{id: "9", input: "closest monday", want: "2026-11-16"},
		{id: "10", input: "today - 1w, monday", want: "2026-11-02"},
		{id: "11", input: "past monday", want: "2026-11-09"},
		{id: "12", input: "end of day, today", want: "2026-11-14T23:59:59-06:00"},
		{id: "13", input: "end of week, this week", want: "2026-11-15T23:59:59-06:00"},
		{id: "14", input: "next month", want: "2026-12-01 → 2026-12-31"},
		{id: "15", input: "in 3 months", want: "2027-02-14T14:30:00-06:00"},
		{id: "16", input: "in Q3", want: "2026-07-01 → 2026-09-30"},
		{id: "17", input: "2nd week, h2", want: "2026-07-06 → 2026-07-12"},
		{id: "17b", input: "week 2", want: "2026-01-05 → 2026-01-11"},
		{id: "18", input: "on aug 20th", want: "2026-08-20"},
		{id: "19", input: "on sept, past year", want: "2025-09-01 → 2025-09-30"},
		{id: "20", input: "2h", want: "2h"},
		{id: "20b", input: "90m", want: "1h 30m"},
		{id: "21", input: "during 2 days", want: "2026-11-14 → 2026-11-15"},
		{id: "22", input: "during 6 weeks, starting next monday", want: "2026-11-16 → 2026-12-27"},
		{id: "23", input: "every monday", want: "2026-11-16\n2026-11-23\n2026-11-30\n2026-12-07\n2026-12-14"},
		{id: "24", input: "every day 2nd", want: "2026-12-02\n2027-01-02\n2027-02-02\n2027-03-02\n2027-04-02"},
		{id: "25", input: "every monday, to cron", want: "0 0 * * 1"},
		{id: "26", input: "2026-12-24 - 2026-11-14", want: "40d"},
		{id: "26b", input: "2026-11-14 - 2026-12-24", want: "-40d"},
		{id: "27", input: "how many days between 2026-11-16 and 2026-11-20", want: "4"},
		{id: "27a", input: "how many days between 2026-11-16 and 2026-11-20 inclusive", want: "5"},
		{id: "27b", input: "how many mondays between today and 2026-12-31", want: "7"},
		{id: "28", input: "2026-11-14 14:30 in Tokyo", want: "2026-11-14T14:30:00+09:00"},
		{id: "28b", input: "now in Tokyo", want: "2026-11-15T05:30:00+09:00"},
		{id: "28c", input: "2pm @ Tokyo", want: "2026-11-14T14:00:00+09:00"},
		{id: "29", input: "2026-11-14 14:30, America/New_York", want: "2026-11-14T14:30:00-05:00"},
		{id: "30", input: "now to Asia/Tokyo", want: "2026-11-15T05:30:00+09:00"},
		{id: "31", input: "now to UTC", want: "2026-11-14T20:30:00Z"},
		{id: "32", input: "2026-11-20 - 2026-11-16, to business days", want: "4"},
		{id: "32_no_holidays", input: "2026-11-20 - 2026-11-16, to business days", noHolidays: true, want: "4"},
		{id: "32a", input: "2026-11-20 - 2026-11-16, to business days inclusive", want: "4"},
		{id: "32a_no_holidays", input: "2026-11-20 - 2026-11-16, to business days inclusive", noHolidays: true, want: "5"},
		{id: "33", input: "2026-11-14, next business day", want: "2026-11-17"},
		{id: "33_no_holidays", input: "2026-11-14, next business day", noHolidays: true, want: "2026-11-16"},
		{id: "33b", input: "2026-11-17, next business day", want: "2026-11-18"},
		{id: "33b_no_holidays", input: "2026-11-17, next business day", noHolidays: true, want: "2026-11-18"},
		{id: "34", input: "2026-11-13 + 3 business days", want: "2026-11-19"},
		{id: "34_no_holidays", input: "2026-11-13 + 3 business days", noHolidays: true, want: "2026-11-18"},
		{id: "34b", input: "2026-11-18 - 2 business days", want: "2026-11-13"},
		{id: "34b_no_holidays", input: "2026-11-18 - 2 business days", noHolidays: true, want: "2026-11-16"},
		{id: "34c", input: "2026-11-14 + 1 business day", want: "2026-11-17"},
		{id: "34c_no_holidays", input: "2026-11-14 + 1 business day", noHolidays: true, want: "2026-11-16"},
		{id: "35", input: "now + 2days", want: "2026-11-16T14:30:00-06:00"},
		{id: "36", input: "now", flagType: "date", want: "2026-11-14"},
		{id: "37", input: "now", flagType: "datetime", want: "2026-11-14T14:30:00-06:00"},
		{id: "38", input: "now", flagType: "time", want: "14:30"},
		{id: "39", input: "now", flagType: "range", want: "0s"},
		{id: "40", input: "now", flagType: "relative", want: "now"},
		{id: "40b", input: "today + 2d", flagType: "relative", want: "in 2 days"},
		{id: "41", input: "2026-06-02", flagFormat: "dd-MM-yyyy", want: "02-06-2026"},
		{id: "42", input: "2026-06-02", flagFormat: "d/M/yy", want: "2/6/26"},
		{id: "43", input: "(now - 1 business day) to los_angeles", want: "2026-11-13T12:30:00-08:00"},
		{id: "43_human", input: "(now - 1 business day) to los_angeles", flagFormat: "human", want: "Fri, Nov 13, 2026, 12:30 PM"},
		{id: "44", input: "now to Los Angeles", want: "2026-11-14T12:30:00-08:00"},
		{id: "45", input: "now to LA", want: "2026-11-14T12:30:00-08:00"},
		{id: "46", input: "now to california", want: "2026-11-14T12:30:00-08:00"},
		{id: "47", input: "now to los_angels", want: "2026-11-14T12:30:00-08:00"},
		{id: "48", input: "2 * (1h + 30m)", want: "3h"},
		{id: "49", input: "today + 3 days", want: "2026-11-17T14:30:00-06:00"},
		{id: "50", input: "days to 2027-01-01", want: "48"},
		{id: "51", input: "days to 2027", want: "48"},
		{id: "52", input: "how many days to 2027", want: "48"},
		{id: "53", input: "days from 2026-11-16 to 2026-11-20", want: "4"},
		{id: "54", input: "EoD", want: "2026-11-14T23:59:59-06:00"},
		{id: "55", input: "SoD", want: "2026-11-14"},
		{id: "56", input: "EoM", want: "2026-11-30T23:59:59-06:00"},
		{id: "57", input: "SoM", want: "2026-11-01"},
		{id: "58", input: "EoY", want: "2026-12-31T23:59:59-06:00"},
		{id: "59", input: "end day", want: "2026-11-14T23:59:59-06:00"},
		{id: "60", input: "start day", want: "2026-11-14"},
		{id: "61", input: "star of day", want: "2026-11-14"},
		{id: "62", input: "end minute", want: "2026-11-14T14:30:59-06:00"},
		{id: "63", input: "yesterday 2pm - now, in hours", want: "-24.5"},
		{id: "64", input: "yesterday 2pm - now, in minutes", want: "-1470"},
		{id: "65", input: "yesterday 2pm - now, in seconds", want: "-88200"},
		{id: "66", input: "yesterday 2pm - now, in millis", want: "-88200000"},
		{id: "67", input: "yesterday 2pm - now, in milliseconds", want: "-88200000"},
		{id: "68", input: "yesterday 2pm - now, in days", want: "-1.02"},
		{id: "69", input: "yesterday 2pm - now, in weeks", want: "-0.15"},
		{id: "70", input: "180s in hours", want: "0.05"},
		{id: "71", input: "1500ms in seconds", want: "1.5"},
		{id: "72", input: "1y in halfs", want: "2"},
		{id: "73", input: "1y in quarters", want: "4"},
		{id: "74", input: "oncall week = from last friday to this friday", want: "2026-11-13 → 2026-11-13"},
		{id: "75", input: "oncall week = if today is friday, from today to next friday, else from last friday to this friday", want: "2026-11-13 → 2026-11-13"},
	}

	for _, tc := range cases {
		t.Run(tc.id+"_"+tc.input, func(t *testing.T) {
			cfgPath := filepath.Join(t.TempDir(), "config.toml")
			if err := os.WriteFile(cfgPath, nil, 0o644); err != nil {
				t.Fatal(err)
			}
			args := []string{"--config-file", cfgPath}
			if tc.noHolidays {
				args = append(args, "--country", "none")
			}
			if tc.flagType != "" {
				args = append(args, "-t", tc.flagType)
			}
			if tc.flagFormat != "" {
				args = append(args, "-f", tc.flagFormat)
			}
			args = append(args, tc.input)

			e := testEnv(t, true)
			e.readFile = os.ReadFile
			var stdout, stderr bytes.Buffer
			code := run(withTZ(args...), strings.NewReader(""), &stdout, &stderr, e)
			if code != 0 {
				t.Fatalf("case %s (%q) exit = %d, want 0 (stderr=%q)", tc.id, tc.input, code, stderr.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("case %s (%q) unexpected stderr: %q", tc.id, tc.input, stderr.String())
			}
			wantOut := tc.want + "\n"
			if stdout.String() != wantOut {
				t.Fatalf("case %s (%q) stdout = %q, want %q", tc.id, tc.input, stdout.String(), wantOut)
			}
		})
	}
}

func TestSaveConfig(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "nested", "config.toml")
	newEnv := func() env {
		e := testEnv(t, true)
		e.readFile = os.ReadFile
		return e
	}

	steps := []struct {
		name     string
		saveArgs []string
		wantOut  string
	}{
		{
			name:     "custom format",
			saveArgs: []string{"--config-file", cfgPath, "--tz", "America/Mexico_City", "-F", "dd-MM-yyyy", "--save-config"},
			wantOut:  "14-11-2026\n",
		},
		{
			name:     "rfc format",
			saveArgs: []string{"--config-file", cfgPath, "-f", "rfc", "--save-config"},
			wantOut:  "2026-11-14T00:00:00-06:00\n",
		},
		{
			name:     "iso format",
			saveArgs: []string{"--config-file", cfgPath, "-f", "iso", "--save-config"},
			wantOut:  "2026-11-14\n",
		},
	}

	for _, st := range steps {
		t.Run(st.name, func(t *testing.T) {
			var out, errBuf bytes.Buffer
			if code := run(st.saveArgs, strings.NewReader(""), &out, &errBuf, newEnv()); code != 0 {
				t.Fatalf("save exit = %d, stderr=%q", code, errBuf.String())
			}
			if out.Len() != 0 {
				t.Fatalf("save stdout = %q, want empty", out.String())
			}

			out.Reset()
			errBuf.Reset()
			if code := run([]string{"--config-file", cfgPath, "today"}, strings.NewReader(""), &out, &errBuf, newEnv()); code != 0 {
				t.Fatalf("eval exit = %d, stderr=%q", code, errBuf.String())
			}
			if out.String() != st.wantOut {
				t.Errorf("today stdout = %q, want %q", out.String(), st.wantOut)
			}
		})
	}

	t.Run("invalid pattern not saved", func(t *testing.T) {
		badPath := filepath.Join(t.TempDir(), "bad.toml")
		var out, errBuf bytes.Buffer
		code := run([]string{"--config-file", badPath, "-F", "dd-QQ-yyyy", "--save-config"}, strings.NewReader(""), &out, &errBuf, newEnv())
		if code != 2 {
			t.Fatalf("exit = %d, want 2 (stderr=%q)", code, errBuf.String())
		}
		if _, err := os.Stat(badPath); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected %s not to be created, stat err = %v", badPath, err)
		}
	})
}

func TestConfigCLI(t *testing.T) {
	runCfg := func(cfgPath string, extraEnv map[string]string, args ...string) (int, string, string) {
		e := testEnv(t, true)
		e.readFile = os.ReadFile
		if extraEnv != nil {
			e.getenv = envMap(extraEnv)
		}
		fullArgs := append([]string{"--config-file", cfgPath}, args...)
		var out, errBuf bytes.Buffer
		code := run(fullArgs, strings.NewReader(""), &out, &errBuf, e)
		return code, out.String(), errBuf.String()
	}

	t.Run("set get eval workflow", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "nested", "config.toml")

		code, out, errStr := runCfg(cfgPath, nil, "--config", "set", "tz=America/Mexico_City")
		if code != 0 || out != "" || errStr != "" {
			t.Fatalf("set tz: code=%d out=%q err=%q", code, out, errStr)
		}

		code, out, errStr = runCfg(cfgPath, nil, "--config", "set", "format=dd-MM-yyyy")
		if code != 0 || out != "" || errStr != "" {
			t.Fatalf("set format: code=%d out=%q err=%q", code, out, errStr)
		}

		code, out, errStr = runCfg(cfgPath, nil, "--config", "get", "format")
		if code != 0 || out != "dd-MM-yyyy\n" || errStr != "" {
			t.Fatalf("get format: code=%d out=%q err=%q", code, out, errStr)
		}

		// get reads config file rather than AHORITA_* env overrides
		code, out, errStr = runCfg(cfgPath, map[string]string{"AHORITA_FORMAT": "human"}, "--config", "get", "format")
		if code != 0 || out != "dd-MM-yyyy\n" || errStr != "" {
			t.Fatalf("get format with env: code=%d out=%q err=%q", code, out, errStr)
		}

		code, out, errStr = runCfg(cfgPath, nil, "today")
		if code != 0 || out != "14-11-2026\n" || errStr != "" {
			t.Fatalf("eval today: code=%d out=%q err=%q", code, out, errStr)
		}
	})

	t.Run("case-insensitive env-style and flag-style keys", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "config.toml")

		for _, args := range [][]string{
			{"--config", "set", "FORMAT=rfc"},
			{"--config", "set", "AHORITA_TZ=America/Mexico_City"},
			{"--config", "set", "day-week-start=sunday"},
		} {
			if code, out, errStr := runCfg(cfgPath, nil, args...); code != 0 || out != "" || errStr != "" {
				t.Fatalf("%v: code=%d out=%q err=%q", args, code, out, errStr)
			}
		}

		if code, out, errStr := runCfg(cfgPath, nil, "--config", "get", "TZ"); code != 0 || out != "America/Mexico_City\n" || errStr != "" {
			t.Fatalf("get TZ: code=%d out=%q err=%q", code, out, errStr)
		}
		if code, out, errStr := runCfg(cfgPath, nil, "--config", "get", "WEEK_START"); code != 0 || out != "sunday\n" || errStr != "" {
			t.Fatalf("get WEEK_START: code=%d out=%q err=%q", code, out, errStr)
		}
		if code, out, errStr := runCfg(cfgPath, nil, "--config", "get", "FORMAT"); code != 0 || out != "rfc\n" || errStr != "" {
			t.Fatalf("get FORMAT: code=%d out=%q err=%q", code, out, errStr)
		}
	})

	t.Run("space-separated set and unquoted multi-word values", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "config.toml")

		if code, out, errStr := runCfg(cfgPath, nil, "--config", "set", "format", "yyyy/MM/dd HH:mm"); code != 0 || errStr != "" {
			t.Fatalf("set format spaced: code=%d out=%q err=%q", code, out, errStr)
		}
		if code, out, _ := runCfg(cfgPath, nil, "--config", "get", "format"); code != 0 || out != "yyyy/MM/dd HH:mm\n" {
			t.Fatalf("get format: code=%d out=%q", code, out)
		}

		if code, out, errStr := runCfg(cfgPath, nil, "--config", "set", "tz", "Los", "Angeles"); code != 0 || errStr != "" {
			t.Fatalf("set tz Los Angeles: code=%d out=%q err=%q", code, out, errStr)
		}
		if code, out, _ := runCfg(cfgPath, nil, "--config", "get", "tz"); code != 0 || out != "America/Los_Angeles\n" {
			t.Fatalf("get tz: code=%d out=%q", code, out)
		}

		if code, out, errStr := runCfg(cfgPath, nil, "--config", "set", "format=yyyy/MM/dd", "HH:mm"); code != 0 || errStr != "" {
			t.Fatalf("set format= unquoted spaces: code=%d out=%q err=%q", code, out, errStr)
		}
		if code, out, _ := runCfg(cfgPath, nil, "--config", "get", "format"); code != 0 || out != "yyyy/MM/dd HH:mm\n" {
			t.Fatalf("get format after unquoted spaces: code=%d out=%q", code, out)
		}
	})

	t.Run("shorthand and quoted --config forms", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "config.toml")

		if code, _, errStr := runCfg(cfgPath, nil, "--config", "format=iso"); code != 0 || errStr != "" {
			t.Fatalf("shorthand set: code=%d err=%q", code, errStr)
		}
		if code, out, errStr := runCfg(cfgPath, nil, "--config", "format"); code != 0 || out != "iso\n" || errStr != "" {
			t.Fatalf("shorthand get: code=%d out=%q err=%q", code, out, errStr)
		}
		if code, _, errStr := runCfg(cfgPath, nil, "--config=set tz=UTC"); code != 0 || errStr != "" {
			t.Fatalf("quoted --config=set: code=%d err=%q", code, errStr)
		}
		if code, out, errStr := runCfg(cfgPath, nil, "--config=get tz"); code != 0 || out != "UTC\n" || errStr != "" {
			t.Fatalf("quoted --config=get: code=%d out=%q err=%q", code, out, errStr)
		}
	})

	t.Run("multiple settings and explicit default persistence", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "config.toml")

		if code, _, errStr := runCfg(cfgPath, nil, "--config", "set", "lang=es", "strict=true"); code != 0 || errStr != "" {
			t.Fatalf("multi set: code=%d err=%q", code, errStr)
		}
		if code, out, _ := runCfg(cfgPath, nil, "--config", "get", "lang"); code != 0 || out != "es\n" {
			t.Fatalf("get lang: code=%d out=%q", code, out)
		}
		if code, out, _ := runCfg(cfgPath, nil, "--config", "get", "strict"); code != 0 || out != "true\n" {
			t.Fatalf("get strict: code=%d out=%q", code, out)
		}

		// Setting lang=en when LANG=es_MX.UTF-8 must persist lang="en" in config.toml.
		if code, _, errStr := runCfg(cfgPath, map[string]string{"LANG": "es_MX.UTF-8"}, "--config", "set", "lang=en", "strict=false"); code != 0 || errStr != "" {
			t.Fatalf("explicit default set: code=%d err=%q", code, errStr)
		}
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", cfgPath, err)
		}
		if !strings.Contains(string(data), `lang = "en"`) || !strings.Contains(string(data), `strict = "false"`) {
			t.Errorf("config.toml =\n%s\nwant lang = \"en\" and strict = \"false\"", string(data))
		}
		if code, out, _ := runCfg(cfgPath, map[string]string{"LANG": "es_MX.UTF-8"}, "--config", "get", "lang"); code != 0 || out != "en\n" {
			t.Fatalf("get lang with LANG=es_MX: code=%d out=%q", code, out)
		}
	})

	t.Run("unset removes key and restores default", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "config.toml")

		if code, _, errStr := runCfg(cfgPath, nil, "--config", "set", "tz=America/Mexico_City", "format=dd-MM-yyyy"); code != 0 || errStr != "" {
			t.Fatalf("setup set: code=%d err=%q", code, errStr)
		}
		if code, _, errStr := runCfg(cfgPath, nil, "--config", "unset", "format"); code != 0 || errStr != "" {
			t.Fatalf("unset format: code=%d err=%q", code, errStr)
		}
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if strings.Contains(string(data), "format =") {
			t.Errorf("config.toml still contains format:\n%s", string(data))
		}
		if code, out, _ := runCfg(cfgPath, nil, "today"); code != 0 || out != "2026-11-14\n" {
			t.Fatalf("eval today after unset: code=%d out=%q", code, out)
		}

		// Empty set on format/type/calendar unsets it.
		if code, _, errStr := runCfg(cfgPath, nil, "--config", "set", "format=human"); code != 0 || errStr != "" {
			t.Fatalf("set format=human: code=%d err=%q", code, errStr)
		}
		if code, _, errStr := runCfg(cfgPath, nil, "--config", "set", "format="); code != 0 || errStr != "" {
			t.Fatalf("set format=: code=%d err=%q", code, errStr)
		}
		data, err = os.ReadFile(cfgPath)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if strings.Contains(string(data), "format =") {
			t.Errorf("config.toml still contains format after empty set:\n%s", string(data))
		}
	})

	t.Run("list get-empty and bare --config", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "config.toml")
		if code, _, errStr := runCfg(cfgPath, nil, "--config", "set", "format=human", "country=US"); code != 0 || errStr != "" {
			t.Fatalf("setup set: code=%d err=%q", code, errStr)
		}

		_, outList, errList := runCfg(cfgPath, nil, "--config", "list")
		codeGet, outGet, errGet := runCfg(cfgPath, nil, "--config", "get")
		codeBare, outBare, errBare := runCfg(cfgPath, nil, "--config")
		if codeGet != 0 || codeBare != 0 || errList != "" || errGet != "" || errBare != "" {
			t.Fatalf("list variants failed: codeGet=%d codeBare=%d errs=%q/%q/%q", codeGet, codeBare, errList, errGet, errBare)
		}
		if outList != outGet || outList != outBare {
			t.Fatalf("list outputs differ:\nlist:\n%s\nget:\n%s\nbare:\n%s", outList, outGet, outBare)
		}
		if !strings.Contains(outList, "format=human\n") || !strings.Contains(outList, "country=US\n") {
			t.Errorf("list output missing expected settings:\n%s", outList)
		}
	})

	t.Run("help and error cases preserve existing config", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "config.toml")
		if code, _, errStr := runCfg(cfgPath, nil, "--config", "set", "format=human"); code != 0 || errStr != "" {
			t.Fatalf("setup set: code=%d err=%q", code, errStr)
		}
		before, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}

		code, out, errStr := runCfg(cfgPath, nil, "--config", "help")
		if code != 2 || out != "" || !strings.Contains(errStr, "Config commands:") {
			t.Errorf("help: code=%d out=%q err=%q", code, out, errStr)
		}

		errCases := []struct {
			name    string
			args    []string
			wantErr string
		}{
			{name: "get unknown", args: []string{"--config", "get", "nope"}, wantErr: "unknown config key"},
			{name: "set unknown", args: []string{"--config", "set", "nope=1"}, wantErr: "unknown config key"},
			{name: "unset unknown", args: []string{"--config", "unset", "nope"}, wantErr: "unknown config key"},
			{name: "set invalid tz", args: []string{"--config", "set", "tz=Nowhere"}, wantErr: "invalid tz"},
			{name: "set invalid format", args: []string{"--config", "set", "format=dd-QQ-yyyy"}, wantErr: "unsupported pattern token"},
			{name: "get with value", args: []string{"--config", "get", "format=rfc"}, wantErr: "unexpected value in get"},
			{name: "set missing args", args: []string{"--config", "set"}, wantErr: "usage:"},
			{name: "set missing value", args: []string{"--config", "set", "format"}, wantErr: "usage:"},
			{name: "unset missing args", args: []string{"--config", "unset"}, wantErr: "usage:"},
			{name: "unknown command", args: []string{"--config", "bogus"}, wantErr: "unknown --config command"},
		}
		for _, tc := range errCases {
			t.Run(tc.name, func(t *testing.T) {
				c, o, e := runCfg(cfgPath, nil, tc.args...)
				if c != 2 || o != "" || !strings.Contains(e, tc.wantErr) {
					t.Errorf("%v: code=%d out=%q err=%q, want code 2 and %q", tc.args, c, o, e, tc.wantErr)
				}
			})
		}

		after, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if string(after) != string(before) {
			t.Errorf("config.toml changed after error cases:\nbefore:\n%s\nafter:\n%s", string(before), string(after))
		}
	})
}

func TestAliasesIfYAMLAndIsValidCLI(t *testing.T) {
	t.Run("alias definition persists to config.toml and resolves across invocations", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(cfgPath, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		runWithCfg := func(stdin string, args ...string) (int, string, string) {
			e := testEnv(t, stdin == "")
			e.readFile = os.ReadFile
			var stdout, stderr bytes.Buffer
			fullArgs := append([]string{"--config-file", cfgPath, "--tz", "America/Mexico_City"}, args...)
			code := run(fullArgs, strings.NewReader(stdin), &stdout, &stderr, e)
			return code, stdout.String(), stderr.String()
		}

		exprStr := "oncall week = if today is friday, from today to next friday, else from last friday to this friday"
		code, out, errStr := runWithCfg("", exprStr)
		if code != 0 || errStr != "" || out != "2026-11-13 → 2026-11-13\n" {
			t.Fatalf("define alias: code=%d out=%q err=%q", code, out, errStr)
		}

		data, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "[aliases]") || !strings.Contains(string(data), `"oncall week" =`) {
			t.Fatalf("config.toml missing [aliases] entry:\n%s", string(data))
		}

		code, out, errStr = runWithCfg("", "oncall week")
		if code != 0 || errStr != "" || out != "2026-11-13 → 2026-11-13\n" {
			t.Fatalf("use persisted alias: code=%d out=%q err=%q", code, out, errStr)
		}

		// --config get/list/unset for alias.<name>
		code, out, errStr = runWithCfg("", "--config", "get", "alias.oncall week")
		if code != 0 || errStr != "" || strings.TrimSpace(out) != "if today is friday, from today to next friday, else from last friday to this friday" {
			t.Fatalf("--config get alias: code=%d out=%q err=%q", code, out, errStr)
		}
		code, out, errStr = runWithCfg("", "--config", "list")
		if code != 0 || errStr != "" || !strings.Contains(out, "alias.oncall week=") {
			t.Fatalf("--config list alias: code=%d out=%q err=%q", code, out, errStr)
		}
		code, _, errStr = runWithCfg("", "--config", "set", "alias.standup=next monday 9am")
		if code != 0 || errStr != "" {
			t.Fatalf("--config set alias.standup: code=%d err=%q", code, errStr)
		}
		code, out, errStr = runWithCfg("", "standup")
		if code != 0 || errStr != "" || out != "2026-11-16T09:00:00-06:00\n" {
			t.Fatalf("eval standup: code=%d out=%q err=%q", code, out, errStr)
		}
		code, _, errStr = runWithCfg("", "--config", "unset", "alias.standup")
		if code != 0 || errStr != "" {
			t.Fatalf("--config unset alias.standup: code=%d err=%q", code, errStr)
		}
	})

	t.Run("stdin defines alias in memory and --is-valid does not write disk", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(cfgPath, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		e := testEnv(t, false)
		e.readFile = os.ReadFile

		var stdout, stderr bytes.Buffer
		stdin := "deploy day = next monday\ndeploy day + 1d\noncall week = if today is friday, 2026-13-45, else today\n"
		code := run([]string{"--config-file", cfgPath, "--tz", "America/Mexico_City", "--is-valid"}, strings.NewReader(stdin), &stdout, &stderr, e)
		if code != 0 || stderr.Len() != 0 {
			t.Fatalf("--is-valid stdin: code=%d stderr=%q", code, stderr.String())
		}
		if stdout.String() != "true\ntrue\nfalse\n" {
			t.Fatalf("--is-valid stdin stdout = %q, want %q", stdout.String(), "true\ntrue\nfalse\n")
		}
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatal(err)
		}
		if len(data) != 0 {
			t.Fatalf("--is-valid wrote to config.toml:\n%s", string(data))
		}
	})

	t.Run("--yaml and -f yaml / -f json output", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(cfgPath, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		runFmt := func(args ...string) (int, string, string) {
			e := testEnv(t, true)
			e.readFile = os.ReadFile
			var stdout, stderr bytes.Buffer
			full := append([]string{"--config-file", cfgPath, "--tz", "America/Mexico_City"}, args...)
			code := run(full, strings.NewReader(""), &stdout, &stderr, e)
			return code, stdout.String(), stderr.String()
		}

		for _, flagArgs := range [][]string{{"--yaml", "today"}, {"-f", "yaml", "today"}, {"-f", "yml", "today"}} {
			code, out, errStr := runFmt(flagArgs...)
			if code != 0 || errStr != "" {
				t.Fatalf("%v: code=%d err=%q", flagArgs, code, errStr)
			}
			if !strings.Contains(out, "input: \"today\"\n") || !strings.Contains(out, "kind: \"date\"\n") || !strings.Contains(out, "text: \"2026-11-14\"\n") {
				t.Fatalf("%v unexpected YAML output:\n%s", flagArgs, out)
			}
		}

		code, out, errStr := runFmt("-f", "json", "today")
		if code != 0 || errStr != "" {
			t.Fatalf("-f json: code=%d err=%q", code, errStr)
		}
		var dec map[string]any
		if err := json.Unmarshal([]byte(out), &dec); err != nil {
			t.Fatalf("-f json invalid JSON %q: %v", out, err)
		}
		if dec["kind"] != "date" || dec["text"] != "2026-11-14" {
			t.Fatalf("-f json decoded = %v", dec)
		}
	})
}
