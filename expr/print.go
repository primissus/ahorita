package expr

import "strings"

// String prints the expression in canonical English, with stages joined by " | ".
func (e Expr) String() string {
	parts := make([]string, len(e.Stages))
	for i, s := range e.Stages {
		parts[i] = s.String()
	}
	return strings.Join(parts, " | ")
}

// String prints the node in canonical English.
func (n Node) String() string {
	switch n.Op {
	case "date", "datetime", "clock", "slash", "num", "year", "weekday", "key":
		return n.Text
	case "unix":
		return "@" + n.Text
	case "now", "today", "yesterday", "tomorrow":
		return n.Op
	case "unit":
		return printUnitSingular(n.Text)
	case "dur":
		parts := make([]string, len(n.Args))
		for i, part := range n.Args {
			parts[i] = printDurPart(part)
		}
		return strings.Join(parts, " ")
	case "+", "-", "*":
		if len(n.Args) < 2 {
			return n.Op
		}
		return n.Args[0].String() + " " + n.Op + " " + n.Args[1].String()
	case "ago":
		if len(n.Args) == 0 {
			return "ago"
		}
		return n.Args[0].String() + " ago"
	case "in":
		if len(n.Args) == 0 {
			return "in"
		}
		return "in " + n.Args[0].String()
	case "rel":
		if len(n.Args) == 0 {
			return n.Text
		}
		return n.Text + " " + n.Args[0].String()
	case "orsame":
		if len(n.Args) == 0 {
			return "on or after"
		}
		phrase := "on or after"
		if n.Args[0].Text == "last" {
			phrase = "on or before"
		}
		out := n.Args[0].String() + " " + phrase
		if len(n.Args) >= 2 {
			out += " " + n.Args[1].String()
		}
		return out
	case "month":
		if len(n.Args) == 0 {
			return n.Text
		}
		parts := make([]string, 1, 1+len(n.Args))
		parts[0] = n.Text
		for _, arg := range n.Args {
			parts = append(parts, arg.String())
		}
		return strings.Join(parts, " ")
	case "period":
		if len(n.Args) == 0 {
			return n.Text
		}
		return n.Text + " " + n.Args[0].String()
	case "week":
		return "week " + n.Text
	case "bound":
		if len(n.Args) == 0 {
			return n.Text
		}
		return n.Text + " of " + n.Args[0].String()
	case "of":
		if len(n.Args) < 2 {
			return "of"
		}
		return n.Args[0].String() + " of " + n.Args[1].String()
	case "closest":
		switch len(n.Args) {
		case 0:
			return "closest"
		case 1:
			return "closest " + n.Args[0].String()
		default:
			return "closest " + n.Args[0].String() + " to " + n.Args[1].String()
		}
	case "during":
		if len(n.Args) == 0 {
			return "during"
		}
		return "during " + n.Args[0].String()
	case "holidays":
		parts := []string{"holidays"}
		if n.Text != "" {
			parts = append(parts, n.Text)
		}
		if len(n.Args) > 0 {
			parts = append(parts, n.Args[0].String())
		}
		return strings.Join(parts, " ")
	case "starting":
		if len(n.Args) == 0 {
			return "starting"
		}
		return "starting " + n.Args[0].String()
	case "span":
		if len(n.Args) < 2 {
			return ".."
		}
		return n.Args[0].String() + " .. " + n.Args[1].String()
	case "count":
		if len(n.Args) < 3 {
			return "how many"
		}
		return "how many " + printPluralTarget(n.Args[0]) + " between " + n.Args[1].String() + " and " + n.Args[2].String()
	case "inclusive":
		if len(n.Args) == 0 {
			return "inclusive"
		}
		return n.Args[0].String() + " inclusive"
	case "to":
		if len(n.Args) == 0 {
			return "to"
		}
		if n.Args[0].Op == "unit" {
			return "to " + printPluralTarget(n.Args[0])
		}
		return "to " + n.Args[0].String()
	case "zone":
		return printZone(n.Text)
	case "interpret":
		switch len(n.Args) {
		case 0:
			return "@"
		case 1:
			return "@ " + n.Args[0].String()
		default:
			return n.Args[0].String() + " @ " + n.Args[1].String()
		}
	case "inzone":
		switch len(n.Args) {
		case 0:
			return "in"
		case 1:
			return "in " + n.Args[0].String()
		default:
			return n.Args[0].String() + " in " + n.Args[1].String()
		}
	case "at":
		if len(n.Args) < 2 {
			return ""
		}
		return n.Args[0].String() + " " + n.Args[1].String()
	default:
		return n.Text
	}
}

func printDurPart(part Node) string {
	switch part.Op {
	case "second":
		return part.Text + "s"
	case "minute":
		return part.Text + "m"
	case "hour":
		return part.Text + "h"
	case "day":
		return part.Text + "d"
	case "week":
		return part.Text + "w"
	case "month":
		return part.Text + "M"
	case "quarter":
		return part.Text + "q"
	case "year":
		return part.Text + "y"
	case "businessday":
		if part.Text == "1" {
			return "1 business day"
		}
		return part.Text + " business days"
	case "half":
		if part.Text == "1" {
			return "1 half"
		}
		return part.Text + " halves"
	case "workweek":
		return part.Text + " work week"
	default:
		return part.Text + " " + part.Op
	}
}

func printUnitSingular(unit string) string {
	switch unit {
	case "workweek":
		return "work week"
	case "businessday":
		return "business day"
	default:
		return unit
	}
}

func printPluralTarget(arg Node) string {
	if arg.Op == "weekday" {
		return arg.Text + "s"
	}
	if arg.Op != "unit" {
		return arg.String()
	}
	switch arg.Text {
	case "half":
		return "halves"
	case "businessday":
		return "business days"
	case "workweek":
		return "work week"
	default:
		return arg.Text + "s"
	}
}

func printZone(text string) string {
	if strings.ContainsRune(text, ' ') {
		return `"` + text + `"`
	}
	return text
}
