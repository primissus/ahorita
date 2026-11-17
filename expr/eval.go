package expr

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Step records one evaluated pipeline stage for Explain.
type Step struct {
	Text  string
	Value Value
}

// Eval lexes, normalizes, parses, and evaluates input against ctx.
func Eval(input string, ctx Context) (Value, error) {
	steps, _, err := evalPipeline(input, ctx)
	if err != nil {
		return Value{}, err
	}
	return steps[len(steps)-1].Value, nil
}

// Explain evaluates input and returns one Step per stage after canonical reordering.
func Explain(input string, ctx Context) ([]Step, error) {
	steps, _, err := evalPipeline(input, ctx)
	if err != nil {
		return nil, err
	}
	return steps, nil
}

type evaluator struct {
	ctx           Context
	loc           *time.Location
	cfgLang       Lang
	resLang       Lang
	priorStages   []string
	activeAliases map[string]bool
}

func evalPipeline(input string, ctx Context) ([]Step, Lang, error) {
	cfgLang := ctx.Cfg.Lang
	if cfgLang == "" {
		cfgLang = EN
	}
	loc := ctx.Loc
	if loc == nil {
		loc = ctx.Now.Location()
		if loc == nil {
			loc = time.UTC
		}
	}

	parsed, err := ParseWithAliases(input, cfgLang, ctx.Cfg.Aliases)
	if err != nil {
		return nil, "", err
	}

	if ctx.Cfg.Strict && len(parsed.Ties) > 0 {
		return nil, "", &Error{Pos: parsed.Ties[0], Msg: "ambiguous word"}
	}

	stages, err := orderStages(parsed.Stages, ctx.Cfg.Strict)
	if err != nil {
		return nil, "", err
	}

	resLang := parsed.Lang
	if resLang == "" {
		resLang = cfgLang
	}

	activeAliases := make(map[string]bool)
	if parsed.AliasName != "" {
		activeAliases[parsed.AliasName] = true
	}

	ev := &evaluator{
		ctx:           ctx,
		loc:           loc,
		cfgLang:       cfgLang,
		resLang:       resLang,
		activeAliases: activeAliases,
	}

	cur := Value{
		Kind:  KDateTime,
		T:     ctx.Now.In(loc),
		Local: true,
	}

	steps := make([]Step, 0, len(stages))
	for _, st := range stages {
		next, err := ev.node(st, cur)
		if err != nil {
			return nil, "", err
		}
		next.Lang = resLang
		stageText := st.String()
		ev.priorStages = append(ev.priorStages, stageText)
		steps = append(steps, Step{
			Text:  stageText,
			Value: next,
		})
		cur = next
	}

	return steps, resLang, nil
}

type stageClass int

const (
	classValue stageClass = iota
	classPeriod
	classOther
)

type rankedStage struct {
	node    Node
	class   stageClass
	rank    int
	origIdx int
}

func orderStages(stages []Node, strict bool) ([]Node, error) {
	ranked := make([]rankedStage, len(stages))
	valueCount := 0
	for i, st := range stages {
		cls, rank := classifyStage(st)
		if cls == classValue {
			valueCount++
			if valueCount > 1 {
				return nil, &Error{
					Pos: st.Pos,
					Msg: "multiple value stages in one pipeline",
				}
			}
		}
		ranked[i] = rankedStage{
			node:    st,
			class:   cls,
			rank:    rank,
			origIdx: i,
		}
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		a, b := ranked[i], ranked[j]
		if a.class != b.class {
			return a.class < b.class
		}
		if a.class == classPeriod && a.rank != b.rank {
			return a.rank > b.rank
		}
		return false
	})

	changed := false
	firstMovedPos := -1
	ordered := make([]Node, len(ranked))
	for i, r := range ranked {
		ordered[i] = r.node
		if r.origIdx != i && !changed {
			changed = true
			firstMovedPos = r.node.Pos
		}
	}

	if changed && strict {
		canonical := Expr{Stages: ordered}.String()
		return nil, &Error{
			Pos: firstMovedPos,
			Msg: fmt.Sprintf("stages are out of order; write: %s", canonical),
		}
	}

	return ordered, nil
}

func classifyStage(n Node) (stageClass, int) {
	switch n.Op {
	case "inclusive":
		if len(n.Args) == 1 {
			return classifyStage(n.Args[0])
		}
		return classOther, 0
	case "interpret", "inzone":
		if len(n.Args) >= 2 {
			return classifyStage(n.Args[0])
		}
		return classOther, 0
	case "at":
		if len(n.Args) >= 1 {
			return classifyStage(n.Args[0])
		}
		return classOther, 0
	case "+", "-", "*":
		cur := n
		for len(cur.Args) > 0 {
			switch cur.Op {
			case "+", "-", "*", "at":
				cur = cur.Args[0]
				continue
			case "interpret", "inzone":
				if len(cur.Args) >= 2 {
					cur = cur.Args[0]
					continue
				}
			}
			break
		}
		if isValueOp(cur.Op) || (cur.Op == "orsame" && len(cur.Args) >= 2) {
			return classValue, 0
		}
		return classOther, 0
	case "year":
		return classPeriod, 6
	case "period":
		switch n.Text {
		case "h1", "h2":
			return classPeriod, 5
		case "q1", "q2", "q3", "q4":
			return classPeriod, 4
		}
		return classOther, 0
	case "month":
		return classPeriod, 3
	case "week":
		return classPeriod, 2
	case "weekday":
		return classPeriod, 1
	case "of":
		if len(n.Args) >= 1 {
			cls, rank := classifyStage(n.Args[0])
			if cls == classPeriod {
				return cls, rank
			}
		}
		return classOther, 0
	case "orsame":
		if len(n.Args) >= 2 {
			return classValue, 0
		}
		if len(n.Args) == 1 {
			return classifyStage(n.Args[0])
		}
		return classOther, 0
	case "rel":
		if len(n.Args) == 0 {
			return classOther, 0
		}
		target := n.Args[0]
		switch target.Op {
		case "month":
			return classPeriod, 3
		case "weekday":
			return classPeriod, 1
		case "unit":
			switch target.Text {
			case "year":
				return classPeriod, 6
			case "half":
				return classPeriod, 5
			case "quarter":
				return classPeriod, 4
			case "month":
				return classPeriod, 3
			case "week", "workweek":
				return classPeriod, 2
			case "day", "businessday":
				return classPeriod, 1
			}
		}
		return classOther, 0
	default:
		if isValueOp(n.Op) {
			return classValue, 0
		}
		return classOther, 0
	}
}

func isValueOp(op string) bool {
	switch op {
	case "date", "datetime", "clock", "slash", "unix",
		"now", "today", "yesterday", "tomorrow",
		"dur", "num", "ago", "in", "span", "count", "during",
		"every", "cron", "alias", "if":
		return true
	default:
		return false
	}
}

func (ev *evaluator) today() (int, time.Month, int) {
	return ev.ctx.Now.In(ev.loc).Date()
}

// wall constructs a wall-clock time in loc and returns an error on a DST gap or overlap.
func wall(y int, m time.Month, d, h, mi, s int, loc *time.Location) (time.Time, error) {
	t := time.Date(y, m, d, h, mi, s, 0, loc)
	ty, tm, td := t.Date()
	th, tmi, ts := t.Clock()
	if ty != y || tm != m || td != d || th != h || tmi != mi || ts != s {
		return t, fmt.Errorf("time %04d-%02d-%02d %02d:%02d:%02d does not exist in %s", y, m, d, h, mi, s, loc)
	}
	_, off := t.Zone()
	for _, delta := range []time.Duration{-time.Hour, time.Hour} {
		alt := t.Add(delta)
		ay, am, ad := alt.Date()
		ah, ami, as := alt.Clock()
		_, aoff := alt.Zone()
		if ay == y && am == m && ad == d && ah == h && ami == mi && as == s && aoff != off {
			return t, fmt.Errorf("time %04d-%02d-%02d %02d:%02d:%02d is ambiguous in %s", y, m, d, h, mi, s, loc)
		}
	}
	return t, nil
}

func (ev *evaluator) wallAt(pos int, y int, m time.Month, d, h, mi, s int, loc *time.Location) (time.Time, error) {
	t, err := wall(y, m, d, h, mi, s, loc)
	if err != nil && ev.ctx.Cfg.Strict {
		return time.Time{}, &Error{Pos: pos, Msg: err.Error()}
	}
	return t, nil
}

func (ev *evaluator) node(n Node, in Value) (Value, error) {
	switch n.Op {
	case "date":
		y, m, d, err := parseISODate(n.Text, n.Pos)
		if err != nil {
			return Value{}, err
		}
		t, err := ev.wallAt(n.Pos, y, m, d, 0, 0, 0, ev.loc)
		if err != nil {
			return Value{}, err
		}
		return Value{Kind: KDate, T: t, Local: true}, nil

	case "datetime":
		return ev.evalDateTime(n, ev.loc)

	case "clock":
		h, mi, s, err := parseClockParts(n.Text, n.Pos)
		if err != nil {
			return Value{}, err
		}
		y, m, d := ev.today()
		t, err := ev.wallAt(n.Pos, y, m, d, h, mi, s, ev.loc)
		if err != nil {
			return Value{}, err
		}
		return Value{Kind: KTime, T: t, Local: true}, nil

	case "slash":
		y, m, d, err := parseSlashDate(n.Text, n.Pos, ev.cfgLang, ev.ctx.Cfg.Strict)
		if err != nil {
			return Value{}, err
		}
		t, err := ev.wallAt(n.Pos, y, m, d, 0, 0, 0, ev.loc)
		if err != nil {
			return Value{}, err
		}
		return Value{Kind: KDate, T: t, Local: true}, nil

	case "unix":
		val, err := strconv.ParseInt(n.Text, 10, 64)
		if err != nil {
			return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("invalid unix timestamp %q", n.Text)}
		}
		var t time.Time
		if len(n.Text) >= 12 {
			t = time.UnixMilli(val).In(ev.loc)
		} else {
			t = time.Unix(val, 0).In(ev.loc)
		}
		return Value{Kind: KDateTime, T: t, Local: false}, nil

	case "num":
		val, err := strconv.ParseFloat(n.Text, 64)
		if err != nil {
			return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("invalid number %q", n.Text)}
		}
		return Value{Kind: KNumber, N: val}, nil

	case "year":
		y, err := strconv.Atoi(n.Text)
		if err != nil || y < 1 {
			return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("invalid year %q", n.Text)}
		}
		start, err := ev.wallAt(n.Pos, y, time.January, 1, 0, 0, 0, ev.loc)
		if err != nil {
			return Value{}, err
		}
		end, err := ev.wallAt(n.Pos, y+1, time.January, 1, 0, 0, 0, ev.loc)
		if err != nil {
			return Value{}, err
		}
		return Value{Kind: KSpan, T: start, End: end, Days: true, Local: true}, nil

	case "now":
		return Value{Kind: KDateTime, T: ev.ctx.Now.In(ev.loc), Local: false}, nil

	case "today":
		y, m, d := ev.today()
		t, err := ev.wallAt(n.Pos, y, m, d, 0, 0, 0, ev.loc)
		if err != nil {
			return Value{}, err
		}
		return Value{Kind: KDate, T: t, Local: true, TodayNow: true}, nil

	case "alias":
		return ev.evalAlias(n, in)

	case "if":
		return ev.evalIf(n, in)

	case "yesterday", "tomorrow":
		delta := -1
		if n.Op == "tomorrow" {
			delta = 1
		}
		base := ev.ctx.Now.In(ev.loc).AddDate(0, 0, delta)
		y, m, d := base.Date()
		t, err := ev.wallAt(n.Pos, y, m, d, 0, 0, 0, ev.loc)
		if err != nil {
			return Value{}, err
		}
		return Value{Kind: KDate, T: t, Local: true}, nil

	case "at":
		return ev.evalAt(n, in, nil)

	case "dur":
		d, err := evalDur(n)
		if err != nil {
			return Value{}, err
		}
		return Value{Kind: KDur, D: d}, nil

	case "+", "-", "*":
		if len(n.Args) < 2 {
			return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("invalid %s expression", n.Op)}
		}
		lhs, err := ev.node(n.Args[0], in)
		if err != nil {
			return Value{}, err
		}
		rhs, err := ev.node(n.Args[1], in)
		if err != nil {
			return Value{}, err
		}
		switch n.Op {
		case "+":
			return evalAdd(n.Pos, lhs, rhs, ev.ctx)
		case "-":
			return evalSub(n.Pos, lhs, rhs, ev.ctx)
		case "*":
			return evalMul(n.Pos, lhs, rhs)
		default:
			return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("unsupported operator %q", n.Op)}
		}

	case "ago", "in":
		if len(n.Args) < 1 {
			return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("invalid %s expression", n.Op)}
		}
		d, err := evalDur(n.Args[0])
		if err != nil {
			return Value{}, err
		}
		if n.Op == "ago" {
			d = negateDuration(d)
		}
		t, err := addDurationCtx(n.Pos, ev.ctx.Now.In(ev.loc), d, ev.ctx)
		if err != nil {
			return Value{}, err
		}
		return Value{Kind: KDateTime, T: t, Local: false}, nil

	case "weekday":
		return ev.evalWeekday(n, in)

	case "rel":
		return ev.evalRel(n, in)

	case "orsame":
		return ev.evalOrSame(n, in)

	case "month":
		return ev.evalMonth(n, in)

	case "period":
		return ev.evalPeriod(n, in)

	case "closest":
		return ev.evalClosest(n, in)

	case "week":
		return ev.evalWeek(n, in)

	case "of":
		return ev.evalOf(n, in)

	case "bound":
		return ev.evalBound(n, in)

	case "span":
		return ev.evalSpan(n, in)

	case "during":
		return ev.evalDuring(n)

	case "holidays":
		return ev.evalHolidays(n, in)

	case "starting":
		return ev.evalStarting(n, in)

	case "every":
		return ev.evalEvery(n)

	case "cron":
		return ev.evalCron(n)

	case "take":
		return ev.evalTake(n, in)

	case "explain":
		return ev.evalExplain(n, in)

	case "count":
		return ev.evalCount(n, in, false)

	case "inclusive":
		if len(n.Args) < 1 {
			return Value{}, &Error{Pos: n.Pos, Msg: "invalid inclusive modifier"}
		}
		switch n.Args[0].Op {
		case "count":
			return ev.evalCount(n.Args[0], in, true)
		case "to":
			return ev.evalTo(n.Args[0], in, true)
		}
		return ev.node(n.Args[0], in)

	case "to":
		return ev.evalTo(n, in, false)

	case "zone":
		return ev.evalZoneStage(n, in)

	case "interpret":
		return ev.evalInterpret(n, in)

	case "inzone":
		return ev.evalInZone(n, in)

	default:
		return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("unsupported expression %q", n.Op)}
	}
}

func lookupAliasExpr(aliases map[string]string, foldedKey string) (string, bool) {
	if raw, ok := aliases[foldedKey]; ok {
		return raw, true
	}
	for k, v := range aliases {
		if fold(strings.Join(strings.Fields(k), " ")) == foldedKey {
			return v, true
		}
	}
	return "", false
}

func (ev *evaluator) evalAlias(n Node, in Value) (Value, error) {
	if ev.activeAliases[n.Text] {
		return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("circular alias reference %q", n.Text)}
	}
	raw, ok := lookupAliasExpr(ev.ctx.Cfg.Aliases, n.Text)
	if !ok {
		return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("unknown alias %q", n.Text)}
	}
	ev.activeAliases[n.Text] = true
	defer delete(ev.activeAliases, n.Text)

	parsed, err := ParseWithAliases(raw, ev.cfgLang, ev.ctx.Cfg.Aliases)
	if err != nil {
		return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("in alias %q: %s", n.Text, err.Error())}
	}
	if parsed.AliasName != "" {
		return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("in alias %q: nested alias assignment is not allowed", n.Text)}
	}
	if ev.ctx.Cfg.Strict && len(parsed.Ties) > 0 {
		return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("in alias %q: ambiguous word", n.Text)}
	}
	stages, err := orderStages(parsed.Stages, ev.ctx.Cfg.Strict)
	if err != nil {
		return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("in alias %q: %s", n.Text, err.Error())}
	}

	cur := in
	for _, st := range stages {
		next, err := ev.node(st, cur)
		if err != nil {
			return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("in alias %q: %s", n.Text, err.Error())}
		}
		cur = next
	}
	return cur, nil
}

func (ev *evaluator) evalIf(n Node, in Value) (Value, error) {
	if len(n.Args) < 3 {
		return Value{}, &Error{Pos: n.Pos, Msg: "invalid if expression"}
	}
	condOK, err := ev.evalCond(n.Args[0], in)
	if err != nil {
		return Value{}, err
	}
	// Evaluate both branches so --is-valid and diagnostics catch errors in untaken branches.
	thenVal, thenErr := ev.node(n.Args[1], in)
	elseVal, elseErr := ev.node(n.Args[2], in)
	if thenErr != nil {
		return Value{}, thenErr
	}
	if elseErr != nil {
		return Value{}, elseErr
	}
	if condOK {
		return thenVal, nil
	}
	return elseVal, nil
}

func (ev *evaluator) evalCond(n Node, in Value) (bool, error) {
	if len(n.Args) < 2 {
		return false, &Error{Pos: n.Pos, Msg: "invalid condition"}
	}
	lhs, err := ev.node(n.Args[0], in)
	if err != nil {
		return false, err
	}
	op := n.Text

	if n.Args[1].Op == "pred" {
		d, err := inputDate(n.Pos, lhs)
		if err != nil {
			return false, err
		}
		pred := n.Args[1].Text
		var match bool
		switch {
		case strings.HasPrefix(pred, "weekday:"):
			wd, err := parseWeekday(n.Args[1].Pos, strings.TrimPrefix(pred, "weekday:"))
			if err != nil {
				return false, err
			}
			match = d.Weekday() == wd
		case pred == "businessday":
			match, err = isBusinessDay(d, ev.ctx)
			if err != nil {
				return false, withErrorPos(n.Pos, err)
			}
		case pred == "weekend":
			match = isWeekendDay(d.Weekday(), ev.ctx.Cfg.Weekend)
		case pred == "weekdaykw":
			match = !isWeekendDay(d.Weekday(), ev.ctx.Cfg.Weekend)
		case pred == "holiday":
			match, err = isHolidayOrDayOff(d, ev.ctx)
			if err != nil {
				return false, withErrorPos(n.Pos, err)
			}
		default:
			return false, &Error{Pos: n.Args[1].Pos, Msg: fmt.Sprintf("unknown predicate %q", pred)}
		}
		if op == "!=" {
			return !match, nil
		}
		return match, nil
	}

	rhs, err := ev.node(n.Args[1], in)
	if err != nil {
		return false, err
	}

	if (op == "==" || op == "!=") && (lhs.Kind == KDate || lhs.Kind == KDateTime) && rhs.Kind == KSpan {
		var eq bool
		if rhs.Days {
			d := dateOnly(lhs.T)
			eq = !d.Before(dateOnly(rhs.T)) && d.Before(dateOnly(rhs.End))
		} else {
			eq = !lhs.T.Before(rhs.T) && lhs.T.Before(rhs.End)
		}
		if op == "!=" {
			return !eq, nil
		}
		return eq, nil
	}

	if (op == "==" || op == "!=") && lhs.Kind == KSpan && rhs.Kind == KSpan {
		eq := lhs.Days == rhs.Days && lhs.T.Equal(rhs.T) && lhs.End.Equal(rhs.End)
		if op == "!=" {
			return !eq, nil
		}
		return eq, nil
	}

	if lhs.Kind == KNumber && rhs.Kind == KNumber {
		return cmpFloat(lhs.N, rhs.N, op), nil
	}

	if lhs.Kind == KDur && rhs.Kind == KDur {
		return compareDurations(n.Pos, lhs.D, rhs.D, op)
	}

	if lhs.Kind == KText && rhs.Kind == KText && (op == "==" || op == "!=") {
		eq := lhs.Str == rhs.Str
		if op == "!=" {
			return !eq, nil
		}
		return eq, nil
	}

	if isPointComparable(lhs.Kind) && isPointComparable(rhs.Kind) {
		if lhs.Kind == KDate && rhs.Kind == KDate {
			return cmpTime(dateOnly(lhs.T), dateOnly(rhs.T), op), nil
		}
		fallback := ev.todayStart()
		if lhs.Kind == KDate || lhs.Kind == KDateTime || lhs.Kind == KSpan {
			fallback = dateOnly(lhs.T)
		} else if rhs.Kind == KDate || rhs.Kind == KDateTime || rhs.Kind == KSpan {
			fallback = dateOnly(rhs.T)
		}
		t1, err := resolvePoint(ev, n.Pos, lhs, fallback)
		if err != nil {
			return false, err
		}
		t2, err := resolvePoint(ev, n.Pos, rhs, fallback)
		if err != nil {
			return false, err
		}
		return cmpTime(t1, t2, op), nil
	}

	return false, &Error{Pos: n.Pos, Msg: fmt.Sprintf("cannot compare %s and %s", kindName(lhs.Kind), kindName(rhs.Kind))}
}

func isPointComparable(k Kind) bool {
	return k == KDate || k == KDateTime || k == KTime || k == KSpan
}

func cmpFloat(a, b float64, op string) bool {
	switch op {
	case "==":
		return a == b
	case "!=":
		return a != b
	case "<":
		return a < b
	case "<=":
		return a <= b
	case ">":
		return a > b
	case ">=":
		return a >= b
	default:
		return false
	}
}

func cmpTime(a, b time.Time, op string) bool {
	switch op {
	case "==":
		return a.Equal(b)
	case "!=":
		return !a.Equal(b)
	case "<":
		return a.Before(b)
	case "<=":
		return !a.After(b)
	case ">":
		return a.After(b)
	case ">=":
		return !a.Before(b)
	default:
		return false
	}
}

func compareDurations(pos int, a, b Duration, op string) (bool, error) {
	if a.Months == 0 && b.Months == 0 && a.BizDays == 0 && b.BizDays == 0 {
		na := float64(a.Days)*float64(24*time.Hour) + float64(a.Nanos)
		nb := float64(b.Days)*float64(24*time.Hour) + float64(b.Nanos)
		return cmpFloat(na, nb, op), nil
	}
	if a.Days == 0 && b.Days == 0 && a.Nanos == 0 && b.Nanos == 0 && a.BizDays == 0 && b.BizDays == 0 {
		return cmpFloat(float64(a.Months), float64(b.Months), op), nil
	}
	if a.Months == 0 && b.Months == 0 && a.Days == 0 && b.Days == 0 && a.Nanos == 0 && b.Nanos == 0 {
		return cmpFloat(float64(a.BizDays), float64(b.BizDays), op), nil
	}
	if op == "==" {
		return a == b, nil
	}
	if op == "!=" {
		return a != b, nil
	}
	return false, &Error{Pos: pos, Msg: "cannot compare durations with mixed calendar and clock units"}
}

func (ev *evaluator) evalAt(n Node, in Value, targetLoc *time.Location) (Value, error) {
	if len(n.Args) < 2 {
		return Value{}, &Error{Pos: n.Pos, Msg: "invalid time attachment"}
	}
	lhs, err := ev.node(n.Args[0], in)
	if err != nil {
		return Value{}, err
	}
	h, mi, s, err := parseClockParts(n.Args[1].Text, n.Args[1].Pos)
	if err != nil {
		return Value{}, err
	}

	var y int
	var m time.Month
	var d int
	var loc *time.Location
	switch lhs.Kind {
	case KDate, KDateTime, KSpan:
		y, m, d = lhs.T.Date()
		loc = lhs.T.Location()
	default:
		return Value{}, &Error{Pos: n.Pos, Msg: "cannot attach a time of day to this value"}
	}
	if targetLoc != nil {
		loc = targetLoc
	}
	t, err := ev.wallAt(n.Args[1].Pos, y, m, d, h, mi, s, loc)
	if err != nil {
		return Value{}, err
	}
	return Value{Kind: KDateTime, T: t, Local: lhs.Local}, nil
}

func (ev *evaluator) evalDateTime(n Node, targetLoc *time.Location) (Value, error) {
	tIdx := strings.IndexByte(n.Text, 'T')
	if tIdx < 0 {
		return Value{}, &Error{Pos: n.Pos, Msg: fmt.Sprintf("invalid datetime %q", n.Text)}
	}
	datePart := n.Text[:tIdx]
	rest := n.Text[tIdx+1:]

	y, m, d, err := parseISODate(datePart, n.Pos)
	if err != nil {
		return Value{}, err
	}

	var timePart string
	var fixedLoc *time.Location
	hasZone := false
	if strings.HasSuffix(rest, "Z") {
		hasZone = true
		fixedLoc = time.UTC
		timePart = rest[:len(rest)-1]
	} else if idx := strings.IndexAny(rest, "+-"); idx >= 0 {
		hasZone = true
		timePart = rest[:idx]
		tzStr := rest[idx:]
		var err error
		fixedLoc, err = parseFixedOffset(tzStr, n.Pos)
		if err != nil {
			return Value{}, err
		}
	} else {
		timePart = rest
	}

	h, mi, s, nsec, err := parseTimeWithFrac(timePart, n.Pos)
	if err != nil {
		return Value{}, err
	}

	if hasZone {
		t := time.Date(y, m, d, h, mi, s, nsec, fixedLoc)
		return Value{Kind: KDateTime, T: t, Local: false}, nil
	}

	if targetLoc == nil {
		targetLoc = ev.loc
	}
	t, err := ev.wallAt(n.Pos, y, m, d, h, mi, s, targetLoc)
	if err != nil {
		return Value{}, err
	}
	if nsec != 0 {
		t = t.Add(time.Duration(nsec))
	}
	return Value{Kind: KDateTime, T: t, Local: true}, nil
}

func parseISODate(s string, pos int) (int, time.Month, int, error) {
	parts := strings.Split(s, "-")
	if len(parts) != 3 || len(parts[0]) != 4 || len(parts[1]) != 2 || len(parts[2]) != 2 {
		return 0, 0, 0, &Error{Pos: pos, Msg: fmt.Sprintf("invalid date %q", s)}
	}
	y, err1 := strconv.Atoi(parts[0])
	mo, err2 := strconv.Atoi(parts[1])
	d, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil || !validDate(y, time.Month(mo), d) {
		return 0, 0, 0, &Error{Pos: pos, Msg: fmt.Sprintf("invalid date %q", s)}
	}
	return y, time.Month(mo), d, nil
}

func parseSlashDate(s string, pos int, lang Lang, strict bool) (int, time.Month, int, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 3 {
		return 0, 0, 0, &Error{Pos: pos, Msg: fmt.Sprintf("invalid date %q", s)}
	}
	a, err1 := strconv.Atoi(parts[0])
	b, err2 := strconv.Atoi(parts[1])
	yr, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, &Error{Pos: pos, Msg: fmt.Sprintf("invalid date %q", s)}
	}
	if len(parts[2]) == 2 {
		if yr <= 68 {
			yr += 2000
		} else {
			yr += 1900
		}
	}
	mdy := a >= 1 && a <= 12 && validDate(yr, time.Month(a), b)
	dmy := b >= 1 && b <= 12 && validDate(yr, time.Month(b), a)
	if !mdy && !dmy {
		return 0, 0, 0, &Error{Pos: pos, Msg: fmt.Sprintf("invalid date %q", s)}
	}
	if mdy && dmy && a != b {
		if strict {
			return 0, 0, 0, &Error{Pos: pos, Msg: fmt.Sprintf("ambiguous slash date %q", s)}
		}
		if lang == ES {
			return yr, time.Month(b), a, nil
		}
		return yr, time.Month(a), b, nil
	}
	if mdy {
		return yr, time.Month(a), b, nil
	}
	return yr, time.Month(b), a, nil
}

func parseClockParts(s string, pos int) (int, int, int, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 && len(parts) != 3 {
		return 0, 0, 0, &Error{Pos: pos, Msg: fmt.Sprintf("invalid time %q", s)}
	}
	h, err1 := strconv.Atoi(parts[0])
	mi, err2 := strconv.Atoi(parts[1])
	sec := 0
	var err3 error
	if len(parts) == 3 {
		sec, err3 = strconv.Atoi(parts[2])
	}
	if err1 != nil || err2 != nil || err3 != nil || h < 0 || h > 23 || mi < 0 || mi > 59 || sec < 0 || sec > 59 {
		return 0, 0, 0, &Error{Pos: pos, Msg: fmt.Sprintf("invalid time %q", s)}
	}
	return h, mi, sec, nil
}

func parseTimeWithFrac(s string, pos int) (int, int, int, int, error) {
	hms := s
	nsec := 0
	if dot := strings.IndexByte(s, '.'); dot >= 0 {
		hms = s[:dot]
		frac := s[dot+1:]
		if frac == "" {
			return 0, 0, 0, 0, &Error{Pos: pos, Msg: fmt.Sprintf("invalid time %q", s)}
		}
		if len(frac) > 9 {
			frac = frac[:9]
		}
		for len(frac) < 9 {
			frac += "0"
		}
		var err error
		nsec, err = strconv.Atoi(frac)
		if err != nil {
			return 0, 0, 0, 0, &Error{Pos: pos, Msg: fmt.Sprintf("invalid time %q", s)}
		}
	}
	h, mi, sec, err := parseClockParts(hms, pos)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return h, mi, sec, nsec, nil
}

func parseFixedOffset(tzStr string, pos int) (*time.Location, error) {
	if len(tzStr) < 2 {
		return nil, &Error{Pos: pos, Msg: fmt.Sprintf("invalid timezone offset %q", tzStr)}
	}
	sign := 1
	if tzStr[0] == '-' {
		sign = -1
	} else if tzStr[0] != '+' {
		return nil, &Error{Pos: pos, Msg: fmt.Sprintf("invalid timezone offset %q", tzStr)}
	}
	digits := strings.ReplaceAll(tzStr[1:], ":", "")
	if len(digits) != 4 {
		return nil, &Error{Pos: pos, Msg: fmt.Sprintf("invalid timezone offset %q", tzStr)}
	}
	oh, err1 := strconv.Atoi(digits[:2])
	om, err2 := strconv.Atoi(digits[2:])
	if err1 != nil || err2 != nil || oh < 0 || oh > 23 || om < 0 || om > 59 {
		return nil, &Error{Pos: pos, Msg: fmt.Sprintf("invalid timezone offset %q", tzStr)}
	}
	off := sign * (oh*3600 + om*60)
	if off == 0 {
		return time.UTC, nil
	}
	return time.FixedZone(fmt.Sprintf("%c%02d:%02d", tzStr[0], oh, om), off), nil
}

func validDate(y int, m time.Month, d int) bool {
	if y < 1 || m < time.January || m > time.December || d < 1 {
		return false
	}
	return d <= daysInMonth(y, m)
}

func daysInMonth(y int, m time.Month) int {
	switch m {
	case time.January, time.March, time.May, time.July, time.August, time.October, time.December:
		return 31
	case time.April, time.June, time.September, time.November:
		return 30
	case time.February:
		if y%4 == 0 && (y%100 != 0 || y%400 == 0) {
			return 29
		}
		return 28
	default:
		return 0
	}
}

func (ev *evaluator) lookupZoneNode(n Node) (*time.Location, error) {
	loc, err := LookupZone(n.Text, ev.ctx.Cfg.Strict)
	if err != nil {
		if e, ok := err.(*Error); ok {
			e.Pos = n.Pos
		}
		return nil, err
	}
	return loc, nil
}

func (ev *evaluator) evalZoneStage(n Node, in Value) (Value, error) {
	loc, err := ev.lookupZoneNode(n)
	if err != nil {
		return Value{}, err
	}
	return interpretInZone(n.Pos, in, loc, ev)
}

func (ev *evaluator) evalInterpret(n Node, in Value) (Value, error) {
	if len(n.Args) == 1 {
		loc, err := ev.lookupZoneNode(n.Args[0])
		if err != nil {
			return Value{}, err
		}
		return interpretInZone(n.Pos, in, loc, ev)
	}
	if len(n.Args) >= 2 {
		loc, err := ev.lookupZoneNode(n.Args[1])
		if err != nil {
			return Value{}, err
		}
		return ev.evalWithZoneLoc(n.Args[0], in, loc, true)
	}
	return Value{}, &Error{Pos: n.Pos, Msg: "invalid timezone interpretation"}
}

func (ev *evaluator) evalInZone(n Node, in Value) (Value, error) {
	if len(n.Args) == 1 {
		loc, err := ev.lookupZoneNode(n.Args[0])
		if err != nil {
			return Value{}, err
		}
		if in.Local {
			return interpretInZone(n.Pos, in, loc, ev)
		}
		return convertToZone(n.Pos, in, loc, ev)
	}
	if len(n.Args) >= 2 {
		loc, err := ev.lookupZoneNode(n.Args[1])
		if err != nil {
			return Value{}, err
		}
		return ev.evalWithZoneLoc(n.Args[0], in, loc, false)
	}
	return Value{}, &Error{Pos: n.Pos, Msg: "invalid timezone expression"}
}

func (ev *evaluator) evalWithZoneLoc(valNode Node, in Value, targetLoc *time.Location, interpretAlways bool) (Value, error) {
	switch valNode.Op {
	case "clock":
		h, mi, s, err := parseClockParts(valNode.Text, valNode.Pos)
		if err != nil {
			return Value{}, err
		}
		y, m, d := ev.today()
		t, err := ev.wallAt(valNode.Pos, y, m, d, h, mi, s, targetLoc)
		if err != nil {
			return Value{}, err
		}
		return Value{Kind: KDateTime, T: t, Local: false}, nil

	case "datetime":
		v, err := ev.evalDateTime(valNode, targetLoc)
		if err != nil {
			return Value{}, err
		}
		if v.Local {
			v.Local = false
			return v, nil
		}
		if interpretAlways {
			return interpretInZone(valNode.Pos, v, targetLoc, ev)
		}
		return convertToZone(valNode.Pos, v, targetLoc, ev)

	case "at":
		v, err := ev.evalAt(valNode, in, targetLoc)
		if err != nil {
			return Value{}, err
		}
		v.Local = false
		return v, nil

	default:
		v, err := ev.node(valNode, in)
		if err != nil {
			return Value{}, err
		}
		if interpretAlways || v.Local {
			return interpretInZone(valNode.Pos, v, targetLoc, ev)
		}
		return convertToZone(valNode.Pos, v, targetLoc, ev)
	}
}

func convertToZone(pos int, v Value, loc *time.Location, ev *evaluator) (Value, error) {
	switch v.Kind {
	case KDateTime:
		return Value{Kind: KDateTime, T: v.T.In(loc), Local: false}, nil
	case KDate:
		return Value{Kind: KDateTime, T: v.T.In(loc), Local: false}, nil
	case KTime:
		y, m, d := ev.today()
		t, err := ev.wallAt(pos, y, m, d, v.T.Hour(), v.T.Minute(), v.T.Second(), ev.loc)
		if err != nil {
			return Value{}, err
		}
		return Value{Kind: KDateTime, T: t.In(loc), Local: false}, nil
	case KSpan:
		return Value{
			Kind:  KSpan,
			T:     v.T.In(loc),
			End:   v.End.In(loc),
			D:     v.D,
			Days:  false,
			Local: false,
		}, nil
	case KRecurrence:
		return convertRecurrenceToZone(pos, v, loc)
	default:
		return Value{}, &Error{
			Pos: pos,
			Msg: fmt.Sprintf("cannot convert %s to timezone", kindName(v.Kind)),
		}
	}
}

func interpretInZone(pos int, v Value, loc *time.Location, ev *evaluator) (Value, error) {
	switch v.Kind {
	case KDateTime:
		t, err := ev.wallAt(pos, v.T.Year(), v.T.Month(), v.T.Day(), v.T.Hour(), v.T.Minute(), v.T.Second(), loc)
		if err != nil {
			return Value{}, err
		}
		if ns := v.T.Nanosecond(); ns != 0 {
			t = t.Add(time.Duration(ns))
		}
		return Value{Kind: KDateTime, T: t, Local: false}, nil
	case KTime:
		y, m, d := ev.today()
		t, err := ev.wallAt(pos, y, m, d, v.T.Hour(), v.T.Minute(), v.T.Second(), loc)
		if err != nil {
			return Value{}, err
		}
		return Value{Kind: KDateTime, T: t, Local: false}, nil
	case KDate:
		t, err := ev.wallAt(pos, v.T.Year(), v.T.Month(), v.T.Day(), 0, 0, 0, loc)
		if err != nil {
			return Value{}, err
		}
		return Value{Kind: KDate, T: t, Local: false}, nil
	case KSpan:
		s, err := ev.wallAt(pos, v.T.Year(), v.T.Month(), v.T.Day(), v.T.Hour(), v.T.Minute(), v.T.Second(), loc)
		if err != nil {
			return Value{}, err
		}
		e, err := ev.wallAt(pos, v.End.Year(), v.End.Month(), v.End.Day(), v.End.Hour(), v.End.Minute(), v.End.Second(), loc)
		if err != nil {
			return Value{}, err
		}
		return Value{
			Kind:  KSpan,
			T:     s,
			End:   e,
			D:     v.D,
			Days:  v.Days,
			Local: false,
		}, nil
	case KRecurrence:
		return interpretRecurrenceInZone(pos, v, loc, ev)
	default:
		return Value{}, &Error{
			Pos: pos,
			Msg: fmt.Sprintf("cannot interpret %s in timezone", kindName(v.Kind)),
		}
	}
}
