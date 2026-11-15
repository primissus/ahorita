package expr

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/teambition/rrule-go"
)

func (ev *evaluator) evalToICS(pos int, in Value) (Value, error) {
	summary := "now"
	if len(ev.priorStages) > 0 {
		summary = strings.Join(ev.priorStages, " | ")
	}
	dtstamp := ev.ctx.Now.UTC().Format("20060102T150405Z")

	var (
		ruleHashKey string
		dtstartLine string
		dtendLine   string
		rruleLine   string
	)

	switch in.Kind {
	case KRecurrence:
		if strings.HasPrefix(in.Str, "CRON ") {
			return Value{}, &Error{Pos: pos, Msg: "cannot export cron schedule to .ics (convert the schedule with \"every …\" instead)"}
		}
		rule, _, err := parseRecurrenceRRule(in)
		if err != nil {
			return Value{}, &Error{Pos: pos, Msg: fmt.Sprintf("invalid recurrence: %v", err)}
		}
		firstList, err := recurTake(in, 1)
		if err != nil || len(firstList.Items) == 0 {
			return Value{}, &Error{Pos: pos, Msg: "recurrence produces no upcoming occurrence"}
		}
		first := firstList.Items[0]
		if in.Days {
			dtstartLine = "DTSTART;VALUE=DATE:" + first.T.Format("20060102")
		} else {
			dtstartLine = formatICSDatetimeProp("DTSTART", first.T)
		}
		rruleStr := rule.OrigOptions.RRuleString()
		rruleLine = "RRULE:" + rruleStr
		ruleHashKey = rruleStr

	case KDate:
		dtstartLine = "DTSTART;VALUE=DATE:" + in.T.Format("20060102")
		ruleHashKey = in.String()

	case KDateTime:
		dtstartLine = formatICSDatetimeProp("DTSTART", in.T)
		dtendLine = formatICSDatetimeProp("DTEND", in.T)
		ruleHashKey = in.String()

	case KSpan:
		if in.Days {
			dtstartLine = "DTSTART;VALUE=DATE:" + in.T.Format("20060102")
			dtendLine = "DTEND;VALUE=DATE:" + in.End.Format("20060102")
		} else {
			dtstartLine = formatICSDatetimeProp("DTSTART", in.T)
			dtendLine = formatICSDatetimeProp("DTEND", in.End)
		}
		ruleHashKey = in.String()

	default:
		return Value{}, &Error{Pos: pos, Msg: fmt.Sprintf("cannot convert %s to .ics", kindName(in.Kind))}
	}

	sum := sha256.Sum256([]byte(ruleHashKey + "\x00" + dtstartLine))
	uid := hex.EncodeToString(sum[:16]) + "@ahorita"

	lines := []string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//ahorita//EN",
		"BEGIN:VEVENT",
		"UID:" + uid,
		"DTSTAMP:" + dtstamp,
		dtstartLine,
	}
	if dtendLine != "" {
		lines = append(lines, dtendLine)
	}
	lines = append(lines, "SUMMARY:"+escapeICSText(summary))
	if rruleLine != "" {
		lines = append(lines, rruleLine)
	}
	lines = append(lines, "END:VEVENT", "END:VCALENDAR")

	return Value{
		Kind: KText,
		Str:  strings.Join(lines, "\r\n") + "\r\n",
	}, nil
}

func formatICSDatetimeProp(prop string, t time.Time) string {
	loc := t.Location()
	if loc == nil || loc == time.UTC || loc.String() == "UTC" {
		return prop + ":" + t.UTC().Format("20060102T150405Z")
	}
	return fmt.Sprintf("%s;TZID=%s:%s", prop, loc.String(), t.Format("20060102T150405"))
}

func escapeICSText(s string) string {
	r := strings.NewReplacer(
		`\`, `\\`,
		";", `\;`,
		",", `\,`,
		"\r\n", `\n`,
		"\n", `\n`,
	)
	return r.Replace(s)
}

type gridMonth struct {
	year  int
	month time.Month
	days  map[int]bool
}

func (ev *evaluator) evalToGrid(pos int, in Value) (Value, error) {
	lang := ev.outLang()
	weekStart := ev.ctx.Cfg.WeekStart

	var months []gridMonth
	switch in.Kind {
	case KDate, KDateTime:
		months = []gridMonth{{
			year:  in.T.Year(),
			month: in.T.Month(),
			days:  map[int]bool{in.T.Day(): true},
		}}

	case KSpan:
		var err error
		months, err = spanGridMonths(pos, in)
		if err != nil {
			return Value{}, err
		}

	case KRecurrence:
		var err error
		months, err = recurGridMonths(pos, in, ev.loc)
		if err != nil {
			return Value{}, err
		}

	default:
		return Value{}, &Error{Pos: pos, Msg: fmt.Sprintf("cannot display %s as a month grid", kindName(in.Kind))}
	}

	rendered := make([]string, len(months))
	for i, gm := range months {
		rendered[i] = renderMonthGrid(gm.year, gm.month, gm.days, weekStart, lang)
	}
	return Value{
		Kind: KText,
		Str:  strings.Join(rendered, "\n\n"),
	}, nil
}

func spanGridMonths(pos int, in Value) ([]gridMonth, error) {
	start := dateOnly(in.T)
	var end time.Time
	if in.Days {
		end = dateOnly(in.End).AddDate(0, 0, -1)
	} else {
		end = dateOnly(in.End.Add(-time.Nanosecond))
	}
	if end.Before(start) {
		end = start
	}

	monthsCount := (end.Year()-start.Year())*12 + int(end.Month()-start.Month()) + 1
	if monthsCount > 12 {
		return nil, &Error{Pos: pos, Msg: fmt.Sprintf("span covers %d months; grid shows at most 12 months", monthsCount)}
	}

	months := make([]gridMonth, 0, monthsCount)
	idx := make(map[[2]int]int, monthsCount)
	curY, curM := start.Year(), start.Month()
	for i := 0; i < monthsCount; i++ {
		idx[[2]int{curY, int(curM)}] = i
		months = append(months, gridMonth{
			year:  curY,
			month: curM,
			days:  make(map[int]bool),
		})
		curM++
		if curM > 12 {
			curM = 1
			curY++
		}
	}

	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if i, ok := idx[[2]int{d.Year(), int(d.Month())}]; ok {
			months[i].days[d.Day()] = true
		}
	}
	return months, nil
}

func recurGridMonths(pos int, in Value, fallbackLoc *time.Location) ([]gridMonth, error) {
	loc := in.T.Location()
	if loc == nil {
		loc = fallbackLoc
		if loc == nil {
			loc = time.UTC
		}
	}
	if in.T.IsZero() {
		return nil, &Error{Pos: pos, Msg: "recurrence has no reference time for grid"}
	}
	now := in.T.In(loc)
	m1 := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	m2 := m1.AddDate(0, 1, 0)
	windowEnd := m1.AddDate(0, 2, 0)

	months := []gridMonth{
		{year: m1.Year(), month: m1.Month(), days: make(map[int]bool)},
		{year: m2.Year(), month: m2.Month(), days: make(map[int]bool)},
	}
	mark := func(t time.Time) {
		lt := t.In(loc)
		for i := range months {
			if lt.Year() == months[i].year && lt.Month() == months[i].month {
				months[i].days[lt.Day()] = true
			}
		}
	}

	if strings.HasPrefix(in.Str, "CRON ") {
		spec := strings.TrimPrefix(in.Str, "CRON ")
		cur := now
		// Reuse cronTake logic indirectly via cronGridOccurrences? Kept cleanly without importing robfig/cron here.
		occs, err := cronOccurrencesUntil(pos, spec, cur, windowEnd)
		if err != nil {
			return nil, err
		}
		for _, occ := range occs {
			mark(occ)
		}
		return months, nil
	}

	rule, _, err := parseRecurrenceRRule(in)
	if err != nil {
		return nil, &Error{Pos: pos, Msg: fmt.Sprintf("invalid recurrence: %v", err)}
	}
	iter := rule.Iterator()
	for {
		occ, ok := iter()
		if !ok {
			break
		}
		if !occ.Before(windowEnd) {
			break
		}
		if !occ.After(now) {
			continue
		}
		mark(occ)
	}
	return months, nil
}

var (
	gridDayHeadersEN = [7]string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}
	gridDayHeadersES = [7]string{"do", "lu", "ma", "mi", "ju", "vi", "sá"}
)

func renderMonthGrid(year int, month time.Month, marked map[int]bool, weekStart time.Weekday, lang Lang) string {
	var title string
	if lang == ES {
		title = fmt.Sprintf("%s %d", cronMonthNamesES[int(month)-1], year)
	} else {
		title = fmt.Sprintf("%s %d", cronMonthNamesEN[int(month)-1], year)
	}
	titleWidth := utf8.RuneCountInString(title)
	if pad := (20 - titleWidth) / 2; pad > 0 {
		title = strings.Repeat(" ", pad) + title
	}

	dayNames := gridDayHeadersEN
	if lang == ES {
		dayNames = gridDayHeadersES
	}
	hdrParts := make([]string, 7)
	for i := 0; i < 7; i++ {
		hdrParts[i] = dayNames[(int(weekStart)+i)%7]
	}
	header := strings.Join(hdrParts, " ")

	lines := []string{title, header}
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	col := (int(first.Weekday()) - int(weekStart) + 7) % 7
	var row strings.Builder
	if col > 0 {
		row.WriteString(strings.Repeat("   ", col))
	}
	totalDays := daysInMonth(year, month)
	for day := 1; day <= totalDays; day++ {
		marker := ' '
		if marked[day] {
			marker = '*'
		}
		fmt.Fprintf(&row, "%2d%c", day, marker)
		col++
		if col == 7 {
			lines = append(lines, strings.TrimRight(row.String(), " "))
			row.Reset()
			col = 0
		}
	}
	if row.Len() > 0 {
		lines = append(lines, strings.TrimRight(row.String(), " "))
	}
	return strings.Join(lines, "\n")
}

// DaysFromICS parses an iCalendar (.ics) document and returns all-day events
// (dates and whole-day spans) falling within [from, to], expanding any RRULEs.
// Timed events are skipped.
func DaysFromICS(data string, from, to time.Time) ([]Value, error) {
	loc := from.Location()
	if loc == nil {
		loc = to.Location()
		if loc == nil {
			loc = time.UTC
		}
	}
	fromDay := dateOnly(from.In(loc))
	toDay := dateOnly(to.In(loc))

	lines := unfoldICSLines(data)
	var (
		hasCalendar bool
		inEvent     bool
		alarmDepth  int
		dtstartVal  string
		dtstartDate bool
		dtendVal    string
		rruleVal    string
		out         []Value
	)

	resetEvent := func() {
		dtstartVal = ""
		dtstartDate = false
		dtendVal = ""
		rruleVal = ""
	}

	for _, line := range lines {
		if line == "" {
			continue
		}
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			continue
		}
		left, val := line[:colon], strings.TrimSpace(line[colon+1:])
		propParts := strings.Split(left, ";")
		prop := strings.ToUpper(strings.TrimSpace(propParts[0]))

		switch prop {
		case "BEGIN":
			comp := strings.ToUpper(val)
			switch comp {
			case "VCALENDAR":
				hasCalendar = true
			case "VEVENT":
				inEvent = true
				alarmDepth = 0
				resetEvent()
			case "VALARM":
				if inEvent {
					alarmDepth++
				}
			}
			continue

		case "END":
			comp := strings.ToUpper(val)
			switch comp {
			case "VALARM":
				if alarmDepth > 0 {
					alarmDepth--
				}
			case "VEVENT":
				if !inEvent {
					continue
				}
				inEvent = false
				if dtstartVal == "" || !isAllDayICSValue(dtstartVal, dtstartDate) {
					resetEvent()
					continue
				}
				vals, err := expandICSEvent(dtstartVal, dtendVal, rruleVal, fromDay, toDay, loc)
				if err != nil {
					return nil, err
				}
				out = append(out, vals...)
				resetEvent()
			}
			continue
		}

		if !inEvent || alarmDepth > 0 {
			continue
		}

		switch prop {
		case "DTSTART":
			dtstartVal = val
			dtstartDate = hasICSParam(propParts[1:], "VALUE", "DATE")
		case "DTEND":
			dtendVal = val
		case "RRULE":
			rruleVal = val
		}
	}

	if !hasCalendar {
		return nil, fmt.Errorf("invalid .ics calendar: missing BEGIN:VCALENDAR")
	}
	if inEvent {
		return nil, fmt.Errorf("invalid .ics calendar: unclosed VEVENT")
	}
	return out, nil
}

func unfoldICSLines(data string) []string {
	normalized := strings.ReplaceAll(data, "\r\n", "\n")
	raw := strings.Split(normalized, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSuffix(line, "\r")
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') && len(out) > 0 {
			out[len(out)-1] += line[1:]
			continue
		}
		out = append(out, line)
	}
	return out
}

func hasICSParam(params []string, key, want string) bool {
	for _, p := range params {
		k, v, ok := strings.Cut(p, "=")
		if ok && strings.EqualFold(strings.TrimSpace(k), key) && strings.EqualFold(strings.TrimSpace(v), want) {
			return true
		}
	}
	return false
}

func isAllDayICSValue(val string, hasDateParam bool) bool {
	if hasDateParam {
		return true
	}
	return len(val) == 8 && allDigits(val)
}

func parseICSDate(val string, loc *time.Location) (time.Time, error) {
	t, err := time.ParseInLocation("20060102", val, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid iCalendar date %q: %w", val, err)
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc), nil
}

func expandICSEvent(dtstartVal, dtendVal, rruleVal string, fromDay, toDay time.Time, loc *time.Location) ([]Value, error) {
	start, err := parseICSDate(dtstartVal, loc)
	if err != nil {
		return nil, err
	}

	spanDays := 1
	if dtendVal != "" {
		end, err := parseICSDate(dtendVal, loc)
		if err != nil {
			return nil, err
		}
		if d := daysFromCivil(end.Date()) - daysFromCivil(start.Date()); d > 1 {
			spanDays = d
		}
	}

	if rruleVal == "" {
		lastDay := start.AddDate(0, 0, spanDays-1)
		if lastDay.Before(fromDay) || start.After(toDay) {
			return nil, nil
		}
		return []Value{makeDayOffValue(start, spanDays, loc)}, nil
	}

	opt, err := rrule.StrToROptionInLocation("RRULE:"+rruleVal, loc)
	if err != nil {
		return nil, fmt.Errorf("invalid iCalendar RRULE %q: %w", rruleVal, err)
	}
	opt.Dtstart = start
	rule, err := rrule.NewRRule(*opt)
	if err != nil {
		return nil, fmt.Errorf("invalid iCalendar RRULE %q: %w", rruleVal, err)
	}

	searchFrom := fromDay.AddDate(0, 0, -(spanDays - 1))
	occs := rule.Between(searchFrom, toDay, true)
	out := make([]Value, 0, len(occs))
	for _, occ := range occs {
		occDay := time.Date(occ.In(loc).Year(), occ.In(loc).Month(), occ.In(loc).Day(), 0, 0, 0, 0, loc)
		lastDay := occDay.AddDate(0, 0, spanDays-1)
		if lastDay.Before(fromDay) || occDay.After(toDay) {
			continue
		}
		out = append(out, makeDayOffValue(occDay, spanDays, loc))
	}
	return out, nil
}

func makeDayOffValue(start time.Time, spanDays int, loc *time.Location) Value {
	s := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
	if spanDays <= 1 {
		return Value{Kind: KDate, T: s, Local: true}
	}
	return Value{
		Kind:  KSpan,
		T:     s,
		End:   s.AddDate(0, 0, spanDays),
		Days:  true,
		Local: true,
	}
}
