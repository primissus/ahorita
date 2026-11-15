package expr

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
	_ "time/tzdata"
)

func resultTestCtx(t *testing.T) Context {
	t.Helper()
	loc, err := time.LoadLocation("America/Mexico_City")
	if err != nil {
		t.Fatalf("load reference location: %v", err)
	}
	return Context{
		Now: time.Date(2026, time.November, 14, 14, 30, 0, 0, loc),
		Loc: loc,
		Cfg: DefaultConfig(),
	}
}

func TestRun(t *testing.T) {
	ctx := resultTestCtx(t)
	nowUnix := ctx.Now.AddDate(0, 0, 2).Unix()
	todayUnix := time.Date(2026, time.November, 14, 0, 0, 0, 0, ctx.Loc).Unix()

	tests := []struct {
		name        string
		input       string
		opts        Options
		explain     bool
		wantKind    string
		wantText    string
		wantISO     string
		wantUnix    *int64
		wantStart   string
		wantEnd     string
		wantExplain []ResultStep
		wantErrPos  *int
		wantErrMsg  string
		wantExpLen  int
	}{
		{
			name:     "now + 2d",
			input:    "now + 2d",
			wantKind: "datetime",
			wantText: "2026-11-16T14:30:00-06:00",
			wantISO:  "2026-11-16T14:30:00-06:00",
			wantUnix: &nowUnix,
		},
		{
			name:      "next month",
			input:     "next month",
			wantKind:  "span",
			wantText:  "2026-12-01 → 2026-12-31",
			wantISO:   "2026-12-01 → 2026-12-31",
			wantStart: "2026-12-01",
			wantEnd:   "2026-12-31",
		},
		{
			name:       "now + parse error",
			input:      "now +",
			wantErrPos: ptr(5),
			wantErrMsg: "unexpected end of input",
			wantExpLen: 1,
		},
		{
			name:     "today - 1w, monday with explain",
			input:    "today - 1w, monday",
			explain:  true,
			wantKind: "date",
			wantText: "2026-11-02",
			wantISO:  "2026-11-02",
			wantUnix: ptr(time.Date(2026, time.November, 2, 0, 0, 0, 0, ctx.Loc).Unix()),
			wantExplain: []ResultStep{
				{Stage: "today - 1w", Value: "2026-11-07"},
				{Stage: "monday", Value: "2026-11-02"},
			},
		},
		{
			name:     "today",
			input:    "today",
			wantKind: "date",
			wantText: "2026-11-14",
			wantISO:  "2026-11-14",
			wantUnix: &todayUnix,
		},
		{
			name:     "time",
			input:    "14:30",
			wantKind: "time",
			wantText: "14:30",
			wantISO:  "14:30",
		},
		{
			name:     "duration without ends",
			input:    "2h",
			wantKind: "duration",
			wantText: "2h",
			wantISO:  "2h",
		},
		{
			name:      "duration with date ends",
			input:     "2026-12-24 - today",
			wantKind:  "duration",
			wantText:  "40d",
			wantISO:   "40d",
			wantStart: "2026-11-14",
			wantEnd:   "2026-12-24",
		},
		{
			name:      "non-whole-day span",
			input:     "now .. now + 2h",
			wantKind:  "span",
			wantText:  "2026-11-14T14:30:00-06:00 → 2026-11-14T16:30:00-06:00",
			wantISO:   "2026-11-14T14:30:00-06:00 → 2026-11-14T16:30:00-06:00",
			wantStart: "2026-11-14T14:30:00-06:00",
			wantEnd:   "2026-11-14T16:30:00-06:00",
		},
		{
			name:      "duration with datetime ends",
			input:     "now + 2h - now",
			wantKind:  "duration",
			wantText:  "2h",
			wantISO:   "2h",
			wantStart: "2026-11-14T14:30:00-06:00",
			wantEnd:   "2026-11-14T16:30:00-06:00",
		},
		{
			name:     "number",
			input:    "how many days between 2026-11-16 and 2026-11-20",
			wantKind: "number",
			wantText: "4",
			wantISO:  "4",
		},
		{
			name:     "options format date on datetime",
			input:    "now + 2d",
			opts:     Options{Type: "date"},
			wantKind: "datetime",
			wantText: "2026-11-16",
			wantISO:  "2026-11-16T14:30:00-06:00",
			wantUnix: &nowUnix,
		},
		{
			name:       "format error gets Pos -1",
			input:      "2h",
			opts:       Options{Type: "date"},
			wantErrPos: ptr(-1),
			wantErrMsg: "doesn't apply to a duration",
		},
		{
			name:       "parse error at pos 0",
			input:      "+ today",
			wantErrPos: ptr(0),
			wantErrMsg: `unexpected "+"`,
			wantExpLen: 1,
		},
		{
			name:       "eval error at pos 0",
			input:      "wat",
			wantErrPos: ptr(0),
			wantErrMsg: `unknown timezone or word "wat"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Run(tc.input, ctx, tc.opts, tc.explain)
			if got.Input != tc.input {
				t.Fatalf("Input = %q, want %q", got.Input, tc.input)
			}

			if tc.wantErrPos != nil {
				if got.Error == nil {
					t.Fatalf("expected Error, got nil (result: %+v)", got)
				}
				if got.Error.Pos != *tc.wantErrPos {
					t.Errorf("Error.Pos = %d, want %d", got.Error.Pos, *tc.wantErrPos)
				}
				if tc.wantErrMsg != "" && !strings.Contains(got.Error.Msg, tc.wantErrMsg) {
					t.Errorf("Error.Msg = %q, want substring %q", got.Error.Msg, tc.wantErrMsg)
				}
				if tc.wantExpLen > 0 && len(got.Error.Expected) < tc.wantExpLen {
					t.Errorf("Error.Expected = %v, want at least %d item(s)", got.Error.Expected, tc.wantExpLen)
				}
				return
			}

			if got.Error != nil {
				t.Fatalf("unexpected Error: %+v", got.Error)
			}
			if got.Kind != tc.wantKind {
				t.Errorf("Kind = %q, want %q", got.Kind, tc.wantKind)
			}
			if got.Text != tc.wantText {
				t.Errorf("Text = %q, want %q", got.Text, tc.wantText)
			}
			if got.ISO != tc.wantISO {
				t.Errorf("ISO = %q, want %q", got.ISO, tc.wantISO)
			}
			if tc.wantUnix == nil {
				if got.Unix != nil {
					t.Errorf("Unix = %d, want nil", *got.Unix)
				}
			} else {
				if got.Unix == nil {
					t.Errorf("Unix = nil, want %d", *tc.wantUnix)
				} else if *got.Unix != *tc.wantUnix {
					t.Errorf("Unix = %d, want %d", *got.Unix, *tc.wantUnix)
				}
			}
			if got.Start != tc.wantStart {
				t.Errorf("Start = %q, want %q", got.Start, tc.wantStart)
			}
			if got.End != tc.wantEnd {
				t.Errorf("End = %q, want %q", got.End, tc.wantEnd)
			}
			if len(got.Explain) != len(tc.wantExplain) {
				t.Fatalf("Explain = %+v, want %+v", got.Explain, tc.wantExplain)
			}
			for i := range tc.wantExplain {
				if got.Explain[i] != tc.wantExplain[i] {
					t.Errorf("Explain[%d] = %+v, want %+v", i, got.Explain[i], tc.wantExplain[i])
				}
			}
		})
	}
}

func TestRunJSONAndNonExprError(t *testing.T) {
	ctx := resultTestCtx(t)

	// Non-*Error conversion
	errRes := toExprError(errors.New("plain failure"))
	if errRes.Pos != -1 || errRes.Msg != "plain failure" {
		t.Fatalf("toExprError = %+v, want Pos:-1 Msg:plain failure", errRes)
	}

	// JSON tags check including pos: 0 (must not be omitted)
	r0 := Run("+ today", ctx, Options{}, false)
	b0, err := json.Marshal(r0)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if !strings.Contains(string(b0), `"pos":0`) {
		t.Errorf("JSON missing pos:0: %s", b0)
	}
	if !strings.Contains(string(b0), `"message":`) || !strings.Contains(string(b0), `"expected":`) {
		t.Errorf("JSON missing message/expected tags: %s", b0)
	}

	// JSON for explain and unix
	r1 := Run("today - 1w, monday", ctx, Options{}, true)
	b1, err := json.Marshal(r1)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(b1, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if decoded["kind"] != "date" || decoded["text"] != "2026-11-02" {
		t.Errorf("unexpected JSON payload: %s", b1)
	}
	if _, ok := decoded["explain"]; !ok {
		t.Errorf("JSON missing explain: %s", b1)
	}
	if _, ok := decoded["unix"]; !ok {
		t.Errorf("JSON missing unix: %s", b1)
	}
}

func ptr[T any](v T) *T {
	return &v
}
