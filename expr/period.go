package expr

import (
	"fmt"
	"strconv"
	"time"
)

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func inputDate(pos int, in Value) (time.Time, error) {
	switch in.Kind {
	case KDate, KDateTime, KSpan:
		return dateOnly(in.T), nil
	default:
		return time.Time{}, &Error{
			Pos: pos,
			Msg: fmt.Sprintf("cannot get a date from %s", kindName(in.Kind)),
		}
	}
}

// weekStart returns the 00:00 date of the most recent cfg.WeekStart on or before d.
func weekStart(d time.Time, cfg Config) time.Time {
	d = dateOnly(d)
	offset := (int(d.Weekday()) - int(cfg.WeekStart) + 7) % 7
	return d.AddDate(0, 0, -offset)
}

func floorDiv(a, b int) int {
	q := a / b
	if a < 0 && a%b != 0 {
		q--
	}
	return q
}

func parseWeekday(pos int, name string) (time.Weekday, error) {
	switch name {
	case "sunday":
		return time.Sunday, nil
	case "monday":
		return time.Monday, nil
	case "tuesday":
		return time.Tuesday, nil
	case "wednesday":
		return time.Wednesday, nil
	case "thursday":
		return time.Thursday, nil
	case "friday":
		return time.Friday, nil
	case "saturday":
		return time.Saturday, nil
	default:
		return 0, &Error{Pos: pos, Msg: fmt.Sprintf("unknown weekday %q", name)}
	}
}

func parseMonthName(pos int, name string) (time.Month, error) {
	switch name {
	case "january":
		return time.January, nil
	case "february":
		return time.February, nil
	case "march":
		return time.March, nil
	case "april":
		return time.April, nil
	case "may":
		return time.May, nil
	case "june":
		return time.June, nil
	case "july":
		return time.July, nil
	case "august":
		return time.August, nil
	case "september":
		return time.September, nil
	case "october":
		return time.October, nil
	case "november":
		return time.November, nil
	case "december":
		return time.December, nil
	default:
		return 0, &Error{Pos: pos, Msg: fmt.Sprintf("unknown month %q", name)}
	}
}

func unitBounds(pos int, anchor time.Time, unit string, cfg Config) (start, end time.Time, dateUnit bool, err error) {
	loc := anchor.Location()
	d := dateOnly(anchor)
	switch unit {
	case "day":
		return d, d.AddDate(0, 0, 1), true, nil
	case "week":
		ws := weekStart(d, cfg)
		return ws, ws.AddDate(0, 0, 7), true, nil
	case "workweek":
		ws := weekStart(d, cfg)
		offsetToMon := (int(time.Monday) - int(cfg.WeekStart) + 7) % 7
		mon := ws.AddDate(0, 0, offsetToMon)
		return mon, mon.AddDate(0, 0, 5), true, nil
	case "month":
		s := time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, loc)
		return s, s.AddDate(0, 1, 0), true, nil
	case "quarter":
		q := (int(d.Month()) - 1) / 3
		s := time.Date(d.Year(), time.Month(q*3+1), 1, 0, 0, 0, 0, loc)
		return s, s.AddDate(0, 3, 0), true, nil
	case "half":
		h := (int(d.Month()) - 1) / 6
		s := time.Date(d.Year(), time.Month(h*6+1), 1, 0, 0, 0, 0, loc)
		return s, s.AddDate(0, 6, 0), true, nil
	case "year":
		s := time.Date(d.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return s, time.Date(d.Year()+1, time.January, 1, 0, 0, 0, 0, loc), true, nil
	case "hour":
		s := time.Date(anchor.Year(), anchor.Month(), anchor.Day(), anchor.Hour(), 0, 0, 0, loc)
		return s, s.Add(time.Hour), false, nil
	case "minute":
		s := time.Date(anchor.Year(), anchor.Month(), anchor.Day(), anchor.Hour(), anchor.Minute(), 0, 0, loc)
		return s, s.Add(time.Minute), false, nil
	case "second":
		s := time.Date(anchor.Year(), anchor.Month(), anchor.Day(), anchor.Hour(), anchor.Minute(), anchor.Second(), 0, loc)
		return s, s.Add(time.Second), false, nil
	case "businessday":
		return time.Time{}, time.Time{}, false, &Error{Pos: pos, Msg: "start/end doesn't apply to business day"}
	default:
		return time.Time{}, time.Time{}, false, &Error{Pos: pos, Msg: fmt.Sprintf("unknown unit %q", unit)}
	}
}

func shiftedUnitSpan(pos int, in Value, unit string, delta int, cfg Config) (Value, error) {
	if unit == "businessday" {
		return Value{}, &Error{Pos: pos, Msg: "business day must be evaluated with context"}
	}

	if unit == "hour" || unit == "minute" || unit == "second" {
		if in.Kind != KDate && in.Kind != KDateTime && in.Kind != KSpan {
			return Value{}, &Error{Pos: pos, Msg: fmt.Sprintf("cannot shift %s by %s", kindName(in.Kind), unit)}
		}
		loc := in.T.Location()
		switch unit {
		case "hour":
			s := time.Date(in.T.Year(), in.T.Month(), in.T.Day(), in.T.Hour(), 0, 0, 0, loc).Add(time.Duration(delta) * time.Hour)
			return Value{Kind: KSpan, T: s, End: s.Add(time.Hour), Days: false, Local: in.Local}, nil
		case "minute":
			s := time.Date(in.T.Year(), in.T.Month(), in.T.Day(), in.T.Hour(), in.T.Minute(), 0, 0, loc).Add(time.Duration(delta) * time.Minute)
			return Value{Kind: KSpan, T: s, End: s.Add(time.Minute), Days: false, Local: in.Local}, nil
		case "second":
			s := time.Date(in.T.Year(), in.T.Month(), in.T.Day(), in.T.Hour(), in.T.Minute(), in.T.Second(), 0, loc).Add(time.Duration(delta) * time.Second)
			return Value{Kind: KSpan, T: s, End: s.Add(time.Second), Days: false, Local: in.Local}, nil
		}
	}

	d, err := inputDate(pos, in)
	if err != nil {
		return Value{}, err
	}
	loc := d.Location()
	switch unit {
	case "day":
		s := d.AddDate(0, 0, delta)
		return Value{Kind: KSpan, T: s, End: s.AddDate(0, 0, 1), Days: true, Local: in.Local}, nil
	case "week":
		ws := weekStart(d, cfg).AddDate(0, 0, 7*delta)
		return Value{Kind: KSpan, T: ws, End: ws.AddDate(0, 0, 7), Days: true, Local: in.Local}, nil
	case "workweek":
		ws := weekStart(d, cfg).AddDate(0, 0, 7*delta)
		offsetToMon := (int(time.Monday) - int(cfg.WeekStart) + 7) % 7
		mon := ws.AddDate(0, 0, offsetToMon)
		return Value{Kind: KSpan, T: mon, End: mon.AddDate(0, 0, 5), Days: true, Local: in.Local}, nil
	case "month":
		totalM := d.Year()*12 + (int(d.Month()) - 1) + delta
		y := floorDiv(totalM, 12)
		m := time.Month(totalM - y*12 + 1)
		s := time.Date(y, m, 1, 0, 0, 0, 0, loc)
		return Value{Kind: KSpan, T: s, End: s.AddDate(0, 1, 0), Days: true, Local: in.Local}, nil
	case "quarter":
		q := (int(d.Month()) - 1) / 3
		totalQ := d.Year()*4 + q + delta
		y := floorDiv(totalQ, 4)
		qq := totalQ - y*4
		s := time.Date(y, time.Month(qq*3+1), 1, 0, 0, 0, 0, loc)
		return Value{Kind: KSpan, T: s, End: s.AddDate(0, 3, 0), Days: true, Local: in.Local}, nil
	case "half":
		h := (int(d.Month()) - 1) / 6
		totalH := d.Year()*2 + h + delta
		y := floorDiv(totalH, 2)
		hh := totalH - y*2
		s := time.Date(y, time.Month(hh*6+1), 1, 0, 0, 0, 0, loc)
		return Value{Kind: KSpan, T: s, End: s.AddDate(0, 6, 0), Days: true, Local: in.Local}, nil
	case "year":
		y := d.Year() + delta
		s := time.Date(y, time.January, 1, 0, 0, 0, 0, loc)
		return Value{Kind: KSpan, T: s, End: time.Date(y+1, time.January, 1, 0, 0, 0, 0, loc), Days: true, Local: in.Local}, nil
	default:
		return Value{}, &Error{Pos: pos, Msg: fmt.Sprintf("unknown unit %q", unit)}
	}
}

func (ev *evaluator) evalWeekday(n Node, in Value) (Value, error) {
	d, err := inputDate(n.Pos, in)
	if err != nil {
		return Value{}, err
	}
	wd, err := parseWeekday(n.Pos, n.Text)
	if err != nil {
		return Value{}, err
	}
	ws := weekStart(d, ev.ctx.Cfg)
	offset := (int(wd) - int(ev.ctx.Cfg.WeekStart) + 7) % 7
	return Value{Kind: KDate, T: ws.AddDate(0, 0, offset), Local: in.Local}, nil
}

func (ev *evaluator) evalRel(n Node, in Value) (Value, error) {
	if len(n.Args) < 1 {
		return Value{}, &Error{Pos: n.Pos, Msg: "invalid relative period expression"}
	}
	var delta int
	switch n.Text {
	case "last":
		delta = -1
	case "this":
		delta = 0
	case "next":
		delta = 1
	default:
		return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("unknown relative modifier %q", n.Text)}
	}

	target := n.Args[0]
	switch target.Op {
	case "unit":
		if target.Text == "businessday" {
			return ev.evalRelBusinessDay(n.Pos, in, delta)
		}
		return shiftedUnitSpan(n.Pos, in, target.Text, delta, ev.ctx.Cfg)

	case "weekday":
		d, err := inputDate(n.Pos, in)
		if err != nil {
			return Value{}, err
		}
		wd, err := parseWeekday(target.Pos, target.Text)
		if err != nil {
			return Value{}, err
		}
		switch delta {
		case 0:
			ws := weekStart(d, ev.ctx.Cfg)
			offset := (int(wd) - int(ev.ctx.Cfg.WeekStart) + 7) % 7
			return Value{Kind: KDate, T: ws.AddDate(0, 0, offset), Local: in.Local}, nil
		case -1:
			diff := (int(d.Weekday()) - int(wd) + 7) % 7
			if diff == 0 {
				diff = 7
			}
			return Value{Kind: KDate, T: d.AddDate(0, 0, -diff), Local: in.Local}, nil
		case 1:
			diff := (int(wd) - int(d.Weekday()) + 7) % 7
			if diff == 0 {
				diff = 7
			}
			return Value{Kind: KDate, T: d.AddDate(0, 0, diff), Local: in.Local}, nil
		}

	case "month":
		return ev.evalRelMonth(n.Pos, target, delta, in)
	}

	return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("unsupported relative target %q", target.Op)}
}

func (ev *evaluator) evalRelBusinessDay(pos int, in Value, delta int) (Value, error) {
	if delta == 0 {
		return Value{}, &Error{Pos: pos, Msg: "\"this business day\" isn't defined — use next or last"}
	}
	d, err := inputDate(pos, in)
	if err != nil {
		return Value{}, err
	}
	res, err := nextBusinessDay(pos, d, delta, false, ev.ctx)
	if err != nil {
		return Value{}, err
	}
	return Value{Kind: KDate, T: res, Local: in.Local}, nil
}

func parseMonthArgs(n Node) (m time.Month, day int, hasDay bool, dayPos int, year int, hasYear bool, err error) {
	m, err = parseMonthName(n.Pos, n.Text)
	if err != nil {
		return 0, 0, false, 0, 0, false, err
	}
	for _, a := range n.Args {
		switch a.Op {
		case "num":
			v, perr := strconv.Atoi(a.Text)
			if perr != nil || v < 1 || v > 31 {
				return 0, 0, false, 0, 0, false, &Error{Pos: a.Pos, Msg: fmt.Sprintf("invalid day %q", a.Text)}
			}
			day = v
			hasDay = true
			dayPos = a.Pos
		case "year":
			v, perr := strconv.Atoi(a.Text)
			if perr != nil {
				return 0, 0, false, 0, 0, false, &Error{Pos: a.Pos, Msg: fmt.Sprintf("invalid year %q", a.Text)}
			}
			year = v
			hasYear = true
		default:
			return 0, 0, false, 0, 0, false, &Error{Pos: a.Pos, Msg: fmt.Sprintf("unexpected month argument %q", a.Op)}
		}
	}
	return m, day, hasDay, dayPos, year, hasYear, nil
}

func (ev *evaluator) evalMonth(n Node, in Value) (Value, error) {
	m, day, hasDay, dayPos, explicitYear, hasYear, err := parseMonthArgs(n)
	if err != nil {
		return Value{}, err
	}
	d, err := inputDate(n.Pos, in)
	if err != nil {
		return Value{}, err
	}
	loc := d.Location()
	y := d.Year()
	if hasYear {
		y = explicitYear
	}

	if hasDay {
		if !validDate(y, m, day) {
			return Value{}, &Error{Pos: dayPos, Msg: fmt.Sprintf("invalid date %04d-%02d-%02d", y, m, day)}
		}
		return Value{Kind: KDate, T: time.Date(y, m, day, 0, 0, 0, 0, loc), Local: in.Local}, nil
	}

	s := time.Date(y, m, 1, 0, 0, 0, 0, loc)
	return Value{Kind: KSpan, T: s, End: s.AddDate(0, 1, 0), Days: true, Local: in.Local}, nil
}

func (ev *evaluator) evalRelMonth(relPos int, n Node, delta int, in Value) (Value, error) {
	m, day, hasDay, dayPos, explicitYear, hasYear, err := parseMonthArgs(n)
	if err != nil {
		return Value{}, err
	}
	if hasYear {
		return Value{}, &Error{Pos: relPos, Msg: fmt.Sprintf("cannot combine relative modifier with explicit year %d", explicitYear)}
	}
	d, err := inputDate(relPos, in)
	if err != nil {
		return Value{}, err
	}
	loc := d.Location()

	if hasDay {
		// Reject unconditionally impossible day-of-month (e.g., Feb 30, Apr 31).
		if day > daysInMonth(2024, m) {
			return Value{}, &Error{Pos: dayPos, Msg: fmt.Sprintf("invalid day %d for %s", day, m)}
		}
		y := d.Year()
		if delta == 0 {
			if !validDate(y, m, day) {
				return Value{}, &Error{Pos: dayPos, Msg: fmt.Sprintf("invalid date %04d-%02d-%02d", y, m, day)}
			}
			return Value{Kind: KDate, T: time.Date(y, m, day, 0, 0, 0, 0, loc), Local: in.Local}, nil
		}
		for step := 0; step < 9; step++ {
			candY := y + delta*step
			if !validDate(candY, m, day) {
				continue
			}
			cand := time.Date(candY, m, day, 0, 0, 0, 0, loc)
			if (delta > 0 && cand.After(d)) || (delta < 0 && cand.Before(d)) {
				return Value{Kind: KDate, T: cand, Local: in.Local}, nil
			}
		}
		return Value{}, &Error{Pos: dayPos, Msg: fmt.Sprintf("no matching date found for %s %d", m, day)}
	}

	y := d.Year()
	if delta > 0 && m <= d.Month() {
		y++
	} else if delta < 0 && m >= d.Month() {
		y--
	}
	s := time.Date(y, m, 1, 0, 0, 0, 0, loc)
	return Value{Kind: KSpan, T: s, End: s.AddDate(0, 1, 0), Days: true, Local: in.Local}, nil
}

func (ev *evaluator) evalPeriod(n Node, in Value) (Value, error) {
	d, err := inputDate(n.Pos, in)
	if err != nil {
		return Value{}, err
	}
	y := d.Year()
	if len(n.Args) >= 1 {
		yy, perr := strconv.Atoi(n.Args[0].Text)
		if perr != nil {
			return Value{}, &Error{Pos: n.Args[0].Pos, Msg: fmt.Sprintf("invalid year %q", n.Args[0].Text)}
		}
		y = yy
	}
	loc := d.Location()
	var startMonth time.Month
	var spanMonths int
	switch n.Text {
	case "q1":
		startMonth, spanMonths = time.January, 3
	case "q2":
		startMonth, spanMonths = time.April, 3
	case "q3":
		startMonth, spanMonths = time.July, 3
	case "q4":
		startMonth, spanMonths = time.October, 3
	case "h1":
		startMonth, spanMonths = time.January, 6
	case "h2":
		startMonth, spanMonths = time.July, 6
	default:
		return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("unknown period %q", n.Text)}
	}
	s := time.Date(y, startMonth, 1, 0, 0, 0, 0, loc)
	return Value{Kind: KSpan, T: s, End: s.AddDate(0, spanMonths, 0), Days: true, Local: in.Local}, nil
}

func (ev *evaluator) evalClosest(n Node, in Value) (Value, error) {
	if len(n.Args) < 1 {
		return Value{}, &Error{Pos: n.Pos, Msg: "invalid closest expression"}
	}
	anchorVal := in
	if len(n.Args) >= 2 {
		v, err := ev.node(n.Args[1], in)
		if err != nil {
			return Value{}, err
		}
		anchorVal = v
	}
	d, err := inputDate(n.Pos, anchorVal)
	if err != nil {
		return Value{}, err
	}
	wd, err := parseWeekday(n.Args[0].Pos, n.Args[0].Text)
	if err != nil {
		return Value{}, err
	}
	fwd := (int(wd) - int(d.Weekday()) + 7) % 7
	if fwd == 0 {
		return Value{Kind: KDate, T: d, Local: anchorVal.Local}, nil
	}
	bwd := 7 - fwd
	if fwd < bwd {
		return Value{Kind: KDate, T: d.AddDate(0, 0, fwd), Local: anchorVal.Local}, nil
	}
	return Value{Kind: KDate, T: d.AddDate(0, 0, -bwd), Local: anchorVal.Local}, nil
}

func (ev *evaluator) evalWeek(n Node, in Value) (Value, error) {
	num, err := strconv.Atoi(n.Text)
	if err != nil || num < 1 {
		return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("week number %q must be at least 1", n.Text)}
	}

	var pStart, pEnd time.Time
	if in.Kind == KSpan {
		pStart = dateOnly(in.T)
		pEnd = dateOnly(in.End)
		if !in.Days && (in.End.Hour() != 0 || in.End.Minute() != 0 || in.End.Second() != 0 || in.End.Nanosecond() != 0) {
			pEnd = pEnd.AddDate(0, 0, 1)
		}
	} else {
		d, err := inputDate(n.Pos, in)
		if err != nil {
			return Value{}, err
		}
		loc := d.Location()
		pStart = time.Date(d.Year(), time.January, 1, 0, 0, 0, 0, loc)
		pEnd = time.Date(d.Year()+1, time.January, 1, 0, 0, 0, 0, loc)
	}

	if !pStart.Before(pEnd) {
		return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("week %d is out of range for empty period", num)}
	}

	cfg := ev.ctx.Cfg
	weekOne := cfg.WeekOne
	if weekOne == "" {
		weekOne = "iso"
	}

	var week1Start time.Time
	switch weekOne {
	case "days":
		week1Start = pStart
	case "first-day":
		week1Start = weekStart(pStart, cfg)
	case "full":
		ws := weekStart(pStart, cfg)
		if ws.Before(pStart) {
			ws = ws.AddDate(0, 0, 7)
		}
		if ws.AddDate(0, 0, 7).After(pEnd) {
			return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("period contains no full week for week %d", num)}
		}
		week1Start = ws
	case "iso":
		ws := weekStart(pStart, cfg)
		wsEnd := ws.AddDate(0, 0, 7)
		if pEnd.Before(wsEnd) {
			wsEnd = pEnd
		}
		if daysBetween(pStart, wsEnd) >= 4 {
			week1Start = ws
		} else {
			ws = ws.AddDate(0, 0, 7)
			wsEnd = ws.AddDate(0, 0, 7)
			if pEnd.Before(wsEnd) {
				wsEnd = pEnd
			}
			if !ws.Before(pEnd) || daysBetween(pStart, wsEnd) < 4 {
				return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("period contains no ISO week for week %d", num)}
			}
			week1Start = ws
		}
	default:
		return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("unknown WeekOne rule %q", weekOne)}
	}

	start := week1Start.AddDate(0, 0, 7*(num-1))
	if !start.Before(pEnd) {
		return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("week %d starts after the period ends", num)}
	}
	return Value{Kind: KSpan, T: start, End: start.AddDate(0, 0, 7), Days: true, Local: in.Local}, nil
}

func (ev *evaluator) evalOf(n Node, in Value) (Value, error) {
	if len(n.Args) < 2 {
		return Value{}, &Error{Pos: n.Pos, Msg: "invalid of expression"}
	}
	periodVal, err := ev.node(n.Args[1], in)
	if err != nil {
		return Value{}, err
	}
	return ev.node(n.Args[0], periodVal)
}

func (ev *evaluator) evalBound(n Node, in Value) (Value, error) {
	if len(n.Args) < 1 {
		return Value{}, &Error{Pos: n.Pos, Msg: "invalid bound expression"}
	}
	isEnd := n.Text == "end"
	if n.Text != "start" && n.Text != "end" {
		return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("unknown bound %q", n.Text)}
	}

	target := n.Args[0]
	if target.Op == "unit" {
		var anchor time.Time
		switch in.Kind {
		case KDate, KDateTime:
			anchor = in.T
		case KSpan:
			if isEnd {
				anchor = in.End.Add(-time.Nanosecond)
			} else {
				anchor = in.T
			}
		default:
			return Value{}, &Error{
				Pos: n.Pos,
				Msg: fmt.Sprintf("cannot take %s of %s", n.Text, kindName(in.Kind)),
			}
		}
		uStart, uEnd, dateUnit, err := unitBounds(target.Pos, anchor, target.Text, ev.ctx.Cfg)
		if err != nil {
			return Value{}, err
		}
		if isEnd {
			return Value{Kind: KDateTime, T: uEnd.Add(-time.Second), Local: in.Local}, nil
		}
		if dateUnit {
			return Value{Kind: KDate, T: uStart, Local: in.Local}, nil
		}
		return Value{Kind: KDateTime, T: uStart, Local: in.Local}, nil
	}

	tv, err := ev.node(target, in)
	if err != nil {
		return Value{}, err
	}
	switch tv.Kind {
	case KSpan:
		if isEnd {
			return Value{Kind: KDateTime, T: tv.End.Add(-time.Second), Local: tv.Local}, nil
		}
		if tv.Days {
			return Value{Kind: KDate, T: dateOnly(tv.T), Local: tv.Local}, nil
		}
		return Value{Kind: KDateTime, T: tv.T, Local: tv.Local}, nil
	case KDate:
		if isEnd {
			dEnd := dateOnly(tv.T).AddDate(0, 0, 1)
			return Value{Kind: KDateTime, T: dEnd.Add(-time.Second), Local: tv.Local}, nil
		}
		return tv, nil
	case KDateTime:
		return tv, nil
	default:
		return Value{}, &Error{
			Pos: n.Pos,
			Msg: fmt.Sprintf("cannot take %s of %s", n.Text, kindName(tv.Kind)),
		}
	}
}

func resolvePoint(ev *evaluator, pos int, v Value, fallbackDate time.Time) (time.Time, error) {
	switch v.Kind {
	case KDate, KDateTime:
		return v.T, nil
	case KTime:
		return ev.wallAt(pos, fallbackDate.Year(), fallbackDate.Month(), fallbackDate.Day(),
			v.T.Hour(), v.T.Minute(), v.T.Second(), fallbackDate.Location())
	default:
		return time.Time{}, &Error{
			Pos: pos,
			Msg: fmt.Sprintf("expected a date or time, got %s", kindName(v.Kind)),
		}
	}
}

func (ev *evaluator) todayStart() time.Time {
	y, m, d := ev.today()
	return time.Date(y, m, d, 0, 0, 0, 0, ev.loc)
}

func resolveSpanBounds(ev *evaluator, pos int, fromVal, toVal Value) (start, end time.Time, bothDates bool, local bool, err error) {
	if fromVal.Kind == KDate && toVal.Kind == KDate {
		if toVal.T.Before(fromVal.T) {
			return time.Time{}, time.Time{}, false, false, &Error{Pos: pos, Msg: "end of range is before start"}
		}
		return fromVal.T, toVal.T, true, fromVal.Local && toVal.Local, nil
	}
	today := ev.todayStart()
	start, err = resolvePoint(ev, pos, fromVal, today)
	if err != nil {
		return time.Time{}, time.Time{}, false, false, err
	}
	fallback := today
	if fromVal.Kind == KDate || fromVal.Kind == KDateTime {
		fallback = dateOnly(start)
	}
	end, err = resolvePoint(ev, pos, toVal, fallback)
	if err != nil {
		return time.Time{}, time.Time{}, false, false, err
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, false, false, &Error{Pos: pos, Msg: "end of range is before start"}
	}
	return start, end, false, fromVal.Local && toVal.Local, nil
}

func (ev *evaluator) evalSpan(n Node, in Value) (Value, error) {
	if len(n.Args) < 2 {
		return Value{}, &Error{Pos: n.Pos, Msg: "invalid span expression"}
	}
	fromVal, err := ev.node(n.Args[0], in)
	if err != nil {
		return Value{}, err
	}
	toVal, err := ev.node(n.Args[1], in)
	if err != nil {
		return Value{}, err
	}
	start, end, bothDates, local, err := resolveSpanBounds(ev, n.Pos, fromVal, toVal)
	if err != nil {
		return Value{}, err
	}
	if bothDates {
		return Value{
			Kind:  KSpan,
			T:     start,
			End:   dateOnly(end).AddDate(0, 0, 1),
			Days:  true,
			Local: local,
		}, nil
	}
	return Value{
		Kind:  KSpan,
		T:     start,
		End:   end,
		Days:  false,
		Local: local,
	}, nil
}

func (ev *evaluator) evalDuring(n Node) (Value, error) {
	if len(n.Args) < 1 {
		return Value{}, &Error{Pos: n.Pos, Msg: "invalid during expression"}
	}
	d, err := evalDur(n.Args[0])
	if err != nil {
		return Value{}, err
	}
	var start time.Time
	var days, local bool
	if d.Nanos == 0 {
		start = ev.todayStart()
		days = true
		local = true
	} else {
		start = ev.ctx.Now.In(ev.loc)
		days = false
		local = false
	}
	end, err := addDurationCtx(n.Pos, start, d, ev.ctx)
	if err != nil {
		return Value{}, err
	}
	if end.Before(start) {
		return Value{}, &Error{Pos: n.Pos, Msg: "end of span is before start"}
	}
	return Value{
		Kind:  KSpan,
		T:     start,
		End:   end,
		D:     d,
		Days:  days,
		Local: local,
	}, nil
}

func (ev *evaluator) evalStarting(n Node, in Value) (Value, error) {
	if len(n.Args) < 1 {
		return Value{}, &Error{Pos: n.Pos, Msg: "invalid starting expression"}
	}
	if in.Kind != KSpan {
		return Value{}, &Error{
			Pos: n.Pos,
			Msg: fmt.Sprintf("starting requires a span input, got %s", kindName(in.Kind)),
		}
	}
	sv, err := ev.node(n.Args[0], in)
	if err != nil {
		return Value{}, err
	}
	var start time.Time
	switch sv.Kind {
	case KDate, KDateTime:
		start = sv.T
	case KTime:
		base := dateOnly(in.T)
		start, err = ev.wallAt(n.Args[0].Pos, base.Year(), base.Month(), base.Day(),
			sv.T.Hour(), sv.T.Minute(), sv.T.Second(), base.Location())
		if err != nil {
			return Value{}, err
		}
	default:
		return Value{}, &Error{
			Pos: n.Pos,
			Msg: fmt.Sprintf("cannot start a span at %s", kindName(sv.Kind)),
		}
	}

	var end time.Time
	var days bool
	if in.D != (Duration{}) {
		days = in.Days && sv.Kind == KDate && in.D.Nanos == 0
		if days {
			start = dateOnly(start)
		}
		end, err = addDurationCtx(n.Pos, start, in.D, ev.ctx)
		if err != nil {
			return Value{}, err
		}
	} else if in.Days && sv.Kind == KDate {
		days = true
		start = dateOnly(start)
		lengthDays := daysBetween(in.T, in.End)
		end = start.AddDate(0, 0, lengthDays)
	} else {
		days = false
		end = start.Add(in.End.Sub(in.T))
	}

	return Value{
		Kind:  KSpan,
		T:     start,
		End:   end,
		D:     in.D,
		Days:  days,
		Local: sv.Local,
	}, nil
}

func isMidnight(t time.Time) bool {
	return t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0
}

func fractionalDayDiff(start, end time.Time) float64 {
	days := float64(daysBetween(start, end))
	if isMidnight(start) && isMidnight(end) {
		return days
	}
	startTOD := float64(time.Duration(start.Hour())*time.Hour+time.Duration(start.Minute())*time.Minute+time.Duration(start.Second())*time.Second+time.Duration(start.Nanosecond())) / float64(24*time.Hour)
	endTOD := float64(time.Duration(end.Hour())*time.Hour+time.Duration(end.Minute())*time.Minute+time.Duration(end.Second())*time.Second+time.Duration(end.Nanosecond())) / float64(24*time.Hour)
	return days + (endTOD - startTOD)
}

func countUnits(pos int, start, end time.Time, unit string, isWeekday bool, inclusive bool, ctx Context) (float64, error) {
	if isWeekday {
		wd, err := parseWeekday(pos, unit)
		if err != nil {
			return 0, err
		}
		sd := dateOnly(start)
		ed := dateOnly(end)
		cur := sd
		if !inclusive {
			cur = cur.AddDate(0, 0, 1)
		}
		if cur.After(ed) {
			return 0, nil
		}
		offset := (int(wd) - int(cur.Weekday()) + 7) % 7
		first := cur.AddDate(0, 0, offset)
		if first.After(ed) {
			return 0, nil
		}
		return float64(1 + daysBetween(first, ed)/7), nil
	}

	switch unit {
	case "day":
		days := fractionalDayDiff(start, end)
		if inclusive {
			days += 1
		}
		return days, nil
	case "week":
		days := fractionalDayDiff(start, end)
		if inclusive {
			days += 1
		}
		return days / 7.0, nil
	case "hour":
		return float64(end.Sub(start)) / float64(time.Hour), nil
	case "minute":
		return float64(end.Sub(start)) / float64(time.Minute), nil
	case "second":
		return float64(end.Sub(start)) / float64(time.Second), nil
	case "month", "quarter", "year":
		unitMonths := 1
		if unit == "quarter" {
			unitMonths = 3
		} else if unit == "year" {
			unitMonths = 12
		}
		effEnd := end
		if inclusive {
			effEnd = effEnd.AddDate(0, 0, 1)
		}
		for n := 0; ; n++ {
			next := addDuration(start, Duration{Months: (n + 1) * unitMonths}, ctx.Cfg)
			if next.After(effEnd) {
				return float64(n), nil
			}
		}
	case "businessday":
		count, err := countBusinessDays(pos, start, end, inclusive, true, ctx)
		if err != nil {
			return 0, err
		}
		return float64(count), nil
	default:
		return 0, &Error{Pos: pos, Msg: fmt.Sprintf("cannot count unit %q", unit)}
	}
}

func (ev *evaluator) evalCount(n Node, in Value, explicitInclusive bool) (Value, error) {
	if len(n.Args) < 3 {
		return Value{}, &Error{Pos: n.Pos, Msg: "invalid count expression"}
	}
	fromVal, err := ev.node(n.Args[1], in)
	if err != nil {
		return Value{}, err
	}
	toVal, err := ev.node(n.Args[2], in)
	if err != nil {
		return Value{}, err
	}
	start, end, _, _, err := resolveSpanBounds(ev, n.Pos, fromVal, toVal)
	if err != nil {
		return Value{}, err
	}
	incl := explicitInclusive || ev.ctx.Cfg.CountInclusive
	what := n.Args[0]
	var count float64
	switch what.Op {
	case "weekday":
		count, err = countUnits(what.Pos, start, end, what.Text, true, incl, ev.ctx)
	case "unit":
		count, err = countUnits(what.Pos, start, end, what.Text, false, incl, ev.ctx)
	default:
		return Value{}, &Error{Pos: what.Pos, Msg: fmt.Sprintf("cannot count %q", what.Op)}
	}
	if err != nil {
		return Value{}, err
	}
	return Value{Kind: KNumber, N: count}, nil
}

func (ev *evaluator) evalTo(n Node, in Value, explicitInclusive bool) (Value, error) {
	if len(n.Args) < 1 {
		return Value{}, &Error{Pos: n.Pos, Msg: "invalid to expression"}
	}
	target := n.Args[0]
	switch target.Op {
	case "key":
		switch target.Text {
		case "cron":
			return ev.evalToCron(n.Pos, in)
		case "ics":
			return ev.evalToICS(n.Pos, in)
		case "grid":
			return ev.evalToGrid(n.Pos, in)
		default:
			return Value{}, &Error{Pos: target.Pos, Msg: fmt.Sprintf("%q aren't available yet", target.Text)}
		}
	case "unit":
		switch in.Kind {
		case KSpan:
			if target.Text == "businessday" {
				res, err := countSpanBusinessDays(target.Pos, in, ev.ctx)
				if err != nil {
					return Value{}, err
				}
				return Value{Kind: KNumber, N: float64(res)}, nil
			}
			res, err := countUnits(target.Pos, in.T, in.End, target.Text, false, false, ev.ctx)
			if err != nil {
				return Value{}, err
			}
			return Value{Kind: KNumber, N: res}, nil
		case KDur:
			if in.HasEnds {
				var res float64
				var err error
				if target.Text == "businessday" {
					incl := explicitInclusive || ev.ctx.Cfg.CountInclusive
					cnt, cerr := countBusinessDays(target.Pos, in.T, in.End, incl, true, ev.ctx)
					res, err = float64(cnt), cerr
				} else {
					res, err = countUnits(target.Pos, in.T, in.End, target.Text, false, explicitInclusive, ev.ctx)
				}
				if err != nil {
					return Value{}, err
				}
				if in.D.Months < 0 || in.D.Days < 0 || in.D.BizDays < 0 || in.D.Nanos < 0 {
					res = -res
				}
				return Value{Kind: KNumber, N: res}, nil
			}
			if in.D.BizDays != 0 {
				if target.Text == "businessday" {
					return Value{Kind: KNumber, N: float64(in.D.BizDays)}, nil
				}
				return Value{}, &Error{
					Pos: n.Pos,
					Msg: fmt.Sprintf("a business day has no fixed length; give a start: during ..., to %ss", target.Text),
				}
			}
			if target.Text == "businessday" {
				return Value{}, &Error{
					Pos: target.Pos,
					Msg: "business days depend on the calendar; give a span or date difference",
				}
			}
			if in.D.Months != 0 {
				return Value{}, &Error{
					Pos: n.Pos,
					Msg: "a month has no fixed length; give a start: during 3 months, to days",
				}
			}
			totalNanos := float64(in.D.Days)*float64(24*time.Hour) + float64(in.D.Nanos)
			switch target.Text {
			case "day":
				return Value{Kind: KNumber, N: totalNanos / float64(24*time.Hour)}, nil
			case "week":
				return Value{Kind: KNumber, N: totalNanos / float64(7*24*time.Hour)}, nil
			case "hour":
				return Value{Kind: KNumber, N: totalNanos / float64(time.Hour)}, nil
			case "minute":
				return Value{Kind: KNumber, N: totalNanos / float64(time.Minute)}, nil
			case "second":
				return Value{Kind: KNumber, N: totalNanos / float64(time.Second)}, nil
			case "month", "quarter", "year":
				return Value{}, &Error{
					Pos: n.Pos,
					Msg: fmt.Sprintf("a %s has no fixed length; give a start: during ..., to %s", target.Text, target.Text),
				}
			default:
				return Value{}, &Error{Pos: target.Pos, Msg: fmt.Sprintf("cannot convert duration to %q", target.Text)}
			}
		default:
			return Value{}, &Error{
				Pos: n.Pos,
				Msg: fmt.Sprintf("cannot convert %s to %s", kindName(in.Kind), target.Text),
			}
		}
	case "zone":
		loc, err := ev.lookupZoneNode(target)
		if err != nil {
			return Value{}, err
		}
		return convertToZone(n.Pos, in, loc, ev)
	default:
		return Value{}, &Error{Pos: target.Pos, Msg: fmt.Sprintf("unsupported to target %q", target.Op)}
	}
}
