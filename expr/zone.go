//go:generate go run zones_gen.go

package expr

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"
	"unicode"
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
	// Existing aliases and exonyms
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

	// US / MX / CA states, provinces, and regions
	"california":       "America/Los_Angeles",
	"texas":            "America/Chicago",
	"florida":          "America/New_York",
	"arizona":          "America/Phoenix",
	"utah":             "America/Denver",
	"colorado":         "America/Denver",
	"illinois":         "America/Chicago",
	"alaska":           "America/Anchorage",
	"hawaii":           "Pacific/Honolulu",
	"jalisco":          "America/Mexico_City",
	"nuevo leon":       "America/Monterrey",
	"baja california":  "America/Tijuana",
	"quintana roo":     "America/Cancun",
	"sonora":           "America/Hermosillo",
	"sinaloa":          "America/Mazatlan",
	"ontario":          "America/Toronto",
	"quebec":           "America/Toronto",
	"alberta":          "America/Edmonton",
	"british columbia": "America/Vancouver",
	"bc":               "America/Vancouver",
	"pacific time":     "America/Los_Angeles",
	"mountain time":    "America/Denver",
	"central time":     "America/Chicago",
	"eastern time":     "America/New_York",
	"pt":               "America/Los_Angeles",
	"mt":               "America/Denver",
	"ct":               "America/Chicago",
	"et":               "America/New_York",
	"ny":               "America/New_York",

	// Countries
	"colombia":       "America/Bogota",
	"peru":           "America/Lima",
	"chile":          "America/Santiago",
	"argentina":      "America/Argentina/Buenos_Aires",
	"brazil":         "America/Sao_Paulo",
	"brasil":         "America/Sao_Paulo",
	"venezuela":      "America/Caracas",
	"uruguay":        "America/Montevideo",
	"ecuador":        "America/Guayaquil",
	"uk":             "Europe/London",
	"united kingdom": "Europe/London",
	"reino unido":    "Europe/London",
	"england":        "Europe/London",
	"inglaterra":     "Europe/London",
	"scotland":       "Europe/London",
	"ireland":        "Europe/Dublin",
	"irlanda":        "Europe/Dublin",
	"spain":          "Europe/Madrid",
	"espana":         "Europe/Madrid",
	"france":         "Europe/Paris",
	"francia":        "Europe/Paris",
	"germany":        "Europe/Berlin",
	"alemania":       "Europe/Berlin",
	"italy":          "Europe/Rome",
	"italia":         "Europe/Rome",
	"switzerland":    "Europe/Zurich",
	"suiza":          "Europe/Zurich",
	"netherlands":    "Europe/Amsterdam",
	"holanda":        "Europe/Amsterdam",
	"belgium":        "Europe/Brussels",
	"belgica":        "Europe/Brussels",
	"portugal":       "Europe/Lisbon",
	"austria":        "Europe/Vienna",
	"poland":         "Europe/Warsaw",
	"polonia":        "Europe/Warsaw",
	"czechia":        "Europe/Prague",
	"sweden":         "Europe/Stockholm",
	"suecia":         "Europe/Stockholm",
	"norway":         "Europe/Oslo",
	"noruega":        "Europe/Oslo",
	"denmark":        "Europe/Copenhagen",
	"dinamarca":      "Europe/Copenhagen",
	"finland":        "Europe/Helsinki",
	"finlandia":      "Europe/Helsinki",
	"greece":         "Europe/Athens",
	"grecia":         "Europe/Athens",
	"turkey":         "Europe/Istanbul",
	"turquia":        "Europe/Istanbul",
	"ukraine":        "Europe/Kyiv",
	"ucrania":        "Europe/Kyiv",
	"russia":         "Europe/Moscow",
	"rusia":          "Europe/Moscow",
	"japan":          "Asia/Tokyo",
	"japon":          "Asia/Tokyo",
	"korea":          "Asia/Seoul",
	"south korea":    "Asia/Seoul",
	"corea":          "Asia/Seoul",
	"china":          "Asia/Shanghai",
	"taiwan":         "Asia/Taipei",
	"india":          "Asia/Kolkata",
	"pakistan":       "Asia/Karachi",
	"bangladesh":     "Asia/Dhaka",
	"thailand":       "Asia/Bangkok",
	"tailandia":      "Asia/Bangkok",
	"vietnam":        "Asia/Ho_Chi_Minh",
	"philippines":    "Asia/Manila",
	"filipinas":      "Asia/Manila",
	"malaysia":       "Asia/Kuala_Lumpur",
	"indonesia":      "Asia/Jakarta",
	"uae":            "Asia/Dubai",
	"emirates":       "Asia/Dubai",
	"saudi arabia":   "Asia/Riyadh",
	"israel":         "Asia/Jerusalem",
	"egypt":          "Africa/Cairo",
	"egipto":         "Africa/Cairo",
	"south africa":   "Africa/Johannesburg",
	"sudafrica":      "Africa/Johannesburg",
	"nigeria":        "Africa/Lagos",
	"kenya":          "Africa/Nairobi",
	"morocco":        "Africa/Casablanca",
	"australia":      "Australia/Sydney",
	"new zealand":    "Pacific/Auckland",
	"nueva zelanda":  "Pacific/Auckland",

	// Non-IANA major cities and exonyms
	"seattle":          "America/Los_Angeles",
	"portland":         "America/Los_Angeles",
	"san francisco":    "America/Los_Angeles",
	"san jose":         "America/Los_Angeles",
	"san diego":        "America/Los_Angeles",
	"las vegas":        "America/Los_Angeles",
	"vegas":            "America/Los_Angeles",
	"dallas":           "America/Chicago",
	"houston":          "America/Chicago",
	"austin":           "America/Chicago",
	"san antonio":      "America/Chicago",
	"new orleans":      "America/Chicago",
	"minneapolis":      "America/Chicago",
	"st louis":         "America/Chicago",
	"kansas city":      "America/Chicago",
	"nashville":        "America/Chicago",
	"memphis":          "America/Chicago",
	"boston":           "America/New_York",
	"miami":            "America/New_York",
	"atlanta":          "America/New_York",
	"philadelphia":     "America/New_York",
	"philly":           "America/New_York",
	"washington":       "America/New_York",
	"washington dc":    "America/New_York",
	"baltimore":        "America/New_York",
	"charlotte":        "America/New_York",
	"orlando":          "America/New_York",
	"tampa":            "America/New_York",
	"pittsburgh":       "America/New_York",
	"cleveland":        "America/New_York",
	"cincinnati":       "America/New_York",
	"columbus":         "America/New_York",
	"tucson":           "America/Phoenix",
	"salt lake city":   "America/Denver",
	"guadalajara":      "America/Mexico_City",
	"puebla":           "America/Mexico_City",
	"queretaro":        "America/Mexico_City",
	"leon":             "America/Mexico_City",
	"oaxaca":           "America/Mexico_City",
	"veracruz":         "America/Mexico_City",
	"mexicali":         "America/Tijuana",
	"playa del carmen": "America/Cancun",
	"tulum":            "America/Cancun",
	"los cabos":        "America/Mazatlan",
	"cabo san lucas":   "America/Mazatlan",
	"culiacan":         "America/Mazatlan",
	"medellin":         "America/Bogota",
	"cali":             "America/Bogota",
	"rio":              "America/Sao_Paulo",
	"rio de janeiro":   "America/Sao_Paulo",
	"brasilia":         "America/Sao_Paulo",
	"quito":            "America/Guayaquil",
	"ottawa":           "America/Toronto",
	"montreal":         "America/Montreal",
	"calgary":          "America/Edmonton",
	"victoria":         "America/Vancouver",
	"edinburgh":        "Europe/London",
	"manchester":       "Europe/London",
	"birmingham":       "Europe/London",
	"barcelona":        "Europe/Madrid",
	"valencia":         "Europe/Madrid",
	"sevilla":          "Europe/Madrid",
	"seville":          "Europe/Madrid",
	"bilbao":           "Europe/Madrid",
	"lyon":             "Europe/Paris",
	"marseille":        "Europe/Paris",
	"nice":             "Europe/Paris",
	"munich":           "Europe/Berlin",
	"munchen":          "Europe/Berlin",
	"frankfurt":        "Europe/Berlin",
	"hamburg":          "Europe/Berlin",
	"cologne":          "Europe/Berlin",
	"koln":             "Europe/Berlin",
	"milan":            "Europe/Rome",
	"milano":           "Europe/Rome",
	"naples":           "Europe/Rome",
	"napoli":           "Europe/Rome",
	"florence":         "Europe/Rome",
	"firenze":          "Europe/Rome",
	"venice":           "Europe/Rome",
	"venezia":          "Europe/Rome",
	"geneva":           "Europe/Zurich",
	"ginebra":          "Europe/Zurich",
	"basel":            "Europe/Zurich",
	"bern":             "Europe/Zurich",
	"rotterdam":        "Europe/Amsterdam",
	"antwerp":          "Europe/Brussels",
	"porto":            "Europe/Lisbon",
	"oporto":           "Europe/Lisbon",
	"lisboa":           "Europe/Lisbon",
	"krakow":           "Europe/Warsaw",
	"praga":            "Europe/Prague",
	"estocolmo":        "Europe/Stockholm",
	"copenhague":       "Europe/Copenhagen",
	"atenas":           "Europe/Athens",
	"ankara":           "Europe/Istanbul",
	"estambul":         "Europe/Istanbul",
	"st petersburg":    "Europe/Moscow",
	"osaka":            "Asia/Tokyo",
	"kyoto":            "Asia/Tokyo",
	"yokohama":         "Asia/Tokyo",
	"nagoya":           "Asia/Tokyo",
	"seul":             "Asia/Seoul",
	"busan":            "Asia/Seoul",
	"shenzhen":         "Asia/Shanghai",
	"guangzhou":        "Asia/Shanghai",
	"chengdu":          "Asia/Shanghai",
	"hangzhou":         "Asia/Shanghai",
	"wuhan":            "Asia/Shanghai",
	"hk":               "Asia/Hong_Kong",
	"singapur":         "Asia/Singapore",
	"sg":               "Asia/Singapore",
	"new delhi":        "Asia/Kolkata",
	"bombay":           "Asia/Kolkata",
	"bangalore":        "Asia/Kolkata",
	"bengaluru":        "Asia/Kolkata",
	"hyderabad":        "Asia/Kolkata",
	"chennai":          "Asia/Kolkata",
	"pune":             "Asia/Kolkata",
	"islamabad":        "Asia/Karachi",
	"lahore":           "Asia/Karachi",
	"phuket":           "Asia/Bangkok",
	"hanoi":            "Asia/Ho_Chi_Minh",
	"hcmc":             "Asia/Ho_Chi_Minh",
	"ho chi minh city": "Asia/Ho_Chi_Minh",
	"cebu":             "Asia/Manila",
	"kl":               "Asia/Kuala_Lumpur",
	"bali":             "Asia/Makassar",
	"abu dhabi":        "Asia/Dubai",
	"jeddah":           "Asia/Riyadh",
	"mecca":            "Asia/Riyadh",
	"jerusalen":        "Asia/Jerusalem",
	"el cairo":         "Africa/Cairo",
	"alexandria":       "Africa/Cairo",
	"cape town":        "Africa/Johannesburg",
	"pretoria":         "Africa/Johannesburg",
	"durban":           "Africa/Johannesburg",
	"abuja":            "Africa/Lagos",
	"marrakech":        "Africa/Casablanca",
	"rabat":            "Africa/Casablanca",
	"wellington":       "Pacific/Auckland",
	"christchurch":     "Pacific/Auckland",
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

type zoneCandidate struct {
	folded  string
	display string
	iana    string
	words   []string
	runes   []rune
}

type zoneIndex struct {
	ianaByFolded map[string]string
	cityByFolded map[string]string
	cities       []cityEntry
	candidates   []zoneCandidate
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

	candidates := make([]zoneCandidate, 0, len(cities)+len(zoneAliases))
	for _, c := range cities {
		candidates = append(candidates, zoneCandidate{
			folded:  c.folded,
			display: c.display,
			iana:    cityByFolded[c.folded],
			words:   strings.Fields(c.folded),
			runes:   []rune(c.folded),
		})
	}

	aliasKeys := make([]string, 0, len(zoneAliases))
	for k := range zoneAliases {
		if len([]rune(k)) >= 5 {
			aliasKeys = append(aliasKeys, k)
		}
	}
	sort.Strings(aliasKeys)
	for _, k := range aliasKeys {
		candidates = append(candidates, zoneCandidate{
			folded:  k,
			display: titleCaseAlias(k),
			iana:    zoneAliases[k],
			words:   strings.Fields(k),
			runes:   []rune(k),
		})
	}

	return zoneIndex{
		ianaByFolded: ianaByFolded,
		cityByFolded: cityByFolded,
		cities:       cities,
		candidates:   candidates,
	}
}

func titleCaseAlias(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		runes := []rune(w)
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
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
	return LookupZone(name, false)
}

// LookupZone resolves a timezone name, offset, abbreviation, city, or alias to a *time.Location.
// When strict is false, unambiguous fuzzy matches (length >= 4) auto-resolve; otherwise an error
// with up to 5 "did you mean" suggestions is returned.
func LookupZone(name string, strict bool) (*time.Location, error) {
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

	// 7. Tiered fuzzy matching (only for non-slashed inputs).
	if strings.ContainsRune(folded, '/') {
		return nil, &Error{Pos: -1, Msg: fmt.Sprintf("unknown timezone or word %q", name)}
	}

	qRunes := []rune(folded)
	if len(qRunes) < 4 {
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

	matches := fuzzyMatchCandidates(folded, qRunes)
	seenZones := make(map[string]bool, len(matches))
	var uniqueZones []string
	var displays []string
	for _, m := range matches {
		if seenZones[m.iana] {
			continue
		}
		seenZones[m.iana] = true
		uniqueZones = append(uniqueZones, m.iana)
		displays = append(displays, m.display)
	}

	if !strict && len(uniqueZones) == 1 {
		return loadZoneLocation(uniqueZones[0], name)
	}
	if len(uniqueZones) >= 1 && len(uniqueZones) <= 5 {
		return nil, &Error{
			Pos: -1,
			Msg: fmt.Sprintf("unknown timezone or word %q; did you mean: %s", name, strings.Join(displays, ", ")),
		}
	}
	return nil, &Error{Pos: -1, Msg: fmt.Sprintf("unknown timezone or word %q", name)}
}

func fuzzyMatchCandidates(folded string, qRunes []rune) []zoneCandidate {
	// Tier 1: Word-prefix match.
	qWords := strings.Fields(folded)
	var tier1 []zoneCandidate
	for _, cand := range zones.candidates {
		if matchesWordPrefix(qWords, cand.words) {
			tier1 = append(tier1, cand)
		}
	}
	if len(tier1) > 0 {
		return tier1
	}

	// Tier 2: Substring containment.
	var tier2 []zoneCandidate
	for _, cand := range zones.candidates {
		if strings.Contains(cand.folded, folded) {
			tier2 = append(tier2, cand)
		}
	}
	if len(tier2) > 0 {
		return tier2
	}

	// Tier 3: Optimal string alignment (Damerau-Levenshtein) typo tolerance.
	bestDist := 3
	var tier3 []zoneCandidate
	for _, cand := range zones.candidates {
		maxDist := 1
		if len(qRunes) >= 9 && len(cand.runes) >= 9 {
			maxDist = 2
		}
		if !firstCharOrTransposed(qRunes, cand.runes) {
			continue
		}
		diff := len(qRunes) - len(cand.runes)
		if diff < 0 {
			diff = -diff
		}
		if diff > maxDist {
			continue
		}
		dist := osaDistance(qRunes, cand.runes)
		if dist > maxDist {
			continue
		}
		if dist < bestDist {
			bestDist = dist
			tier3 = append(tier3[:0], cand)
		} else if dist == bestDist {
			tier3 = append(tier3, cand)
		}
	}
	return tier3
}

func matchesWordPrefix(qWords, candWords []string) bool {
	if len(qWords) == 0 || len(qWords) > len(candWords) {
		return false
	}
	if len(qWords) == 1 {
		q := qWords[0]
		for _, w := range candWords {
			if strings.HasPrefix(w, q) {
				return true
			}
		}
		return false
	}
	for i, q := range qWords {
		if !strings.HasPrefix(candWords[i], q) {
			return false
		}
	}
	return true
}

func firstCharOrTransposed(a, b []rune) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	if a[0] == b[0] {
		return true
	}
	return len(a) >= 2 && len(b) >= 2 && a[0] == b[1] && a[1] == b[0]
}

// osaDistance computes the Optimal String Alignment (restricted Damerau-Levenshtein) distance.
func osaDistance(a, b []rune) int {
	m, n := len(a), len(b)
	cols := n + 1
	d := make([]int, (m+1)*cols)
	for i := 0; i <= m; i++ {
		d[i*cols] = i
	}
	for j := 0; j <= n; j++ {
		d[j] = j
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			best := d[(i-1)*cols+j] + 1
			if ins := d[i*cols+(j-1)] + 1; ins < best {
				best = ins
			}
			if sub := d[(i-1)*cols+(j-1)] + cost; sub < best {
				best = sub
			}
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				if trans := d[(i-2)*cols+(j-2)] + 1; trans < best {
					best = trans
				}
			}
			d[i*cols+j] = best
		}
	}
	return d[m*cols+n]
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
