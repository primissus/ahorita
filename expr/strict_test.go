package expr

import (
	"errors"
	"strings"
	"testing"
)

func TestStrict(t *testing.T) {
	ny := mustLoc(t, "America/New_York")
	ref := refCtx(t)

	strictOf := func(base Context) Context {
		base.Cfg.Strict = true
		return base
	}
	withLang := func(base Context, lang Lang) Context {
		base.Cfg.Lang = lang
		return base
	}

	cases := []struct {
		name    string
		input   string
		ctx     Context
		wantVal string
		wantPos int
		wantSub string
	}{
		{
			name:    "slash_date_valid_both_ways",
			input:   "02/06/2026",
			ctx:     withLang(ref, EN),
			wantVal: "2026-02-06",
			wantPos: 0,
			wantSub: "ambiguous slash date",
		},
		{
			name:    "out_of_order_stages",
			input:   "end of day, today",
			ctx:     ref,
			wantVal: "2026-11-14T23:59:59-06:00",
			wantPos: 12,
			wantSub: "stage",
		},
		{
			name:    "word_tie_fin_de_semana",
			input:   "fin de semana",
			ctx:     ref,
			wantVal: "2026-11-15T23:59:59-06:00",
			wantPos: 0,
			wantSub: "ambiguous word",
		},
		{
			name:    "word_tie_bare_ago_ES",
			input:   "ago",
			ctx:     withLang(ref, ES),
			wantVal: "2026-08-01 → 2026-08-31",
			wantPos: 0,
			wantSub: "ambiguous word",
		},
		{
			name:    "dst_gap",
			input:   "2026-03-08 02:30",
			ctx:     Context{Now: ref.Now, Loc: ny, Cfg: DefaultConfig()},
			wantVal: "2026-03-08T01:30:00-05:00",
			wantPos: 11,
			wantSub: "does not exist",
		},
		{
			name:    "dst_overlap",
			input:   "2026-11-01 01:30",
			ctx:     Context{Now: ref.Now, Loc: ny, Cfg: DefaultConfig()},
			wantVal: "2026-11-01T01:30:00-04:00",
			wantPos: 11,
			wantSub: "ambiguous",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, err := Eval(tc.input, tc.ctx)
			if err != nil {
				t.Fatalf("non-strict Eval(%q) unexpected error: %v", tc.input, err)
			}
			if got := v.String(); got != tc.wantVal {
				t.Fatalf("non-strict Eval(%q) = %q, want %q", tc.input, got, tc.wantVal)
			}

			sctx := strictOf(tc.ctx)
			sv, err := Eval(tc.input, sctx)
			if err == nil {
				t.Fatalf("strict Eval(%q) = %q, want *Error", tc.input, sv.String())
			}
			var e *Error
			if !errors.As(err, &e) {
				t.Fatalf("strict Eval(%q) error type = %T (%v), want *expr.Error", tc.input, err, err)
			}
			if e.Pos != tc.wantPos {
				t.Errorf("strict Eval(%q) Pos = %d, want %d (msg: %s)", tc.input, e.Pos, tc.wantPos, e.Msg)
			}
			if tc.wantSub != "" && !strings.Contains(strings.ToLower(e.Msg), strings.ToLower(tc.wantSub)) {
				t.Errorf("strict Eval(%q) Msg = %q, want substring %q", tc.input, e.Msg, tc.wantSub)
			}
		})
	}

	// Documented tie-break rule (plan 2 step 5, .context/known-issues.md):
	// "20 ago" next to a bare Number resolves unambiguously by position to August 20
	// even when Lang is EN and Strict is true; only bare "ago" carries a tie.
	t.Run("20_ago_resolved_by_position_under_strict", func(t *testing.T) {
		ctx := strictOf(withLang(ref, EN))
		v, err := Eval("20 ago", ctx)
		if err != nil {
			t.Fatalf("Eval(\"20 ago\", EN strict) error: %v", err)
		}
		if got := v.String(); got != "2026-08-20" {
			t.Fatalf("Eval(\"20 ago\", EN strict) = %q, want %q", got, "2026-08-20")
		}
	})
}
