package expr

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rickar/cal/v2"
	"github.com/rickar/cal/v2/ar"
	"github.com/rickar/cal/v2/at"
	"github.com/rickar/cal/v2/be"
	"github.com/rickar/cal/v2/bg"
	"github.com/rickar/cal/v2/bj"
	"github.com/rickar/cal/v2/br"
	"github.com/rickar/cal/v2/ca"
	"github.com/rickar/cal/v2/ch"
	"github.com/rickar/cal/v2/cy"
	"github.com/rickar/cal/v2/cz"
	"github.com/rickar/cal/v2/de"
	"github.com/rickar/cal/v2/dk"
	"github.com/rickar/cal/v2/ee"
	"github.com/rickar/cal/v2/es"
	"github.com/rickar/cal/v2/fi"
	"github.com/rickar/cal/v2/fr"
	"github.com/rickar/cal/v2/gb"
	"github.com/rickar/cal/v2/gr"
	"github.com/rickar/cal/v2/hr"
	"github.com/rickar/cal/v2/hu"
	"github.com/rickar/cal/v2/ie"
	"github.com/rickar/cal/v2/is"
	"github.com/rickar/cal/v2/it"
	"github.com/rickar/cal/v2/jp"
	"github.com/rickar/cal/v2/ke"
	"github.com/rickar/cal/v2/lt"
	"github.com/rickar/cal/v2/lu"
	"github.com/rickar/cal/v2/lv"
	"github.com/rickar/cal/v2/mt"
	"github.com/rickar/cal/v2/mw"
	"github.com/rickar/cal/v2/nc"
	"github.com/rickar/cal/v2/nl"
	"github.com/rickar/cal/v2/no"
	"github.com/rickar/cal/v2/nz"
	"github.com/rickar/cal/v2/pl"
	"github.com/rickar/cal/v2/pt"
	"github.com/rickar/cal/v2/ro"
	"github.com/rickar/cal/v2/rs"
	"github.com/rickar/cal/v2/ru"
	"github.com/rickar/cal/v2/se"
	"github.com/rickar/cal/v2/si"
	"github.com/rickar/cal/v2/sk"
	"github.com/rickar/cal/v2/th"
	"github.com/rickar/cal/v2/ua"
	"github.com/rickar/cal/v2/us"
	"github.com/rickar/cal/v2/za"
)

type holiday struct {
	Date   time.Time
	Name   string
	NameES string
}

type holidayRec struct {
	year   int
	month  time.Month
	day    int
	name   string
	nameES string
}

type holidayCacheKey struct {
	country string
	year    int
}

var (
	holidayCacheMu sync.Mutex
	holidayCache   = make(map[holidayCacheKey][]holidayRec)
)

var countryHolidays = map[string][]*cal.Holiday{
	"AR": ar.Holidays,
	"AT": at.Holidays,
	"BE": be.Holidays,
	"BG": bg.Holidays,
	"BJ": bj.Holidays,
	"BR": br.Holidays,
	"CA": ca.Holidays,
	"CH": ch.Holidays,
	"CY": cy.Holidays,
	"CZ": cz.Holidays,
	"DE": de.Holidays,
	"DK": dk.Holidays,
	"EE": ee.Holidays,
	"ES": es.Holidays,
	"FI": fi.Holidays,
	"FR": fr.Holidays,
	"GB": gb.Holidays,
	"GR": gr.Holidays,
	"HR": hr.Holidays,
	"HU": hu.Holidays,
	"IE": ie.Holidays,
	"IS": is.Holidays,
	"IT": it.Holidays,
	"JP": jp.Holidays,
	"KE": ke.Holidays,
	"LT": lt.Holidays,
	"LU": lu.Holidays,
	"LV": lv.Holidays,
	"MT": mt.Holidays,
	"MW": mw.Holidays,
	"NC": nc.Holidays,
	"NL": nl.Holidays,
	"NO": no.Holidays,
	"NZ": nz.Holidays,
	"PL": pl.Holidays,
	"PT": pt.Holidays,
	"RO": ro.Holidays,
	"RS": rs.Holidays,
	"RU": ru.Holidays,
	"SE": se.Holidays,
	"SI": si.Holidays,
	"SK": sk.Holidays,
	"TH": th.Holidays,
	"UA": ua.Holidays,
	"US": us.Holidays,
	"ZA": za.Holidays,
}

func supportedCountries() []string {
	codes := make([]string, 0, len(countryHolidays)+1)
	codes = append(codes, "MX")
	for code := range countryHolidays {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes
}

func nthWeekdayOfMonth(year int, month time.Month, wd time.Weekday, n int) int {
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).Weekday()
	offset := (int(wd) - int(first) + 7) % 7
	return 1 + offset + (n-1)*7
}

func mxHolidaysForYear(year int) []holidayRec {
	recs := []holidayRec{
		{year: year, month: time.January, day: 1, name: "New Year's Day", nameES: "Año Nuevo"},
		{year: year, month: time.February, day: nthWeekdayOfMonth(year, time.February, time.Monday, 1), name: "Constitution Day", nameES: "Día de la Constitución"},
		{year: year, month: time.March, day: nthWeekdayOfMonth(year, time.March, time.Monday, 3), name: "Benito Juárez's Birthday", nameES: "Natalicio de Benito Juárez"},
		{year: year, month: time.May, day: 1, name: "Labour Day", nameES: "Día del Trabajo"},
		{year: year, month: time.September, day: 16, name: "Independence Day", nameES: "Día de la Independencia"},
	}
	if year >= 2024 && (year-2024)%6 == 0 {
		recs = append(recs, holidayRec{
			year: year, month: time.October, day: 1,
			name: "Inauguration Day", nameES: "Transmisión del Poder Ejecutivo Federal",
		})
	}
	recs = append(recs,
		holidayRec{year: year, month: time.November, day: nthWeekdayOfMonth(year, time.November, time.Monday, 3), name: "Revolution Day", nameES: "Día de la Revolución"},
		holidayRec{year: year, month: time.December, day: 25, name: "Christmas Day", nameES: "Navidad"},
	)
	return recs
}

func holidays(country string, year int, loc *time.Location) ([]holiday, error) {
	if loc == nil {
		loc = time.UTC
	}
	code := strings.ToUpper(strings.TrimSpace(country))
	if code == "" {
		return nil, &Error{
			Pos: -1,
			Msg: fmt.Sprintf("unknown holiday country %q (supported: %s)", country, strings.Join(supportedCountries(), ", ")),
		}
	}
	var defs []*cal.Holiday
	if code != "MX" {
		var ok bool
		defs, ok = countryHolidays[code]
		if !ok {
			return nil, &Error{
				Pos: -1,
				Msg: fmt.Sprintf("unknown holiday country %q (supported: %s)", country, strings.Join(supportedCountries(), ", ")),
			}
		}
	}

	key := holidayCacheKey{country: code, year: year}
	holidayCacheMu.Lock()
	recs, cached := holidayCache[key]
	if !cached {
		if code == "MX" {
			recs = mxHolidaysForYear(year)
		} else {
			for _, calcYear := range []int{year - 1, year, year + 1} {
				for _, h := range defs {
					if h.Type != cal.ObservancePublic {
						continue
					}
					_, obs := h.Calc(calcYear)
					if obs.IsZero() {
						continue
					}
					oy, om, od := obs.Date()
					if oy != year {
						continue
					}
					recs = append(recs, holidayRec{
						year:  oy,
						month: om,
						day:   od,
						name:  h.Name,
					})
				}
			}
			sort.SliceStable(recs, func(i, j int) bool {
				if recs[i].month != recs[j].month {
					return recs[i].month < recs[j].month
				}
				return recs[i].day < recs[j].day
			})
		}
		holidayCache[key] = recs
	}
	holidayCacheMu.Unlock()

	out := make([]holiday, len(recs))
	for i, r := range recs {
		out[i] = holiday{
			Date:   time.Date(r.year, r.month, r.day, 0, 0, 0, 0, loc),
			Name:   r.name,
			NameES: r.nameES,
		}
	}
	return out, nil
}

func isBusinessDay(d time.Time, ctx Context) (bool, error) {
	wd := d.Weekday()
	for _, w := range ctx.Cfg.Weekend {
		if w == wd {
			return false, nil
		}
	}

	y, m, day := d.Date()
	if ctx.Cfg.Country != "" {
		hs, err := holidays(ctx.Cfg.Country, y, d.Location())
		if err != nil {
			return false, err
		}
		for _, h := range hs {
			hy, hm, hd := h.Date.Date()
			if hy == y && hm == m && hd == day {
				return false, nil
			}
		}
	}

	dc := daysFromCivil(y, m, day)
	for _, off := range ctx.Cfg.DaysOff {
		switch off.Kind {
		case KDate:
			oy, om, od := off.T.Date()
			if oy == y && om == m && od == day {
				return false, nil
			}
		case KSpan:
			if off.Days {
				sy, sm, sd := off.T.Date()
				ey, em, ed := off.End.Date()
				sc := daysFromCivil(sy, sm, sd)
				ec := daysFromCivil(ey, em, ed)
				if dc >= sc && dc < ec {
					return false, nil
				}
			} else if !d.Before(off.T) && d.Before(off.End) {
				return false, nil
			}
		}
	}

	return true, nil
}

func withErrorPos(pos int, err error) error {
	if e, ok := err.(*Error); ok && e.Pos < 0 {
		copy := *e
		copy.Pos = pos
		return &copy
	}
	return err
}

const maxBusinessSearchDays = 10000

func addBusinessDays(pos int, t time.Time, n int, ctx Context) (time.Time, error) {
	// ponytail: walks day by day; fine for thousands of days, jump by weeks if someone needs millions
	if n == 0 {
		return t, nil
	}
	step := 1
	remaining := n
	if n < 0 {
		step = -1
		remaining = -n
	}
	cur := t
	skipped := 0
	for remaining > 0 {
		cur = cur.AddDate(0, 0, step)
		ok, err := isBusinessDay(cur, ctx)
		if err != nil {
			return time.Time{}, withErrorPos(pos, err)
		}
		if ok {
			remaining--
			skipped = 0
			continue
		}
		skipped++
		if skipped > maxBusinessSearchDays {
			return time.Time{}, &Error{Pos: pos, Msg: "no business day found"}
		}
	}
	return cur, nil
}

func nextBusinessDay(pos int, d time.Time, dir int, allowSame bool, ctx Context) (time.Time, error) {
	cur := dateOnly(d)
	if allowSame {
		ok, err := isBusinessDay(cur, ctx)
		if err != nil {
			return time.Time{}, withErrorPos(pos, err)
		}
		if ok {
			return cur, nil
		}
	}
	for i := 0; i <= maxBusinessSearchDays; i++ {
		cur = cur.AddDate(0, 0, dir)
		ok, err := isBusinessDay(cur, ctx)
		if err != nil {
			return time.Time{}, withErrorPos(pos, err)
		}
		if ok {
			return cur, nil
		}
	}
	return time.Time{}, &Error{Pos: pos, Msg: "no business day found"}
}

func countBusinessDays(pos int, start, end time.Time, startInc, endInc bool, ctx Context) (int, error) {
	cur := dateOnly(start)
	if !startInc {
		cur = cur.AddDate(0, 0, 1)
	}
	nDays := daysBetween(cur, dateOnly(end))
	limit := nDays
	if endInc {
		limit++
	}
	count := 0
	for i := 0; i < limit; i++ {
		ok, err := isBusinessDay(cur, ctx)
		if err != nil {
			return 0, withErrorPos(pos, err)
		}
		if ok {
			count++
		}
		cur = cur.AddDate(0, 0, 1)
	}
	return count, nil
}

func countSpanBusinessDays(pos int, span Value, ctx Context) (int, error) {
	if !span.Days && !span.End.After(span.T) {
		return 0, nil
	}
	endInc := !span.Days && !isMidnight(span.End)
	return countBusinessDays(pos, span.T, span.End, true, endInc, ctx)
}

func (ev *evaluator) evalOrSame(n Node, in Value) (Value, error) {
	if len(n.Args) < 1 {
		return Value{}, &Error{Pos: n.Pos, Msg: "invalid on or after/before expression"}
	}
	rel := n.Args[0]
	var dir int
	switch rel.Text {
	case "next":
		dir = 1
	case "last":
		dir = -1
	default:
		return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("invalid business day direction %q", rel.Text)}
	}
	anchorVal := in
	if len(n.Args) >= 2 {
		var err error
		anchorVal, err = ev.node(n.Args[1], in)
		if err != nil {
			return Value{}, err
		}
	}
	d, err := inputDate(n.Pos, anchorVal)
	if err != nil {
		return Value{}, err
	}
	res, err := nextBusinessDay(n.Pos, d, dir, true, ev.ctx)
	if err != nil {
		return Value{}, err
	}
	return Value{Kind: KDate, T: res, Local: anchorVal.Local}, nil
}

func (ev *evaluator) holidayItem(h holiday) Value {
	name := h.Name
	if ev.resLang == ES && h.NameES != "" {
		name = h.NameES
	}
	return Value{
		Kind:  KDate,
		T:     h.Date,
		Local: true,
		Str:   name,
	}
}

func (ev *evaluator) evalHolidays(n Node, in Value) (Value, error) {
	country := n.Text
	if country == "" {
		country = ev.ctx.Cfg.Country
	}

	if len(n.Args) > 0 {
		y, err := strconv.Atoi(n.Args[0].Text)
		if err != nil || y < 1 {
			return Value{}, &Error{Pos: n.Args[0].Pos, Msg: fmt.Sprintf("invalid year %q", n.Args[0].Text)}
		}
		if country == "" {
			return Value{Kind: KList}, nil
		}
		hs, err := holidays(country, y, ev.loc)
		if err != nil {
			return Value{}, withErrorPos(n.Pos, err)
		}
		items := make([]Value, len(hs))
		for i, h := range hs {
			items[i] = ev.holidayItem(h)
		}
		return Value{Kind: KList, Items: items}, nil
	}

	if in.Kind == KSpan {
		startDate := dateOnly(in.T)
		endInc := !in.Days && !isMidnight(in.End)
		lastDate := dateOnly(in.End)
		if !endInc {
			lastDate = lastDate.AddDate(0, 0, -1)
		}
		if country == "" {
			return Value{Kind: KList}, nil
		}
		if (!in.Days && !in.End.After(in.T)) || lastDate.Before(startDate) {
			if _, err := holidays(country, startDate.Year(), ev.loc); err != nil {
				return Value{}, withErrorPos(n.Pos, err)
			}
			return Value{Kind: KList}, nil
		}
		startDC := daysFromCivil(startDate.Year(), startDate.Month(), startDate.Day())
		lastDC := daysFromCivil(lastDate.Year(), lastDate.Month(), lastDate.Day())
		var items []Value
		for y := startDate.Year(); y <= lastDate.Year(); y++ {
			hs, err := holidays(country, y, ev.loc)
			if err != nil {
				return Value{}, withErrorPos(n.Pos, err)
			}
			for _, h := range hs {
				hy, hm, hd := h.Date.Date()
				dc := daysFromCivil(hy, hm, hd)
				if dc >= startDC && dc <= lastDC {
					items = append(items, ev.holidayItem(h))
				}
			}
		}
		return Value{Kind: KList, Items: items}, nil
	}

	d, err := inputDate(n.Pos, in)
	if err != nil {
		return Value{}, err
	}
	if country == "" {
		return Value{Kind: KList}, nil
	}
	hs, err := holidays(country, d.Year(), ev.loc)
	if err != nil {
		return Value{}, withErrorPos(n.Pos, err)
	}
	items := make([]Value, len(hs))
	for i, h := range hs {
		items[i] = ev.holidayItem(h)
	}
	return Value{Kind: KList, Items: items}, nil
}
