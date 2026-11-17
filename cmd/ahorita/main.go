package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/primissus/ahorita/expr"
	internalsettings "github.com/primissus/ahorita/internal/settings"
)

type env struct {
	getenv          func(string) string
	now             func() time.Time
	readFile        func(string) ([]byte, error)
	stdinIsTerminal bool
	writeFile       func(string, []byte, os.FileMode) error
	mkdirAll        func(string, os.FileMode) error
	goos            string
	lookPath        func(string) (string, error)
	runCmd          func(name string, args []string, input string) error
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, env{
		getenv:          os.Getenv,
		now:             time.Now,
		readFile:        os.ReadFile,
		stdinIsTerminal: isTerminal(os.Stdin),
		writeFile:       os.WriteFile,
		mkdirAll:        os.MkdirAll,
		goos:            runtime.GOOS,
		lookPath:        exec.LookPath,
		runCmd:          runClipboardCmd,
	}))
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return true
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func runClipboardCmd(name string, args []string, input string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}
	return nil
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, e env) int {
	if e.now == nil {
		e.now = time.Now
	}
	if e.writeFile == nil {
		e.writeFile = os.WriteFile
	}
	if e.mkdirAll == nil {
		e.mkdirAll = os.MkdirAll
	}
	if e.lookPath == nil {
		e.lookPath = exec.LookPath
	}
	if e.runCmd == nil {
		e.runCmd = runClipboardCmd
	}

	s, rest, err := load(args, e.getenv, e.readFile)
	if err != nil {
		var he helpError
		if errors.As(err, &he) {
			fmt.Fprintln(stderr, he.text)
			return 2
		}
		fmt.Fprintf(stderr, "ahorita: %v\n", err)
		return 2
	}

	if s.configCmd != "" {
		return runConfig(s, rest, stdout, stderr, e)
	}

	inputs, err := readInputs(rest, stdin, e.stdinIsTerminal)
	if err != nil {
		fmt.Fprintf(stderr, "ahorita: %v\n", err)
		return 2
	}

	ctx := expr.Context{
		Now: e.now(),
		Loc: s.loc,
		Cfg: s.cfg,
	}
	if err := loadDaysOff(s.calendars, e.readFile, &ctx); err != nil {
		fmt.Fprintf(stderr, "ahorita: %v\n", err)
		return 2
	}

	if s.saveConfig {
		if err := saveConfigFile(s, ctx, e); err != nil {
			fmt.Fprintf(stderr, "ahorita: %v\n", err)
			return 2
		}
		fmt.Fprintf(stderr, "saved config to %s\n", s.configPath)
		if len(rest) == 0 && e.stdinIsTerminal {
			return 0
		}
	}

	var outLines []string
	exitCode := 0

	for _, input := range inputs {
		res := expr.Run(input, ctx, s.opts, s.explain)
		if s.json {
			data, err := json.Marshal(res)
			if err != nil {
				fmt.Fprintf(stderr, "ahorita: %v\n", err)
				return 1
			}
			outLines = append(outLines, string(data))
			if res.Error != nil {
				exitCode = 1
			}
			continue
		}

		if res.Error != nil {
			writeError(stderr, input, res.Error)
			exitCode = 1
			continue
		}

		if s.explain {
			for _, st := range res.Explain {
				fmt.Fprintf(stderr, "%s  →  %s\n", st.Stage, st.Value)
			}
		}
		if res.Note != "" {
			fmt.Fprintf(stderr, "ahorita: %s\n", res.Note)
		}
		outLines = append(outLines, res.Text)
	}

	if len(outLines) == 0 {
		return exitCode
	}

	if writeCode := writeOutput(outLines, s, stdout, stderr, e); writeCode != 0 {
		return writeCode
	}
	return exitCode
}

func writeOutput(outLines []string, s settings, stdout, stderr io.Writer, e env) int {
	body := strings.Join(outLines, "\n")
	payload := body
	if !s.noNewline {
		payload += "\n"
	}

	if s.output != "" {
		if err := e.writeFile(s.output, []byte(payload), 0o644); err != nil {
			fmt.Fprintf(stderr, "ahorita: %v\n", err)
			return 2
		}
	} else {
		fmt.Fprint(stdout, payload)
	}

	if s.copy {
		if err := copyToClipboard(body, e); err != nil {
			fmt.Fprintf(stderr, "ahorita: can't copy: %v\n", err)
			return 1
		}
	}
	return 0
}

func readInputs(rest []string, stdin io.Reader, stdinIsTerminal bool) ([]string, error) {
	if len(rest) > 0 {
		return []string{strings.Join(rest, " ")}, nil
	}
	if stdinIsTerminal {
		return []string{"now"}, nil
	}
	if stdin == nil {
		return nil, nil
	}

	var inputs []string
	scanner := bufio.NewScanner(stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		inputs = append(inputs, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading stdin: %w", err)
	}
	return inputs, nil
}

func writeError(w io.Writer, input string, e *expr.Error) {
	fmt.Fprintf(w, "ahorita: %s\n", e.Msg)
	if e.Pos >= 0 {
		pos := e.Pos
		if pos > len(input) {
			pos = len(input)
		}
		cols := utf8.RuneCountInString(input[:pos])
		fmt.Fprintf(w, "  %s\n", input)
		fmt.Fprintf(w, "  %s^\n", strings.Repeat(" ", cols))
	}
	if len(e.Expected) > 0 {
		fmt.Fprintf(w, "  expected: %s\n", strings.Join(e.Expected, ", "))
	}
}

func clipboardCandidates(goos string) [][]string {
	switch goos {
	case "darwin":
		return [][]string{{"pbcopy"}}
	case "windows":
		return [][]string{{"clip.exe"}}
	default:
		return [][]string{
			{"wl-copy"},
			{"xclip", "-selection", "clipboard"},
			{"xsel", "--clipboard", "--input"},
		}
	}
}

func copyToClipboard(text string, e env) error {
	goos := e.goos
	if goos == "" {
		goos = runtime.GOOS
	}
	for _, cmd := range clipboardCandidates(goos) {
		if _, err := e.lookPath(cmd[0]); err == nil {
			return e.runCmd(cmd[0], cmd[1:], text)
		}
	}
	return errors.New("no clipboard command found")
}

func validatePattern(vals internalsettings.Values, now time.Time) error {
	if vals.Opts.Pattern == "" {
		return nil
	}
	loc := vals.Loc
	if loc == nil {
		loc = time.Local
	}
	ctx := expr.Context{Now: now, Loc: loc, Cfg: vals.Cfg}
	sample := expr.Value{Kind: expr.KDateTime, T: now.In(loc), Lang: vals.Cfg.Lang}
	_, err := expr.Format(sample, ctx, expr.Options{Pattern: vals.Opts.Pattern})
	return err
}

func writeConfigFile(path, body string, e env) error {
	if path == "" {
		return errors.New("cannot determine config file path")
	}
	if err := e.mkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if err := e.writeFile(path, []byte(body), 0o644); err != nil {
		return fmt.Errorf("write config %q: %w", path, err)
	}
	return nil
}

func saveConfigFile(s settings, ctx expr.Context, e env) error {
	if err := validatePattern(s.vals, ctx.Now); err != nil {
		return err
	}
	body := formatConfigWithKeys(s.vals, s.fileKeys)
	return writeConfigFile(s.configPath, body, e)
}

func runConfig(s settings, rest []string, stdout, stderr io.Writer, e env) int {
	cmdStr := strings.TrimSpace(s.configCmd)
	lower := strings.ToLower(cmdStr)
	for _, prefix := range []string{"get ", "set ", "unset ", "reset "} {
		if strings.HasPrefix(lower, prefix) {
			cmd, argStr, _ := strings.Cut(cmdStr, " ")
			cmdStr = cmd
			argStr = strings.TrimSpace(argStr)
			if argStr != "" {
				var extra []string
				if strings.EqualFold(cmd, "set") && strings.Contains(argStr, "=") {
					fields := strings.Fields(argStr)
					allHaveEq := true
					for _, f := range fields {
						if !strings.Contains(f, "=") {
							allHaveEq = false
							break
						}
					}
					if allHaveEq {
						extra = fields
					} else {
						extra = []string{argStr}
					}
				} else {
					extra = strings.Fields(argStr)
				}
				rest = append(extra, rest...)
			}
			break
		}
	}

	cmd := strings.ToLower(cmdStr)
	switch cmd {
	case "help", "?":
		fmt.Fprintln(stderr, configHelp())
		return 2
	case "list":
		if len(rest) > 0 {
			fmt.Fprintln(stderr, "ahorita: usage: ahorita --config list")
			return 2
		}
		return listConfig(s, stdout, stderr, e)
	case "get":
		if len(rest) == 0 {
			return listConfig(s, stdout, stderr, e)
		}
		return getConfig(s, rest, stdout, stderr, e)
	case "set":
		return setConfig(s, rest, stderr, e)
	case "unset", "reset":
		return unsetConfig(s, rest, stderr, e)
	default:
		if strings.Contains(cmdStr, "=") {
			return setConfig(s, append([]string{cmdStr}, rest...), stderr, e)
		}
		if _, ok := internalsettings.Lookup(cmdStr); ok {
			if len(rest) == 0 {
				return getConfig(s, []string{cmdStr}, stdout, stderr, e)
			}
			return setConfig(s, []string{cmdStr, strings.Join(rest, " ")}, stderr, e)
		}
		fmt.Fprintf(stderr, "ahorita: unknown --config command or key %q (try --config help)\n", s.configCmd)
		return 2
	}
}

func listConfig(s settings, stdout, stderr io.Writer, e env) int {
	lines := make([]string, 0, len(internalsettings.Table))
	for i := range internalsettings.Table {
		def := &internalsettings.Table[i]
		lines = append(lines, def.Key+"="+def.Get(&s.fileVals))
	}
	return writeOutput(lines, s, stdout, stderr, e)
}

func getConfig(s settings, rest []string, stdout, stderr io.Writer, e env) int {
	if len(rest) > 1 {
		fmt.Fprintln(stderr, "ahorita: usage: ahorita --config get <key>")
		return 2
	}
	rawKey := rest[0]
	if eq := strings.IndexByte(rawKey, '='); eq >= 0 && rawKey[eq+1:] != "" {
		fmt.Fprintf(stderr, "ahorita: unexpected value in get %q; use \"ahorita --config set %s\" to set a value\n", rawKey, rawKey)
		return 2
	}
	key := strings.TrimSuffix(rawKey, "=")
	def, ok := internalsettings.Lookup(key)
	if !ok {
		fmt.Fprintf(stderr, "ahorita: unknown config key %q\n", key)
		return 2
	}
	return writeOutput([]string{def.Get(&s.fileVals)}, s, stdout, stderr, e)
}

type configKV struct {
	key string
	val string
}

func parseSetPairs(rest []string) ([]configKV, bool) {
	if len(rest) == 0 {
		return nil, false
	}
	if strings.Contains(rest[0], "=") {
		allHaveEq := true
		for _, item := range rest {
			if !strings.Contains(item, "=") {
				allHaveEq = false
				break
			}
		}
		if allHaveEq {
			pairs := make([]configKV, 0, len(rest))
			for _, item := range rest {
				k, v, _ := strings.Cut(item, "=")
				pairs = append(pairs, configKV{key: k, val: v})
			}
			return pairs, true
		}
		k, v, _ := strings.Cut(rest[0], "=")
		tail := strings.Join(rest[1:], " ")
		if v == "" {
			v = tail
		} else {
			v = v + " " + tail
		}
		return []configKV{{key: k, val: v}}, true
	}
	if len(rest) < 2 {
		return nil, false
	}
	return []configKV{{key: rest[0], val: strings.Join(rest[1:], " ")}}, true
}

func setConfig(s settings, rest []string, stderr io.Writer, e env) int {
	pairs, ok := parseSetPairs(rest)
	if !ok {
		fmt.Fprintln(stderr, "ahorita: usage: ahorita --config set <key>=<value>")
		return 2
	}
	if s.fileKeys == nil {
		s.fileKeys = make(map[string]bool)
	}
	defaults := defaultValuesForEnv(e.getenv)
	for _, p := range pairs {
		def, found := internalsettings.Lookup(p.key)
		if !found {
			fmt.Fprintf(stderr, "ahorita: unknown config key %q\n", p.key)
			return 2
		}
		if p.val == "" && (def.Key == "format" || def.Key == "type" || def.Key == "calendar") {
			internalsettings.Reset(&s.fileVals, def.Key, defaults)
			delete(s.fileKeys, def.Key)
			continue
		}
		if err := def.Apply(&s.fileVals, p.val); err != nil {
			fmt.Fprintf(stderr, "ahorita: %v\n", err)
			return 2
		}
		if def.Key == "format" && s.fileVals.Opts.Pattern != "" {
			if err := validatePattern(s.fileVals, e.now()); err != nil {
				fmt.Fprintf(stderr, "ahorita: %v\n", err)
				return 2
			}
		}
		s.fileKeys[def.Key] = true
	}
	body := formatConfigWithKeys(s.fileVals, s.fileKeys)
	if err := writeConfigFile(s.configPath, body, e); err != nil {
		fmt.Fprintf(stderr, "ahorita: %v\n", err)
		return 2
	}
	return 0
}

func unsetConfig(s settings, rest []string, stderr io.Writer, e env) int {
	if len(rest) == 0 {
		fmt.Fprintln(stderr, "ahorita: usage: ahorita --config unset <key>...")
		return 2
	}
	defaults := defaultValuesForEnv(e.getenv)
	for _, rawKey := range rest {
		key := strings.TrimSuffix(rawKey, "=")
		def, ok := internalsettings.Lookup(key)
		if !ok {
			fmt.Fprintf(stderr, "ahorita: unknown config key %q\n", key)
			return 2
		}
		internalsettings.Reset(&s.fileVals, def.Key, defaults)
		delete(s.fileKeys, def.Key)
	}
	body := formatConfigWithKeys(s.fileVals, s.fileKeys)
	if err := writeConfigFile(s.configPath, body, e); err != nil {
		fmt.Fprintf(stderr, "ahorita: %v\n", err)
		return 2
	}
	return 0
}
