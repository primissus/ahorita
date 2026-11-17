package expr

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ResultStep records one evaluated pipeline stage for Result.Explain.
type ResultStep struct {
	Stage string `json:"stage"`
	Value string `json:"value"`
}

// Result holds the evaluation, formatted output, optional explanation steps, and error for an expression.
type Result struct {
	Input   string       `json:"input"`
	Alias   string       `json:"alias,omitempty"`
	Kind    string       `json:"kind,omitempty"`  // "date", "datetime", "time", "duration", "span", "number", "list", "text", "recurrence"
	Text    string       `json:"text,omitempty"`  // formatted per Options
	ISO     string       `json:"iso,omitempty"`   // Value.String()
	Unix    *int64       `json:"unix,omitempty"`  // dates and date-times
	Start   string       `json:"start,omitempty"` // spans, and durations that remember their ends
	End     string       `json:"end,omitempty"`
	Note    string       `json:"note,omitempty"`
	Explain []ResultStep `json:"explain,omitempty"`
	Error   *Error       `json:"error,omitempty"`
}

// Run evaluates, formats and explains in one call. It never returns a Go error:
// failures are in Result.Error.
func Run(input string, ctx Context, o Options, explain bool) Result {
	res := Result{Input: input}

	if rawName, _, isAssign := SplitAliasAssignment(input); isAssign {
		if foldedName, err := ValidateAliasName(rawName); err == nil {
			res.Alias = foldedName
		}
	}

	steps, _, err := evalPipeline(input, ctx)
	if err != nil {
		res.Error = toExprError(err)
		return res
	}

	v := steps[len(steps)-1].Value
	fmtOpts := o
	if strings.EqualFold(o.Pattern, "json") || strings.EqualFold(o.Pattern, "yaml") || strings.EqualFold(o.Pattern, "yml") {
		fmtOpts.Pattern = ""
	}
	text, err := Format(v, ctx, fmtOpts)
	if err != nil {
		res.Error = toExprError(err)
		return res
	}

	res.Kind = resultKind(v.Kind)
	res.Text = text
	res.ISO = v.String()
	res.Note = v.Note

	switch v.Kind {
	case KDate:
		u := dateOnly(v.T).Unix()
		res.Unix = &u
	case KDateTime:
		u := v.T.Unix()
		res.Unix = &u
	case KSpan:
		if v.Days {
			res.Start = v.T.Format("2006-01-02")
			res.End = v.End.AddDate(0, 0, -1).Format("2006-01-02")
		} else {
			res.Start = v.T.Format(time.RFC3339)
			res.End = v.End.Format(time.RFC3339)
		}
	case KDur:
		if v.HasEnds {
			if v.Days {
				res.Start = v.T.Format("2006-01-02")
				res.End = v.End.Format("2006-01-02")
			} else {
				res.Start = v.T.Format(time.RFC3339)
				res.End = v.End.Format(time.RFC3339)
			}
		}
	}

	if explain {
		res.Explain = make([]ResultStep, len(steps))
		for i, st := range steps {
			res.Explain[i] = ResultStep{
				Stage: st.Text,
				Value: st.Value.String(),
			}
		}
	}

	return res
}

func toExprError(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return &Error{Pos: -1, Msg: err.Error()}
}

func resultKind(k Kind) string {
	switch k {
	case KDate:
		return "date"
	case KDateTime:
		return "datetime"
	case KTime:
		return "time"
	case KDur:
		return "duration"
	case KSpan:
		return "span"
	case KNumber:
		return "number"
	case KList:
		return "list"
	case KText:
		return "text"
	case KRecurrence:
		return "recurrence"
	default:
		return ""
	}
}

// MarshalYAML formats a Result as a YAML 1.2 document using only the Go standard library.
func MarshalYAML(r Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "input: %s\n", strconv.Quote(r.Input))
	if r.Alias != "" {
		fmt.Fprintf(&b, "alias: %s\n", strconv.Quote(r.Alias))
	}
	if r.Kind != "" {
		fmt.Fprintf(&b, "kind: %s\n", strconv.Quote(r.Kind))
	}
	if r.Text != "" {
		fmt.Fprintf(&b, "text: %s\n", strconv.Quote(r.Text))
	}
	if r.ISO != "" {
		fmt.Fprintf(&b, "iso: %s\n", strconv.Quote(r.ISO))
	}
	if r.Unix != nil {
		fmt.Fprintf(&b, "unix: %d\n", *r.Unix)
	}
	if r.Start != "" {
		fmt.Fprintf(&b, "start: %s\n", strconv.Quote(r.Start))
	}
	if r.End != "" {
		fmt.Fprintf(&b, "end: %s\n", strconv.Quote(r.End))
	}
	if r.Note != "" {
		fmt.Fprintf(&b, "note: %s\n", strconv.Quote(r.Note))
	}
	if len(r.Explain) > 0 {
		b.WriteString("explain:\n")
		for _, st := range r.Explain {
			fmt.Fprintf(&b, "  - stage: %s\n", strconv.Quote(st.Stage))
			fmt.Fprintf(&b, "    value: %s\n", strconv.Quote(st.Value))
		}
	}
	if r.Error != nil {
		b.WriteString("error:\n")
		fmt.Fprintf(&b, "  pos: %d\n", r.Error.Pos)
		fmt.Fprintf(&b, "  message: %s\n", strconv.Quote(r.Error.Msg))
		if len(r.Error.Expected) > 0 {
			b.WriteString("  expected:\n")
			for _, exp := range r.Error.Expected {
				fmt.Fprintf(&b, "    - %s\n", strconv.Quote(exp))
			}
		}
	}
	return b.String()
}
