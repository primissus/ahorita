package expr

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/teambition/rrule-go"
)

const maxTakeCount = 1000

var rruleWeekdays = [7]rrule.Weekday{
	rrule.SU, // time.Sunday = 0
	rrule.MO, // time.Monday = 1
	rrule.TU, // time.Tuesday = 2
	rrule.WE, // time.Wednesday = 3
	rrule.TH, // time.Thursday = 4
	rrule.FR, // time.Friday = 5
	rrule.SA, // time.Saturday = 6
}

func timeWeekdayToRRule(wd time.Weekday) rrule.Weekday {
	return rruleWeekdays[int(wd)%7]
}

func rruleWeekdayFromKey(key string) (rrule.Weekday, bool) {
	switch key {
	case "monday":
		return rrule.MO, true
	case "tuesday":
		return rrule.TU, true
	case "wednesday":
		return rrule.WE, true
	case "thursday":
		return rrule.TH, true
	case "friday":
		return rrule.FR, true
	case "saturday":
		return rrule.SA, true
	case "sunday":
		return rrule.SU, true
	default:
		return rrule.MO, false
	}
}

func everyFreqFromUnit(unit string) (rrule.Frequency, bool) {
	switch unit {
	case "minute":
		return rrule.MINUTELY, true
	case "hour":
		return rrule.HOURLY, true
	case "day":
		return rrule.DAILY, true
	case "week":
		return rrule.WEEKLY, true
	case "month":
		return rrule.MONTHLY, true
	case "year":
		return rrule.YEARLY, true
	default:
		return rrule.DAILY, false
	}
}

func isDailyOrSlower(freq rrule.Frequency) bool {
	switch freq {
	case rrule.YEARLY, rrule.MONTHLY, rrule.WEEKLY, rrule.DAILY:
		return true
	default:
		return false
	}
}

func (ev *evaluator) evalEvery(n Node) (Value, error) {
	if len(n.Args) == 0 {
		return Value{}, &Error{Pos: n.Pos, Msg: "invalid every expression"}
	}

	args := n.Args
	var (
		hasClock bool
		h, mi, s int
	)
	if args[len(args)-1].Op == "clock" {
		clk := args[len(args)-1]
		var err error
		h, mi, s, err = parseClockParts(clk.Text, clk.Pos)
		if err != nil {
			return Value{}, err
		}
		hasClock = true
		args = args[:len(args)-1]
	}

	y, mo, da := ev.today()
	dtstart := time.Date(y, mo, da, h, mi, s, 0, ev.loc)
	opt := rrule.ROption{
		Dtstart: dtstart,
		Wkst:    timeWeekdayToRRule(ev.ctx.Cfg.WeekStart),
	}
	if hasClock {
		opt.Byhour = []int{h}
		opt.Byminute = []int{mi}
		opt.Bysecond = []int{s}
	}

	if err := fillEveryOption(args, ev.ctx.Cfg.WeekStart, &opt); err != nil {
		return Value{}, err
	}

	rule, err := rrule.NewRRule(opt)
	if err != nil {
		return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("invalid recurrence: %v", err)}
	}

	return Value{
		Kind: KRecurrence,
		T:    ev.ctx.Now.In(ev.loc),
		Days: !hasClock && isDailyOrSlower(opt.Freq),
		Str:  rule.String(),
	}, nil
}

func fillEveryOption(args []Node, weekStart time.Weekday, opt *rrule.ROption) error {
	switch len(args) {
	case 1:
		a := args[0]
		switch a.Op {
		case "weekday":
			wd, ok := rruleWeekdayFromKey(a.Text)
			if !ok {
				return &Error{Pos: a.Pos, Msg: fmt.Sprintf("unknown weekday %q", a.Text)}
			}
			opt.Freq = rrule.WEEKLY
			opt.Byweekday = []rrule.Weekday{wd}
			return nil
		case "unit":
			freq, ok := everyFreqFromUnit(a.Text)
			if !ok {
				return &Error{Pos: a.Pos, Msg: fmt.Sprintf("unsupported recurrence unit %q", a.Text)}
			}
			opt.Freq = freq
			return nil
		case "key":
			if a.Text == "weekdaykw" {
				opt.Freq = rrule.WEEKLY
				opt.Byweekday = []rrule.Weekday{rrule.MO, rrule.TU, rrule.WE, rrule.TH, rrule.FR}
				return nil
			}
		}

	case 2:
		a, b := args[0], args[1]
		switch {
		case a.Op == "num" && b.Op == "unit":
			interval, err := strconv.Atoi(a.Text)
			if err != nil || interval < 1 {
				return &Error{Pos: a.Pos, Msg: "interval must be at least 1"}
			}
			freq, ok := everyFreqFromUnit(b.Text)
			if !ok {
				return &Error{Pos: b.Pos, Msg: fmt.Sprintf("unsupported recurrence unit %q", b.Text)}
			}
			opt.Freq = freq
			opt.Interval = interval
			return nil

		case a.Op == "unit" && (a.Text == "day" || a.Text == "month") && b.Op == "ord":
			ord, err := parseOrdinalBound(b, 1, 31, "day of month")
			if err != nil {
				return err
			}
			opt.Freq = rrule.MONTHLY
			opt.Bymonthday = []int{ord}
			return nil

		case a.Op == "key" && a.Text == "weekdaykw" && b.Op == "ord":
			ord, err := parseOrdinalBound(b, 1, 7, "day of week")
			if err != nil {
				return err
			}
			wd := time.Weekday((int(weekStart) + (ord - 1)) % 7)
			opt.Freq = rrule.WEEKLY
			opt.Byweekday = []rrule.Weekday{timeWeekdayToRRule(wd)}
			return nil

		case a.Op == "key" && a.Text == "dayyear" && b.Op == "ord":
			ord, err := parseOrdinalBound(b, 1, 366, "day of year")
			if err != nil {
				return err
			}
			opt.Freq = rrule.YEARLY
			opt.Byyearday = []int{ord}
			return nil

		case a.Op == "ord" && b.Op == "weekday":
			ord, err := parseOrdinalBound(a, 1, 5, "weekday occurrence")
			if err != nil {
				return err
			}
			wd, ok := rruleWeekdayFromKey(b.Text)
			if !ok {
				return &Error{Pos: b.Pos, Msg: fmt.Sprintf("unknown weekday %q", b.Text)}
			}
			opt.Freq = rrule.MONTHLY
			opt.Byweekday = []rrule.Weekday{wd.Nth(ord)}
			return nil

		case a.Op == "unit" && a.Text == "year" && b.Op == "month" && len(b.Args) == 1 && b.Args[0].Op == "num":
			m, err := parseMonthName(b.Pos, b.Text)
			if err != nil {
				return err
			}
			d, err := strconv.Atoi(b.Args[0].Text)
			if err != nil || d < 1 || d > daysInMonth(2024, m) {
				return &Error{Pos: b.Args[0].Pos, Msg: fmt.Sprintf("day %q out of range for %s", b.Args[0].Text, b.Text)}
			}
			opt.Freq = rrule.YEARLY
			opt.Bymonth = []int{int(m)}
			opt.Bymonthday = []int{d}
			return nil
		}
	}
	return &Error{Pos: args[0].Pos, Msg: "invalid every schedule"}
}

func parseOrdinalBound(n Node, min, max int, label string) (int, error) {
	v, err := strconv.Atoi(n.Text)
	if err != nil || v < min || v > max {
		return 0, &Error{Pos: n.Pos, Msg: fmt.Sprintf("%s %q must be between %d and %d", label, n.Text, min, max)}
	}
	return v, nil
}

func (ev *evaluator) evalTake(n Node, in Value) (Value, error) {
	count, err := strconv.Atoi(n.Text)
	if err != nil || count < 1 || count > maxTakeCount {
		return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("next count must be between 1 and %d (got %q)", maxTakeCount, n.Text)}
	}
	if in.Kind != KRecurrence {
		return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("cannot take next %d from %s", count, kindName(in.Kind))}
	}
	list, err := recurTake(in, count)
	if err != nil {
		return Value{}, &Error{Pos: n.Pos, Msg: err.Error()}
	}
	list.Note = in.Note
	return list, nil
}

func recurString(v Value) string {
	if v.T.IsZero() && !strings.Contains(v.Str, "DTSTART") {
		return v.Str
	}
	list, err := recurTake(v, 5)
	if err != nil {
		return v.Str
	}
	return list.String()
}

func recurTake(v Value, count int) (Value, error) {
	if strings.HasPrefix(v.Str, "CRON ") {
		return cronTake(v, count)
	}
	rule, loc, err := parseRecurrenceRRule(v)
	if err != nil {
		return Value{}, err
	}
	after := v.T.In(loc)
	if v.T.IsZero() {
		after = rule.GetDTStart().Add(-time.Nanosecond)
	}
	items := make([]Value, 0, count)
	iter := rule.Iterator()
	for len(items) < count {
		occ, ok := iter()
		if !ok {
			break
		}
		if !occ.After(after) {
			continue
		}
		occ = occ.In(loc)
		if v.Days {
			items = append(items, Value{
				Kind: KDate,
				T:    time.Date(occ.Year(), occ.Month(), occ.Day(), 0, 0, 0, 0, loc),
				Lang: v.Lang,
			})
		} else {
			items = append(items, Value{
				Kind: KDateTime,
				T:    occ,
				Lang: v.Lang,
			})
		}
	}
	return Value{Kind: KList, Items: items, Lang: v.Lang}, nil
}

func parseRecurrenceRRule(v Value) (*rrule.RRule, *time.Location, error) {
	loc := v.T.Location()
	if loc == nil {
		loc = time.UTC
	}
	opt, err := parseROptionInLoc(v.Str, loc)
	if err != nil {
		return nil, nil, err
	}
	if opt.Dtstart.IsZero() {
		if v.T.IsZero() {
			return nil, nil, fmt.Errorf("recurrence has no start reference")
		}
		y, m, d := v.T.In(loc).Date()
		opt.Dtstart = time.Date(y, m, d, 0, 0, 0, 0, loc)
	}
	rule, err := rrule.NewRRule(*opt)
	if err != nil {
		return nil, nil, err
	}
	return rule, loc, nil
}

func parseROptionInLoc(rfc string, loc *time.Location) (*rrule.ROption, error) {
	opt, err := rrule.StrToROptionInLocation(rfc, loc)
	if err == nil {
		return opt, nil
	}
	// If loc is a FixedZone (not in tzdb), rrule-go's parseTZID rejects TZID=<offset>.
	// Fall back to parsing the DTSTART timestamp directly in the known loc.
	lines := strings.Split(strings.TrimSpace(rfc), "\n")
	if len(lines) == 2 && strings.HasPrefix(lines[0], "DTSTART;TZID=") {
		if colon := strings.IndexByte(lines[0], ':'); colon >= 0 {
			fallback := "DTSTART" + lines[0][colon:] + "\n" + lines[1]
			if opt2, err2 := rrule.StrToROptionInLocation(fallback, loc); err2 == nil {
				return opt2, nil
			}
		}
	}
	return nil, err
}

func rruleWeekdayToTime(wd rrule.Weekday) time.Weekday {
	// rrule.Weekday.Day() has MO=0..SA=5, SU=6.
	return time.Weekday((wd.Day() + 1) % 7)
}

func convertRecurrenceToZone(pos int, v Value, targetLoc *time.Location) (Value, error) {
	if strings.HasPrefix(v.Str, "CRON ") {
		return convertCronToZone(pos, v, targetLoc)
	}
	rule, srcLoc, err := parseRecurrenceRRule(v)
	if err != nil {
		return Value{}, &Error{Pos: pos, Msg: fmt.Sprintf("invalid recurrence: %v", err)}
	}
	opt := rule.OrigOptions
	if opt.Dtstart.IsZero() {
		opt.Dtstart = rule.GetDTStart()
	}

	if opt.Freq == rrule.MINUTELY || opt.Freq == rrule.HOURLY {
		opt.Dtstart = opt.Dtstart.In(targetLoc)
		newRule, err := rrule.NewRRule(opt)
		if err != nil {
			return Value{}, &Error{Pos: pos, Msg: fmt.Sprintf("invalid recurrence: %v", err)}
		}
		note := joinNotes(v.Note, buildCronZoneNote(srcLoc, targetLoc, v.T.Year(), 0, nil, nil))
		return Value{
			Kind: KRecurrence,
			T:    v.T.In(targetLoc),
			Days: false,
			Str:  newRule.String(),
			Note: note,
			Lang: v.Lang,
		}, nil
	}

	list, err := recurTake(v, 1)
	if err != nil || len(list.Items) == 0 {
		return Value{}, &Error{Pos: pos, Msg: "recurrence has no upcoming occurrence to convert"}
	}
	anchor := list.Items[0].T.In(srcLoc)
	converted := anchor.In(targetLoc)
	dayDelta := daysFromCivil(converted.Date()) - daysFromCivil(anchor.Date())

	opt.Byhour = []int{converted.Hour()}
	opt.Byminute = []int{converted.Minute()}
	opt.Bysecond = []int{converted.Second()}

	var origWDs, newWDs []time.Weekday
	switch opt.Freq {
	case rrule.WEEKLY:
		if len(opt.Byweekday) > 0 {
			origWDs = make([]time.Weekday, len(opt.Byweekday))
			newWDs = make([]time.Weekday, len(opt.Byweekday))
			shifted := make([]rrule.Weekday, len(opt.Byweekday))
			for i, wd := range opt.Byweekday {
				ow := rruleWeekdayToTime(wd)
				nw := time.Weekday((int(ow) + dayDelta + 7) % 7)
				origWDs[i] = ow
				newWDs[i] = nw
				sw := timeWeekdayToRRule(nw)
				if n := wd.N(); n != 0 {
					sw = sw.Nth(n)
				}
				shifted[i] = sw
			}
			opt.Byweekday = shifted
		} else if dayDelta != 0 {
			ow := opt.Dtstart.In(srcLoc).Weekday()
			nw := time.Weekday((int(ow) + dayDelta + 7) % 7)
			origWDs = []time.Weekday{ow}
			newWDs = []time.Weekday{nw}
		}

	case rrule.MONTHLY:
		if len(opt.Byweekday) > 0 {
			if dayDelta != 0 {
				return Value{}, &Error{Pos: pos, Msg: "cannot shift monthly Nth-weekday rule across midnight to another timezone"}
			}
		} else if len(opt.Bymonthday) > 0 && dayDelta != 0 {
			shifted := make([]int, len(opt.Bymonthday))
			for i, md := range opt.Bymonthday {
				nmd := md + dayDelta
				if nmd < 1 || nmd > 28 {
					return Value{}, &Error{Pos: pos, Msg: fmt.Sprintf("cannot shift day %d across month boundary in %s", md, targetLoc)}
				}
				shifted[i] = nmd
			}
			opt.Bymonthday = shifted
		}

	case rrule.YEARLY:
		if len(opt.Bymonth) == 1 && len(opt.Bymonthday) == 1 && dayDelta != 0 {
			opt.Bymonth = []int{int(converted.Month())}
			opt.Bymonthday = []int{converted.Day()}
		} else if len(opt.Byyearday) == 1 && dayDelta != 0 {
			opt.Byyearday = []int{converted.YearDay()}
		}
	}

	opt.Dtstart = opt.Dtstart.In(targetLoc)
	newRule, err := rrule.NewRRule(opt)
	if err != nil {
		return Value{}, &Error{Pos: pos, Msg: fmt.Sprintf("invalid recurrence after timezone conversion: %v", err)}
	}
	note := joinNotes(v.Note, buildCronZoneNote(srcLoc, targetLoc, v.T.Year(), dayDelta, origWDs, newWDs))
	return Value{
		Kind: KRecurrence,
		T:    v.T.In(targetLoc),
		Days: false,
		Str:  newRule.String(),
		Note: note,
		Lang: v.Lang,
	}, nil
}

func interpretRecurrenceInZone(pos int, v Value, targetLoc *time.Location, ev *evaluator) (Value, error) {
	if strings.HasPrefix(v.Str, "CRON ") {
		return Value{
			Kind: KRecurrence,
			T:    v.T.In(targetLoc),
			Days: v.Days,
			Str:  v.Str,
			Note: v.Note,
			Lang: v.Lang,
		}, nil
	}
	rule, srcLoc, err := parseRecurrenceRRule(v)
	if err != nil {
		return Value{}, &Error{Pos: pos, Msg: fmt.Sprintf("invalid recurrence: %v", err)}
	}
	opt := rule.OrigOptions
	dt := rule.GetDTStart().In(srcLoc)
	newDT, err := ev.wallAt(pos, dt.Year(), dt.Month(), dt.Day(), dt.Hour(), dt.Minute(), dt.Second(), targetLoc)
	if err != nil {
		return Value{}, err
	}
	opt.Dtstart = newDT
	newRule, err := rrule.NewRRule(opt)
	if err != nil {
		return Value{}, &Error{Pos: pos, Msg: fmt.Sprintf("invalid recurrence: %v", err)}
	}
	return Value{
		Kind: KRecurrence,
		T:    ev.ctx.Now.In(targetLoc),
		Days: v.Days,
		Str:  newRule.String(),
		Note: v.Note,
		Lang: v.Lang,
	}, nil
}
