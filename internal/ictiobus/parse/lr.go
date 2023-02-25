package parse

import (
	"fmt"
	"strings"

	"github.com/dekarrin/tunaq/internal/ictiobus/automaton"
	"github.com/dekarrin/tunaq/internal/ictiobus/grammar"
	"github.com/dekarrin/tunaq/internal/ictiobus/icterrors"
	"github.com/dekarrin/tunaq/internal/ictiobus/types"
	"github.com/dekarrin/tunaq/internal/util"
)

// LRParseTable is a table of information passed to an LR parser. These will be
// generated from a grammar for the purposes of performing bottom-up parsing.
type LRParseTable interface {
	// Shift reads one token of input. For SR parsers that are implemented with
	// a stack, this will push a terminal onto the stack.
	//
	// ABC|xyz => ABCx|yz
	//Shift()

	// Reduce applies an inverse production at the right end of the left string.
	// For SR parsers that are implemented with a stack, this will pop 0 or more
	// terminals off of the stack (production rhs), then will push a
	// non-terminal onto the stack (production lhs).
	//
	// Given A -> xy is a production, then:
	// Cbxy|ijk => CbA|ijk
	//Reduce()

	// Initial returns the initial state of the parse table, if that is
	// applicable for the table.
	Initial() string

	// Action gets the next action to take based on a state i and terminal a.
	Action(state, symbol string) LRAction

	// Goto maps a state and a grammar symbol to some other state.
	Goto(state, symbol string) (string, error)

	// String prints a string representation of the table. If two LRParseTables
	// produce the same String() output, they are considered equal.
	String() string

	// GetDFA returns the DFA simulated by the table. Some tables may in fact
	// be the DFA itself along with supplementary info.
	GetDFA() automaton.DFA[string]
}

type lrParser struct {
	table     LRParseTable
	parseType types.ParserType
	gram      grammar.Grammar
	trace     func(s string)
}

func (lr *lrParser) GetDFA() *automaton.DFA[string] {
	dfa := lr.table.GetDFA()
	return &dfa
}

func (lr *lrParser) RegisterTraceListener(listener func(s string)) {
	lr.trace = listener
}

func (lr *lrParser) Type() types.ParserType {
	return lr.parseType
}

func (lr *lrParser) TableString() string {
	return lr.table.String()
}

func (lr lrParser) notifyTraceFn(fn func() string) {
	if lr.trace != nil {
		lr.trace(fn())
	}
}

func (lr lrParser) notifyTrace(fmtStr string, args ...interface{}) {
	lr.notifyTraceFn(func() string { return fmt.Sprintf(fmtStr, args...) })
}

func (lr lrParser) notifyStatePeek(s string) {
	lr.notifyTrace("states.peek(): %s", s)
}

func (lr lrParser) notifyStatePush(s string) {
	lr.notifyTrace("states.push(): %s", s)
}

func (lr lrParser) notifyStatePop(s string) {
	if s == "" {
		lr.notifyTrace("states.pop()")
	} else {
		lr.notifyTrace("states.pop(): %s", s)
	}
}

func (lr lrParser) notifyAction(act LRAction) {
	lr.notifyTrace("Action: %s", act.Type.String())
}

func (lr lrParser) notifyNextToken(tok types.Token) {
	lr.notifyTrace("Got next token: %s", tok.String())
}

func (lr lrParser) notifyTokenStack(st util.Stack[types.Token]) {
	lr.notifyTraceFn(func() string {
		var lexStr strings.Builder
		var tokStr strings.Builder
		for i := range st.Of {
			tok := st.Of[i]
			lexStr.WriteRune('"')
			lexStr.WriteString(tok.Lexeme())
			lexStr.WriteRune('"')

			tokStr.WriteString(strings.ToUpper(tok.Class().ID()))

			if i+1 < len(st.Of) {
				lexStr.WriteString(", ")
				tokStr.WriteString(", ")
			}
		}
		if st.Empty() {
			lexStr.WriteString("(empty)")
			tokStr.WriteString("(empty)")
		}

		str := fmt.Sprintf("Token stack (lexed): %s", lexStr.String())
		str += "\n"
		str += fmt.Sprintf("Token stack (ttype): %s", tokStr.String())

		return str
	})
}

// Parse parses the input stream with the internal LR parse table.
//
// This is an implementation of Algorithm 4.44, "LR-parsing algorithm", from
// the purple dragon book.
func (lr *lrParser) Parse(stream types.TokenStream) (types.ParseTree, error) {
	stateStack := util.Stack[string]{Of: []string{lr.table.Initial()}}

	// we will use these to build our parse tree
	tokenBuffer := util.Stack[types.Token]{}
	subTreeRoots := util.Stack[*types.ParseTree]{}

	// let a be the first symbol of w$;
	a := stream.Next()
	lr.notifyNextToken(a)

	for { /* repeat forever */
		lr.notifyTokenStack(tokenBuffer)

		// let s be the state on top of the stack;
		s := stateStack.Peek()
		lr.notifyStatePeek(s)

		ACTION := lr.table.Action(s, a.Class().ID())
		lr.notifyAction(ACTION)

		switch ACTION.Type {
		case LRShift: // if ( ACTION[s, a] = shift t )
			// add token to our buffer
			tokenBuffer.Push(a)

			t := ACTION.State

			// push t onto the stack
			stateStack.Push(t)
			lr.notifyStatePush(t)

			// let a be the next input symbol
			a = stream.Next()
			lr.notifyNextToken(a)
		case LRReduce: // else if ( ACTION[s, a] = reduce A -> β )
			A := ACTION.Symbol
			beta := ACTION.Production

			// use the reduce to create a node in the parse tree
			node := &types.ParseTree{Value: A, Children: make([]*types.ParseTree, 0)}
			// we need to go from right to left of the production to pop things
			// from the stacks in the correct order
			for i := len(beta) - 1; i >= 0; i-- {
				sym := beta[i]
				if strings.ToLower(sym) == sym {
					// it is a terminal. read the source from the token buffer
					tok := tokenBuffer.Pop()
					subNode := &types.ParseTree{Terminal: true, Value: tok.Class().ID(), Source: tok}
					node.Children = append([]*types.ParseTree{subNode}, node.Children...)
				} else {
					// it is a non-terminal. it should be in our stack of
					// current tree roots.
					subNode := subTreeRoots.Pop()
					node.Children = append([]*types.ParseTree{subNode}, node.Children...)
				}
			}
			// remember it for next time
			subTreeRoots.Push(node)

			// pop |β| symbols off the stack;
			for i := 0; i < len(beta); i++ {
				stateStack.Pop()
				lr.notifyStatePop("")
			}

			// let state t now be on top of the stack
			t := stateStack.Peek()
			lr.notifyStatePeek(t)

			// push GOTO[t, A] onto the stack
			toPush, err := lr.table.Goto(t, A)
			if err != nil {
				return types.ParseTree{}, icterrors.NewSyntaxErrorFromToken(fmt.Sprintf("LR parsing error; DFA has no valid transition from here on %q", A), a)
			}
			stateStack.Push(toPush)
			lr.notifyStatePush(toPush)

			// output the production A -> β
			// (TODO: put it on the parse tree)
		case LRAccept: // else if ( ACTION[s, a] = accept )
			// parsing is done. there should be at least one item on the stack
			pt := subTreeRoots.Pop()
			return *pt, nil
		case LRError:
			// call error-recovery routine
			// TODO: error recovery, for now, just report it
			expMessage := lr.getExpectedString(s)
			return types.ParseTree{}, icterrors.NewSyntaxErrorFromToken(fmt.Sprintf("unexpected %s; %s", a.Class().Human(), expMessage), a)
		}
	}
}

func (lr lrParser) getExpectedString(stateName string) string {
	expected := lr.findExpectedTokens(stateName)

	var sb strings.Builder

	sb.WriteString("expected ")

	commas := false
	finalOr := false

	if len(expected) > 1 {
		finalOr = true
		if len(expected) > 2 {
			commas = true
		}
	}
	for i := range expected {
		t := expected[i]

		if i == 0 {
			sb.WriteString(util.ArticleFor(t.Human(), false))
			sb.WriteRune(' ')
		}

		if finalOr && i+1 == len(expected) {
			sb.WriteString(" or ")
		}

		sb.WriteString(t.Human())
		if commas && i+1 < len(expected) {
			sb.WriteString(", ")
		}
	}

	return sb.String()
}

// findExpectedAt returns all token classes that are allowed/expected for
// the given state, that is, those symbols that result in a non-error entry.
func (lr lrParser) findExpectedTokens(stateName string) []types.TokenClass {
	terms := lr.gram.Terminals()

	classes := make([]types.TokenClass, 0)
	for i := range terms {
		t := lr.gram.Term(terms[i])
		act := lr.table.Action(stateName, t.ID())
		if act.Type != LRError {
			classes = append(classes, t)
		}
	}

	return classes
}
