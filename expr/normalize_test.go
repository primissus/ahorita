package expr

import "testing"

func TestFold(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"MAÑANA", "manana"},
		{"Miércoles", "miercoles"},
		{"áéíóúüñ", "aeiouun"},
		{"ÁÉÍÓÚÜÑ", "aeiouun"},
		{"Días Hábiles", "dias habiles"},
		{"Próximo", "proximo"},
		{"pingüino", "pinguino"},
		{"today", "today"},
		{"", ""},
	}

	for _, tc := range tests {
		if got := fold(tc.in); got != tc.want {
			t.Errorf("fold(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestVocabKeysFolded(t *testing.T) {
	for k := range vocabEN {
		if got := fold(k); got != k {
			t.Errorf("vocabEN key %q differs from fold(k) = %q", k, got)
		}
	}
	for k := range vocabES {
		if got := fold(k); got != k {
			t.Errorf("vocabES key %q differs from fold(k) = %q", k, got)
		}
	}
	for k, enVal := range vocabEN {
		if esVal, ok := vocabES[k]; ok && enVal != esVal && k != "ago" {
			t.Errorf("vocabEN[%q]=%q collides with vocabES[%q]=%q without a tie-break rule", k, enVal, k, esVal)
		}
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantKeys []string // excludes trailing EOF token
	}{
		{
			name:     "now plus 2d",
			input:    "now + 2d",
			wantKeys: []string{"now", "", "day"},
		},
		{
			name:     "2M uppercase month",
			input:    "2M",
			wantKeys: []string{"month"},
		},
		{
			name:     "2m lowercase minute",
			input:    "2m",
			wantKeys: []string{"minute"},
		},
		{
			name:     "2D uppercase day",
			input:    "2D",
			wantKeys: []string{"day"},
		},
		{
			name:     "now plus 2days",
			input:    "now + 2days",
			wantKeys: []string{"now", "", "day"},
		},
		{
			name:     "past monday",
			input:    "past monday",
			wantKeys: []string{"last", "monday"},
		},
		{
			name:     "how many days between today and tomorrow",
			input:    "how many days between today and tomorrow",
			wantKeys: []string{"howmany", "day", "between", "today", "and", "tomorrow"},
		},
		{
			name:     "filler dropped on aug 20th",
			input:    "on aug 20th",
			wantKeys: []string{"august", "ordinal"},
		},
		{
			name:     "filler dropped end of the day",
			input:    "end of the day",
			wantKeys: []string{"end", "of", "day"},
		},
		{
			name:     "compound business days",
			input:    "3 business days",
			wantKeys: []string{"", "businessday"},
		},
		{
			name:     "2pm at Tokyo",
			input:    "2pm @ Tokyo",
			wantKeys: []string{"pm", "", ""},
		},
		{
			name:     "now to IANA zone",
			input:    "now to America/Mexico_City",
			wantKeys: []string{"now", "to", ""},
		},
		{
			name:     "2nd week H2",
			input:    "2nd week, H2",
			wantKeys: []string{"ordinal", "week", "", "h2"},
		},
		{
			name:     "unknown quantity unit",
			input:    "2x",
			wantKeys: []string{""},
		},
		{
			name:     "case preserved on Text and Unit",
			input:    "2Days",
			wantKeys: []string{"day"},
		},
		// Step 2: English synonym groups
		{
			name:     "synonym last: previous prev",
			input:    "previous monday, prev week",
			wantKeys: []string{"last", "monday", "", "last", "week"},
		},
		{
			name:     "synonym next: coming following",
			input:    "coming friday, following month",
			wantKeys: []string{"next", "friday", "", "next", "month"},
		},
		{
			name:     "synonym this: current",
			input:    "current week",
			wantKeys: []string{"this", "week"},
		},
		{
			name:     "synonym start: beginning",
			input:    "beginning of year",
			wantKeys: []string{"start", "of", "year"},
		},
		{
			name:     "synonym closest: nearest",
			input:    "nearest monday",
			wantKeys: []string{"closest", "monday"},
		},
		{
			name:     "synonym to: until till through",
			input:    "from today until tomorrow, till monday, through friday",
			wantKeys: []string{"from", "today", "to", "tomorrow", "", "to", "monday", "", "to", "friday"},
		},
		{
			name:     "synonym from: since",
			input:    "since yesterday",
			wantKeys: []string{"from", "yesterday"},
		},
		{
			name:     "synonym inclusive: inclusively including",
			input:    "inclusive inclusively including",
			wantKeys: []string{"inclusive", "inclusive", "inclusive"},
		},
		{
			name:     "synonym ago: back",
			input:    "2d back",
			wantKeys: []string{"day", "ago"},
		},
		{
			name:     "synonym utc: gmt z",
			input:    "now to gmt, now to Z",
			wantKeys: []string{"now", "to", "utc", "", "now", "to", "utc"},
		},
		{
			name:     "synonym ordinals: first third fourth fifth",
			input:    "first third fourth fifth",
			wantKeys: []string{"1st", "3rd", "4th", "5th"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Normalize(Lex(tc.input), EN)
			if len(got) != len(tc.wantKeys)+1 {
				t.Fatalf("Normalize(Lex(%q)) returned %d tokens %+v, want %d + EOF",
					tc.input, len(got), got, len(tc.wantKeys))
			}
			for i, wantKey := range tc.wantKeys {
				if got[i].Key != wantKey {
					t.Errorf("tok[%d] (%v %q) Key = %q, want %q",
						i, got[i].Kind, got[i].Text, got[i].Key, wantKey)
				}
			}
			if got[len(got)-1].Kind != EOF {
				t.Errorf("last token = %v, want EOF", got[len(got)-1].Kind)
			}
		})
	}
}

func TestNormalizePreservesCaseAndPos(t *testing.T) {
	got := Normalize(Lex("How Many Business Days"), EN)
	if len(got) != 3 {
		t.Fatalf("got %d tokens %+v, want 3 (2 phrases + EOF)", len(got), got)
	}
	if got[0].Key != "howmany" || got[0].Text != "How Many" || got[0].Pos != 0 {
		t.Errorf("got[0] = %+v, want Key=howmany Text=%q Pos=0", got[0], "How Many")
	}
	if got[1].Key != "businessday" || got[1].Text != "Business Days" || got[1].Pos != 9 {
		t.Errorf("got[1] = %+v, want Key=businessday Text=%q Pos=9", got[1], "Business Days")
	}
	qty := Normalize(Lex("2Days"), EN)
	if qty[0].Unit != "Days" || qty[0].Text != "2Days" || qty[0].Key != "day" {
		t.Errorf("qty[0] = %+v, want Unit=Days Text=2Days Key=day", qty[0])
	}
}

func TestNormalizeSpanish(t *testing.T) {
	tests := []struct {
		input    string
		wantKeys []string
	}{
		{"hoy", []string{"today"}},
		{"ahora - 1d", []string{"now", "", "day"}},
		{"ahorita", []string{"now"}},
		{"ayer", []string{"yesterday"}},
		{"mañana", []string{"tomorrow"}},
		{"manana", []string{"tomorrow"}},
		{"hace 2 días", []string{"ago", "", "day"}},
		{"la semana pasada", []string{"week", "last"}},
		{"próximo mes", []string{"next", "month"}},
		{"el lunes que viene", []string{"monday", "next"}},
		{"en 3 meses", []string{"in", "", "month"}},
		{"dentro de 3 meses", []string{"in", "", "month"}},
		{"fin del día", []string{"end", "of", "day"}},
		{"20 de agosto", []string{"", "of", "august"}},
		{"cuántos días entre hoy y mañana", []string{"howmany", "day", "between", "today", "and", "tomorrow"}},
		{"3 días hábiles", []string{"", "businessday"}},
		{"desde hoy hasta mañana", []string{"from", "today", "to", "tomorrow"}},
		{"durante 2 semanas, a partir del lunes", []string{"during", "", "week", "", "starting", "monday"}},
		{"8 de la mañana", []string{"", "am"}},
		{"ahora a UTC", []string{"now", "to", "utc"}},
		{"hoy + 2 days", []string{"today", "", "", "day"}},
		// Additional Spanish vocabulary and ordinal suffixes
		{"semana laboral", []string{"workweek"}},
		{"2 días laborables", []string{"", "businessday"}},
		{"inicio del trimestre", []string{"start", "of", "quarter"}},
		{"comienzo del semestre", []string{"start", "of", "half"}},
		{"final del año", []string{"end", "of", "year"}},
		{"miércoles sábado domingo", []string{"wednesday", "saturday", "sunday"}},
		{"lun mar mie jue vie sab dom", []string{"monday", "march", "wednesday", "thursday", "friday", "saturday", "sunday"}},
		{"ene feb abr mayo jun jul ago setiembre oct nov dic", []string{"january", "february", "april", "may", "june", "july", "august", "september", "october", "november", "december"}},
		{"lunes más cercano", []string{"monday", "closest"}},
		{"2 de la tarde, 9 de la noche", []string{"", "pm", "", "", "pm"}},
		{"a las 14:30", []string{""}},
		{"a la 1 de la tarde", []string{"", "pm"}},
		{"primer tercer cuarto quinto", []string{"1st", "3rd", "4th", "5th"}},
		{"2o 2a 2º 2ª 1er 1ro 2do 2da 4to 4ta", []string{"ordinal", "ordinal", "ordinal", "ordinal", "ordinal", "ordinal", "ordinal", "ordinal", "ordinal", "ordinal"}},
		{"2d 2m 2M 2días", []string{"day", "minute", "month", "day"}},
		{"2ago 2e", []string{"", ""}},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := Normalize(Lex(tc.input), ES)
			if len(got) != len(tc.wantKeys)+1 {
				t.Fatalf("Normalize(Lex(%q)) returned %d tokens %+v, want %d + EOF",
					tc.input, len(got), got, len(tc.wantKeys))
			}
			for i, wantKey := range tc.wantKeys {
				if got[i].Key != wantKey {
					t.Errorf("tok[%d] (%v %q) Key = %q, want %q",
						i, got[i].Kind, got[i].Text, got[i].Key, wantKey)
				}
			}
		})
	}
}

func TestTokenLang(t *testing.T) {
	hoy := Normalize(Lex("hoy"), EN)
	if hoy[0].Lang != ES {
		t.Errorf("hoy Lang = %q, want %q", hoy[0].Lang, ES)
	}

	today := Normalize(Lex("today"), ES)
	if today[0].Lang != EN {
		t.Errorf("today Lang = %q, want %q", today[0].Lang, EN)
	}

	date := Normalize(Lex("2026-06-02"), EN)
	if date[0].Lang != "" {
		t.Errorf("2026-06-02 Lang = %q, want empty", date[0].Lang)
	}

	incl := Normalize(Lex("inclusive"), EN)
	if incl[0].Lang != "" {
		t.Errorf("inclusive Lang = %q, want empty", incl[0].Lang)
	}

	mixed := Normalize(Lex("hoy + 2 days"), EN)
	if mixed[0].Lang != ES || mixed[3].Lang != EN {
		t.Errorf("mixed langs = (%q, %q), want (%q, %q)", mixed[0].Lang, mixed[3].Lang, ES, EN)
	}
}

func TestInputLang(t *testing.T) {
	tests := []struct {
		input string
		want  Lang
	}{
		{"hace 2 días", ES},
		{"now + 2d", EN},
		{"2026-06-02 + 2d", ""},
		{"hoy + 2 days", ES},
		{"2026-06-02", ""},
		{"el lunes", ES},
		{"2º", ES},
		{"20th", EN},
		{"20 feb", ""},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			if got := InputLang(Normalize(Lex(tc.input), EN)); got != tc.want {
				t.Errorf("InputLang(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestTies(t *testing.T) {
	type wantTok struct {
		key  string
		lang Lang
		tie  bool
	}
	tests := []struct {
		name  string
		input string
		lang  Lang
		want  []wantTok
	}{
		// ago
		{
			name:  "ago after quantity under ES",
			input: "2d ago",
			lang:  ES,
			want:  []wantTok{{"day", "", false}, {"ago", EN, false}},
		},
		{
			name:  "ago after unit word under ES",
			input: "2 days ago",
			lang:  ES,
			want:  []wantTok{{"", "", false}, {"day", EN, false}, {"ago", EN, false}},
		},
		{
			name:  "ago after business day unit word",
			input: "3 business days ago",
			lang:  EN,
			want:  []wantTok{{"", "", false}, {"businessday", EN, false}, {"ago", EN, false}},
		},
		{
			name:  "ago after bare number under EN",
			input: "20 ago",
			lang:  EN,
			want:  []wantTok{{"", "", false}, {"august", ES, false}},
		},
		{
			name:  "ago before bare number under EN",
			input: "ago 20",
			lang:  EN,
			want:  []wantTok{{"august", ES, false}, {"", "", false}},
		},
		{
			name:  "bare ago under EN ties",
			input: "ago",
			lang:  EN,
			want:  []wantTok{{"ago", EN, true}},
		},
		{
			name:  "bare ago under ES ties",
			input: "ago",
			lang:  ES,
			want:  []wantTok{{"august", ES, true}},
		},
		{
			name:  "bare ago under empty lang defaults EN and ties",
			input: "ago",
			lang:  "",
			want:  []wantTok{{"ago", EN, true}},
		},
		// mar
		{
			name:  "mar always march",
			input: "mar 20",
			lang:  ES,
			want:  []wantTok{{"march", "", false}, {"", "", false}},
		},
		{
			name:  "martes in full",
			input: "martes",
			lang:  ES,
			want:  []wantTok{{"tuesday", ES, false}},
		},
		// shared month abbreviations
		{
			name:  "shared month abbrevs no tie no lang",
			input: "sep jun jul nov oct feb",
			lang:  ES,
			want: []wantTok{
				{"september", "", false},
				{"june", "", false},
				{"july", "", false},
				{"november", "", false},
				{"october", "", false},
				{"february", "", false},
			},
		},
		// second, segundo, segunda
		{
			name:  "30 seconds plural",
			input: "30 seconds",
			lang:  EN,
			want:  []wantTok{{"", "", false}, {"second", EN, false}},
		},
		{
			name:  "30 second singular after number",
			input: "30 second",
			lang:  EN,
			want:  []wantTok{{"", "", false}, {"second", EN, false}},
		},
		{
			name:  "1 segundo after number",
			input: "1 segundo",
			lang:  ES,
			want:  []wantTok{{"", "", false}, {"second", ES, false}},
		},
		{
			name:  "1 segunda after number",
			input: "1 segunda",
			lang:  ES,
			want:  []wantTok{{"", "", false}, {"second", ES, false}},
		},
		{
			name:  "second week ordinal",
			input: "second week",
			lang:  EN,
			want:  []wantTok{{"2nd", EN, false}, {"week", EN, false}},
		},
		{
			name:  "segundo mes ordinal",
			input: "segundo mes",
			lang:  ES,
			want:  []wantTok{{"2nd", ES, false}, {"month", ES, false}},
		},
		{
			name:  "segunda semana ordinal",
			input: "segunda semana",
			lang:  ES,
			want:  []wantTok{{"2nd", ES, false}, {"week", ES, false}},
		},
		{
			name:  "la segunda semana ordinal",
			input: "la segunda semana",
			lang:  ES,
			want:  []wantTok{{"2nd", ES, false}, {"week", ES, false}},
		},
		// manana
		{
			name:  "manana alone is tomorrow",
			input: "mañana",
			lang:  ES,
			want:  []wantTok{{"tomorrow", ES, false}},
		},
		{
			name:  "de la manana is am",
			input: "8 de la mañana",
			lang:  ES,
			want:  []wantTok{{"", "", false}, {"am", ES, false}},
		},
		{
			name:  "en la manana stays unknown",
			input: "en la mañana",
			lang:  ES,
			want:  []wantTok{{"", "", false}, {"", "", false}, {"", "", false}},
		},
		{
			name:  "por la manana stays unknown",
			input: "por la mañana",
			lang:  ES,
			want:  []wantTok{{"", "", false}, {"", "", false}, {"", "", false}},
		},
		{
			name:  "manana por la manana keeps tomorrow and unknown",
			input: "mañana por la mañana",
			lang:  ES,
			want:  []wantTok{{"tomorrow", ES, false}, {"", "", false}, {"", "", false}, {"", "", false}},
		},
		// de
		{
			name:  "de at start of input is from",
			input: "de hoy a mañana",
			lang:  ES,
			want:  []wantTok{{"from", ES, false}, {"today", ES, false}, {"to", ES, false}, {"tomorrow", ES, false}},
		},
		{
			name:  "de after comma is from",
			input: "hoy, de lunes a viernes",
			lang:  ES,
			want:  []wantTok{{"today", ES, false}, {"", "", false}, {"from", ES, false}, {"monday", ES, false}, {"to", ES, false}, {"friday", ES, false}},
		},
		{
			name:  "de after pipe is from",
			input: "hoy | de lunes a viernes",
			lang:  ES,
			want:  []wantTok{{"today", ES, false}, {"", "", false}, {"from", ES, false}, {"monday", ES, false}, {"to", ES, false}, {"friday", ES, false}},
		},
		{
			name:  "fin de mes uses of",
			input: "fin de mes",
			lang:  ES,
			want:  []wantTok{{"end", ES, false}, {"of", ES, false}, {"month", ES, false}},
		},
		// a
		{
			name:  "a as word is to and 2a is ordinal",
			input: "2a a hoy",
			lang:  ES,
			want:  []wantTok{{"ordinal", ES, false}, {"to", ES, false}, {"today", ES, false}},
		},
		// fin de semana
		{
			name:  "fin de semana sets Tie on end",
			input: "fin de semana",
			lang:  ES,
			want:  []wantTok{{"end", ES, true}, {"of", ES, false}, {"week", ES, false}},
		},
		{
			name:  "el fin de semana drops filler and sets Tie on end",
			input: "el fin de semana",
			lang:  ES,
			want:  []wantTok{{"end", ES, true}, {"of", ES, false}, {"week", ES, false}},
		},
		// in / en and m / M
		{
			name:  "in and en and m vs M",
			input: "in 2m, en 2M",
			lang:  EN,
			want:  []wantTok{{"in", EN, false}, {"minute", "", false}, {"", "", false}, {"in", ES, false}, {"month", "", false}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Normalize(Lex(tc.input), tc.lang)
			if len(got) != len(tc.want)+1 {
				t.Fatalf("Normalize(Lex(%q), %q) returned %d tokens %+v, want %d + EOF",
					tc.input, tc.lang, len(got), got, len(tc.want))
			}
			for i, w := range tc.want {
				g := got[i]
				if g.Key != w.key || g.Lang != w.lang || g.Tie != w.tie {
					t.Errorf("tok[%d] (%v %q) = (Key=%q, Lang=%q, Tie=%v), want (Key=%q, Lang=%q, Tie=%v)",
						i, g.Kind, g.Text, g.Key, g.Lang, g.Tie, w.key, w.lang, w.tie)
				}
			}
			if got[len(got)-1].Kind != EOF {
				t.Errorf("last token = %v, want EOF", got[len(got)-1].Kind)
			}
		})
	}
}

func TestREADMESpanishBlock(t *testing.T) {
	lines := []string{
		"hoy",
		"ahora - 1d",
		"ayer",
		"mañana",
		"hace 2 días",
		"la semana pasada",
		"próximo mes",
		"en 3 meses",
		"fin del día",
	}

	for _, line := range lines {
		toks := Normalize(Lex(line), ES)
		if len(toks) <= 1 {
			t.Fatalf("line %q produced no tokens before EOF", line)
		}
		for i, tok := range toks {
			switch tok.Kind {
			case EOF, Number, Op, String, Date, DateTime, Clock, SlashDate:
				continue
			case Illegal:
				t.Errorf("line %q tok[%d] unexpected Illegal token %q", line, i, tok.Text)
			default:
				if tok.Key == "" {
					t.Errorf("line %q tok[%d] (%v %q) has empty Key", line, i, tok.Kind, tok.Text)
				}
			}
		}
	}
}

func TestNormalizeRecurrenceVocab(t *testing.T) {
	cases := []struct {
		input    string
		lang     Lang
		wantKeys []string
		wantLang Lang
	}{
		{"every monday", "", []string{"every", "monday"}, EN},
		{"each weekday", "", []string{"every", "weekdaykw"}, EN},
		{"every weekdays, to cron", "", []string{"every", "weekdaykw", ",", "to", "cron"}, EN},
		{"to ics, to grid, explain", "", []string{"to", "ics", ",", "to", "grid", ",", "explain"}, EN},
		{"cada lunes", "", []string{"every", "monday"}, ES},
		{"todos los días de semana", "", []string{"every", "weekdaykw"}, ES},
		{"todas las semanas", "", []string{"every", "week"}, ES},
		{"a calendario, explicar", "", []string{"to", "grid", ",", "explain"}, ES},
		{`cron "0 9 * * 1", explica`, "", []string{"cron", `0 9 * * 1`, ",", "explain"}, ES},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			toks := Normalize(Lex(tc.input), tc.lang)
			if got := InputLang(toks); got != tc.wantLang {
				t.Errorf("InputLang(%q) = %q, want %q", tc.input, got, tc.wantLang)
			}
			var got []string
			for _, tok := range toks {
				if tok.Kind == EOF {
					continue
				}
				if tok.Key != "" {
					got = append(got, tok.Key)
				} else {
					got = append(got, tok.Text)
				}
			}
			if len(got) != len(tc.wantKeys) {
				t.Fatalf("keys(%q) = %v, want %v", tc.input, got, tc.wantKeys)
			}
			for i := range got {
				if got[i] != tc.wantKeys[i] {
					t.Fatalf("keys(%q)[%d] = %q, want %q (all=%v)", tc.input, i, got[i], tc.wantKeys[i], got)
				}
			}
		})
	}
}
