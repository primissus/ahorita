package expr

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/teambition/rrule-go"
)

func (ev *evaluator) evalCron(n Node) (Value, error) {
	fields, _, err := parseFiveFieldCron(n.Pos, n.Text)
	if err != nil {
		return Value{}, err
	}
	return Value{
		Kind: KRecurrence,
		T:    ev.ctx.Now.In(ev.loc),
		Days: false,
		Str:  "CRON " + fields,
	}, nil
}

func parseFiveFieldCron(pos int, spec string) (string, cron.Schedule, error) {
	trimmed := strings.TrimSpace(spec)
	if strings.HasPrefix(trimmed, "@") {
		return "", nil, &Error{Pos: pos, Msg: fmt.Sprintf("non-standard cron descriptor %q is not supported; use 5 standard fields", trimmed)}
	}
	upper := strings.ToUpper(trimmed)
	if strings.HasPrefix(upper, "CRON_TZ=") || strings.HasPrefix(upper, "TZ=") {
		return "", nil, &Error{Pos: pos, Msg: "inline cron TZ prefix is not supported; use 5 standard fields and \"to <zone>\""}
	}
	fields := strings.Join(strings.Fields(trimmed), " ")
	sched, err := cron.ParseStandard(fields)
	if err != nil {
		return "", nil, &Error{Pos: pos, Msg: err.Error()}
	}
	return fields, sched, nil
}

func cronTake(v Value, count int) (Value, error) {
	if v.T.IsZero() {
		return Value{}, fmt.Errorf("cron recurrence has no start reference")
	}
	spec := strings.TrimPrefix(v.Str, "CRON ")
	_, sched, err := parseFiveFieldCron(0, spec)
	if err != nil {
		return Value{}, err
	}
	loc := v.T.Location()
	if loc == nil {
		loc = time.UTC
	}
	cur := v.T.In(loc)
	items := make([]Value, 0, count)
	for len(items) < count {
		next := sched.Next(cur)
		if next.IsZero() {
			break
		}
		items = append(items, Value{
			Kind: KDateTime,
			T:    next.In(loc),
			Lang: v.Lang,
		})
		cur = next
	}
	return Value{Kind: KList, Items: items, Lang: v.Lang}, nil
}

func cronOccurrencesUntil(pos int, spec string, after, end time.Time) ([]time.Time, error) {
	_, sched, err := parseFiveFieldCron(pos, spec)
	if err != nil {
		return nil, err
	}
	cur := after
	var out []time.Time
	for {
		next := sched.Next(cur)
		if next.IsZero() || !next.Before(end) {
			break
		}
		out = append(out, next)
		cur = next
	}
	return out, nil
}

func (ev *evaluator) evalExplain(n Node, in Value) (Value, error) {
	if in.Kind != KRecurrence {
		return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("cannot explain %s (expected a recurrence or cron expression)", kindName(in.Kind))}
	}
	var cronSpec string
	if strings.HasPrefix(in.Str, "CRON ") {
		cronSpec = strings.TrimPrefix(in.Str, "CRON ")
	} else {
		var err error
		cronSpec, err = rruleToCron(n.Pos, in, ev.ctx.Cfg.Strict)
		if err != nil {
			return Value{}, err
		}
	}

	lang := ev.outLang()
	rawLines := strings.Split(cronSpec, "\n")
	explained := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields, _, err := parseFiveFieldCron(n.Pos, line)
		if err != nil {
			return Value{}, err
		}
		explained = append(explained, explainCronLine(fields, lang))
	}
	return Value{
		Kind: KText,
		Str:  strings.Join(explained, "\n"),
		Note: in.Note,
	}, nil
}

func (ev *evaluator) outLang() Lang {
	if ev.resLang == ES {
		return ES
	}
	return EN
}

func (ev *evaluator) evalToCron(pos int, in Value) (Value, error) {
	if in.Kind != KRecurrence {
		return Value{}, &Error{Pos: pos, Msg: fmt.Sprintf("cannot convert %s to cron (expected a recurrence)", kindName(in.Kind))}
	}
	if strings.HasPrefix(in.Str, "CRON ") {
		return Value{
			Kind: KText,
			Str:  strings.TrimPrefix(in.Str, "CRON "),
			Note: in.Note,
		}, nil
	}
	cronText, err := rruleToCron(pos, in, ev.ctx.Cfg.Strict)
	if err != nil {
		return Value{}, err
	}
	return Value{
		Kind: KText,
		Str:  cronText,
		Note: in.Note,
	}, nil
}

func rruleToCron(pos int, in Value, strict bool) (string, error) {
	rule, _, err := parseRecurrenceRRule(in)
	if err != nil {
		return "", &Error{Pos: pos, Msg: fmt.Sprintf("invalid recurrence: %v", err)}
	}
	opt := rule.Options
	orig := rule.OrigOptions

	interval := orig.Interval
	if interval == 0 {
		interval = 1
	}

	min := 0
	hr := 0
	if len(orig.Byminute) > 0 {
		min = orig.Byminute[0]
	}
	if len(orig.Byhour) > 0 {
		hr = orig.Byhour[0]
	}

	switch opt.Freq {
	case rrule.WEEKLY:
		if interval > 1 {
			return "", &Error{Pos: pos, Msg: fmt.Sprintf("cron has no week interval (cannot express every %d weeks)", interval)}
		}
		var dowField string
		if len(orig.Byweekday) > 0 {
			dowField = formatCronWeekdays(orig.Byweekday)
		} else {
			dowField = strconv.Itoa(int(rule.GetDTStart().Weekday()))
		}
		return fmt.Sprintf("%d %d * * %s", min, hr, dowField), nil

	case rrule.DAILY:
		if interval > 1 {
			return "", &Error{Pos: pos, Msg: fmt.Sprintf("cron restarts its day-of-month count every month (cannot express every %d days)", interval)}
		}
		return fmt.Sprintf("%d %d * * *", min, hr), nil

	case rrule.MONTHLY:
		if interval > 1 {
			return "", &Error{Pos: pos, Msg: fmt.Sprintf("cron cannot express every %d months", interval)}
		}
		if len(orig.Byweekday) > 0 {
			return "", &Error{Pos: pos, Msg: "cron can't combine day-of-month and day-of-week (cannot express Nth weekday of month)"}
		}
		var doms []int
		if len(orig.Bymonthday) > 0 {
			doms = orig.Bymonthday
		} else {
			doms = []int{rule.GetDTStart().Day()}
		}
		return fmt.Sprintf("%d %d %s * *", min, hr, joinIntsComma(doms)), nil

	case rrule.YEARLY:
		if interval > 1 {
			return "", &Error{Pos: pos, Msg: fmt.Sprintf("cron has no year field (cannot express every %d years)", interval)}
		}
		if len(orig.Byweekday) > 0 {
			return "", &Error{Pos: pos, Msg: "cron can't express Nth weekday rules"}
		}
		if len(orig.Byyearday) == 1 {
			yd := orig.Byyearday[0]
			if yd < 1 || yd > 59 {
				return "", &Error{Pos: pos, Msg: "year-day above 59 differs in leap years and cannot be expressed in cron"}
			}
			if yd <= 31 {
				return fmt.Sprintf("%d %d %d 1 *", min, hr, yd), nil
			}
			return fmt.Sprintf("%d %d %d 2 *", min, hr, yd-31), nil
		}
		if len(orig.Bymonth) == 1 && len(orig.Bymonthday) == 1 {
			return fmt.Sprintf("%d %d %d %d *", min, hr, orig.Bymonthday[0], orig.Bymonth[0]), nil
		}
		dt := rule.GetDTStart()
		return fmt.Sprintf("%d %d %d %d *", min, hr, dt.Day(), int(dt.Month())), nil

	case rrule.HOURLY:
		if len(orig.Byhour) > 0 || len(orig.Byminute) > 0 {
			return "", &Error{Pos: pos, Msg: "cron cannot combine an hourly interval with a fixed clock time"}
		}
		if 24%interval != 0 {
			return "", &Error{Pos: pos, Msg: fmt.Sprintf("hour interval %d does not divide a 24-hour day", interval)}
		}
		if interval == 1 {
			return "0 * * * *", nil
		}
		return fmt.Sprintf("0 */%d * * *", interval), nil

	case rrule.MINUTELY:
		if len(orig.Byhour) > 0 || len(orig.Byminute) > 0 {
			return "", &Error{Pos: pos, Msg: "cron cannot combine a minute interval with a fixed clock time"}
		}
		if interval == 1 {
			return "* * * * *", nil
		}
		if 60%interval == 0 {
			return fmt.Sprintf("*/%d * * * *", interval), nil
		}
		if 1440%interval == 0 {
			if strict {
				return "", &Error{Pos: pos, Msg: fmt.Sprintf("every %d minutes needs multiple cron lines, which is disallowed under --strict", interval)}
			}
			return multiLineMinuteCron(interval), nil
		}
		return "", &Error{Pos: pos, Msg: fmt.Sprintf("minute interval %d does not divide a 60-minute hour or 24-hour day", interval)}

	default:
		return "", &Error{Pos: pos, Msg: fmt.Sprintf("recurrence frequency %v cannot be expressed in cron", opt.Freq)}
	}
}

func multiLineMinuteCron(interval int) string {
	var minsOrder []int
	hoursByMin := make(map[int][]int)
	for total := 0; total < 1440; total += interval {
		m := total % 60
		h := total / 60
		if _, seen := hoursByMin[m]; !seen {
			minsOrder = append(minsOrder, m)
		}
		hoursByMin[m] = append(hoursByMin[m], h)
	}
	lines := make([]string, len(minsOrder))
	for i, m := range minsOrder {
		lines[i] = fmt.Sprintf("%d %s * * *", m, joinIntsComma(hoursByMin[m]))
	}
	return strings.Join(lines, "\n")
}

func formatCronWeekdays(wdays []rrule.Weekday) string {
	nums := make([]int, 0, len(wdays))
	seen := make(map[int]bool, len(wdays))
	for _, wd := range wdays {
		// rrule.Weekday.Day() has MO=0..SU=6; cron uses SU=0, MO=1..SA=6.
		c := (wd.Day() + 1) % 7
		if !seen[c] {
			seen[c] = true
			nums = append(nums, c)
		}
	}
	sort.Ints(nums)
	if len(nums) == 5 && nums[0] == 1 && nums[1] == 2 && nums[2] == 3 && nums[3] == 4 && nums[4] == 5 {
		return "1-5"
	}
	return joinIntsComma(nums)
}

func joinIntsComma(vals []int) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, ",")
}

func explainCronLine(spec string, lang Lang) string {
	parts := strings.Fields(spec)
	if len(parts) != 5 {
		return spec
	}
	m, h, dom, mon, dow := parts[0], parts[1], parts[2], parts[3], parts[4]

	if strings.HasPrefix(m, "*/") && h == "*" && dom == "*" && mon == "*" && dow == "*" {
		if iv, ok := parseExactInt(m[2:], 1, 59); ok {
			if lang == ES {
				if iv == 1 {
					return "Cada minuto"
				}
				return fmt.Sprintf("Cada %d minutos", iv)
			}
			if iv == 1 {
				return "Every minute"
			}
			return fmt.Sprintf("Every %d minutes", iv)
		}
	}

	if m == "0" && strings.HasPrefix(h, "*/") && dom == "*" && mon == "*" && dow == "*" {
		if iv, ok := parseExactInt(h[2:], 1, 23); ok {
			if lang == ES {
				if iv == 1 {
					return "Cada hora"
				}
				return fmt.Sprintf("Cada %d horas", iv)
			}
			if iv == 1 {
				return "Every hour"
			}
			return fmt.Sprintf("Every %d hours", iv)
		}
	}

	if mn, okM := parseExactInt(m, 0, 59); okM {
		if hr, okH := parseExactInt(h, 0, 23); okH {
			clockStr := fmt.Sprintf("%02d:%02d", hr, mn)

			if dom == "*" && mon == "*" && dow == "*" {
				if lang == ES {
					return fmt.Sprintf("A las %s todos los días", clockStr)
				}
				return fmt.Sprintf("At %s every day", clockStr)
			}

			if dom == "*" && mon == "*" && dow != "*" {
				if strings.EqualFold(dow, "1-5") || strings.EqualFold(dow, "mon-fri") {
					if lang == ES {
						return fmt.Sprintf("A las %s, de lunes a viernes", clockStr)
					}
					return fmt.Sprintf("At %s, Monday through Friday", clockStr)
				}
				if wdays, ok := parseCronWeekdayList(dow); ok {
					return explainCronWeekdays(clockStr, wdays, lang)
				}
			}

			if domNum, okDom := parseExactInt(dom, 1, 31); okDom && dow == "*" {
				if mon == "*" {
					if lang == ES {
						return fmt.Sprintf("A las %s el día %d de cada mes", clockStr, domNum)
					}
					return fmt.Sprintf("At %s on day %d of every month", clockStr, domNum)
				}
				if monNum, okMon := parseCronMonth(mon); okMon {
					if lang == ES {
						return fmt.Sprintf("A las %s el %d de %s", clockStr, domNum, cronMonthNamesES[monNum-1])
					}
					return fmt.Sprintf("At %s on %s %d", clockStr, cronMonthNamesEN[monNum-1], domNum)
				}
			}
		}
	}

	if lang == ES {
		return fmt.Sprintf("En el minuto %s, hora %s, día del mes %s, mes %s, día de la semana %s", m, h, dom, mon, dow)
	}
	return fmt.Sprintf("At minute %s, hour %s, day of month %s, month %s, day of week %s", m, h, dom, mon, dow)
}

var (
	cronWeekdayNamesEN = [7]string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	cronWeekdayNamesES = [7]string{"domingos", "lunes", "martes", "miércoles", "jueves", "viernes", "sábados"}
	cronMonthNamesEN   = [12]string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
	cronMonthNamesES   = [12]string{"enero", "febrero", "marzo", "abril", "mayo", "junio", "julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"}
)

func parseExactInt(s string, min, max int) (int, bool) {
	if s == "" {
		return 0, false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < min || v > max {
		return 0, false
	}
	return v, true
}

func parseCronWeekdayList(field string) ([]int, bool) {
	parts := strings.Split(field, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		wd, ok := parseCronWeekdayToken(p)
		if !ok {
			return nil, false
		}
		out = append(out, wd)
	}
	return out, len(out) > 0
}

func parseCronWeekdayToken(tok string) (int, bool) {
	if v, ok := parseExactInt(tok, 0, 7); ok {
		if v == 7 {
			v = 0
		}
		return v, true
	}
	switch strings.ToUpper(tok) {
	case "SUN":
		return 0, true
	case "MON":
		return 1, true
	case "TUE":
		return 2, true
	case "WED":
		return 3, true
	case "THU":
		return 4, true
	case "FRI":
		return 5, true
	case "SAT":
		return 6, true
	default:
		return 0, false
	}
}

func parseCronMonth(tok string) (int, bool) {
	if v, ok := parseExactInt(tok, 1, 12); ok {
		return v, true
	}
	switch strings.ToUpper(tok) {
	case "JAN":
		return 1, true
	case "FEB":
		return 2, true
	case "MAR":
		return 3, true
	case "APR":
		return 4, true
	case "MAY":
		return 5, true
	case "JUN":
		return 6, true
	case "JUL":
		return 7, true
	case "AUG":
		return 8, true
	case "SEP":
		return 9, true
	case "OCT":
		return 10, true
	case "NOV":
		return 11, true
	case "DEC":
		return 12, true
	default:
		return 0, false
	}
}

func explainCronWeekdays(clockStr string, wdays []int, lang Lang) string {
	names := make([]string, len(wdays))
	for i, wd := range wdays {
		if lang == ES {
			names[i] = cronWeekdayNamesES[wd]
		} else {
			names[i] = cronWeekdayNamesEN[wd]
		}
	}
	joined := joinNaturalList(names, lang)
	if lang == ES {
		return fmt.Sprintf("A las %s los %s", clockStr, joined)
	}
	return fmt.Sprintf("At %s on %s", clockStr, joined)
}

func joinNaturalList(items []string, lang Lang) string {
	conj := " and "
	if lang == ES {
		conj = " y "
	}
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + conj + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + conj + items[len(items)-1]
	}
}

func zoneHasDST(loc *time.Location, year int) bool {
	if loc == nil {
		return false
	}
	base := time.Date(year, time.January, 1, 12, 0, 0, 0, loc)
	_, baseOff := base.Zone()
	if base.IsDST() {
		return true
	}
	for m := time.February; m <= time.December; m++ {
		t := time.Date(year, m, 1, 12, 0, 0, 0, loc)
		_, off := t.Zone()
		if t.IsDST() || off != baseOff {
			return true
		}
	}
	return false
}

func buildCronZoneNote(srcLoc, dstLoc *time.Location, year int, dayDelta int, origWDs, newWDs []time.Weekday) string {
	var parts []string
	if dayDelta != 0 && len(origWDs) > 0 && len(origWDs) == len(newWDs) {
		switch {
		case len(origWDs) == 1:
			parts = append(parts, fmt.Sprintf("weekday shifts from %s to %s in %s", origWDs[0], newWDs[0], dstLoc.String()))
		case len(origWDs) == 5 && origWDs[0] == time.Monday && origWDs[4] == time.Friday:
			parts = append(parts, fmt.Sprintf("weekdays shift from Monday..Friday to %s..%s in %s", newWDs[0], newWDs[4], dstLoc.String()))
		default:
			parts = append(parts, fmt.Sprintf("weekdays shift by %+d day in %s", dayDelta, dstLoc.String()))
		}
	}
	srcDST := zoneHasDST(srcLoc, year)
	dstDST := zoneHasDST(dstLoc, year)
	if srcDST != dstDST {
		if srcDST {
			parts = append(parts, fmt.Sprintf("%s observes DST while %s does not", srcLoc.String(), dstLoc.String()))
		} else {
			parts = append(parts, fmt.Sprintf("%s does not observe DST while %s does", srcLoc.String(), dstLoc.String()))
		}
	}
	return strings.Join(parts, "; ")
}

func joinNotes(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	default:
		return a + "; " + b
	}
}

func convertCronToZone(pos int, v Value, targetLoc *time.Location) (Value, error) {
	spec := strings.TrimPrefix(v.Str, "CRON ")
	fields, _, err := parseFiveFieldCron(pos, spec)
	if err != nil {
		return Value{}, err
	}
	srcLoc := v.T.Location()
	if srcLoc == nil {
		srcLoc = time.UTC
	}
	if v.T.IsZero() {
		return Value{}, &Error{Pos: pos, Msg: "cron recurrence has no reference time for timezone conversion"}
	}

	parts := strings.Fields(fields)
	mField, hField, domField, monField, dowField := parts[0], parts[1], parts[2], parts[3], parts[4]
	if domField != "*" || monField != "*" {
		return Value{}, &Error{Pos: pos, Msg: "cannot shift cron expression with fixed day-of-month or month across timezones"}
	}

	mn, okM := parseExactInt(mField, 0, 59)
	hr, okH := parseExactInt(hField, 0, 23)
	if !okM || !okH {
		note := joinNotes(v.Note, buildCronZoneNote(srcLoc, targetLoc, v.T.Year(), 0, nil, nil))
		return Value{
			Kind: KRecurrence,
			T:    v.T.In(targetLoc),
			Days: false,
			Str:  "CRON " + fields,
			Note: note,
			Lang: v.Lang,
		}, nil
	}

	refNow := v.T.In(srcLoc)
	anchor := time.Date(refNow.Year(), refNow.Month(), refNow.Day(), hr, mn, 0, 0, srcLoc)
	converted := anchor.In(targetLoc)
	dayDelta := daysFromCivil(converted.Date()) - daysFromCivil(anchor.Date())

	var origWDs, newWDs []time.Weekday
	newDow := dowField
	if dowField != "*" {
		isMonFri := strings.EqualFold(dowField, "1-5") || strings.EqualFold(dowField, "mon-fri")
		var wints []int
		if isMonFri {
			wints = []int{1, 2, 3, 4, 5}
		} else {
			var ok bool
			wints, ok = parseCronWeekdayList(dowField)
			if !ok {
				return Value{}, &Error{Pos: pos, Msg: fmt.Sprintf("cannot shift cron day-of-week %q across timezones", dowField)}
			}
		}
		shifted := make([]int, len(wints))
		origWDs = make([]time.Weekday, len(wints))
		newWDs = make([]time.Weekday, len(wints))
		for i, wd := range wints {
			origWDs[i] = time.Weekday(wd)
			nw := (wd + dayDelta + 7) % 7
			newWDs[i] = time.Weekday(nw)
			shifted[i] = nw
		}
		if dayDelta != 0 {
			sort.Ints(shifted)
			newDow = joinIntsComma(shifted)
		}
	}

	newFields := fmt.Sprintf("%d %d * * %s", converted.Minute(), converted.Hour(), newDow)
	note := joinNotes(v.Note, buildCronZoneNote(srcLoc, targetLoc, refNow.Year(), dayDelta, origWDs, newWDs))
	return Value{
		Kind: KRecurrence,
		T:    v.T.In(targetLoc),
		Days: false,
		Str:  "CRON " + newFields,
		Note: note,
		Lang: v.Lang,
	}, nil
}
