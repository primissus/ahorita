package expr

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

func evalDur(n Node) (Duration, error) {
	var d Duration
	var hasBiz, hasOther bool
	for _, part := range n.Args {
		switch part.Op {
		case "businessday":
			hasBiz = true
			if hasOther {
				return Duration{}, &Error{Pos: part.Pos, Msg: "business days can't be combined with other units"}
			}
			count, err := strconv.Atoi(part.Text)
			if err != nil {
				if strings.ContainsRune(part.Text, '.') {
					return Duration{}, &Error{Pos: part.Pos, Msg: "fractional values aren't supported for business days"}
				}
				return Duration{}, &Error{Pos: part.Pos, Msg: fmt.Sprintf("invalid business day count %q", part.Text)}
			}
			d.BizDays += count
		case "millisecond", "second", "minute", "hour":
			hasOther = true
			if hasBiz {
				return Duration{}, &Error{Pos: part.Pos, Msg: "business days can't be combined with other units"}
			}
			f, err := strconv.ParseFloat(part.Text, 64)
			if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
				return Duration{}, &Error{Pos: part.Pos, Msg: fmt.Sprintf("invalid duration value %q", part.Text)}
			}
			var unit time.Duration
			switch part.Op {
			case "millisecond":
				unit = time.Millisecond
			case "second":
				unit = time.Second
			case "minute":
				unit = time.Minute
			case "hour":
				unit = time.Hour
			}
			prod := f * float64(unit)
			if prod > float64(math.MaxInt64) || prod < float64(math.MinInt64) {
				return Duration{}, &Error{Pos: part.Pos, Msg: "duration out of range"}
			}
			d.Nanos += time.Duration(math.Round(prod))
		case "day", "week", "month", "quarter", "half", "year":
			hasOther = true
			if hasBiz {
				return Duration{}, &Error{Pos: part.Pos, Msg: "business days can't be combined with other units"}
			}
			count, err := strconv.Atoi(part.Text)
			if err != nil {
				if strings.ContainsRune(part.Text, '.') {
					return Duration{}, &Error{Pos: part.Pos, Msg: fmt.Sprintf("fractional values aren't supported for %ss", part.Op)}
				}
				return Duration{}, &Error{Pos: part.Pos, Msg: fmt.Sprintf("invalid %s count %q", part.Op, part.Text)}
			}
			switch part.Op {
			case "day":
				d.Days += count
			case "week":
				d.Days += 7 * count
			case "month":
				d.Months += count
			case "quarter":
				d.Months += 3 * count
			case "half":
				d.Months += 6 * count
			case "year":
				d.Months += 12 * count
			}
		default:
			return Duration{}, &Error{Pos: part.Pos, Msg: fmt.Sprintf("%s can't be used as a duration", printUnitSingular(part.Op))}
		}
	}
	return d, nil
}

func addDuration(t time.Time, d Duration, cfg Config) time.Time {
	if d.Months != 0 {
		if cfg.MonthOverflow {
			t = t.AddDate(0, d.Months, 0)
		} else {
			y, m, day := t.Date()
			h, mi, s := t.Clock()
			nsec := t.Nanosecond()
			loc := t.Location()
			totalMonths := y*12 + int(m-1) + d.Months
			ty := totalMonths / 12
			rem := totalMonths % 12
			if rem < 0 {
				ty--
				rem += 12
			}
			tm := time.Month(rem + 1)
			maxDay := daysInMonth(ty, tm)
			if day > maxDay {
				day = maxDay
			}
			t = time.Date(ty, tm, day, h, mi, s, nsec, loc)
		}
	}
	if d.Days != 0 {
		t = t.AddDate(0, 0, d.Days)
	}
	if d.Nanos != 0 {
		t = t.Add(d.Nanos)
	}
	return t
}

func addDurationCtx(pos int, t time.Time, d Duration, ctx Context) (time.Time, error) {
	t = addDuration(t, d, ctx.Cfg)
	if d.BizDays != 0 {
		return addBusinessDays(pos, t, d.BizDays, ctx)
	}
	return t, nil
}

func negateDuration(d Duration) Duration {
	return Duration{
		Months:  -d.Months,
		Days:    -d.Days,
		BizDays: -d.BizDays,
		Nanos:   -d.Nanos,
	}
}

func hasNonBiz(d Duration) bool {
	return d.Months != 0 || d.Days != 0 || d.Nanos != 0
}

func evalAdd(pos int, lhs, rhs Value, ctx Context) (Value, error) {
	if rhs.Kind == KDur {
		return shiftValueByDur(pos, lhs, rhs.D, ctx, false)
	}
	return Value{}, &Error{
		Pos: pos,
		Msg: fmt.Sprintf("cannot add %s to %s", kindName(rhs.Kind), kindName(lhs.Kind)),
	}
}

func evalSub(pos int, lhs, rhs Value, ctx Context) (Value, error) {
	if rhs.Kind == KDur {
		return shiftValueByDur(pos, lhs, negateDuration(rhs.D), ctx, true)
	}
	if (lhs.Kind == KDate || lhs.Kind == KDateTime) && (rhs.Kind == KDate || rhs.Kind == KDateTime) {
		if lhs.Kind == KDate && rhs.Kind == KDate {
			days := daysBetween(rhs.T, lhs.T)
			t, end := rhs.T, lhs.T
			if days < 0 {
				t, end = lhs.T, rhs.T
			}
			return Value{
				Kind:    KDur,
				T:       t,
				End:     end,
				D:       Duration{Days: days},
				Days:    true,
				HasEnds: true,
			}, nil
		}
		diff := lhs.T.Sub(rhs.T)
		t, end := rhs.T, lhs.T
		if t.After(end) {
			t, end = end, t
		}
		return Value{
			Kind:    KDur,
			T:       t,
			End:     end,
			D:       Duration{Nanos: diff},
			Days:    false,
			HasEnds: true,
		}, nil
	}
	return Value{}, &Error{
		Pos: pos,
		Msg: fmt.Sprintf("cannot subtract %s from %s", kindName(rhs.Kind), kindName(lhs.Kind)),
	}
}

func shiftValueByDur(pos int, lhs Value, d Duration, ctx Context, isSub bool) (Value, error) {
	switch lhs.Kind {
	case KDate:
		base := lhs.T
		if lhs.TodayNow {
			base = ctx.Now.In(ctxLoc(ctx))
		}
		t, err := addDurationCtx(pos, base, d, ctx)
		if err != nil {
			return Value{}, err
		}
		if lhs.TodayNow {
			return Value{Kind: KDateTime, T: t, Local: false}, nil
		}
		if d.Nanos == 0 {
			return Value{Kind: KDate, T: t, Local: lhs.Local}, nil
		}
		return Value{Kind: KDateTime, T: t, Local: lhs.Local}, nil
	case KDateTime:
		t, err := addDurationCtx(pos, lhs.T, d, ctx)
		if err != nil {
			return Value{}, err
		}
		return Value{Kind: KDateTime, T: t, Local: lhs.Local}, nil
	case KTime:
		if d.Months != 0 || d.Days != 0 || d.BizDays != 0 {
			if isSub {
				return Value{}, &Error{Pos: pos, Msg: "cannot subtract calendar units from a time of day"}
			}
			return Value{}, &Error{Pos: pos, Msg: "cannot add calendar units to a time of day"}
		}
		return addTimeNanos(lhs, d.Nanos), nil
	case KSpan:
		t, err := addDurationCtx(pos, lhs.T, d, ctx)
		if err != nil {
			return Value{}, err
		}
		end, err := addDurationCtx(pos, lhs.End, d, ctx)
		if err != nil {
			return Value{}, err
		}
		return Value{
			Kind:  KSpan,
			T:     t,
			End:   end,
			Days:  lhs.Days && d.Nanos == 0,
			D:     lhs.D,
			Local: lhs.Local,
		}, nil
	case KDur:
		if (lhs.D.BizDays != 0 && hasNonBiz(d)) || (d.BizDays != 0 && hasNonBiz(lhs.D)) {
			return Value{}, &Error{Pos: pos, Msg: "business days can't be combined with other units"}
		}
		return Value{
			Kind: KDur,
			D: Duration{
				Months:  lhs.D.Months + d.Months,
				Days:    lhs.D.Days + d.Days,
				BizDays: lhs.D.BizDays + d.BizDays,
				Nanos:   lhs.D.Nanos + d.Nanos,
			},
		}, nil
	default:
		if isSub {
			return Value{}, &Error{Pos: pos, Msg: fmt.Sprintf("cannot subtract a duration from %s", kindName(lhs.Kind))}
		}
		return Value{}, &Error{Pos: pos, Msg: fmt.Sprintf("cannot add a duration to %s", kindName(lhs.Kind))}
	}
}

func evalMul(pos int, lhs, rhs Value) (Value, error) {
	var dur Duration
	var factor float64
	switch {
	case lhs.Kind == KDur && rhs.Kind == KNumber:
		dur = lhs.D
		factor = rhs.N
	case lhs.Kind == KNumber && rhs.Kind == KDur:
		dur = rhs.D
		factor = lhs.N
	default:
		return Value{}, &Error{
			Pos: pos,
			Msg: fmt.Sprintf("cannot multiply %s by %s", kindName(lhs.Kind), kindName(rhs.Kind)),
		}
	}

	if (dur.Months != 0 || dur.Days != 0 || dur.BizDays != 0) && factor != math.Trunc(factor) {
		return Value{}, &Error{
			Pos: pos,
			Msg: "cannot multiply months or days by a fractional number",
		}
	}
	intFactor := int(factor)
	prod := float64(dur.Nanos) * factor
	if prod > float64(math.MaxInt64) || prod < float64(math.MinInt64) {
		return Value{}, &Error{Pos: pos, Msg: "duration out of range"}
	}
	return Value{
		Kind: KDur,
		D: Duration{
			Months:  dur.Months * intFactor,
			Days:    dur.Days * intFactor,
			BizDays: dur.BizDays * intFactor,
			Nanos:   time.Duration(math.Round(prod)),
		},
	}, nil
}

func addTimeNanos(v Value, delta time.Duration) Value {
	const day = 24 * time.Hour
	h, mi, s := v.T.Clock()
	cur := time.Duration(h)*time.Hour + time.Duration(mi)*time.Minute + time.Duration(s)*time.Second + time.Duration(v.T.Nanosecond())
	total := ((cur+delta)%day + day) % day
	nh := int(total / time.Hour)
	nmi := int((total % time.Hour) / time.Minute)
	ns := int((total % time.Minute) / time.Second)
	nnsec := int(total % time.Second)
	y, m, d := v.T.Date()
	return Value{
		Kind:  KTime,
		T:     time.Date(y, m, d, nh, nmi, ns, nnsec, v.T.Location()),
		Local: v.Local,
	}
}

func daysFromCivil(y int, m time.Month, d int) int {
	yy := y
	if m <= 2 {
		yy--
	}
	era := yy / 400
	if yy < 0 && yy%400 != 0 {
		era--
	}
	yoe := yy - era*400
	mm := int(m)
	var doy int
	if mm > 2 {
		doy = (153*(mm-3)+2)/5 + d - 1
	} else {
		doy = (153*(mm+9)+2)/5 + d - 1
	}
	doe := yoe*365 + yoe/4 - yoe/100 + doy
	return era*146097 + doe - 719468
}

func daysBetween(from, to time.Time) int {
	fy, fm, fd := from.Date()
	ty, tm, td := to.Date()
	return daysFromCivil(ty, tm, td) - daysFromCivil(fy, fm, fd)
}

func kindName(k Kind) string {
	switch k {
	case KDate:
		return "a date"
	case KDateTime:
		return "a date-time"
	case KTime:
		return "a time"
	case KDur:
		return "a duration"
	case KSpan:
		return "a span"
	case KNumber:
		return "a number"
	case KRecurrence:
		return "a recurrence"
	case KList:
		return "a list"
	case KText:
		return "text"
	default:
		return "a value"
	}
}
