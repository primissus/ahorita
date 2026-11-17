package expr

import (
	"fmt"
	"strconv"
	"strings"
)

/*
Grammar of ahorita expressions (canonical English keys after Lex + Normalize):

	expr         = stage { ( "," | "|" | "to"-stage ) stage } ;
	stage        = stage-body [ "inclusive" ] ;
	stage-body   = to-stage
	             | "explain"
	             | "next" Number            (* take N occurrences *)
	             | "@" zone
	             | "in" zone                (* only when not dur / period / year / rel *)
	             | zone                     (* bare zone stage *)
	             | span-expr ;

	to-stage     = "to" ( unit | to-key | zone ) ;

	span-expr    = add-expr [ ".." add-expr ] ;
	add-expr     = mul-expr { ( "+" | "-" ) mul-expr } ;
	mul-expr     = postfix-expr { "*" mul-rhs } ;
	mul-rhs      = Number                   (* parsed as num even if 4 digits *)
	             | postfix-expr ;

	postfix-expr = of-expr { clock | "@" zone | "in" zone } ;
	of-expr      = primary [ "of" of-expr ] ;  (* allowed after week, weekday, month, period *)

	primary      = Date | DateTime | SlashDate
	             | "(" span-expr ")"
	             | "@" Number               (* unix *)
	             | clock                    (* Clock [am|pm], Quantity(am|pm), Number (am|pm) *)
	             | dur [ "ago" ]
	             | Number                   (* 4 digits -> year, else num *)
	             | day-first-month [ "last" | "next" | "this" ]
	             | ( Quantity(ordinal) | "1st".."5th" ) "week"
	             | "now" | "today" | "yesterday" | "tomorrow"
	             | ( "last" | "next" | "this" ) ( unit | weekday | month )
	             | ( "next" "businessday" "onorafter" | "last" "businessday" "onorbefore" ) [ of-expr ]
	             | weekday [ "last" | "next" | "this" | "closest" [ "to" postfix-expr ] ]
	             | month [ "last" | "next" | "this" ]
	             | period                   (* q1..q4 | h1..h2, optional 4-digit year *)
	             | "week" ( Number | "last" | "next" | "this" )
	             | unit ( "last" | "next" | "this" )
	             | "ago" dur
	             | "in" ( dur | month | period | 4-digit-year | rel )
	             | ( "start" | "end" ) "of" ( unit | of-expr )
	             | "closest" weekday [ "to" postfix-expr ]
	             | "during" dur
	             | "holidays" [ Word(2 letters) ] [ 4-digit-year ]
	             | "starting" add-expr
	             | "every" every-spec
	             | "cron" String
	             | "from" add-expr "to" add-expr
	             | "between" add-expr "and" add-expr
	             | "howmany" ( unit | weekday ) ( "between" add-expr "and" add-expr | "from" add-expr "to" add-expr ) ;

	dur          = dur-part { dur-part } ;
	dur-part     = Quantity(duration-unit) | Number unit ;
	clock        = Clock [ "am" | "pm" ] | Quantity("am" | "pm") | Number ( "am" | "pm" ) ;
	month        = month-name [ day ] [ 4-digit-year ]
	             | day [ "of" ] month-name [ [ "of" ] 4-digit-year ] ;
	zone         = String
	             | "utc"                    (* includes gmt, z; utc/gmt may take +/- offset *)
	             | ( "utc" | "gmt" ) ( "+" | "-" ) ( Clock | Number )
	             | ( "+" | "-" ) ( Clock | Number )
	             | Word(Key=="") { Word(Key=="") } ;

AST Node Op table (contract with the evaluator):

	Op            | Text                    | Args
	--------------+-------------------------+--------------------------------------------
	date          | as typed (2026-06-02)   | -
	datetime      | as typed                | -
	clock         | 24h HH:MM or HH:MM:SS   | -
	slash         | as typed                | -
	unix          | digits                  | -
	num           | digits                  | -
	year          | 4-digit year            | -
	dur           | -                       | parts (Op=unit key, Text=number)
	+  -  *       | -                       | left, right
	now today     | -                       | -
	yesterday     | -                       | -
	tomorrow      | -                       | -
	ago           | -                       | dur
	in            | -                       | dur
	rel           | last | next | this      | unit | weekday | month
	orsame        | -                       | rel [, anchor value]
	unit          | unit key                | -
	weekday       | monday..sunday          | -
	month         | january..december       | optional day (num), optional year
	period        | q1..q4 | h1 | h2        | optional year
	week          | week number             | -
	bound         | start | end             | unit | target node
	of            | -                       | selector, period
	closest       | -                       | weekday [, anchor value]
	during        | -                       | dur
	holidays      | 2-letter code or ""     | [year]
	starting      | -                       | value
	every         | -                       | schedule parts (num, unit, weekday, key, ord, month, clock)
	cron          | 5 cron fields           | -
	take          | count                   | -
	explain       | -                       | -
	span          | -                       | from, to
	count         | -                       | unit | weekday, from, to
	inclusive     | -                       | wrapped stage node
	to            | -                       | unit | zone | key
	key           | keyword text            | -
	zone          | space-joined zone name  | -
	interpret     | -                       | [value,] zone
	inzone        | -                       | [value,] zone
	at            | -                       | value, clock
*/

const (
	bpLowest = 0
	bpSpan   = 1 // ..
	bpAdd    = 2 // + -
	bpMul    = 3 // *
)

// toKeys holds extra keyword targets allowed after a stage-level "to".
var toKeys = map[string]bool{
	"cron": true,
	"ics":  true,
	"grid": true,
}

// prefix maps canonical starting keywords to their prefix parser functions.
// Later plans add entries in init().
var prefix = map[string]func(*parser) (Node, error){}

func init() {
	for _, kw := range []string{"now", "today", "yesterday", "tomorrow"} {
		prefix[kw] = (*parser).parseSimpleValue
	}
	for _, kw := range []string{"last", "next", "this"} {
		prefix[kw] = (*parser).parseRelPrefix
	}
	for _, kw := range []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"} {
		prefix[kw] = (*parser).parseWeekday
	}
	for _, kw := range []string{"january", "february", "march", "april", "may", "june", "july", "august", "september", "october", "november", "december"} {
		prefix[kw] = (*parser).parseMonthPrefix
	}
	for _, kw := range []string{"q1", "q2", "q3", "q4", "h1", "h2"} {
		prefix[kw] = (*parser).parsePeriod
	}
	prefix["week"] = (*parser).parseWeekPrefix
	for _, kw := range []string{"second", "minute", "hour", "day", "month", "quarter", "half", "year", "workweek", "businessday"} {
		prefix[kw] = (*parser).parseUnitRel
	}
	for _, kw := range []string{"1st", "2nd", "3rd", "4th", "5th"} {
		prefix[kw] = (*parser).parseOrdWordWeek
	}
	prefix["ago"] = (*parser).parseAgoPrefix
	prefix["in"] = (*parser).parseInPrefix
	prefix["start"] = (*parser).parseBound
	prefix["end"] = (*parser).parseBound
	prefix["closest"] = (*parser).parseClosestPrefix
	prefix["during"] = (*parser).parseDuring
	prefix["holidays"] = (*parser).parseHolidays
	prefix["starting"] = (*parser).parseStarting
	prefix["every"] = (*parser).parseEvery
	prefix["cron"] = (*parser).parseCron
	prefix["from"] = (*parser).parseFromSpan
	prefix["between"] = (*parser).parseBetweenSpan
	prefix["howmany"] = (*parser).parseCount
}

type parser struct {
	toks  []Token
	pos   int
	input string
}

// Parse lexes, normalizes, and parses an expression into a pipeline of AST stages.
func Parse(input string, lang Lang) (Expr, error) {
	toks := Normalize(Lex(input), lang)

	var ties []int
	for _, tok := range toks {
		if tok.Tie {
			ties = append(ties, tok.Pos)
		}
	}

	expr := Expr{
		Input: input,
		Lang:  InputLang(toks),
		Ties:  ties,
	}

	if len(toks) == 1 && toks[0].Kind == EOF {
		return Expr{}, &Error{
			Pos:      0,
			Msg:      "empty expression",
			Expected: []string{"a value"},
		}
	}

	p := &parser{
		toks:  toks,
		input: input,
	}

	for {
		stage, err := p.parseStage()
		if err != nil {
			return Expr{}, err
		}
		expr.Stages = append(expr.Stages, stage)

		tok := p.cur()
		if tok.Kind == EOF {
			break
		}
		if tok.Kind == Op && (tok.Text == "," || tok.Text == "|") {
			p.next()
			if p.cur().Kind == EOF {
				return Expr{}, p.unexpectedErr(p.cur(), "a value")
			}
			continue
		}
		if tok.Kind == Word && tok.Key == "to" {
			// Stage-level "to" starts a new stage without consuming "to".
			continue
		}
		return Expr{}, p.unexpectedErr(tok)
	}

	return expr, nil
}

func (p *parser) cur() Token {
	return p.peek(0)
}

func (p *parser) peek(offset int) Token {
	idx := p.pos + offset
	if idx >= len(p.toks) {
		return p.toks[len(p.toks)-1]
	}
	return p.toks[idx]
}

func (p *parser) next() Token {
	tok := p.cur()
	if p.pos < len(p.toks)-1 {
		p.pos++
	}
	return tok
}

func (p *parser) parseStage() (Node, error) {
	n, err := p.parseStageBody()
	if err != nil {
		return Node{}, err
	}
	if p.cur().Kind == Word && p.cur().Key == "inclusive" {
		p.next()
		n = Node{Op: "inclusive", Pos: n.Pos, Args: []Node{n}}
	}
	return n, nil
}

func (p *parser) parseStageBody() (Node, error) {
	tok := p.cur()
	if tok.Kind == Illegal || (tok.Kind == Op && tok.Text == ")") {
		return Node{}, p.unexpectedErr(tok)
	}
	if tok.Kind == Word {
		switch tok.Key {
		case "to":
			return p.parseToStage()
		case "explain":
			p.next()
			return Node{Op: "explain", Pos: tok.Pos}, nil
		case "next":
			if p.startsTakeStage() {
				return p.parseTakeStage()
			}
		case "in":
			if p.startsInZone() {
				return p.parseInZoneStage()
			}
		case "utc", "":
			return p.parseZone()
		}
	}
	if tok.Kind == String || p.startsZoneOffset(0) {
		return p.parseZone()
	}
	if tok.Kind == Op && tok.Text == "@" && p.peek(1).Kind != Number {
		return p.parseAtZoneStage()
	}
	return p.parseExpr(bpLowest)
}

func (p *parser) parseToStage() (Node, error) {
	toTok := p.next()
	next := p.cur()
	if next.Kind == EOF {
		return Node{}, p.unexpectedErr(next, "a unit", "a timezone")
	}
	if next.Kind == Illegal || (next.Kind == Op && (next.Text == "(" || next.Text == ")")) {
		return Node{}, p.unexpectedErr(next)
	}
	if next.Kind == Word && isUnitKey(next.Key) {
		p.next()
		return Node{
			Op:   "to",
			Pos:  toTok.Pos,
			Args: []Node{{Op: "unit", Pos: next.Pos, Text: next.Key}},
		}, nil
	}
	if next.Kind == Word && toKeys[next.Key] {
		p.next()
		return Node{
			Op:   "to",
			Pos:  toTok.Pos,
			Args: []Node{{Op: "key", Pos: next.Pos, Text: next.Key}},
		}, nil
	}
	if p.canStartZone() {
		z, err := p.parseZone()
		if err != nil {
			return Node{}, err
		}
		return Node{Op: "to", Pos: toTok.Pos, Args: []Node{z}}, nil
	}
	return Node{}, p.unexpectedErr(next, "a unit", "a timezone")
}

func (p *parser) parseAtZoneStage() (Node, error) {
	atTok := p.next()
	if p.cur().Kind == EOF {
		return Node{}, p.unexpectedErr(p.cur(), "a unix timestamp", "a timezone")
	}
	z, err := p.parseZone()
	if err != nil {
		return Node{}, err
	}
	return Node{Op: "interpret", Pos: atTok.Pos, Args: []Node{z}}, nil
}

func (p *parser) parseInZoneStage() (Node, error) {
	inTok := p.next()
	if p.cur().Kind == EOF {
		return Node{}, p.unexpectedErr(p.cur(), "a duration", "a period", "a timezone")
	}
	z, err := p.parseZone()
	if err != nil {
		return Node{}, err
	}
	return Node{Op: "inzone", Pos: inTok.Pos, Args: []Node{z}}, nil
}

func (p *parser) startsInZone() bool {
	tok1 := p.peek(1)
	if tok1.Kind == String || tok1.Kind == EOF {
		return true
	}
	if tok1.Kind == Word && (tok1.Key == "utc" || tok1.Key == "") {
		return true
	}
	return p.startsZoneOffset(1)
}

func (p *parser) parseExpr(minBP int) (Node, error) {
	lhs, err := p.parsePostfix(false)
	if err != nil {
		return Node{}, err
	}
	for {
		tok := p.cur()
		if tok.Kind != Op {
			break
		}
		var bp int
		switch tok.Text {
		case "..":
			if lhs.Op == "span" {
				return lhs, nil
			}
			bp = bpSpan
		case "+", "-":
			bp = bpAdd
		case "*":
			bp = bpMul
		}
		if bp == 0 || bp < minBP {
			break
		}
		opTok := p.next()
		var rhs Node
		if opTok.Text == "*" {
			rhs, err = p.parsePostfix(true)
		} else {
			rhs, err = p.parseExpr(bp + 1)
		}
		if err != nil {
			return Node{}, err
		}
		if opTok.Text == ".." {
			lhs = Node{Op: "span", Pos: lhs.Pos, Args: []Node{lhs, rhs}}
		} else {
			lhs = Node{Op: opTok.Text, Pos: lhs.Pos, Args: []Node{lhs, rhs}}
		}
	}
	return lhs, nil
}

func (p *parser) parsePostfix(afterMul bool) (Node, error) {
	n, err := p.parseAtomWithOf(afterMul)
	if err != nil {
		return Node{}, err
	}
	for {
		if canTakeClock(n.Op) && p.startsClock() {
			c, err := p.parseClock()
			if err != nil {
				return Node{}, err
			}
			n = Node{Op: "at", Pos: n.Pos, Args: []Node{n, c}}
			continue
		}
		if p.cur().Kind == Op && p.cur().Text == "@" {
			p.next()
			z, err := p.parseZone()
			if err != nil {
				return Node{}, err
			}
			n = Node{Op: "interpret", Pos: n.Pos, Args: []Node{n, z}}
			continue
		}
		if p.cur().Kind == Word && p.cur().Key == "in" {
			p.next()
			z, err := p.parseZone()
			if err != nil {
				return Node{}, err
			}
			n = Node{Op: "inzone", Pos: n.Pos, Args: []Node{n, z}}
			continue
		}
		break
	}
	return n, nil
}

func (p *parser) parseAtomWithOf(afterMul bool) (Node, error) {
	n, err := p.parsePrimary(afterMul)
	if err != nil {
		return Node{}, err
	}
	if canTakeOf(n.Op) && p.cur().Kind == Word && p.cur().Key == "of" {
		p.next()
		rhs, err := p.parseAtomWithOf(false)
		if err != nil {
			return Node{}, err
		}
		n = Node{Op: "of", Pos: n.Pos, Args: []Node{n, rhs}}
	}
	return n, nil
}

func (p *parser) parsePrimary(afterMul bool) (Node, error) {
	tok := p.cur()
	switch tok.Kind {
	case Date:
		p.next()
		return Node{Op: "date", Pos: tok.Pos, Text: tok.Text}, nil
	case DateTime:
		p.next()
		return Node{Op: "datetime", Pos: tok.Pos, Text: tok.Text}, nil
	case SlashDate:
		p.next()
		return Node{Op: "slash", Pos: tok.Pos, Text: tok.Text}, nil
	case Clock:
		return p.parseClock()
	case Op:
		if tok.Text == "(" {
			p.next()
			if p.cur().Kind == Op && p.cur().Text == ")" {
				return Node{}, &Error{
					Pos:      p.cur().Pos,
					Msg:      "empty parentheses",
					Expected: []string{"a value"},
				}
			}
			inner, err := p.parseExpr(bpLowest)
			if err != nil {
				return Node{}, err
			}
			if p.cur().Kind != Op || p.cur().Text != ")" {
				return Node{}, p.unexpectedErr(p.cur(), ")")
			}
			p.next()
			return inner, nil
		}
		if tok.Text == "@" && p.peek(1).Kind == Number {
			atTok := p.next()
			numTok := p.next()
			if !allDigits(numTok.Text) {
				return Node{}, &Error{Pos: numTok.Pos, Msg: fmt.Sprintf("invalid unix timestamp %q", numTok.Text)}
			}
			return Node{Op: "unix", Pos: atTok.Pos, Text: numTok.Text}, nil
		}
		return Node{}, p.unexpectedErr(tok, "a value")
	case Quantity:
		if tok.Key == "am" || tok.Key == "pm" {
			return p.parseClock()
		}
		if isDurationUnitKey(tok.Key) {
			return p.parseDurWithAgo()
		}
		if tok.Key == "ordinal" {
			if p.startsDayFirstMonth() {
				return p.parseDayFirstMonth(true)
			}
			return p.parseOrdinalQtyWeek()
		}
		return Node{}, p.unexpectedErr(tok)
	case Number:
		if p.startsDayFirstMonth() {
			return p.parseDayFirstMonth(true)
		}
		if p.peek(1).Kind == Word && isUnitKey(p.peek(1).Key) {
			return p.parseDurWithAgo()
		}
		if p.peek(1).Kind == Word && (p.peek(1).Key == "am" || p.peek(1).Key == "pm") {
			return p.parseClock()
		}
		p.next()
		if !afterMul && isYearToken(tok) {
			return Node{Op: "year", Pos: tok.Pos, Text: tok.Text}, nil
		}
		return Node{Op: "num", Pos: tok.Pos, Text: tok.Text}, nil
	case Word:
		if fn, ok := prefix[tok.Key]; ok {
			return fn(p)
		}
		return Node{}, p.unexpectedErr(tok, "a value")
	default:
		return Node{}, p.unexpectedErr(tok, "a value")
	}
}

func (p *parser) parseSimpleValue() (Node, error) {
	tok := p.next()
	return Node{Op: tok.Key, Pos: tok.Pos}, nil
}

func (p *parser) parseDurWithAgo() (Node, error) {
	d, err := p.parseDur()
	if err != nil {
		return Node{}, err
	}
	if p.cur().Kind == Word && p.cur().Key == "ago" {
		p.next()
		return Node{Op: "ago", Pos: d.Pos, Args: []Node{d}}, nil
	}
	return d, nil
}

func (p *parser) parseDur() (Node, error) {
	if !p.startsDur() {
		return Node{}, p.unexpectedErr(p.cur(), "a duration")
	}
	pos := p.cur().Pos
	var parts []Node
	for {
		tok := p.cur()
		if tok.Kind == Quantity && isDurationUnitKey(tok.Key) {
			p.next()
			parts = append(parts, Node{Op: tok.Key, Pos: tok.Pos, Text: tok.Num})
			continue
		}
		if tok.Kind == Number && p.peek(1).Kind == Word && isUnitKey(p.peek(1).Key) {
			numTok := p.next()
			unitTok := p.next()
			parts = append(parts, Node{Op: unitTok.Key, Pos: numTok.Pos, Text: numTok.Text})
			continue
		}
		break
	}
	return Node{Op: "dur", Pos: pos, Args: parts}, nil
}

func (p *parser) parseAgoPrefix() (Node, error) {
	agoTok := p.next()
	d, err := p.parseDur()
	if err != nil {
		return Node{}, err
	}
	return Node{Op: "ago", Pos: agoTok.Pos, Args: []Node{d}}, nil
}

func (p *parser) parseInPrefix() (Node, error) {
	inTok := p.next()
	next := p.cur()
	if p.startsDur() {
		d, err := p.parseDur()
		if err != nil {
			return Node{}, err
		}
		return Node{Op: "in", Pos: inTok.Pos, Args: []Node{d}}, nil
	}
	if next.Kind == Quantity && next.Key == "" {
		return Node{}, p.unexpectedErr(next)
	}
	if next.Kind == Number {
		if p.startsDayFirstMonth() {
			return p.parseDayFirstMonth(true)
		}
		if isYearToken(next) {
			p.next()
			return Node{Op: "year", Pos: next.Pos, Text: next.Text}, nil
		}
		return Node{}, p.unexpectedErr(p.peek(1), "a unit")
	}
	if next.Kind == Word {
		switch {
		case isMonthKey(next.Key):
			return p.parseMonthPrefix()
		case isPeriodKey(next.Key):
			return p.parsePeriod()
		case isRelKey(next.Key):
			return p.parseRelPrefix()
		case isUnitKey(next.Key) && isRelKey(p.peek(1).Key):
			return p.parseUnitRel()
		case isWeekdayKey(next.Key) && isRelKey(p.peek(1).Key):
			return p.parseWeekday()
		}
	}
	return Node{}, p.unexpectedErr(next, "a duration", "a period")
}

func (p *parser) parseRelPrefix() (Node, error) {
	relTok := p.next()
	next := p.cur()
	if next.Kind == Word {
		if isUnitKey(next.Key) {
			uTok := p.next()
			rel := Node{
				Op:   "rel",
				Pos:  relTok.Pos,
				Text: relTok.Key,
				Args: []Node{{Op: "unit", Pos: uTok.Pos, Text: uTok.Key}},
			}
			if uTok.Key == "businessday" {
				return p.finishBusinessRel(rel)
			}
			return rel, nil
		}
		if isWeekdayKey(next.Key) {
			wdTok := p.next()
			return Node{
				Op:   "rel",
				Pos:  relTok.Pos,
				Text: relTok.Key,
				Args: []Node{{Op: "weekday", Pos: wdTok.Pos, Text: wdTok.Key}},
			}, nil
		}
		if isMonthKey(next.Key) {
			m, err := p.parseMonthFirst(false)
			if err != nil {
				return Node{}, err
			}
			return Node{Op: "rel", Pos: relTok.Pos, Text: relTok.Key, Args: []Node{m}}, nil
		}
	}
	if p.startsDayFirstMonth() {
		m, err := p.parseDayFirstMonth(false)
		if err != nil {
			return Node{}, err
		}
		return Node{Op: "rel", Pos: relTok.Pos, Text: relTok.Key, Args: []Node{m}}, nil
	}
	return Node{}, p.unexpectedErr(next, "a unit", "a weekday", "a month")
}

func (p *parser) parseUnitRel() (Node, error) {
	uTok := p.next()
	if p.cur().Kind == Word && isRelKey(p.cur().Key) {
		relTok := p.next()
		rel := Node{
			Op:   "rel",
			Pos:  uTok.Pos,
			Text: relTok.Key,
			Args: []Node{{Op: "unit", Pos: uTok.Pos, Text: uTok.Key}},
		}
		if uTok.Key == "businessday" {
			return p.finishBusinessRel(rel)
		}
		return rel, nil
	}
	return Node{}, p.unexpectedErr(p.cur(), "last", "next", "this")
}

func (p *parser) finishBusinessRel(rel Node) (Node, error) {
	tok := p.cur()
	if tok.Kind != Word || (tok.Key != "onorafter" && tok.Key != "onorbefore") {
		return rel, nil
	}
	if tok.Key == "onorafter" && rel.Text != "next" {
		return Node{}, &Error{Pos: tok.Pos, Msg: "\"on or after\" must be paired with \"next business day\""}
	}
	if tok.Key == "onorbefore" && rel.Text != "last" {
		return Node{}, &Error{Pos: tok.Pos, Msg: "\"on or before\" must be paired with \"last business day\""}
	}
	p.next()
	args := []Node{rel}
	if p.startsOrSameAnchor() {
		anchor, err := p.parseAtomWithOf(false)
		if err != nil {
			return Node{}, err
		}
		args = append(args, anchor)
	}
	return Node{Op: "orsame", Pos: rel.Pos, Args: args}, nil
}

func (p *parser) startsOrSameAnchor() bool {
	tok := p.cur()
	if tok.Kind == EOF || tok.Kind == Illegal {
		return false
	}
	if tok.Kind == Op {
		switch tok.Text {
		case ",", "|", "..", "+", "-", "*", ")":
			return false
		case "@":
			return p.peek(1).Kind == Number
		}
	}
	if tok.Kind == Word {
		switch tok.Key {
		case "inclusive", "to", "and":
			return false
		case "in":
			return !p.startsInZone()
		}
	}
	if p.startsClock() {
		return false
	}
	return true
}

func (p *parser) parseWeekPrefix() (Node, error) {
	wTok := p.next()
	if p.cur().Kind == Number {
		nTok := p.cur()
		if !allDigits(nTok.Text) {
			return Node{}, &Error{Pos: nTok.Pos, Msg: fmt.Sprintf("invalid week number %q", nTok.Text)}
		}
		p.next()
		return Node{Op: "week", Pos: wTok.Pos, Text: nTok.Text}, nil
	}
	if p.cur().Kind == Word && isRelKey(p.cur().Key) {
		relTok := p.next()
		return Node{
			Op:   "rel",
			Pos:  wTok.Pos,
			Text: relTok.Key,
			Args: []Node{{Op: "unit", Pos: wTok.Pos, Text: "week"}},
		}, nil
	}
	return Node{}, p.unexpectedErr(p.cur(), "a week number", "last", "next", "this")
}

func (p *parser) parseOrdinalQtyWeek() (Node, error) {
	ordTok := p.next()
	if !allDigits(ordTok.Num) {
		return Node{}, &Error{Pos: ordTok.Pos, Msg: fmt.Sprintf("invalid ordinal %q", ordTok.Text)}
	}
	if p.cur().Kind == Word && p.cur().Key == "week" {
		p.next()
		return Node{Op: "week", Pos: ordTok.Pos, Text: ordTok.Num}, nil
	}
	return Node{}, p.unexpectedErr(p.cur(), "week", "a month")
}

func (p *parser) parseOrdWordWeek() (Node, error) {
	ordTok := p.next()
	if p.cur().Kind == Word && p.cur().Key == "week" {
		p.next()
		return Node{Op: "week", Pos: ordTok.Pos, Text: ordTok.Key[:1]}, nil
	}
	return Node{}, p.unexpectedErr(p.cur(), "week")
}

func (p *parser) parseWeekday() (Node, error) {
	wdTok := p.next()
	wd := Node{Op: "weekday", Pos: wdTok.Pos, Text: wdTok.Key}
	if p.cur().Kind == Word {
		if isRelKey(p.cur().Key) {
			relTok := p.next()
			return Node{Op: "rel", Pos: wd.Pos, Text: relTok.Key, Args: []Node{wd}}, nil
		}
		if p.cur().Key == "closest" {
			p.next()
			return p.finishClosest(wd.Pos, wd)
		}
	}
	return wd, nil
}

func (p *parser) parseMonthPrefix() (Node, error) {
	return p.parseMonthFirst(true)
}

func (p *parser) parseMonthFirst(allowPostfixRel bool) (Node, error) {
	mTok := p.next()
	var args []Node

	if dayNode, ok := p.tryConsumeMonthDay(); ok {
		args = append(args, dayNode)
	}
	if yNode, ok := p.tryConsumeOptionalYear(false); ok {
		args = append(args, yNode)
	}

	m := Node{Op: "month", Pos: mTok.Pos, Text: mTok.Key, Args: args}
	if allowPostfixRel && p.cur().Kind == Word && isRelKey(p.cur().Key) {
		relTok := p.next()
		return Node{Op: "rel", Pos: m.Pos, Text: relTok.Key, Args: []Node{m}}, nil
	}
	return m, nil
}

func (p *parser) parseDayFirstMonth(allowPostfixRel bool) (Node, error) {
	dTok := p.next()
	dayText := dTok.Text
	if dTok.Kind == Quantity {
		dayText = dTok.Num
	}
	if p.cur().Kind == Word && p.cur().Key == "of" {
		p.next()
	}
	mTok := p.next()
	args := []Node{{Op: "num", Pos: dTok.Pos, Text: dayText}}
	if yNode, ok := p.tryConsumeOptionalYear(true); ok {
		args = append(args, yNode)
	}
	m := Node{Op: "month", Pos: dTok.Pos, Text: mTok.Key, Args: args}
	if allowPostfixRel && p.cur().Kind == Word && isRelKey(p.cur().Key) {
		relTok := p.next()
		return Node{Op: "rel", Pos: m.Pos, Text: relTok.Key, Args: []Node{m}}, nil
	}
	return m, nil
}

func (p *parser) tryConsumeMonthDay() (Node, bool) {
	tok := p.cur()
	if tok.Kind == Number && isDayDigits(tok.Text) {
		next := p.peek(1)
		if next.Kind == Word && (next.Key == "am" || next.Key == "pm" || isUnitKey(next.Key)) {
			return Node{}, false
		}
		p.next()
		return Node{Op: "num", Pos: tok.Pos, Text: tok.Text}, true
	}
	if tok.Kind == Quantity && tok.Key == "ordinal" && isDayDigits(tok.Num) {
		p.next()
		return Node{Op: "num", Pos: tok.Pos, Text: tok.Num}, true
	}
	return Node{}, false
}

func (p *parser) tryConsumeOptionalYear(allowOf bool) (Node, bool) {
	tok := p.cur()
	if isYearToken(tok) && !(p.peek(1).Kind == Word && isUnitKey(p.peek(1).Key)) {
		p.next()
		return Node{Op: "year", Pos: tok.Pos, Text: tok.Text}, true
	}
	if allowOf && tok.Kind == Word && tok.Key == "of" {
		yTok := p.peek(1)
		if isYearToken(yTok) && !(p.peek(2).Kind == Word && isUnitKey(p.peek(2).Key)) {
			p.next() // consume "of"
			p.next() // consume year
			return Node{Op: "year", Pos: yTok.Pos, Text: yTok.Text}, true
		}
	}
	return Node{}, false
}

func (p *parser) parsePeriod() (Node, error) {
	pTok := p.next()
	var args []Node
	if yNode, ok := p.tryConsumeOptionalYear(false); ok {
		args = append(args, yNode)
	}
	return Node{Op: "period", Pos: pTok.Pos, Text: pTok.Key, Args: args}, nil
}

func (p *parser) parseBound() (Node, error) {
	bTok := p.next()
	if p.cur().Kind != Word || p.cur().Key != "of" {
		return Node{}, p.unexpectedErr(p.cur(), "of")
	}
	p.next() // consume "of"

	next := p.cur()
	if next.Kind == EOF {
		return Node{}, p.unexpectedErr(next, "a unit", "a value")
	}
	if next.Kind == Word && isUnitKey(next.Key) {
		if (next.Key == "week" && p.peek(1).Kind == Number) || isRelKey(p.peek(1).Key) {
			target, err := p.parseAtomWithOf(false)
			if err != nil {
				return Node{}, err
			}
			return Node{Op: "bound", Pos: bTok.Pos, Text: bTok.Key, Args: []Node{target}}, nil
		}
		p.next()
		return Node{
			Op:   "bound",
			Pos:  bTok.Pos,
			Text: bTok.Key,
			Args: []Node{{Op: "unit", Pos: next.Pos, Text: next.Key}},
		}, nil
	}
	target, err := p.parseAtomWithOf(false)
	if err != nil {
		return Node{}, err
	}
	return Node{Op: "bound", Pos: bTok.Pos, Text: bTok.Key, Args: []Node{target}}, nil
}

func (p *parser) parseClosestPrefix() (Node, error) {
	cTok := p.next()
	wdTok := p.cur()
	if wdTok.Kind != Word || !isWeekdayKey(wdTok.Key) {
		return Node{}, p.unexpectedErr(wdTok, "a weekday")
	}
	p.next()
	wd := Node{Op: "weekday", Pos: wdTok.Pos, Text: wdTok.Key}
	return p.finishClosest(cTok.Pos, wd)
}

func (p *parser) finishClosest(pos int, wd Node) (Node, error) {
	if p.cur().Kind == Word && p.cur().Key == "to" {
		p.next()
		target, err := p.parsePostfix(false)
		if err != nil {
			return Node{}, err
		}
		return Node{Op: "closest", Pos: pos, Args: []Node{wd, target}}, nil
	}
	return Node{Op: "closest", Pos: pos, Args: []Node{wd}}, nil
}

func (p *parser) parseDuring() (Node, error) {
	dTok := p.next()
	d, err := p.parseDur()
	if err != nil {
		return Node{}, err
	}
	return Node{Op: "during", Pos: dTok.Pos, Args: []Node{d}}, nil
}

func (p *parser) parseHolidays() (Node, error) {
	hTok := p.next()
	n := Node{Op: "holidays", Pos: hTok.Pos}
	if tok := p.cur(); tok.Kind == Word && tok.Key == "" && len([]rune(tok.Text)) == 2 {
		p.next()
		n.Text = strings.ToUpper(tok.Text)
	}
	if tok := p.cur(); isYearToken(tok) {
		p.next()
		n.Args = []Node{{Op: "year", Pos: tok.Pos, Text: tok.Text}}
	}
	return n, nil
}

func (p *parser) parseStarting() (Node, error) {
	sTok := p.next()
	val, err := p.parseExpr(bpAdd)
	if err != nil {
		return Node{}, err
	}
	return Node{Op: "starting", Pos: sTok.Pos, Args: []Node{val}}, nil
}

func (p *parser) parseFromSpan() (Node, error) {
	fTok := p.next()
	fromVal, err := p.parseExpr(bpAdd)
	if err != nil {
		return Node{}, err
	}
	if p.cur().Kind != Word || p.cur().Key != "to" {
		return Node{}, p.unexpectedErr(p.cur(), "to")
	}
	p.next()
	toVal, err := p.parseExpr(bpAdd)
	if err != nil {
		return Node{}, err
	}
	return Node{Op: "span", Pos: fTok.Pos, Args: []Node{fromVal, toVal}}, nil
}

func (p *parser) parseBetweenSpan() (Node, error) {
	bTok := p.next()
	fromVal, err := p.parseExpr(bpAdd)
	if err != nil {
		return Node{}, err
	}
	if p.cur().Kind != Word || p.cur().Key != "and" {
		return Node{}, p.unexpectedErr(p.cur(), "and")
	}
	p.next()
	toVal, err := p.parseExpr(bpAdd)
	if err != nil {
		return Node{}, err
	}
	return Node{Op: "span", Pos: bTok.Pos, Args: []Node{fromVal, toVal}}, nil
}

func (p *parser) parseCount() (Node, error) {
	hmTok := p.next()
	wTok := p.cur()
	var what Node
	switch {
	case wTok.Kind == Word && isUnitKey(wTok.Key):
		p.next()
		what = Node{Op: "unit", Pos: wTok.Pos, Text: wTok.Key}
	case wTok.Kind == Word && isWeekdayKey(wTok.Key):
		p.next()
		what = Node{Op: "weekday", Pos: wTok.Pos, Text: wTok.Key}
	default:
		return Node{}, p.unexpectedErr(wTok, "a unit", "a weekday")
	}

	var sep string
	switch {
	case p.cur().Kind == Word && p.cur().Key == "between":
		p.next()
		sep = "and"
	case p.cur().Kind == Word && p.cur().Key == "from":
		p.next()
		sep = "to"
	default:
		return Node{}, p.unexpectedErr(p.cur(), "between", "from")
	}

	fromVal, err := p.parseExpr(bpAdd)
	if err != nil {
		return Node{}, err
	}
	if p.cur().Kind != Word || p.cur().Key != sep {
		return Node{}, p.unexpectedErr(p.cur(), sep)
	}
	p.next()
	toVal, err := p.parseExpr(bpAdd)
	if err != nil {
		return Node{}, err
	}
	return Node{Op: "count", Pos: hmTok.Pos, Args: []Node{what, fromVal, toVal}}, nil
}

func (p *parser) parseClock() (Node, error) {
	tok := p.cur()
	switch tok.Kind {
	case Clock:
		p.next()
		text := tok.Text
		if p.cur().Kind == Word && (p.cur().Key == "am" || p.cur().Key == "pm") {
			ampm := p.next().Key
			converted, err := formatClock24(text, ampm, tok.Pos)
			if err != nil {
				return Node{}, err
			}
			text = converted
		} else if colon := strings.IndexByte(text, ':'); colon == 1 {
			text = "0" + text
		}
		return Node{Op: "clock", Pos: tok.Pos, Text: text}, nil
	case Quantity:
		if tok.Key == "am" || tok.Key == "pm" {
			p.next()
			text, err := formatClock24(tok.Num, tok.Key, tok.Pos)
			if err != nil {
				return Node{}, err
			}
			return Node{Op: "clock", Pos: tok.Pos, Text: text}, nil
		}
	case Number:
		if p.peek(1).Kind == Word && (p.peek(1).Key == "am" || p.peek(1).Key == "pm") {
			numTok := p.next()
			ampmTok := p.next()
			text, err := formatClock24(numTok.Text, ampmTok.Key, numTok.Pos)
			if err != nil {
				return Node{}, err
			}
			return Node{Op: "clock", Pos: numTok.Pos, Text: text}, nil
		}
	}
	return Node{}, p.unexpectedErr(tok, "a time")
}

func (p *parser) parseZone() (Node, error) {
	tok := p.cur()
	switch {
	case tok.Kind == String:
		p.next()
		if tok.Text == "" {
			return Node{}, &Error{Pos: tok.Pos, Msg: "empty timezone name"}
		}
		return Node{Op: "zone", Pos: tok.Pos, Text: tok.Text}, nil
	case p.startsPrefixedZoneOffset():
		utcTok := p.next()
		signTok := p.next()
		valTok := p.next()
		return Node{Op: "zone", Pos: utcTok.Pos, Text: "UTC" + signTok.Text + valTok.Text}, nil
	case p.startsZoneOffset(0):
		signTok := p.next()
		valTok := p.next()
		return Node{Op: "zone", Pos: signTok.Pos, Text: signTok.Text + valTok.Text}, nil
	case tok.Kind == Word && tok.Key == "utc":
		p.next()
		return Node{Op: "zone", Pos: tok.Pos, Text: "UTC"}, nil
	case tok.Kind == Word && tok.Key == "":
		pos := tok.Pos
		var parts []string
		for p.cur().Kind == Word && p.cur().Key == "" {
			parts = append(parts, p.next().Text)
		}
		return Node{Op: "zone", Pos: pos, Text: strings.Join(parts, " ")}, nil
	default:
		return Node{}, p.unexpectedErr(tok, "a timezone")
	}
}

func (p *parser) startsDur() bool {
	tok := p.cur()
	if tok.Kind == Quantity && isDurationUnitKey(tok.Key) {
		return true
	}
	return tok.Kind == Number && p.peek(1).Kind == Word && isUnitKey(p.peek(1).Key)
}

func (p *parser) startsDayFirstMonth() bool {
	return p.startsDayFirstMonthAt(0)
}

func (p *parser) startsDayFirstMonthAt(offset int) bool {
	tok := p.peek(offset)
	var ok bool
	switch tok.Kind {
	case Number:
		ok = isDayDigits(tok.Text)
	case Quantity:
		ok = tok.Key == "ordinal" && isDayDigits(tok.Num)
	}
	if !ok {
		return false
	}
	next := p.peek(offset + 1)
	if next.Kind == Word && isMonthKey(next.Key) {
		return true
	}
	if next.Kind == Word && next.Key == "of" && p.peek(offset+2).Kind == Word && isMonthKey(p.peek(offset+2).Key) {
		return true
	}
	return false
}

func (p *parser) startsClock() bool {
	tok := p.cur()
	if tok.Kind == Clock {
		return true
	}
	if tok.Kind == Quantity && (tok.Key == "am" || tok.Key == "pm") {
		return true
	}
	return tok.Kind == Number && p.peek(1).Kind == Word && (p.peek(1).Key == "am" || p.peek(1).Key == "pm")
}

func (p *parser) unexpectedErr(tok Token, expected ...string) error {
	if tok.Kind == Illegal {
		return &Error{Pos: tok.Pos, Msg: fmt.Sprintf("unexpected character %q", tok.Text)}
	}
	if tok.Kind == Quantity && tok.Key == "" {
		return &Error{Pos: tok.Pos, Msg: fmt.Sprintf("unknown unit %q", tok.Unit)}
	}
	if tok.Kind == EOF {
		msg := "unexpected end of input"
		if len(expected) == 1 && !strings.HasPrefix(expected[0], "a ") && !strings.HasPrefix(expected[0], "an ") {
			msg = fmt.Sprintf("unexpected end of input, expected %q", expected[0])
		}
		return &Error{Pos: tok.Pos, Msg: msg, Expected: expected}
	}
	return &Error{Pos: tok.Pos, Msg: fmt.Sprintf("unexpected %q", tok.Text), Expected: expected}
}

func formatClock24(raw, ampm string, pos int) (string, error) {
	if strings.ContainsRune(raw, '.') {
		return "", &Error{Pos: pos, Msg: fmt.Sprintf("invalid clock %q", raw+ampm)}
	}
	if colon := strings.IndexByte(raw, ':'); colon >= 0 {
		h, err := strconv.Atoi(raw[:colon])
		if err != nil || h < 1 || h > 12 {
			return "", &Error{Pos: pos, Msg: fmt.Sprintf("invalid 12-hour clock %q", raw+ampm)}
		}
		if ampm == "am" && h == 12 {
			h = 0
		} else if ampm == "pm" && h < 12 {
			h += 12
		}
		return fmt.Sprintf("%02d%s", h, raw[colon:]), nil
	}
	h, err := strconv.Atoi(raw)
	if err != nil || h < 1 || h > 12 {
		return "", &Error{Pos: pos, Msg: fmt.Sprintf("invalid 12-hour clock %q", raw+ampm)}
	}
	if ampm == "am" && h == 12 {
		h = 0
	} else if ampm == "pm" && h < 12 {
		h += 12
	}
	return fmt.Sprintf("%02d:00", h), nil
}

func (p *parser) canStartZone() bool {
	tok := p.cur()
	if tok.Kind == String {
		return true
	}
	if tok.Kind == Word && (tok.Key == "utc" || tok.Key == "") {
		return true
	}
	return p.startsZoneOffset(0)
}

func (p *parser) startsZoneOffset(offset int) bool {
	signTok := p.peek(offset)
	if signTok.Kind != Op || (signTok.Text != "+" && signTok.Text != "-") {
		return false
	}
	valTok := p.peek(offset + 1)
	if valTok.Pos != signTok.Pos+len(signTok.Text) {
		return false
	}
	switch valTok.Kind {
	case Clock:
		return true
	case Number:
		next := p.peek(offset + 2)
		if next.Kind == Word && (isUnitKey(next.Key) || next.Key == "am" || next.Key == "pm") {
			return false
		}
		return true
	default:
		return false
	}
}

func (p *parser) startsPrefixedZoneOffset() bool {
	tok := p.cur()
	if tok.Kind != Word {
		return false
	}
	low := fold(tok.Text)
	if low != "utc" && low != "gmt" {
		return false
	}
	signTok := p.peek(1)
	if signTok.Kind != Op || signTok.Pos != tok.Pos+len(tok.Text) {
		return false
	}
	return p.startsZoneOffset(1)
}

func canTakeOf(op string) bool {
	switch op {
	case "week", "weekday", "month", "period":
		return true
	default:
		return false
	}
}

func canTakeClock(op string) bool {
	switch op {
	case "date", "slash", "today", "yesterday", "tomorrow",
		"rel", "orsame", "weekday", "month", "period", "week", "of", "closest", "bound",
		"+", "-":
		return true
	default:
		return false
	}
}

func isRelKey(key string) bool {
	return key == "last" || key == "next" || key == "this"
}

func isWeekdayKey(key string) bool {
	switch key {
	case "monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday":
		return true
	default:
		return false
	}
}

func isMonthKey(key string) bool {
	switch key {
	case "january", "february", "march", "april", "may", "june",
		"july", "august", "september", "october", "november", "december":
		return true
	default:
		return false
	}
}

func isPeriodKey(key string) bool {
	switch key {
	case "q1", "q2", "q3", "q4", "h1", "h2":
		return true
	default:
		return false
	}
}

func isYearToken(tok Token) bool {
	return tok.Kind == Number && len(tok.Text) == 4 && allDigits(tok.Text)
}

func isDayDigits(s string) bool {
	return (len(s) == 1 || len(s) == 2) && allDigits(s)
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func (p *parser) startsTakeStage() bool {
	next := p.peek(1)
	if next.Kind != Number || !allDigits(next.Text) {
		return false
	}
	if p.startsDayFirstMonthAt(1) {
		return false
	}
	after := p.peek(2)
	if after.Kind == Word && (after.Key == "am" || after.Key == "pm" || isUnitKey(after.Key)) {
		return false
	}
	return true
}

func (p *parser) parseTakeStage() (Node, error) {
	nextTok := p.next()
	numTok := p.next()
	return Node{Op: "take", Pos: nextTok.Pos, Text: numTok.Text}, nil
}

func (p *parser) parseCron() (Node, error) {
	cronTok := p.next()
	strTok := p.cur()
	if strTok.Kind != String {
		return Node{}, p.unexpectedErr(strTok, "a quoted 5-field cron string")
	}
	p.next()
	fields := strings.Join(strings.Fields(strTok.Text), " ")
	return Node{Op: "cron", Pos: cronTok.Pos, Text: fields}, nil
}

func (p *parser) parseEvery() (Node, error) {
	everyTok := p.next()
	var args []Node
	tok := p.cur()

	switch {
	case tok.Kind == Word && tok.Key == "day" && p.peek(1).Kind == Op && p.peek(1).Text == "-" && p.peek(2).Kind == Word && p.peek(2).Key == "year":
		dayTok := p.next()
		p.next() // '-'
		p.next() // 'year'
		ordNode, err := p.parseOrd()
		if err != nil {
			return Node{}, err
		}
		args = []Node{{Op: "key", Pos: dayTok.Pos, Text: "dayyear"}, ordNode}

	case tok.Kind == Number:
		if !allDigits(tok.Text) {
			return Node{}, p.everyErr(tok.Pos)
		}
		n, err := strconv.Atoi(tok.Text)
		if err != nil || n < 1 {
			return Node{}, &Error{Pos: tok.Pos, Msg: "interval must be at least 1"}
		}
		p.next()
		uTok := p.cur()
		if uTok.Kind != Word || !isEveryUnitKey(uTok.Key) {
			return Node{}, p.everyErr(uTok.Pos)
		}
		p.next()
		args = []Node{
			{Op: "num", Pos: tok.Pos, Text: tok.Text},
			{Op: "unit", Pos: uTok.Pos, Text: uTok.Key},
		}

	case p.startsOrd():
		ordNode, err := p.parseOrd()
		if err != nil {
			return Node{}, err
		}
		wdTok := p.cur()
		if wdTok.Kind != Word || !isWeekdayKey(wdTok.Key) {
			return Node{}, p.everyErr(wdTok.Pos)
		}
		p.next()
		args = []Node{ordNode, {Op: "weekday", Pos: wdTok.Pos, Text: wdTok.Key}}

	case tok.Kind == Word && isWeekdayKey(tok.Key):
		p.next()
		args = []Node{{Op: "weekday", Pos: tok.Pos, Text: tok.Key}}

	case tok.Kind == Word && tok.Key == "weekdaykw":
		p.next()
		args = []Node{{Op: "key", Pos: tok.Pos, Text: "weekdaykw"}}
		if p.startsOrd() {
			ordNode, err := p.parseOrd()
			if err != nil {
				return Node{}, err
			}
			args = append(args, ordNode)
		}

	case tok.Kind == Word && isEveryUnitKey(tok.Key):
		p.next()
		uNode := Node{Op: "unit", Pos: tok.Pos, Text: tok.Key}
		switch tok.Key {
		case "day", "month":
			if p.startsOrd() {
				ordNode, err := p.parseOrd()
				if err != nil {
					return Node{}, err
				}
				args = []Node{uNode, ordNode}
			} else {
				args = []Node{uNode}
			}
		case "year":
			switch {
			case p.cur().Kind == Word && isMonthKey(p.cur().Key):
				m, err := p.parseMonthFirst(false)
				if err != nil {
					return Node{}, err
				}
				if len(m.Args) != 1 || m.Args[0].Op != "num" {
					return Node{}, p.everyErr(m.Pos)
				}
				args = []Node{uNode, m}
			case p.startsDayFirstMonth():
				m, err := p.parseDayFirstMonth(false)
				if err != nil {
					return Node{}, err
				}
				if len(m.Args) != 1 || m.Args[0].Op != "num" {
					return Node{}, p.everyErr(m.Pos)
				}
				args = []Node{uNode, m}
			default:
				args = []Node{uNode}
			}
		default:
			args = []Node{uNode}
		}

	default:
		return Node{}, p.everyErr(tok.Pos)
	}

	if p.startsClock() {
		clk, err := p.parseClock()
		if err != nil {
			return Node{}, err
		}
		args = append(args, clk)
	}

	return Node{Op: "every", Pos: everyTok.Pos, Args: args}, nil
}

func (p *parser) everyErr(pos int) error {
	return &Error{
		Pos: pos,
		Msg: `invalid every schedule (expected e.g. "every monday", "every 2 days", "every month 15th", "every 2nd monday", "every weekday", "every year aug 20")`,
	}
}

func (p *parser) startsOrd() bool {
	tok := p.cur()
	if tok.Kind == Quantity && tok.Key == "ordinal" && allDigits(tok.Num) {
		return true
	}
	return tok.Kind == Word && (tok.Key == "1st" || tok.Key == "2nd" || tok.Key == "3rd" || tok.Key == "4th" || tok.Key == "5th")
}

func (p *parser) parseOrd() (Node, error) {
	tok := p.cur()
	var num string
	switch {
	case tok.Kind == Quantity && tok.Key == "ordinal" && allDigits(tok.Num):
		num = tok.Num
	case tok.Kind == Word && (tok.Key == "1st" || tok.Key == "2nd" || tok.Key == "3rd" || tok.Key == "4th" || tok.Key == "5th"):
		num = tok.Key[:1]
	default:
		return Node{}, p.everyErr(tok.Pos)
	}
	n, err := strconv.Atoi(num)
	if err != nil || n < 1 {
		return Node{}, &Error{Pos: tok.Pos, Msg: "ordinal must be at least 1"}
	}
	p.next()
	return Node{Op: "ord", Pos: tok.Pos, Text: num}, nil
}

func isEveryUnitKey(key string) bool {
	switch key {
	case "minute", "hour", "day", "week", "month", "year":
		return true
	default:
		return false
	}
}
