// Package expr lexes, normalizes, parses, evaluates, and formats date and time expressions.
package expr

import "fmt"

type TokenKind int

const (
	EOF       TokenKind = iota
	Illegal             // a character the lexer doesn't know
	Number              // 2, 1700000000, 1.5
	Quantity            // number glued to letters: 2d, 30m, 2pm, 20th, 2days
	Word                // letters (any language), may contain digits: now, días, Q3, America/Mexico_City
	String              // "…" or '…'; Text holds the contents without the quotes
	Date                // 2026-06-02
	DateTime            // 2026-06-02T14:30:00-06:00
	Clock               // 14:30, 2:30, 14:30:05
	SlashDate           // 02/06/2026
	Op                  // + - * , | @ .. ( )
)

func (k TokenKind) String() string {
	switch k {
	case EOF:
		return "EOF"
	case Illegal:
		return "Illegal"
	case Number:
		return "Number"
	case Quantity:
		return "Quantity"
	case Word:
		return "Word"
	case String:
		return "String"
	case Date:
		return "Date"
	case DateTime:
		return "DateTime"
	case Clock:
		return "Clock"
	case SlashDate:
		return "SlashDate"
	case Op:
		return "Op"
	default:
		return fmt.Sprintf("TokenKind(%d)", int(k))
	}
}

type Lang string

const (
	EN Lang = "en"
	ES Lang = "es"
)

type Token struct {
	Kind TokenKind
	Text string // exact input text, case preserved
	Pos  int    // byte offset in the input
	Num  string // Quantity only: "2", "1.5"
	Unit string // Quantity only: letters, case preserved
	Key  string // canonical keyword, set by Normalize; "" = not a keyword
	Lang Lang   // set by Normalize when the word belongs to one language only
	Tie  bool   // set by Normalize when it had to guess between two meanings
}
