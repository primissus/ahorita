package expr

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	bigTable := []struct {
		input     string
		canonical string // blank means equal to input
	}{
		{input: "today"},
		{input: "now - 1d"},
		{input: "2026-06-02 + 2d"},
		{input: "2026-06-02 - 1h"},
		{input: "2026-12-24 - 2026-11-14"},
		{input: "2h * 3"},
		{input: "now+2d", canonical: "now + 2d"},
		{input: "now + 2days", canonical: "now + 2d"},
		{input: "1h30m", canonical: "1h 30m"},
		{input: "2 days 4 hours", canonical: "2d 4h"},
		{input: "2h"},
		{input: "90m"},
		{input: "yesterday"},
		{input: "2d ago"},
		{input: "last week"},
		{input: "week last", canonical: "last week"},
		{input: "past monday", canonical: "last monday"},
		{input: "monday next", canonical: "next monday"},
		{input: "today - 1w, monday", canonical: "today - 1w | monday"},
		{input: "closest monday"},
		{input: "closest monday to 2026-12-01"},
		{input: "monday closest", canonical: "closest monday"},
		{input: "end of day, today", canonical: "end of day | today"},
		{input: "end of week, this week", canonical: "end of week | this week"},
		{input: "end of work week"},
		{input: "end of next month"},
		{input: "next month"},
		{input: "in 3 months", canonical: "in 3M"},
		{input: "in Q3", canonical: "q3"},
		{input: "2nd week, H2", canonical: "week 2 | h2"},
		{input: "week 2"},
		{input: "2nd week of march", canonical: "week 2 of march"},
		{input: "on aug 20th", canonical: "august 20"},
		{input: "20 aug", canonical: "august 20"},
		{input: "aug 20 2027", canonical: "august 20 2027"},
		{input: "next aug 20", canonical: "next august 20"},
		{input: "on sept, past year", canonical: "september | last year"},
		{input: "2026"},
		{input: "during 2 days", canonical: "during 2d"},
		{input: "during 6 weeks, starting next monday", canonical: "during 6w | starting next monday"},
		{input: "2026-11-16 .. 2026-11-20"},
		{input: "today .. today + 3d"},
		{input: "from today to tomorrow", canonical: "today .. tomorrow"},
		{input: "between today and tomorrow", canonical: "today .. tomorrow"},
		{input: "how many days between 2026-11-16 and 2026-11-20"},
		{input: "how many days between 2026-11-16 and 2026-11-20 inclusive"},
		{input: "how many mondays between today and 2026-12-31"},
		{input: "2026-11-14 14:30"},
		{input: "tomorrow 2pm", canonical: "tomorrow 14:00"},
		{input: "2:30pm", canonical: "14:30"},
		{input: "9 am", canonical: "09:00"},
		{input: "12am", canonical: "00:00"},
		{input: "@1700000000"},
		{input: "02/06/2026"},
		{input: "2pm @ Tokyo", canonical: "14:00 @ Tokyo"},
		{input: "2pm @ Tokyo + 1h", canonical: "14:00 @ Tokyo + 1h"},
		{input: "2026-11-14 14:30 in Tokyo"},
		{input: "2026-11-14 14:30, America/New_York", canonical: "2026-11-14 14:30 | America/New_York"},
		{input: "now to Asia/Tokyo", canonical: "now | to Asia/Tokyo"},
		{input: "now, to UTC", canonical: "now | to UTC"},
		{input: "now to New York", canonical: `now | to "New York"`},
		{input: `now to "Port of Spain"`, canonical: `now | to "Port of Spain"`},
		{input: "90m to hours", canonical: "90m | to hours"},
		{input: "2026-11-20 - 2026-11-16, to business days", canonical: "2026-11-20 - 2026-11-16 | to business days"},
		{input: "2026-11-14, next business day", canonical: "2026-11-14 | next business day"},
		{input: "2026-11-13 + 3 business days"},
		{input: "today | end of day | start of hour | to UTC"},

		// Additional coverage beyond the big table
		{input: "12pm", canonical: "12:00"},
		{input: "12:30am", canonical: "00:30"},
		{input: "12:30pm", canonical: "12:30"},
		{input: "2:30:05pm", canonical: "14:30:05"},
		{input: "9:30", canonical: "09:30"},
		{input: "2026-06-02-1h", canonical: "2026-06-02 - 1h"},
		{input: "2026-06-02T14:30:00-06:00"},
		{input: "20 of august of 2027", canonical: "august 20 2027"},
		{input: "20th aug", canonical: "august 20"},
		{input: "aug 2027", canonical: "august 2027"},
		{input: "q3 2027"},
		{input: "2nd week", canonical: "week 2"},
		{input: "second week", canonical: "week 2"},
		{input: "first week", canonical: "week 1"},
		{input: "third week", canonical: "week 3"},
		{input: "fourth week", canonical: "week 4"},
		{input: "fifth week", canonical: "week 5"},
		{input: "monday of next week"},
		{input: "q2 of 2027"},
		{input: "march of last year"},
		{input: "end of week 2 of march"},
		{input: "in 2026", canonical: "2026"},
		{input: "in march", canonical: "march"},
		{input: "in next month", canonical: "next month"},
		{input: "in 3 months + 1d", canonical: "in 3M + 1d"},
		{input: "@ Tokyo"},
		{input: "in Tokyo"},
		{input: "now in Tokyo"},
		{input: "America/New_York"},
		{input: "foo bar", canonical: `"foo bar"`},
		{input: "UTC"},
		{input: "now to gmt", canonical: "now | to UTC"},
		{input: "now to Z", canonical: "now | to UTC"},
		{input: "2026-11-20 - 2026-11-16, to business days inclusive", canonical: "2026-11-20 - 2026-11-16 | to business days inclusive"},
		{input: "2026-11-18 - 2 business days"},
		{input: "2026-11-14 + 1 business day"},
		{input: "how many days from 2026-11-16 to 2026-11-20", canonical: "how many days between 2026-11-16 and 2026-11-20"},
		{input: "how many halves between today and tomorrow"},
		{input: "3 * 2h"},
		{input: "2h * 2026"},
		{input: "3 business days ago"},
		{input: "aug 9 am", canonical: "august 09:00"},
		{input: "monday closest to 2026-12-01", canonical: "closest monday to 2026-12-01"},
		{input: "2026-13-45"}, // no date validation in parser
		// Plan 6 timezone offsets
		{input: "now to +09:00", canonical: "now | to +09:00"},
		{input: "now to -06:00", canonical: "now | to -06:00"},
		{input: "now to +9", canonical: "now | to +9"},
		{input: "now to -6", canonical: "now | to -6"},
		{input: "now to +0900", canonical: "now | to +0900"},
		{input: "now to UTC+2", canonical: "now | to UTC+2"},
		{input: "now to utc+9", canonical: "now | to UTC+9"},
		{input: "now to GMT-5", canonical: "now | to UTC-5"},
		{input: "now to UTC+09:00", canonical: "now | to UTC+09:00"},
		{input: "2pm @ -06:00", canonical: "14:00 @ -06:00"},
		{input: "2pm @ UTC+2", canonical: "14:00 @ UTC+2"},
		{input: "2pm @ UTC + 1h", canonical: "14:00 @ UTC + 1h"},
		{input: "2pm @ UTC + 1 day", canonical: "14:00 @ UTC + 1d"},
		{input: "2pm @ +09:00 + 1h", canonical: "14:00 @ +09:00 + 1h"},
		{input: "2026-11-14 14:30 in +09:00"},
		{input: "now | in -06:00"},
		{input: "now | @ +09:00"},
		{input: "2026-11-14 14:30, +09:00", canonical: "2026-11-14 14:30 | +09:00"},
	}

	for _, tc := range bigTable {
		want := tc.canonical
		if want == "" {
			want = tc.input
		}
		t.Run(tc.input, func(t *testing.T) {
			got, err := Parse(tc.input, EN)
			if err != nil {
				t.Fatalf("Parse(%q, EN) unexpected error: %v", tc.input, err)
			}
			if got.String() != want {
				t.Errorf("Parse(%q, EN).String() = %q, want %q", tc.input, got.String(), want)
			}

			// Round-trip: parsing canonical output produces equal canonical string and equal AST.
			rt, err := Parse(want, EN)
			if err != nil {
				t.Fatalf("round-trip Parse(%q, EN) unexpected error: %v", want, err)
			}
			if rt.String() != want {
				t.Errorf("round-trip Parse(%q, EN).String() = %q, want %q", want, rt.String(), want)
			}
			if !nodesEqualIgnorePos(got.Stages, rt.Stages) {
				t.Errorf("round-trip AST mismatch for %q:\ngot: %#v\nrt:  %#v", tc.input, got.Stages, rt.Stages)
			}
		})
	}
}

func nodesEqualIgnorePos(a, b []Node) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Op != b[i].Op || a[i].Text != b[i].Text {
			return false
		}
		if !nodesEqualIgnorePos(a[i].Args, b[i].Args) {
			return false
		}
	}
	return true
}

func TestParseASTStructure(t *testing.T) {
	t.Run("2pm @ Tokyo + 1h precedence", func(t *testing.T) {
		got, err := Parse("2pm @ Tokyo + 1h", EN)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []Node{{
			Op:  "+",
			Pos: 0,
			Args: []Node{
				{
					Op:  "interpret",
					Pos: 0,
					Args: []Node{
						{Op: "clock", Pos: 0, Text: "14:00"},
						{Op: "zone", Pos: 6, Text: "Tokyo"},
					},
				},
				{
					Op:  "dur",
					Pos: 14,
					Args: []Node{
						{Op: "hour", Pos: 14, Text: "1"},
					},
				},
			},
		}}
		if !reflect.DeepEqual(got.Stages, want) {
			t.Fatalf("got %#v\nwant %#v", got.Stages, want)
		}
	})

	t.Run("2h * 2026 is num while bare 2026 is year", func(t *testing.T) {
		mul, err := Parse("2h * 2026", EN)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mul.Stages[0].Op != "*" || mul.Stages[0].Args[1].Op != "num" || mul.Stages[0].Args[1].Text != "2026" {
			t.Errorf("2h * 2026 = %#v, want *(dur, num(2026))", mul.Stages[0])
		}

		yr, err := Parse("2026", EN)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if yr.Stages[0].Op != "year" || yr.Stages[0].Text != "2026" {
			t.Errorf("2026 = %#v, want year(2026)", yr.Stages[0])
		}
	})

	t.Run("Expr Lang and Ties", func(t *testing.T) {
		got, err := Parse("hoy + 2 days", EN)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Lang != ES {
			t.Errorf("Lang = %q, want %q", got.Lang, ES)
		}
		if len(got.Ties) != 0 {
			t.Errorf("Ties = %v, want empty", got.Ties)
		}

		tieExpr, err := Parse("fin de semana", ES)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(tieExpr.Ties, []int{0}) {
			t.Errorf("Ties = %v, want [0]", tieExpr.Ties)
		}
		if tieExpr.Lang != ES {
			t.Errorf("Lang = %q, want %q", tieExpr.Lang, ES)
		}

		noWords, err := Parse("2026-06-02 + 2d", EN)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if noWords.Lang != "" {
			t.Errorf("Lang = %q, want empty", noWords.Lang)
		}
	})

	t.Run("single-arg interpret and inzone stages", func(t *testing.T) {
		atStage, err := Parse("@ Tokyo", EN)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantAt := []Node{{
			Op:   "interpret",
			Pos:  0,
			Args: []Node{{Op: "zone", Pos: 2, Text: "Tokyo"}},
		}}
		if !reflect.DeepEqual(atStage.Stages, wantAt) {
			t.Errorf("@ Tokyo = %#v, want %#v", atStage.Stages, wantAt)
		}

		inStage, err := Parse("in Tokyo", EN)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantIn := []Node{{
			Op:   "inzone",
			Pos:  0,
			Args: []Node{{Op: "zone", Pos: 3, Text: "Tokyo"}},
		}}
		if !reflect.DeepEqual(inStage.Stages, wantIn) {
			t.Errorf("in Tokyo = %#v, want %#v", inStage.Stages, wantIn)
		}
	})

	t.Run("next aug 20 vs next month", func(t *testing.T) {
		m, err := Parse("next aug 20", EN)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Stages[0].Op != "rel" || m.Stages[0].Args[0].Op != "month" {
			t.Errorf("next aug 20 = %#v, want rel wrapping month", m.Stages[0])
		}

		u, err := Parse("next month", EN)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.Stages[0].Op != "rel" || u.Stages[0].Args[0].Op != "unit" {
			t.Errorf("next month = %#v, want rel wrapping unit", u.Stages[0])
		}
	})
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantPos      int
		wantMsg      string
		wantExpected bool
		wantExpItem  string
	}{
		// Required 10 rows from Step 5
		{
			name:         "now plus EOF",
			input:        "now +",
			wantPos:      5,
			wantMsg:      "end of input",
			wantExpected: true,
		},
		{
			name:    "unknown unit 2x",
			input:   "now + 2x",
			wantPos: 6,
			wantMsg: `unknown unit "x"`,
		},
		{
			name:    "leftover today today",
			input:   "today today",
			wantPos: 6,
			wantMsg: "unexpected",
		},
		{
			name:    "parentheses",
			input:   "(today)",
			wantPos: 0,
			wantMsg: "parentheses",
		},
		{
			name:         "end of EOF",
			input:        "end of",
			wantPos:      6,
			wantMsg:      "end of input",
			wantExpected: true,
		},
		{
			name:         "from today EOF",
			input:        "from today",
			wantPos:      10,
			wantMsg:      "end of input",
			wantExpected: true,
			wantExpItem:  "to",
		},
		{
			name:         "how many days between today EOF",
			input:        "how many days between today",
			wantPos:      27,
			wantMsg:      "end of input",
			wantExpected: true,
			wantExpItem:  "and",
		},
		{
			name:    "unclosed string",
			input:   `"abc`,
			wantPos: 0,
			wantMsg: "unexpected character",
		},
		{
			name:    "illegal hash",
			input:   "#",
			wantPos: 0,
			wantMsg: "unexpected character",
		},
		{
			name:    "empty",
			input:   "",
			wantPos: 0,
			wantMsg: "empty",
		},

		// Additional error edge cases
		{
			name:    "whitespace only",
			input:   "   ",
			wantPos: 0,
			wantMsg: "empty",
		},
		{
			name:         "trailing comma",
			input:        "today,",
			wantPos:      6,
			wantMsg:      "end of input",
			wantExpected: true,
		},
		{
			name:         "leading comma",
			input:        ", today",
			wantPos:      0,
			wantMsg:      "unexpected",
			wantExpected: true,
		},
		{
			name:         "empty middle stage",
			input:        "today | | now",
			wantPos:      8,
			wantMsg:      "unexpected",
			wantExpected: true,
		},
		{
			name:         "now to EOF",
			input:        "now to",
			wantPos:      6,
			wantMsg:      "end of input",
			wantExpected: true,
		},
		{
			name:         "2pm @ EOF",
			input:        "2pm @",
			wantPos:      5,
			wantMsg:      "end of input",
			wantExpected: true,
		},
		{
			name:         "today in EOF",
			input:        "today in",
			wantPos:      8,
			wantMsg:      "end of input",
			wantExpected: true,
		},
		{
			name:         "between today EOF",
			input:        "between today",
			wantPos:      13,
			wantMsg:      "end of input",
			wantExpected: true,
			wantExpItem:  "and",
		},
		{
			name:    "closing paren",
			input:   "today)",
			wantPos: 5,
			wantMsg: "parentheses",
		},
		{
			name:    "paren inside expr",
			input:   "now + (1d)",
			wantPos: 6,
			wantMsg: "parentheses",
		},
		{
			name:    "unknown word after plus",
			input:   "today + foo",
			wantPos: 8,
			wantMsg: "unexpected",
		},
		{
			name:    "unquoted zone stops at keyword of",
			input:   "now to Port of Spain",
			wantPos: 12,
			wantMsg: `unexpected "of"`,
		},
		{
			name:    "bare unit day",
			input:   "day",
			wantPos: 3,
			wantMsg: "end of input",
		},
		{
			name:    "bare week",
			input:   "week",
			wantPos: 4,
			wantMsg: "end of input",
		},
		{
			name:    "chained span dots",
			input:   "today .. tomorrow .. friday",
			wantPos: 18,
			wantMsg: `unexpected ".."`,
		},
		{
			name:    "invalid 12h clock",
			input:   "13pm",
			wantPos: 0,
			wantMsg: "invalid 12-hour clock",
		},
		{
			name:    "invalid unix decimal",
			input:   "@1700.5",
			wantPos: 1,
			wantMsg: "invalid unix timestamp",
		},
		{
			name:    "in followed by bare non-year number",
			input:   "in 3",
			wantPos: 4,
			wantMsg: "end of input",
		},
		{
			name:    "empty quoted zone",
			input:   `now to ""`,
			wantPos: 7,
			wantMsg: "empty timezone",
		},
		{
			name:         "end missing of",
			input:        "end today",
			wantPos:      4,
			wantMsg:      "unexpected",
			wantExpected: true,
			wantExpItem:  "of",
		},
		{
			name:         "from missing to before dots",
			input:        "from today .. tomorrow",
			wantPos:      11,
			wantMsg:      "unexpected",
			wantExpected: true,
			wantExpItem:  "to",
		},
		{
			name:         "how many missing between/from",
			input:        "how many days today",
			wantPos:      14,
			wantMsg:      "unexpected",
			wantExpected: true,
			wantExpItem:  "between",
		},
		{
			name:         "plus at zone invalid",
			input:        "today + @ Tokyo",
			wantPos:      8,
			wantMsg:      "unexpected",
			wantExpected: true,
		},
		{
			name:    "to UTC plus quantity is not offset",
			input:   "now to UTC + 1h",
			wantPos: 11,
			wantMsg: `unexpected "+"`,
		},
		{
			name:         "to plus quantity is not offset",
			input:        "now to +1h",
			wantPos:      7,
			wantMsg:      `unexpected "+"`,
			wantExpected: true,
			wantExpItem:  "a timezone",
		},
		{
			name:         "to plus number and unit is not offset",
			input:        "now to +1 day",
			wantPos:      7,
			wantMsg:      `unexpected "+"`,
			wantExpected: true,
			wantExpItem:  "a timezone",
		},
		{
			name:    "to Z+2 is not a prefixed offset",
			input:   "now to Z+2",
			wantPos: 8,
			wantMsg: `unexpected "+"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.input, EN)
			if err == nil {
				t.Fatalf("Parse(%q) succeeded, expected error", tc.input)
			}
			var parseErr *Error
			if !errors.As(err, &parseErr) {
				t.Fatalf("error type = %T, want *expr.Error", err)
			}
			if parseErr.Pos != tc.wantPos {
				t.Errorf("Pos = %d, want %d (msg: %q)", parseErr.Pos, tc.wantPos, parseErr.Msg)
			}
			if !strings.Contains(parseErr.Msg, tc.wantMsg) {
				t.Errorf("Msg = %q, want substring %q", parseErr.Msg, tc.wantMsg)
			}
			if tc.wantExpected && len(parseErr.Expected) == 0 {
				t.Errorf("Expected is empty, want non-empty (msg: %q)", parseErr.Msg)
			}
			if tc.wantExpItem != "" {
				found := false
				for _, exp := range parseErr.Expected {
					if exp == tc.wantExpItem {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected = %v, want %q in list", parseErr.Expected, tc.wantExpItem)
				}
			}
		})
	}
}

func TestSpanishEnglishParseParity(t *testing.T) {
	tests := []struct {
		es string
		en string
	}{
		// Required 8 pairs from Plan 2 Step 6
		{"hoy", "today"},
		{"hace 2 días", "2 days ago"},
		{"la semana pasada", "last week"},
		{"próximo mes", "next month"},
		{"en 3 meses", "in 3 months"},
		{"fin del día", "end of day"},
		{"20 de agosto", "aug 20"},
		{"cuántos días entre hoy y mañana", "how many days between today and tomorrow"},

		// Additional Spanish parity pairs
		{"ahora - 1d", "now - 1d"},
		{"ahorita", "now"},
		{"ayer", "yesterday"},
		{"mañana", "tomorrow"},
		{"manana", "tomorrow"},
		{"el lunes que viene", "next monday"},
		{"dentro de 3 meses", "in 3 months"},
		{"3 días hábiles", "3 business days"},
		{"desde hoy hasta mañana", "from today to tomorrow"},
		{"de hoy a mañana", "from today to tomorrow"},
		{"durante 2 semanas, a partir del lunes", "during 2 weeks, starting monday"},
		{"8 de la mañana", "8 am"},
		{"2 de la tarde", "2 pm"},
		{"ahora a UTC", "now to UTC"},
		{"lunes más cercano", "closest monday"},
		{"segunda semana", "2nd week"},
		{"20 de agosto de 2027", "aug 20 2027"},
		{"fin de semana", "end of week"},
		{"20 ago", "aug 20"},
		{"ago 20", "aug 20"},
		{"hoy + 2 days", "today + 2d"},
	}

	for _, tc := range tests {
		t.Run(tc.es+" == "+tc.en, func(t *testing.T) {
			esExpr, err := Parse(tc.es, ES)
			if err != nil {
				t.Fatalf("Parse(%q, ES) error: %v", tc.es, err)
			}
			enExpr, err := Parse(tc.en, EN)
			if err != nil {
				t.Fatalf("Parse(%q, EN) error: %v", tc.en, err)
			}
			if esExpr.String() != enExpr.String() {
				t.Errorf("Parse(%q, ES) = %q, Parse(%q, EN) = %q", tc.es, esExpr.String(), tc.en, enExpr.String())
			}
		})
	}
}

func TestREADMEExpressionsParse(t *testing.T) {
	// All concrete expressions from README.md "Expression examples" (excluding Recurrence & cron).
	cases := []struct {
		input string
		lang  Lang
	}{
		{"now + 2d", EN},
		{"2026-06-02 - 1h", EN},
		{"next month", EN},
		{"2026-12-24 - today", EN},
		{"now, to UTC", EN},
		{"hace 2 días", ES},
		{"today | end of day | start of hour | to UTC", EN},
		{"today", EN},
		{"now - 1d", EN},
		{"2026-06-02 + 2d", EN},
		{"2h * 3", EN},
		{"yesterday", EN},
		{"2d ago", EN},
		{"last week", EN},
		{"today - 1w, monday", EN},
		{"past monday", EN},
		{"closest monday", EN},
		{"end of day, today", EN},
		{"end of week, this week", EN},
		{"in 3 months", EN},
		{"in Q3", EN},
		{"2nd week, H2", EN},
		{"week 2", EN},
		{"on aug 20th", EN},
		{"on sept, past year", EN},
		{"2h", EN},
		{"90m", EN},
		{"during 2 days", EN},
		{"during 6 weeks, starting next monday", EN},
		{"2026-11-16 .. 2026-11-20", EN},
		{"how many days between 2026-11-16 and 2026-11-20", EN},
		{"2pm @ Tokyo", EN},
		{"2026-11-14 14:30 in Tokyo", EN},
		{"2026-11-14 14:30, America/New_York", EN},
		{"now, to Asia/Tokyo", EN},
		{"now to Tokyo", EN},
		{"now to CDMX", EN},
		{"now to JST", EN},
		{"now to +09:00", EN},
		{`now to "Port of Spain"`, EN},
		{`now to "Ciudad de México"`, EN},
		{"2026-11-20 - 2026-11-16, to business days", EN},
		{"2026-11-14, next business day", EN},
		{"2026-11-13 + 3 business days", EN},
		{"next business day on or after 2026-11-14", EN},
		{"2026-11-14, next business day on or after", EN},
		{"last business day on or before 2026-11-16", EN},
		{"2026-11-16, last business day on or before", EN},
		{"holidays", EN},
		{"holidays MX", EN},
		{"holidays 2026", EN},
		{"holidays MX 2026", EN},
		{"feriados MX 2026", ES},
		{"next month, holidays", EN},
		{"hoy", ES},
		{"ahora - 1d", ES},
		{"ayer", ES},
		{"mañana", ES},
		{"la semana pasada", ES},
		{"próximo mes", ES},
		{"en 3 meses", ES},
		{"fin del día", ES},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			e, err := Parse(tc.input, tc.lang)
			if err != nil {
				t.Fatalf("Parse(%q, %q) failed: %v", tc.input, tc.lang, err)
			}
			rt, err := Parse(e.String(), EN)
			if err != nil {
				t.Fatalf("round-trip Parse(%q, EN) failed: %v", e.String(), err)
			}
			if rt.String() != e.String() {
				t.Errorf("round-trip mismatch: %q != %q", rt.String(), e.String())
			}
		})
	}
}
