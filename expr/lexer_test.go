package expr

import "testing"

func TestLex(t *testing.T) {
	type wantTok struct {
		kind TokenKind
		text string
		num  string
		unit string
	}
	tok := func(kind TokenKind, text string) wantTok {
		return wantTok{kind: kind, text: text}
	}
	qty := func(num, unit string) wantTok {
		return wantTok{kind: Quantity, text: num + unit, num: num, unit: unit}
	}

	tests := []struct {
		name  string
		input string
		want  []wantTok
	}{
		{
			name:  "empty",
			input: "",
			want:  []wantTok{},
		},
		{
			name:  "date minus quantity spaced",
			input: "2026-06-02 - 1h",
			want:  []wantTok{tok(Date, "2026-06-02"), tok(Op, "-"), qty("1", "h")},
		},
		{
			name:  "date minus quantity tight",
			input: "2026-06-02-1h",
			want:  []wantTok{tok(Date, "2026-06-02"), tok(Op, "-"), qty("1", "h")},
		},
		{
			name:  "datetime offset",
			input: "2026-06-02T14:30:00-06:00",
			want:  []wantTok{tok(DateTime, "2026-06-02T14:30:00-06:00")},
		},
		{
			name:  "datetime zulu",
			input: "2026-06-02T14:30:00Z",
			want:  []wantTok{tok(DateTime, "2026-06-02T14:30:00Z")},
		},
		{
			name:  "datetime short",
			input: "2026-06-02T14:30",
			want:  []wantTok{tok(DateTime, "2026-06-02T14:30")},
		},
		{
			name:  "date and clock",
			input: "2026-11-14 14:30",
			want:  []wantTok{tok(Date, "2026-11-14"), tok(Clock, "14:30")},
		},
		{
			name:  "now plus 2d tight",
			input: "now+2d",
			want:  []wantTok{tok(Word, "now"), tok(Op, "+"), qty("2", "d")},
		},
		{
			name:  "combined quantity",
			input: "1h30m",
			want:  []wantTok{qty("1", "h"), qty("30", "m")},
		},
		{
			name:  "case kept M vs m",
			input: "2M and 2m",
			want:  []wantTok{qty("2", "M"), tok(Word, "and"), qty("2", "m")},
		},
		{
			name:  "decimal quantity",
			input: "1.5h",
			want:  []wantTok{qty("1.5", "h")},
		},
		{
			name:  "word unit quantity",
			input: "now + 2days",
			want:  []wantTok{tok(Word, "now"), tok(Op, "+"), qty("2", "days")},
		},
		{
			name:  "2pm",
			input: "2pm",
			want:  []wantTok{qty("2", "pm")},
		},
		{
			name:  "2:30pm",
			input: "2:30pm",
			want:  []wantTok{tok(Clock, "2:30"), tok(Word, "pm")},
		},
		{
			name:  "14:30",
			input: "14:30",
			want:  []wantTok{tok(Clock, "14:30")},
		},
		{
			name:  "spanish accented word",
			input: "hace 2 días",
			want:  []wantTok{tok(Word, "hace"), tok(Number, "2"), tok(Word, "días")},
		},
		{
			name:  "manana",
			input: "mañana",
			want:  []wantTok{tok(Word, "mañana")},
		},
		{
			name:  "comma pipe op",
			input: "today - 1w, monday",
			want:  []wantTok{tok(Word, "today"), tok(Op, "-"), qty("1", "w"), tok(Op, ","), tok(Word, "monday")},
		},
		{
			name:  "vertical bar pipe",
			input: "today | end of day",
			want:  []wantTok{tok(Word, "today"), tok(Op, "|"), tok(Word, "end"), tok(Word, "of"), tok(Word, "day")},
		},
		{
			name:  "unix timestamp",
			input: "@1700000000",
			want:  []wantTok{tok(Op, "@"), tok(Number, "1700000000")},
		},
		{
			name:  "at zone",
			input: "2pm @ Tokyo",
			want:  []wantTok{qty("2", "pm"), tok(Op, "@"), tok(Word, "Tokyo")},
		},
		{
			name:  "iana zone with underscore",
			input: "now to America/Mexico_City",
			want:  []wantTok{tok(Word, "now"), tok(Word, "to"), tok(Word, "America/Mexico_City")},
		},
		{
			name:  "iana zone with hyphen",
			input: "America/Port-au-Prince - 1h",
			want:  []wantTok{tok(Word, "America/Port-au-Prince"), tok(Op, "-"), qty("1", "h")},
		},
		{
			name:  "slash date",
			input: "02/06/2026",
			want:  []wantTok{tok(SlashDate, "02/06/2026")},
		},
		{
			name:  "ordinal english",
			input: "aug 20th",
			want:  []wantTok{tok(Word, "aug"), qty("20", "th")},
		},
		{
			name:  "ordinal and period H2",
			input: "2nd week, H2",
			want:  []wantTok{qty("2", "nd"), tok(Word, "week"), tok(Op, ","), tok(Word, "H2")},
		},
		{
			name:  "spanish ordinal indicator",
			input: "2º",
			want:  []wantTok{qty("2", "º")},
		},
		{
			name:  "multiply",
			input: "2h * 3",
			want:  []wantTok{qty("2", "h"), tok(Op, "*"), tok(Number, "3")},
		},
		{
			name:  "range dots",
			input: "2026-11-16..2026-11-20",
			want:  []wantTok{tok(Date, "2026-11-16"), tok(Op, ".."), tok(Date, "2026-11-20")},
		},
		{
			name:  "cron string",
			input: `cron "0 9 * * 1,3"`,
			want:  []wantTok{tok(Word, "cron"), tok(String, "0 9 * * 1,3")},
		},
		{
			name:  "unclosed string",
			input: `"open`,
			want:  []wantTok{tok(Illegal, `"open`)},
		},
		{
			name:  "illegal hash",
			input: "#",
			want:  []wantTok{tok(Illegal, "#")},
		},
		{
			name:  "number range dots not decimal",
			input: "1..5",
			want:  []wantTok{tok(Number, "1"), tok(Op, ".."), tok(Number, "5")},
		},
		{
			name:  "parens and single quotes",
			input: `('a')`,
			want:  []wantTok{tok(Op, "("), tok(String, "a"), tok(Op, ")")},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Lex(tc.input)
			if len(got) != len(tc.want)+1 {
				t.Fatalf("Lex(%q) returned %d tokens %+v, want %d + EOF", tc.input, len(got), got, len(tc.want))
			}
			for i, w := range tc.want {
				g := got[i]
				if g.Kind != w.kind || g.Text != w.text || g.Num != w.num || g.Unit != w.unit {
					t.Errorf("tok[%d] = (%v, %q, num=%q, unit=%q), want (%v, %q, num=%q, unit=%q)",
						i, g.Kind, g.Text, g.Num, g.Unit, w.kind, w.text, w.num, w.unit)
				}
			}
			eof := got[len(got)-1]
			if eof.Kind != EOF || eof.Pos != len(tc.input) {
				t.Errorf("last token = (%v, pos=%d), want (EOF, pos=%d)", eof.Kind, eof.Pos, len(tc.input))
			}
		})
	}
}

func TestLexPositions(t *testing.T) {
	input := "hace 2 días, mañana"
	toks := Lex(input)
	var found *Token
	for i := range toks {
		if toks[i].Text == "mañana" {
			found = &toks[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("token %q not found in %+v", "mañana", toks)
	}
	if found.Pos != 14 {
		t.Errorf("mañana Pos = %d, want 14", found.Pos)
	}
}
