package expr

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const supportedPatternTokens = "yyyy, yy, MMMM, MMM, MM, M, dd, d, EEEE, EEE, HH, H, hh, h, mm, m, ss, s, a, XXX, 'text'"

type langNames struct {
	monthsFull    [12]string
	monthsShort   [12]string
	weekdaysFull  [7]string
	weekdaysShort [7]string
	am            string
	pm            string
}

var enNames = langNames{
	monthsFull: [12]string{
		"January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December",
	},
	monthsShort: [12]string{
		"Jan", "Feb", "Mar", "Apr", "May", "Jun",
		"Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
	},
	weekdaysFull: [7]string{
		"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday",
	},
	weekdaysShort: [7]string{
		"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat",
	},
	am: "AM",
	pm: "PM",
}

var esNames = langNames{
	monthsFull: [12]string{
		"enero", "febrero", "marzo", "abril", "mayo", "junio",
		"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre",
	},
	monthsShort: [12]string{
		"ene", "feb", "mar", "abr", "may", "jun",
		"jul", "ago", "sep", "oct", "nov", "dic",
	},
	weekdaysFull: [7]string{
		"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado",
	},
	weekdaysShort: [7]string{
		"dom", "lun", "mar", "mié", "jue", "vie", "sáb",
	},
	am: "a. m.",
	pm: "p. m.",
}

func namesForLang(lang Lang) langNames {
	if lang == ES {
		return esNames
	}
	return enNames
}

// renderPattern formats t with a Java-style date/time pattern in lang.
func renderPattern(t time.Time, pattern string, lang Lang) (string, error) {
	names := namesForLang(lang)
	var b strings.Builder

	for i := 0; i < len(pattern); {
		r, w := utf8.DecodeRuneInString(pattern[i:])
		if r == '\'' {
			if i+w < len(pattern) && pattern[i+w] == '\'' {
				b.WriteByte('\'')
				i += w + 1
				continue
			}
			i += w
			closed := false
			for i < len(pattern) {
				cr, cw := utf8.DecodeRuneInString(pattern[i:])
				if cr == '\'' {
					if i+cw < len(pattern) && pattern[i+cw] == '\'' {
						b.WriteByte('\'')
						i += cw + 1
						continue
					}
					i += cw
					closed = true
					break
				}
				b.WriteRune(cr)
				i += cw
			}
			if !closed {
				return "", fmt.Errorf("unterminated quoted text in pattern %q", pattern)
			}
			continue
		}

		if isPatternLetter(r) {
			start := i
			i += w
			for i < len(pattern) {
				nr, nw := utf8.DecodeRuneInString(pattern[i:])
				if nr != r {
					break
				}
				i += nw
			}
			tok := pattern[start:i]
			if err := writePatternToken(&b, t, tok, r, names); err != nil {
				return "", err
			}
			continue
		}

		b.WriteRune(r)
		i += w
	}

	return b.String(), nil
}

func isPatternLetter(r rune) bool {
	return ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z') || unicode.IsLetter(r)
}

func writePatternToken(b *strings.Builder, t time.Time, tok string, letter rune, names langNames) error {
	switch tok {
	case "yyyy":
		fmt.Fprintf(b, "%04d", t.Year())
	case "yy":
		y := t.Year() % 100
		if y < 0 {
			y += 100
		}
		fmt.Fprintf(b, "%02d", y)
	case "MMMM":
		b.WriteString(names.monthsFull[t.Month()-1])
	case "MMM":
		b.WriteString(names.monthsShort[t.Month()-1])
	case "MM":
		fmt.Fprintf(b, "%02d", int(t.Month()))
	case "M":
		b.WriteString(strconv.Itoa(int(t.Month())))
	case "dd":
		fmt.Fprintf(b, "%02d", t.Day())
	case "d":
		b.WriteString(strconv.Itoa(t.Day()))
	case "EEEE":
		b.WriteString(names.weekdaysFull[t.Weekday()])
	case "EEE":
		b.WriteString(names.weekdaysShort[t.Weekday()])
	case "HH":
		fmt.Fprintf(b, "%02d", t.Hour())
	case "H":
		b.WriteString(strconv.Itoa(t.Hour()))
	case "hh":
		fmt.Fprintf(b, "%02d", hour12(t.Hour()))
	case "h":
		b.WriteString(strconv.Itoa(hour12(t.Hour())))
	case "mm":
		fmt.Fprintf(b, "%02d", t.Minute())
	case "m":
		b.WriteString(strconv.Itoa(t.Minute()))
	case "ss":
		fmt.Fprintf(b, "%02d", t.Second())
	case "s":
		b.WriteString(strconv.Itoa(t.Second()))
	case "a":
		if t.Hour() < 12 {
			b.WriteString(names.am)
		} else {
			b.WriteString(names.pm)
		}
	case "XXX":
		b.WriteString(formatOffsetXXX(t))
	default:
		return fmt.Errorf("unsupported pattern token %q (letter %q); supported tokens: %s", tok, string(letter), supportedPatternTokens)
	}
	return nil
}

func hour12(h int) int {
	h %= 12
	if h == 0 {
		return 12
	}
	return h
}

func formatOffsetXXX(t time.Time) string {
	_, offset := t.Zone()
	if offset == 0 {
		return "Z"
	}
	sign := '+'
	if offset < 0 {
		sign = '-'
		offset = -offset
	}
	hours := offset / 3600
	mins := (offset % 3600) / 60
	return fmt.Sprintf("%c%02d:%02d", sign, hours, mins)
}
