package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/primissus/ahorita/expr"
)

type env struct {
	getenv          func(string) string
	now             func() time.Time
	readFile        func(string) ([]byte, error)
	stdinIsTerminal bool
	writeFile       func(string, []byte, os.FileMode) error
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

	return exitCode
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
