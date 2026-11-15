package expr

// Node is one node in a stage's syntax tree.
type Node struct {
	Op   string // see the node table in parser.go
	Pos  int    // byte offset of the node's first token
	Text string // meaning depends on Op
	Args []Node
}

// Expr is a parsed expression: a pipeline of stages in order.
type Expr struct {
	Input  string
	Stages []Node
	Lang   Lang  // Lang of the first token that has one; "" if none
	Ties   []int // Pos of every token Normalize flagged with Tie
}

// Error is a parse or evaluation error with source position and expected next tokens.
type Error struct {
	Pos      int      `json:"pos"`                // byte offset, -1 if unknown
	Msg      string   `json:"message"`            // human-readable error message
	Expected []string `json:"expected,omitempty"` // what could have come next (parse errors)
}

func (e *Error) Error() string {
	return e.Msg
}
