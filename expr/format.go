package expr

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Options controls how Format converts and renders a Value.
type Options struct {
	Type    string // "", "date", "datetime", "time", "range", "relative"
	Pattern string // "", a preset name, or a Java-style pattern
}

// Format renders v according to ctx and o.
// With empty options, it returns v.String() byte for byte.
func Format(v Value, ctx Context, o Options) (string, error) {
	if o.Type == "" && o.Pattern == "" {
		return v.String(), nil
	}

	switch o.Type {
	case "", "date", "datetime", "time", "range":
	case "relative":
		return formatRelative(v, ctx)
	default:
		return "", fmt.Errorf("unknown output type %q; valid types: date, datetime, time, range, relative", o.Type)
	}

	converted, err := convertType(v, ctx, o.Type)
	if err != nil {
		return "", err
	}

	return renderValue(converted, ctx, o.Pattern)
}

func resolveLang(v Value, ctx Context) Lang {
	lang := v.Lang
	if lang == "" {
		lang = ctx.Cfg.Lang
	}
	if lang == ES {
		return ES
	}
	return EN
}

type presetPatterns struct {
	date     string
	dateTime string
	timeOnly string
}

var namedPresets = map[string]map[Lang]presetPatterns{
	"human": {
		EN: {date: "EEE, MMM d, yyyy", dateTime: "EEE, MMM d, yyyy, h:mm a", timeOnly: "h:mm a"},
		ES: {date: "EEE d MMM yyyy", dateTime: "EEE d MMM yyyy, HH:mm", timeOnly: "HH:mm"},
	},
	"short": {
		EN: {date: "MM-dd-yyyy", dateTime: "MM-dd-yyyy HH:mm", timeOnly: "HH:mm"},
		ES: {date: "dd-MM-yyyy", dateTime: "dd-MM-yyyy HH:mm", timeOnly: "HH:mm"},
	},
}

func renderValue(v Value, ctx Context, pattern string) (string, error) {
	if pattern == "" {
		return v.String(), nil
	}

	preset := strings.ToLower(pattern)
	if preset == "iso" {
		return v.String(), nil
	}

	switch v.Kind {
	case KSpan:
		endKind := KDateTime
		endT := v.End
		if v.Days {
			endKind = KDate
			endT = v.End.AddDate(0, 0, -1)
		}
		left, err := renderValue(Value{Kind: endKind, T: v.T, Local: v.Local, Lang: v.Lang}, ctx, pattern)
		if err != nil {
			return "", err
		}
		right, err := renderValue(Value{Kind: endKind, T: endT, Local: v.Local, Lang: v.Lang}, ctx, pattern)
		if err != nil {
			return "", err
		}
		return left + " → " + right, nil

	case KList:
		lines := make([]string, len(v.Items))
		for i, item := range v.Items {
			if item.Lang == "" {
				item.Lang = v.Lang
			}
			s, err := renderValue(item, ctx, pattern)
			if err != nil {
				return "", err
			}
			if item.Str != "" && item.Kind != KText && item.Kind != KRecurrence {
				s += "  " + item.Str
			}
			lines[i] = s
		}
		return strings.Join(lines, "\n"), nil
	}

	switch preset {
	case "rfc3339":
		switch v.Kind {
		case KDate:
			return dateOnly(v.T).Format(time.RFC3339), nil
		case KDateTime:
			return v.T.Format(time.RFC3339), nil
		default:
			return "", fmt.Errorf("preset %q doesn't apply to %s", "rfc3339", kindName(v.Kind))
		}
	case "unix":
		switch v.Kind {
		case KDate:
			return strconv.FormatInt(dateOnly(v.T).Unix(), 10), nil
		case KDateTime:
			return strconv.FormatInt(v.T.Unix(), 10), nil
		default:
			return "", fmt.Errorf("preset %q doesn't apply to %s", "unix", kindName(v.Kind))
		}
	}

	lang := resolveLang(v, ctx)
	if table, ok := namedPresets[preset]; ok {
		pats := table[lang]
		switch v.Kind {
		case KDate:
			return renderPattern(dateOnly(v.T), pats.date, lang)
		case KDateTime:
			return renderPattern(v.T, pats.dateTime, lang)
		case KTime:
			return renderPattern(v.T, pats.timeOnly, lang)
		default:
			return v.String(), nil
		}
	}

	switch v.Kind {
	case KDate:
		return renderPattern(dateOnly(v.T), pattern, lang)
	case KDateTime, KTime:
		return renderPattern(v.T, pattern, lang)
	default:
		return v.String(), nil
	}
}

type relUnit struct {
	enSing, enPlur string
	esSing, esPlur string
}

var relUnits = map[string]relUnit{
	"minute": {"minute", "minutes", "minuto", "minutos"},
	"hour":   {"hour", "hours", "hora", "horas"},
	"day":    {"day", "days", "día", "días"},
	"week":   {"week", "weeks", "semana", "semanas"},
	"month":  {"month", "months", "mes", "meses"},
	"year":   {"year", "years", "año", "años"},
}

func formatRelative(v Value, ctx Context) (string, error) {
	lang := resolveLang(v, ctx)

	switch v.Kind {
	case KSpan:
		if v.Days {
			v = Value{Kind: KDate, T: dateOnly(v.T), Local: v.Local, Lang: v.Lang}
		} else {
			v = Value{Kind: KDateTime, T: v.T, Local: v.Local, Lang: v.Lang}
		}
	case KTime:
		v = Value{Kind: KDateTime, T: timeOnToday(v, ctx), Local: v.Local, Lang: v.Lang}
	case KDate, KDateTime:
	default:
		return "", fmt.Errorf("-t relative doesn't apply to %s", kindName(v.Kind))
	}

	now := ctxNow(ctx)
	if v.Kind == KDate {
		n := daysBetween(dateOnly(now), v.T)
		switch n {
		case 0:
			if lang == ES {
				return "hoy", nil
			}
			return "today", nil
		case 1:
			if lang == ES {
				return "mañana", nil
			}
			return "tomorrow", nil
		case -1:
			if lang == ES {
				return "ayer", nil
			}
			return "yesterday", nil
		}
		future := n > 0
		if n < 0 {
			n = -n
		}
		return formatRelativeDays(n, future, lang), nil
	}

	diff := v.T.Sub(now)
	future := diff >= 0
	absDiff := diff
	if absDiff < 0 {
		if absDiff == math.MinInt64 {
			absDiff = math.MaxInt64
		} else {
			absDiff = -absDiff
		}
	}

	if absDiff < time.Minute {
		if lang == ES {
			return "ahora", nil
		}
		return "now", nil
	}
	if absDiff < time.Hour {
		return formatRelativeUnit("minute", int(absDiff/time.Minute), future, lang), nil
	}
	if absDiff < 24*time.Hour {
		return formatRelativeUnit("hour", int(absDiff/time.Hour), future, lang), nil
	}
	days := int(absDiff / (24 * time.Hour))
	return formatRelativeDays(days, future, lang), nil
}

func formatRelativeDays(absDays int, future bool, lang Lang) string {
	switch {
	case absDays < 7:
		return formatRelativeUnit("day", absDays, future, lang)
	case absDays < 30:
		return formatRelativeUnit("week", absDays/7, future, lang)
	case absDays < 365:
		return formatRelativeUnit("month", absDays/30, future, lang)
	default:
		return formatRelativeUnit("year", absDays/365, future, lang)
	}
}

func formatRelativeUnit(unit string, count int, future bool, lang Lang) string {
	u := relUnits[unit]
	if lang == ES {
		name := u.esPlur
		if count == 1 {
			name = u.esSing
		}
		if future {
			return fmt.Sprintf("en %d %s", count, name)
		}
		return fmt.Sprintf("hace %d %s", count, name)
	}
	name := u.enPlur
	if count == 1 {
		name = u.enSing
	}
	if future {
		return fmt.Sprintf("in %d %s", count, name)
	}
	return fmt.Sprintf("%d %s ago", count, name)
}

func ctxLoc(ctx Context) *time.Location {
	if ctx.Loc != nil {
		return ctx.Loc
	}
	if loc := ctx.Now.Location(); loc != nil {
		return loc
	}
	return time.UTC
}

func ctxNow(ctx Context) time.Time {
	return ctx.Now.In(ctxLoc(ctx))
}

func timeOnToday(v Value, ctx Context) time.Time {
	loc := ctxLoc(ctx)
	now := ctx.Now.In(loc)
	y, m, d := now.Date()
	return time.Date(y, m, d, v.T.Hour(), v.T.Minute(), v.T.Second(), v.T.Nanosecond(), loc)
}

func convertType(v Value, ctx Context, typ string) (Value, error) {
	switch typ {
	case "":
		return v, nil

	case "date":
		switch v.Kind {
		case KDate:
			return v, nil
		case KDateTime, KSpan:
			return Value{Kind: KDate, T: dateOnly(v.T), Local: v.Local, Lang: v.Lang}, nil
		case KTime:
			return Value{Kind: KDate, T: dateOnly(ctxNow(ctx)), Local: v.Local, Lang: v.Lang}, nil
		default:
			return Value{}, fmt.Errorf("-t date doesn't apply to %s", kindName(v.Kind))
		}

	case "datetime":
		switch v.Kind {
		case KDate:
			return Value{Kind: KDateTime, T: dateOnly(v.T), Local: v.Local, Lang: v.Lang}, nil
		case KDateTime, KSpan:
			return Value{Kind: KDateTime, T: v.T, Local: v.Local, Lang: v.Lang}, nil
		case KTime:
			return Value{Kind: KDateTime, T: timeOnToday(v, ctx), Local: v.Local, Lang: v.Lang}, nil
		default:
			return Value{}, fmt.Errorf("-t datetime doesn't apply to %s", kindName(v.Kind))
		}

	case "time":
		switch v.Kind {
		case KDate:
			return Value{Kind: KTime, T: dateOnly(v.T), Local: v.Local, Lang: v.Lang}, nil
		case KDateTime, KSpan:
			return Value{Kind: KTime, T: v.T, Local: v.Local, Lang: v.Lang}, nil
		case KTime:
			return v, nil
		default:
			return Value{}, fmt.Errorf("-t time doesn't apply to %s", kindName(v.Kind))
		}

	case "range":
		switch v.Kind {
		case KDate:
			today := dateOnly(ctxNow(ctx))
			days := daysBetween(today, v.T)
			return Value{Kind: KDur, D: Duration{Days: days}, Lang: v.Lang}, nil
		case KDateTime:
			diff := v.T.Sub(ctxNow(ctx))
			return Value{Kind: KDur, D: Duration{Nanos: diff}, Lang: v.Lang}, nil
		case KTime:
			diff := timeOnToday(v, ctx).Sub(ctxNow(ctx))
			return Value{Kind: KDur, D: Duration{Nanos: diff}, Lang: v.Lang}, nil
		case KSpan:
			return v, nil
		case KDur:
			if !v.HasEnds {
				return v, nil
			}
			start, end := v.T, v.End
			if end.Before(start) {
				start, end = end, start
			}
			if v.Days {
				start = dateOnly(start)
				end = dateOnly(end).AddDate(0, 0, 1)
			}
			return Value{Kind: KSpan, T: start, End: end, Days: v.Days, Local: v.Local, Lang: v.Lang}, nil
		default:
			return Value{}, fmt.Errorf("-t range doesn't apply to %s", kindName(v.Kind))
		}

	default:
		return Value{}, fmt.Errorf("unknown output type %q; valid types: date, datetime, time, range, relative", typ)
	}
}
