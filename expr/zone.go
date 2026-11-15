//go:generate go run zones_gen.go

package expr

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"
)

var continentPrefixes = []string{
	"Africa/",
	"America/",
	"Antarctica/",
	"Asia/",
	"Atlantic/",
	"Australia/",
	"Europe/",
	"Indian/",
	"Pacific/",
}

var zoneAliases = map[string]string{
	"cdmx":             "America/Mexico_City",
	"nyc":              "America/New_York",
	"dc":               "America/New_York",
	"sf":               "America/Los_Angeles",
	"la":               "America/Los_Angeles",
	"mexico":           "America/Mexico_City",
	"ciudad de mexico": "America/Mexico_City",
	"nueva york":       "America/New_York",
	"londres":          "Europe/London",
	"tokio":            "Asia/Tokyo",
	"moscu":            "Europe/Moscow",
	"pekin":            "Asia/Shanghai",
	"beijing":          "Asia/Shanghai",
	"nueva delhi":      "Asia/Kolkata",
	"delhi":            "Asia/Kolkata",
	"mumbai":           "Asia/Kolkata",
}

var zoneAbbrevs = map[string]int{
	"est":  -5,
	"edt":  -4,
	"pst":  -8,
	"pdt":  -7,
	"mst":  -7,
	"mdt":  -6,
	"akst": -9,
	"akdt": -8,
	"hst":  -10,
	"cet":  1,
	"cest": 2,
	"eet":  2,
	"eest": 3,
	"wet":  0,
	"west": 1,
	"jst":  9,
	"kst":  9,
	"aest": 10,
	"aedt": 11,
	"nzst": 12,
	"nzdt": 13,
}

var ambiguousAbbrevs = map[string][]string{
	"cst": {"America/Chicago", "Asia/Shanghai", "America/Havana"},
	"cdt": {"America/Chicago", "America/Havana"},
	"ist": {"Asia/Kolkata", "Europe/Dublin", "Asia/Jerusalem"},
	"bst": {"Europe/London", "Asia/Dhaka"},
	"ast": {"America/Halifax", "Asia/Riyadh"},
}

type cityEntry struct {
	folded  string
	display string
}

type zoneIndex struct {
	ianaByFolded map[string]string
	cityByFolded map[string]string
	cities       []cityEntry
}

var zones = buildZoneIndex()

func buildZoneIndex() zoneIndex {
	ianaByFolded := make(map[string]string, len(zoneNames))
	cityByFolded := make(map[string]string, len(zoneNames))
	cities := make([]cityEntry, 0, len(zoneNames))

	for _, name := range zoneNames {
		ianaByFolded[foldZone(name)] = name
		if !hasContinentPrefix(name) {
			continue
		}
		slash := strings.LastIndexByte(name, '/')
		rawCity := name[slash+1:]
		foldedCity := foldZone(rawCity)
		if _, exists := cityByFolded[foldedCity]; !exists {
			cityByFolded[foldedCity] = name
			cities = append(cities, cityEntry{
				folded:  foldedCity,
				display: strings.ReplaceAll(rawCity, "_", " "),
			})
		}
	}

	sort.Slice(cities, func(i, j int) bool {
		return cities[i].display < cities[j].display
	})

	return zoneIndex{
		ianaByFolded: ianaByFolded,
		cityByFolded: cityByFolded,
		cities:       cities,
	}
}

func hasContinentPrefix(name string) bool {
	for _, p := range continentPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// foldZone normalizes case, Latin accents, and underscores/spaces for timezone matching.
func foldZone(s string) string {
	mapped := strings.Map(func(r rune) rune {
		switch r {
		case '_':
			return ' '
		case 'à', 'â', 'ã', 'ä', 'å':
			return 'a'
		case 'ç':
			return 'c'
		case 'è', 'ê', 'ë':
			return 'e'
		case 'ì', 'î', 'ï':
			return 'i'
		case 'ò', 'ô', 'õ', 'ö':
			return 'o'
		case 'ù', 'û':
			return 'u'
		case 'ý', 'ÿ':
			return 'y'
		default:
			return r
		}
	}, fold(s))
	return strings.Join(strings.Fields(mapped), " ")
}

func lookupZone(name string) (*time.Location, error) {
	folded := foldZone(name)
	if folded == "" || folded == "local" {
		return nil, &Error{Pos: -1, Msg: fmt.Sprintf("unknown timezone or word %q", name)}
	}

	// 1. UTC / GMT / Z.
	if folded == "utc" || folded == "gmt" || folded == "z" {
		return time.UTC, nil
	}

	// 2. Fixed offsets (+09:00, -06:00, +0900, +9, -6, optionally prefixed with utc or gmt).
	if loc, ok := parseZoneOffset(folded); ok {
		return loc, nil
	}

	// 3. Full IANA name containing '/', any case.
	if strings.ContainsRune(folded, '/') {
		if exact, ok := zones.ianaByFolded[folded]; ok {
			return loadZoneLocation(exact, name)
		}
	}

	// 4. Short aliases.
	if target, ok := zoneAliases[folded]; ok {
		return loadZoneLocation(target, name)
	}

	// 5. Abbreviations (fixed offsets) and ambiguous abbreviations.
	if hours, ok := zoneAbbrevs[folded]; ok {
		return time.FixedZone(strings.ToUpper(folded), hours*3600), nil
	}
	if candidates, ok := ambiguousAbbrevs[folded]; ok {
		return nil, &Error{
			Pos: -1,
			Msg: fmt.Sprintf("%q is ambiguous: %s", strings.ToUpper(folded), joinOr(candidates)),
		}
	}

	// 6. City name from the 9 continent prefixes (first in sorted order).
	if target, ok := zones.cityByFolded[folded]; ok {
		return loadZoneLocation(target, name)
	}

	// Non-slashed legacy IANA entries that do not collide with abbreviations (e.g. CST6CDT, Japan).
	if exact, ok := zones.ianaByFolded[folded]; ok {
		return loadZoneLocation(exact, name)
	}

	// 7. Unknown timezone, with did-you-mean suggestions for up to 5 matching cities.
	var matches []string
	for _, c := range zones.cities {
		if strings.Contains(c.folded, folded) {
			matches = append(matches, c.display)
			if len(matches) > 5 {
				break
			}
		}
	}
	msg := fmt.Sprintf("unknown timezone or word %q", name)
	if len(matches) > 0 && len(matches) <= 5 {
		msg += "; did you mean: " + strings.Join(matches, ", ")
	}
	return nil, &Error{Pos: -1, Msg: msg}
}

func loadZoneLocation(ianaName, origInput string) (*time.Location, error) {
	loc, err := time.LoadLocation(ianaName)
	if err != nil {
		return nil, &Error{Pos: -1, Msg: fmt.Sprintf("unknown timezone or word %q", origInput)}
	}
	return loc, nil
}

func parseZoneOffset(folded string) (*time.Location, bool) {
	s := folded
	if strings.HasPrefix(s, "utc") {
		s = strings.TrimSpace(s[3:])
	} else if strings.HasPrefix(s, "gmt") {
		s = strings.TrimSpace(s[3:])
	}
	if len(s) < 2 || (s[0] != '+' && s[0] != '-') {
		return nil, false
	}
	sign := 1
	if s[0] == '-' {
		sign = -1
	}
	rest := s[1:]

	var oh, om int
	if colon := strings.IndexByte(rest, ':'); colon >= 0 {
		hStr := rest[:colon]
		mStr := rest[colon+1:]
		if (len(hStr) != 1 && len(hStr) != 2) || len(mStr) != 2 || !allDigits(hStr) || !allDigits(mStr) {
			return nil, false
		}
		var err1, err2 error
		oh, err1 = strconv.Atoi(hStr)
		om, err2 = strconv.Atoi(mStr)
		if err1 != nil || err2 != nil {
			return nil, false
		}
	} else {
		if !allDigits(rest) {
			return nil, false
		}
		switch len(rest) {
		case 1, 2:
			var err error
			oh, err = strconv.Atoi(rest)
			if err != nil {
				return nil, false
			}
		case 4:
			var err1, err2 error
			oh, err1 = strconv.Atoi(rest[:2])
			om, err2 = strconv.Atoi(rest[2:])
			if err1 != nil || err2 != nil {
				return nil, false
			}
		default:
			return nil, false
		}
	}

	if oh < 0 || oh > 23 || om < 0 || om > 59 {
		return nil, false
	}
	off := sign * (oh*3600 + om*60)
	signCh := '+'
	if sign < 0 && off != 0 {
		signCh = '-'
	}
	return time.FixedZone(fmt.Sprintf("%c%02d:%02d", signCh, oh, om), off), true
}

func joinOr(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	default:
		return strings.Join(items[:len(items)-1], ", ") + " or " + items[len(items)-1]
	}
}
