package expr

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	reDateTime  = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}(?::[0-9]{2}(?:\.[0-9]+)?)?(?:Z|[+-][0-9]{2}:?[0-9]{2})?`)
	reDate      = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}`)
	reSlashDate = regexp.MustCompile(`^[0-9]{1,2}/[0-9]{1,2}/(?:[0-9]{4}|[0-9]{2})`)
	reClock     = regexp.MustCompile(`^[0-9]{1,2}:[0-9]{2}(?::[0-9]{2})?`)
	reNumber    = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?`)
)

// Lex splits input into tokens without validating values or changing case.
// It never fails and always ends with an EOF token at Pos len(input).
func Lex(input string) []Token {
	var toks []Token
	i := 0
	for i < len(input) {
		r, size := utf8.DecodeRuneInString(input[i:])
		if unicode.IsSpace(r) {
			i += size
			continue
		}

		if r >= '0' && r <= '9' {
			rest := input[i:]
			if m := reDateTime.FindString(rest); m != "" && !digitAt(rest, len(m)) {
				toks = append(toks, Token{Kind: DateTime, Text: m, Pos: i})
				i += len(m)
				continue
			}
			if m := reDate.FindString(rest); m != "" && !digitAt(rest, len(m)) {
				toks = append(toks, Token{Kind: Date, Text: m, Pos: i})
				i += len(m)
				continue
			}
			if m := reSlashDate.FindString(rest); m != "" && !digitAt(rest, len(m)) {
				toks = append(toks, Token{Kind: SlashDate, Text: m, Pos: i})
				i += len(m)
				continue
			}
			if m := reClock.FindString(rest); m != "" && !digitAt(rest, len(m)) {
				toks = append(toks, Token{Kind: Clock, Text: m, Pos: i})
				i += len(m)
				continue
			}
			num := reNumber.FindString(rest)
			j := i + len(num)
			unitStart := j
			for j < len(input) {
				lr, lsize := utf8.DecodeRuneInString(input[j:])
				if !unicode.IsLetter(lr) {
					break
				}
				j += lsize
			}
			if j > unitStart {
				toks = append(toks, Token{
					Kind: Quantity,
					Text: input[i:j],
					Pos:  i,
					Num:  num,
					Unit: input[unitStart:j],
				})
			} else {
				toks = append(toks, Token{
					Kind: Number,
					Text: num,
					Pos:  i,
				})
			}
			i = j
			continue
		}

		if unicode.IsLetter(r) {
			start := i
			i += size
			hasSlash := false
			for i < len(input) {
				nr, nsize := utf8.DecodeRuneInString(input[i:])
				if unicode.IsLetter(nr) || unicode.IsDigit(nr) || nr == '_' {
					i += nsize
					continue
				}
				if nr == '/' && i+nsize < len(input) {
					nextR, _ := utf8.DecodeRuneInString(input[i+nsize:])
					if unicode.IsLetter(nextR) {
						hasSlash = true
						i += nsize
						continue
					}
				}
				if nr == '-' && hasSlash && i+nsize < len(input) {
					nextR, _ := utf8.DecodeRuneInString(input[i+nsize:])
					if unicode.IsLetter(nextR) {
						i += nsize
						continue
					}
				}
				break
			}
			toks = append(toks, Token{Kind: Word, Text: input[start:i], Pos: start})
			continue
		}

		if r == '"' || r == '\'' {
			if end := strings.IndexByte(input[i+1:], byte(r)); end >= 0 {
				toks = append(toks, Token{Kind: String, Text: input[i+1 : i+1+end], Pos: i})
				i += end + 2
			} else {
				toks = append(toks, Token{Kind: Illegal, Text: input[i:], Pos: i})
				i = len(input)
			}
			continue
		}

		if op, ok := matchTwoCharOp(input[i:]); ok {
			toks = append(toks, Token{Kind: Op, Text: op, Pos: i})
			i += 2
			continue
		}

		if isSingleOp(r) {
			toks = append(toks, Token{Kind: Op, Text: input[i : i+size], Pos: i})
			i += size
			continue
		}

		toks = append(toks, Token{Kind: Illegal, Text: input[i : i+size], Pos: i})
		i += size
	}

	toks = append(toks, Token{Kind: EOF, Pos: len(input)})
	return toks
}

func matchTwoCharOp(rest string) (string, bool) {
	for _, op := range []string{"..", "==", "!=", "<=", ">="} {
		if strings.HasPrefix(rest, op) {
			return op, true
		}
	}
	return "", false
}

func digitAt(s string, idx int) bool {
	return idx < len(s) && s[idx] >= '0' && s[idx] <= '9'
}

func isSingleOp(r rune) bool {
	switch r {
	case '+', '-', '*', ',', '|', '@', '(', ')', '=', '<', '>':
		return true
	default:
		return false
	}
}
