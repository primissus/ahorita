package expr

import "strings"

const maxPhraseWords = 4

// Normalize assigns canonical keywords (Key) to words and quantities,
// merges multi-word phrases, and drops filler words.
func Normalize(toks []Token, lang Lang) []Token {
	_ = lang
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
				phrase := joinWordsLower(toks[i : i+n])
				if key, ok := vocabEN[phrase]; ok {
					if key != "" {
						out = append(out, Token{
							Kind: Word,
							Text: joinWordsRaw(toks[i : i+n]),
							Pos:  tok.Pos,
							Key:  key,
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
			tok.Key = quantityKey(tok.Unit)
			out = append(out, tok)
			i++

		default:
			out = append(out, tok)
			i++
		}
	}

	return out
}

func joinWordsLower(toks []Token) string {
	if len(toks) == 1 {
		return strings.ToLower(toks[0].Text)
	}
	parts := make([]string, len(toks))
	for i, t := range toks {
		parts[i] = strings.ToLower(t.Text)
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

func quantityKey(unit string) string {
	switch unit {
	case "m":
		return "minute"
	case "M":
		return "month"
	}

	lower := strings.ToLower(unit)
	switch lower {
	case "s":
		return "second"
	case "h":
		return "hour"
	case "d":
		return "day"
	case "w":
		return "week"
	case "q":
		return "quarter"
	case "y":
		return "year"
	case "am":
		return "am"
	case "pm":
		return "pm"
	case "st", "nd", "rd", "th":
		return "ordinal"
	}

	if key, ok := vocabEN[lower]; ok {
		return key
	}
	return ""
}
