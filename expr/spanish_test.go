package expr

import "testing"

func TestSpanish(t *testing.T) {
	ctx := refCtx(t)

	pairs := []struct {
		es []string
		en string
	}{
		{es: []string{"hoy"}, en: "today"},
		{es: []string{"ahora - 1d"}, en: "now - 1d"},
		{es: []string{"ahorita"}, en: "now"},
		{es: []string{"ayer"}, en: "yesterday"},
		{es: []string{"mañana", "manana"}, en: "tomorrow"},
		{es: []string{"hace 2 días", "hace 2 dias"}, en: "2 days ago"},
		{es: []string{"la semana pasada"}, en: "last week"},
		{es: []string{"el próximo mes", "próximo mes", "mes próximo"}, en: "next month"},
		{es: []string{"en 3 meses", "dentro de 3 meses"}, en: "in 3 months"},
		{es: []string{"fin del día"}, en: "end of day"},
		{es: []string{"fin de mes"}, en: "end of month"},
		{es: []string{"lunes pasado", "el lunes pasado"}, en: "last monday"},
		{es: []string{"el lunes que viene"}, en: "next monday"},
		{es: []string{"lunes más cercano"}, en: "closest monday"},
		{es: []string{"20 de agosto"}, en: "aug 20"},
		{es: []string{"segunda semana, h2"}, en: "2nd week, h2"},
		{es: []string{"septiembre, año pasado"}, en: "sept, past year"},
		{es: []string{"durante 2 días"}, en: "during 2 days"},
		{es: []string{"desde hoy hasta mañana"}, en: "from today to tomorrow"},
		{es: []string{"cuántos días entre 2026-11-16 y 2026-11-20"}, en: "how many days between 2026-11-16 and 2026-11-20"},
		{es: []string{"cuántos lunes entre hoy y 2026-12-31"}, en: "how many mondays between today and 2026-12-31"},
		{es: []string{"ahora a UTC"}, en: "now to UTC"},
		{es: []string{"8 de la mañana"}, en: "8am"},
		{es: []string{"hoy + 2 days"}, en: "today + 2d"},
	}

	for _, pair := range pairs {
		wantVal, err := Eval(pair.en, ctx)
		if err != nil {
			t.Fatalf("Eval(%q) error: %v", pair.en, err)
		}
		for _, es := range pair.es {
			es := es
			t.Run(es+" == "+pair.en, func(t *testing.T) {
				gotVal, err := Eval(es, ctx)
				if err != nil {
					t.Fatalf("Eval(%q) error: %v", es, err)
				}
				if gotVal.String() != wantVal.String() {
					t.Fatalf("Eval(%q) = %q, want Eval(%q) = %q", es, gotVal.String(), pair.en, wantVal.String())
				}
			})
		}
	}

	t.Run("output language relative", func(t *testing.T) {
		relCases := []struct {
			input string
			want  string
		}{
			{input: "hace 2 días", want: "hace 2 días"},
			{input: "2 days ago", want: "2 days ago"},
		}
		for _, tc := range relCases {
			v, err := Eval(tc.input, ctx)
			if err != nil {
				t.Fatalf("Eval(%q) error: %v", tc.input, err)
			}
			got, err := Format(v, ctx, Options{Type: "relative"})
			if err != nil {
				t.Fatalf("Format(%q, relative) error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("Format(%q, relative) = %q, want %q", tc.input, got, tc.want)
			}
		}
	})
}
