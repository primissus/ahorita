package expr

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestToCron(t *testing.T) {
	ctx := recurRefCtx(t, time.Monday)

	cases := []struct {
		input  string
		strict bool
		want   string
	}{
		{input: "every monday, to cron", want: "0 0 * * 1"},
		{input: "every monday 9am, to cron", want: "0 9 * * 1"},
		{input: "every weekday 8:30, to cron", want: "30 8 * * 1-5"},
		{input: "every sunday, to cron", want: "0 0 * * 0"},
		{input: "every day 2nd, to cron", want: "0 0 2 * *"},
		{input: "every day 14:30, to cron", want: "30 14 * * *"},
		{input: "every year aug 20, to cron", want: "0 0 20 8 *"},
		{input: "every day-year 2nd, to cron", want: "0 0 2 1 *"},
		{input: "every day-year 59th, to cron", want: "0 0 28 2 *"},
		{input: "every 15 minutes, to cron", want: "*/15 * * * *"},
		{input: "every 2 hours, to cron", want: "0 */2 * * *"},
		{
			input: "every 90 minutes, to cron",
			want:  "0 0,3,6,9,12,15,18,21 * * *\n30 1,4,7,10,13,16,19,22 * * *",
		},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			c := ctx
			c.Cfg.Strict = tc.strict
			v, err := Eval(tc.input, c)
			if err != nil {
				t.Fatalf("Eval(%q) unexpected error: %v", tc.input, err)
			}
			if v.Kind != KText {
				t.Fatalf("Eval(%q).Kind = %v, want KText", tc.input, v.Kind)
			}
			if got := v.String(); got != tc.want {
				t.Errorf("Eval(%q) =\n%q\nwant:\n%q", tc.input, got, tc.want)
			}
		})
	}

	t.Run("errors", func(t *testing.T) {
		bad := []struct {
			input   string
			strict  bool
			wantSub string
		}{
			{input: "every 90 minutes, to cron", strict: true, wantSub: "strict"},
			{input: "every 2 weeks, to cron", wantSub: "week"},
			{input: "every 2 days, to cron", wantSub: "month"},
			{input: "every 2nd monday, to cron", wantSub: "combine"},
			{input: "every 7 minutes, to cron", wantSub: "divide"},
			{input: "every 5 hours, to cron", wantSub: "divide"},
			{input: "every day-year 60th, to cron", wantSub: "leap"},
			{input: "today, to cron", wantSub: "recurrence"},
		}
		for _, tc := range bad {
			c := ctx
			c.Cfg.Strict = tc.strict
			_, err := Eval(tc.input, c)
			if err == nil {
				t.Errorf("Eval(%q, strict=%v) unexpectedly succeeded", tc.input, tc.strict)
				continue
			}
			if tc.wantSub != "" && !strings.Contains(strings.ToLower(err.Error()), tc.wantSub) {
				t.Errorf("Eval(%q) error = %q, want substring %q", tc.input, err.Error(), tc.wantSub)
			}
		}
	})
}

func TestCron(t *testing.T) {
	ctx := recurRefCtx(t, time.Monday)

	explainCases := []struct {
		input string
		lang  Lang
		want  string
	}{
		{`cron "0 9 * * 1", explain`, EN, "At 09:00 on Monday"},
		{`cron "0 9 * * 1", explica`, "", "A las 09:00 los lunes"},
		{`cron "30 8 * * 1-5", explain`, EN, "At 08:30, Monday through Friday"},
		{`cron "30 8 * * 1-5", explica`, "", "A las 08:30, de lunes a viernes"},
		{`cron "0 0 * * *", explain`, EN, "At 00:00 every day"},
		{`cron "0 0 * * *", explica`, "", "A las 00:00 todos los días"},
		{`cron "*/15 * * * *", explain`, EN, "Every 15 minutes"},
		{`cron "*/15 * * * *", explica`, "", "Cada 15 minutos"},
		{`cron "0 */2 * * *", explain`, EN, "Every 2 hours"},
		{`cron "0 */2 * * *", explica`, "", "Cada 2 horas"},
		{`cron "0 0 2 * *", explain`, EN, "At 00:00 on day 2 of every month"},
		{`cron "0 0 2 * *", explica`, "", "A las 00:00 el día 2 de cada mes"},
		{`cron "0 0 20 8 *", explain`, EN, "At 00:00 on August 20"},
		{`cron "0 0 20 8 *", explica`, "", "A las 00:00 el 20 de agosto"},
		{`cron "0 9 * * 1,3", explain`, EN, "At 09:00 on Monday and Wednesday"},
		{`cron "0 9 * * 1,3", explica`, "", "A las 09:00 los lunes y miércoles"},
		{`cron "0 9 1-5 * 1", explain`, EN, "At minute 0, hour 9, day of month 1-5, month *, day of week 1"},
		{`cron "0 9 1-5 * 1", explica`, "", "En el minuto 0, hora 9, día del mes 1-5, mes *, día de la semana 1"},
		{`every monday 9am, explain`, EN, "At 09:00 on Monday"},
		{`cada lunes 9am, explica`, "", "A las 09:00 los lunes"},
	}

	for _, tc := range explainCases {
		t.Run("explain/"+tc.input, func(t *testing.T) {
			c := ctx
			if tc.lang != "" {
				c.Cfg.Lang = tc.lang
			}
			v, err := Eval(tc.input, c)
			if err != nil {
				t.Fatalf("Eval(%q) unexpected error: %v", tc.input, err)
			}
			if v.Kind != KText {
				t.Fatalf("Kind = %v, want KText", v.Kind)
			}
			if got := v.String(); got != tc.want {
				t.Errorf("Eval(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}

	t.Run("next_runs", func(t *testing.T) {
		v, err := Eval(`cron "0 9 * * 1"`, ctx)
		if err != nil {
			t.Fatal(err)
		}
		if v.Kind != KRecurrence {
			t.Fatalf("Kind = %v, want KRecurrence", v.Kind)
		}
		lines := strings.Split(v.String(), "\n")
		if len(lines) != 5 {
			t.Fatalf("got %d lines, want 5:\n%s", len(lines), v.String())
		}
		if lines[0] != "2026-11-16T09:00:00-06:00" {
			t.Errorf("first run = %q, want 2026-11-16T09:00:00-06:00", lines[0])
		}

		v2, err := Eval(`cron "0 9 * * 1", next 2`, ctx)
		if err != nil {
			t.Fatal(err)
		}
		want2 := "2026-11-16T09:00:00-06:00\n2026-11-23T09:00:00-06:00"
		if got := v2.String(); got != want2 {
			t.Errorf("got\n%s\nwant\n%s", got, want2)
		}
	})

	t.Run("errors", func(t *testing.T) {
		bad := []string{
			`cron "61 9 * * 1"`,
			`cron "0 9 * *"`,
			`cron "0 0 9 * * 1"`,
			`cron "@daily"`,
			`every 2 weeks, explain`,
			`today, explain`,
		}
		for _, in := range bad {
			if _, err := Eval(in, ctx); err == nil {
				t.Errorf("Eval(%q) unexpectedly succeeded", in)
			}
		}
	})
}

func TestCronZone(t *testing.T) {
	mxCtx := recurRefCtx(t, time.Monday)

	t.Run("every_monday_8pm_to_UTC_to_cron", func(t *testing.T) {
		res := Run("every monday 8pm, to UTC, to cron", mxCtx, Options{}, false)
		if res.Error != nil {
			t.Fatalf("unexpected error: %v", res.Error)
		}
		if res.Text != "0 2 * * 2" {
			t.Errorf("Text = %q, want %q", res.Text, "0 2 * * 2")
		}
		wantNote := "weekday shifts from Monday to Tuesday in UTC"
		if res.Note != wantNote {
			t.Errorf("Note = %q, want %q", res.Note, wantNote)
		}

		data, err := json.Marshal(res)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), `"note":"`+wantNote+`"`) {
			t.Errorf("JSON missing note field: %s", string(data))
		}
	})

	t.Run("every_monday_9am_no_shift_no_note", func(t *testing.T) {
		res := Run("every monday 9am, to UTC, to cron", mxCtx, Options{}, false)
		if res.Error != nil {
			t.Fatalf("unexpected error: %v", res.Error)
		}
		if res.Text != "0 15 * * 1" {
			t.Errorf("Text = %q, want %q", res.Text, "0 15 * * 1")
		}
		if res.Note != "" {
			t.Errorf("unexpected Note = %q", res.Note)
		}
	})

	t.Run("dst_zone_to_UTC_warns", func(t *testing.T) {
		nyLoc, err := time.LoadLocation("America/New_York")
		if err != nil {
			t.Fatal(err)
		}
		nyCtx := Context{
			Now: time.Date(2026, 11, 14, 14, 30, 0, 0, nyLoc),
			Loc: nyLoc,
			Cfg: Config{WeekStart: time.Monday, Lang: EN},
		}
		res := Run("every monday 9am, to UTC, to cron", nyCtx, Options{}, false)
		if res.Error != nil {
			t.Fatalf("unexpected error: %v", res.Error)
		}
		if res.Text != "0 14 * * 1" {
			t.Errorf("Text = %q, want %q", res.Text, "0 14 * * 1")
		}
		if !strings.Contains(res.Note, "DST") {
			t.Errorf("Note = %q, want DST warning", res.Note)
		}
	})

	t.Run("every_monday_8pm_to_UTC_next_1", func(t *testing.T) {
		v, err := Eval("every monday 8pm, to UTC, next 1", mxCtx)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := v.String(), "2026-11-17T02:00:00Z"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
		if v.Note != "weekday shifts from Monday to Tuesday in UTC" {
			t.Errorf("Note = %q", v.Note)
		}
	})

	t.Run("cron_to_UTC", func(t *testing.T) {
		res := Run(`cron "0 20 * * 1", to UTC, to cron`, mxCtx, Options{}, false)
		if res.Error != nil {
			t.Fatalf("unexpected error: %v", res.Error)
		}
		if res.Text != "0 2 * * 2" {
			t.Errorf("Text = %q, want %q", res.Text, "0 2 * * 2")
		}
		if res.Note != "weekday shifts from Monday to Tuesday in UTC" {
			t.Errorf("Note = %q", res.Note)
		}
	})
}
