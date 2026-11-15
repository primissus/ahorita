package expr

import (
	"fmt"
	"strings"
	"time"
)

// Kind identifies the type of value produced by evaluating an expression.
type Kind int

// The K prefix avoids a clash with the token kinds Date, DateTime, Number.
const (
	KDate       Kind = iota + 1 // a calendar day
	KDateTime                   // an instant with a zone
	KTime                       // a time of day
	KDur                        // a duration
	KSpan                       // [T, End)
	KNumber                     // a count or scalar
	KRecurrence                 // plan 10
	KList                       // plans 9, 10
	KText                       // plan 10
)

// Duration stores calendar and fixed time components separately.
type Duration struct {
	Months  int           // years and quarters are folded into months
	Days    int           // weeks are folded into days
	BizDays int           // business days; cannot be combined with other units
	Nanos   time.Duration // hours, minutes, seconds
}

// Value is the unified result type returned by the evaluator.
type Value struct {
	Kind    Kind
	T       time.Time // KDate: 00:00 in its zone. KDateTime. KTime: on ctx's today. KSpan: start. KDur with ends: the earlier end.
	End     time.Time // KSpan: exclusive end. KDur with ends: the later end.
	D       Duration  // KDur; also the length a `during` span was built from
	N       float64   // KNumber
	Days    bool      // KSpan: made of whole days (or KDur with ends: both ends were dates)
	HasEnds bool      // KDur: T and End are set (from a - b)
	Local   bool      // typed without a zone, so @ / in may still reinterpret it
	Items   []Value   // KList
	Str     string    // KText, KRecurrence
	Note    string    // non-fatal warning (e.g. cron weekday/DST shift across zones)
	Lang    Lang      // language of the input; set by Eval on the final value
}

// String returns the plain ISO rendering of the value.
func (v Value) String() string {
	switch v.Kind {
	case KDate:
		return v.T.Format("2006-01-02")
	case KDateTime:
		return v.T.Format(time.RFC3339)
	case KTime:
		if v.T.Second() != 0 {
			return v.T.Format("15:04:05")
		}
		return v.T.Format("15:04")
	case KDur:
		return formatDuration(v.D)
	case KSpan:
		if v.Days {
			endDay := v.End.AddDate(0, 0, -1)
			return v.T.Format("2006-01-02") + " → " + endDay.Format("2006-01-02")
		}
		return v.T.Format(time.RFC3339) + " → " + v.End.Format(time.RFC3339)
	case KNumber:
		return formatNumber(v.N)
	case KList:
		lines := make([]string, len(v.Items))
		for i, item := range v.Items {
			s := item.String()
			if item.Str != "" && item.Kind != KText && item.Kind != KRecurrence {
				s += "  " + item.Str
			}
			lines[i] = s
		}
		return strings.Join(lines, "\n")
	case KText:
		return v.Str
	case KRecurrence:
		return recurString(v)
	default:
		return ""
	}
}

func formatDuration(d Duration) string {
	const dayNanos = 24 * time.Hour

	months := d.Months
	days := d.Days
	bizDays := d.BizDays
	nanos := d.Nanos

	if nanos >= dayNanos || nanos <= -dayNanos {
		extra := int(nanos / dayNanos)
		days += extra
		nanos -= time.Duration(extra) * dayNanos
	}
	if days > 0 && nanos < 0 {
		days--
		nanos += dayNanos
	} else if days < 0 && nanos > 0 {
		days++
		nanos -= dayNanos
	}

	if months == 0 && days == 0 && bizDays == 0 && nanos == 0 {
		return "0s"
	}

	prefix := ""
	if months <= 0 && days <= 0 && bizDays <= 0 && nanos <= 0 {
		prefix = "-"
		months = -months
		days = -days
		bizDays = -bizDays
		nanos = -nanos
	}

	var parts []string
	years := months / 12
	remMonths := months % 12
	if years != 0 {
		parts = append(parts, fmt.Sprintf("%dy", years))
	}
	if remMonths != 0 {
		parts = append(parts, fmt.Sprintf("%dM", remMonths))
	}
	if days != 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if bizDays != 0 {
		if bizDays == 1 {
			parts = append(parts, "1 business day")
		} else {
			parts = append(parts, fmt.Sprintf("%d business days", bizDays))
		}
	}

	hours := nanos / time.Hour
	rem := nanos % time.Hour
	mins := rem / time.Minute
	rem %= time.Minute
	secs := rem / time.Second
	remNanos := rem % time.Second

	if hours != 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if mins != 0 {
		parts = append(parts, fmt.Sprintf("%dm", mins))
	}
	if secs != 0 || remNanos != 0 {
		if remNanos == 0 {
			parts = append(parts, fmt.Sprintf("%ds", secs))
		} else {
			sign := ""
			if secs < 0 || remNanos < 0 {
				sign = "-"
				secs = -secs
				remNanos = -remNanos
			}
			frac := strings.TrimRight(fmt.Sprintf("%09d", remNanos), "0")
			parts = append(parts, fmt.Sprintf("%s%d.%ss", sign, secs, frac))
		}
	}

	if len(parts) == 0 {
		return "0s"
	}
	return prefix + strings.Join(parts, " ")
}

func formatNumber(n float64) string {
	s := fmt.Sprintf("%.2f", n)
	if strings.ContainsRune(s, '.') {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	if s == "-0" {
		return "0"
	}
	return s
}
