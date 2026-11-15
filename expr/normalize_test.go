package expr

import "testing"

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
