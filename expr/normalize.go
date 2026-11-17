package expr

import (
	"strings"
	"unicode"
)

const maxPhraseWords = 4

// Normalize assigns canonical keywords (Key) to words and quantities,
// merges multi-word phrases, sets Lang and Tie, and drops filler words.
func Normalize(toks []Token, lang Lang) []Token {
	return NormalizeWithAliases(toks, lang, nil)
}

// NormalizeWithAliases performs Normalize while matching longest folded word
// sequences against aliases and expanding boundary acronyms.
func NormalizeWithAliases(toks []Token, lang Lang, aliases map[string]string) []Token {
	var foldedAliases map[string]string
	maxAliasWords := 0
	if len(aliases) > 0 {
		foldedAliases = make(map[string]string, len(aliases))
		for k := range aliases {
			fields := strings.Fields(k)
			if len(fields) == 0 {
				continue
			}
			folded := fold(strings.Join(fields, " "))
			foldedAliases[folded] = folded
			if len(fields) > maxAliasWords {
				maxAliasWords = len(fields)
			}
		}
	}

	out := make([]Token, 0, len(toks))

	for i := 0; i < len(toks); {
		tok := toks[i]
		switch tok.Kind {
		case Word:
			if strings.Contains(tok.Text, "/") {
				tok.Key = ""
				out = append(out, tok)
				i++
				continue
			}
			if len(out) > 0 && out[len(out)-1].Key == "holidays" && isUpperASCII2(tok.Text) {
				tok.Key = ""
				tok.Lang = ""
				tok.Tie = false
				out = append(out, tok)
				i++
				continue
			}

			if len(foldedAliases) > 0 {
				maxA := 0
				for maxA < maxAliasWords && i+maxA < len(toks) {
					next := toks[i+maxA]
					if next.Kind != Word || strings.Contains(next.Text, "/") {
						break
					}
					maxA++
				}
				aliasMatched := false
				for n := maxA; n >= 1; n-- {
					phrase := joinWordsFolded(toks[i : i+n])
					if aliasKey, ok := foldedAliases[phrase]; ok {
						out = append(out, Token{
							Kind: Word,
							Text: joinWordsRaw(toks[i : i+n]),
							Pos:  tok.Pos,
							Key:  "alias",
							Unit: aliasKey,
						})
						i += n
						aliasMatched = true
						break
					}
				}
				if aliasMatched {
					continue
				}
			}

			if expanded, ok := expandBoundaryAcronym(tok); ok {
				out = append(out, expanded...)
				i++
				continue
			}

			maxN := 0
			for maxN < maxPhraseWords && i+maxN < len(toks) {
				next := toks[i+maxN]
				if next.Kind != Word || strings.Contains(next.Text, "/") {
					break
				}
				maxN++
			}

			matched := false
			for n := maxN; n >= 1; n-- {
				phrase := joinWordsFolded(toks[i : i+n])
				if n == 3 {
					if special, ok := resolveSpecialPhrase(phrase, toks, i); ok {
						out = append(out, special...)
						i += 3
						matched = true
						break
					}
				}
				if key, tokLang, tie, ok := lookupPhrase(phrase, toks, i, out, lang); ok {
					if key != "" {
						out = append(out, Token{
							Kind: Word,
							Text: joinWordsRaw(toks[i : i+n]),
							Pos:  tok.Pos,
							Key:  key,
							Lang: tokLang,
							Tie:  tie,
						})
					}
					i += n
					matched = true
					break
				}
			}
			if !matched {
				tok.Key = ""
				out = append(out, tok)
				i++
			}

		case Quantity:
			tok.Key, tok.Lang = quantityKey(tok.Unit)
			out = append(out, tok)
			i++

		default:
			out = append(out, tok)
			i++
		}
	}

	return out
}

var boundaryAcronyms = map[string][2]string{
	"eod":  {"end", "day"},
	"sod":  {"start", "day"},
	"bod":  {"start", "day"},
	"eow":  {"end", "week"},
	"sow":  {"start", "week"},
	"bow":  {"start", "week"},
	"eoww": {"end", "workweek"},
	"soww": {"start", "workweek"},
	"boww": {"start", "workweek"},
	"eom":  {"end", "month"},
	"som":  {"start", "month"},
	"bom":  {"start", "month"},
	"eoq":  {"end", "quarter"},
	"soq":  {"start", "quarter"},
	"boq":  {"start", "quarter"},
	"eoy":  {"end", "year"},
	"soy":  {"start", "year"},
	"boy":  {"start", "year"},
}

func isBoundaryAcronym(folded string) bool {
	_, ok := boundaryAcronyms[folded]
	return ok
}

func expandBoundaryAcronym(tok Token) ([]Token, bool) {
	spec, ok := boundaryAcronyms[fold(tok.Text)]
	if !ok {
		return nil, false
	}
	return []Token{
		{Kind: Word, Text: tok.Text, Pos: tok.Pos, Key: spec[0], Lang: EN},
		{Kind: Word, Text: tok.Text, Pos: tok.Pos, Key: "of", Lang: EN},
		{Kind: Word, Text: tok.Text, Pos: tok.Pos, Key: spec[1], Lang: EN},
	}, true
}

// InputLang returns the Lang of the first token that has one, or "".
func InputLang(toks []Token) Lang {
	for _, tok := range toks {
		if tok.Lang != "" {
			return tok.Lang
		}
	}
	return ""
}

func fold(s string) string {
	return strings.Map(func(r rune) rune {
		r = unicode.ToLower(r)
		switch r {
		case 'á':
			return 'a'
		case 'é':
			return 'e'
		case 'í':
			return 'i'
		case 'ó':
			return 'o'
		case 'ú', 'ü':
			return 'u'
		case 'ñ':
			return 'n'
		default:
			return r
		}
	}, s)
}

func resolveSpecialPhrase(phrase string, toks []Token, i int) ([]Token, bool) {
	switch phrase {
	case "fin de semana":
		return []Token{
			{Kind: Word, Text: toks[i].Text, Pos: toks[i].Pos, Key: "end", Lang: ES, Tie: true},
			{Kind: Word, Text: toks[i+1].Text, Pos: toks[i+1].Pos, Key: "of", Lang: ES},
			{Kind: Word, Text: toks[i+2].Text, Pos: toks[i+2].Pos, Key: "week", Lang: ES},
		}, true
	case "en la manana", "por la manana":
		res := make([]Token, 3)
		for k := 0; k < 3; k++ {
			t := toks[i+k]
			t.Key = ""
			t.Lang = ""
			t.Tie = false
			res[k] = t
		}
		return res, true
	default:
		return nil, false
	}
}

func lookupPhrase(phrase string, toks []Token, i int, out []Token, lang Lang) (string, Lang, bool, bool) {
	switch phrase {
	case "ago":
		key, tokLang, tie := resolveAgo(toks, i, out, lang)
		return key, tokLang, tie, true
	case "second", "segundo", "segunda":
		key, tokLang := resolveSecond(phrase, toks, i)
		return key, tokLang, false, true
	case "de":
		return resolveDe(toks, i), ES, false, true
	case "a la", "a las":
		if i+2 < len(toks) && (toks[i+2].Kind == Clock || toks[i+2].Kind == Number || toks[i+2].Kind == Quantity) {
			return "", ES, false, true
		}
		return "", "", false, false
	}

	enKey, inEN := vocabEN[phrase]
	esKey, inES := vocabES[phrase]
	switch {
	case inEN && inES:
		if enKey == esKey {
			return enKey, "", false, true
		}
		if lang == ES {
			return esKey, ES, true, true
		}
		return enKey, EN, true, true
	case inEN:
		return enKey, EN, false, true
	case inES:
		if esKey == "" && !hasSpanishFollower(toks, i+1) {
			return "", "", false, false
		}
		return esKey, ES, false, true
	default:
		return "", "", false, false
	}
}

func hasSpanishFollower(toks []Token, start int) bool {
	if start >= len(toks) {
		return false
	}
	if toks[start].Kind == Number || toks[start].Kind == Quantity {
		return true
	}
	for n := 1; n <= maxPhraseWords && start+n <= len(toks); n++ {
		next := toks[start+n-1]
		if next.Kind != Word || strings.Contains(next.Text, "/") {
			break
		}
		phrase := joinWordsFolded(toks[start : start+n])
		if _, ok := vocabES[phrase]; ok {
			return true
		}
		if n == 3 {
			if _, ok := resolveSpecialPhrase(phrase, toks, start); ok {
				return true
			}
		}
	}
	return false
}

func resolveAgo(toks []Token, i int, out []Token, lang Lang) (string, Lang, bool) {
	if len(out) > 0 {
		prev := out[len(out)-1]
		if (prev.Kind == Quantity && prev.Key != "ordinal" && prev.Key != "am" && prev.Key != "pm") ||
			(prev.Kind == Word && isUnitKey(prev.Key)) {
			return "ago", EN, false
		}
		if prev.Kind == Number {
			return "august", ES, false
		}
	}
	if i+1 < len(toks) && toks[i+1].Kind == Number {
		return "august", ES, false
	}
	if lang == ES {
		return "august", ES, true
	}
	return "ago", EN, true
}

func resolveSecond(phrase string, toks []Token, i int) (string, Lang) {
	tokLang := ES
	if phrase == "second" {
		tokLang = EN
	}
	if i > 0 && toks[i-1].Kind == Number {
		return "second", tokLang
	}
	return "2nd", tokLang
}

func resolveDe(toks []Token, i int) string {
	if i == 0 || (toks[i-1].Kind == Op && (toks[i-1].Text == "," || toks[i-1].Text == "|")) {
		return "from"
	}
	return "of"
}

func joinWordsFolded(toks []Token) string {
	if len(toks) == 1 {
		return fold(toks[0].Text)
	}
	parts := make([]string, len(toks))
	for i, t := range toks {
		parts[i] = fold(t.Text)
	}
	return strings.Join(parts, " ")
}

func joinWordsRaw(toks []Token) string {
	if len(toks) == 1 {
		return toks[0].Text
	}
	parts := make([]string, len(toks))
	for i, t := range toks {
		parts[i] = t.Text
	}
	return strings.Join(parts, " ")
}

func quantityKey(unit string) (string, Lang) {
	switch unit {
	case "m":
		return "minute", ""
	case "M":
		return "month", ""
	}

	folded := fold(unit)
	switch folded {
	case "ms":
		return "millisecond", ""
	case "s":
		return "second", ""
	case "h":
		return "hour", ""
	case "d":
		return "day", ""
	case "w":
		return "week", ""
	case "q":
		return "quarter", ""
	case "y":
		return "year", ""
	case "am":
		return "am", ""
	case "pm":
		return "pm", ""
	case "st", "nd", "rd", "th":
		return "ordinal", EN
	case "o", "a", "º", "ª", "er", "ro", "do", "da", "to", "ta":
		return "ordinal", ES
	}

	enKey, inEN := vocabEN[folded]
	esKey, inES := vocabES[folded]
	if inEN && !isDurationUnitKey(enKey) {
		inEN = false
	}
	if inES && !isDurationUnitKey(esKey) {
		inES = false
	}
	switch {
	case inEN && inES:
		if enKey == esKey {
			return enKey, ""
		}
		return enKey, EN
	case inEN:
		return enKey, EN
	case inES:
		return esKey, ES
	default:
		return "", ""
	}
}

func isUpperASCII2(s string) bool {
	return len(s) == 2 && s[0] >= 'A' && s[0] <= 'Z' && s[1] >= 'A' && s[1] <= 'Z'
}

func isUnitKey(key string) bool {
	if isDurationUnitKey(key) {
		return true
	}
	switch key {
	case "workweek", "businessday":
		return true
	default:
		return false
	}
}

func isDurationUnitKey(key string) bool {
	switch key {
	case "millisecond", "second", "minute", "hour", "day", "week", "month", "quarter", "half", "year":
		return true
	default:
		return false
	}
}
