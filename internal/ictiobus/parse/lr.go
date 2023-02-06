package parse

import (
	"strings"

	"github.com/dekarrin/tunaq/internal/ictiobus/icterrors"
	"github.com/dekarrin/tunaq/internal/ictiobus/lex"
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
}

type lrParser struct {
	table LRParseTable
}

// Parse parses the input stream with the internal LR parse table.
//
// This is an implementation of Algorithm 4.44, "LR-parsing algorithm", from
// the purple dragon book.
func (lr lrParser) Parse(stream lex.TokenStream) (Tree, error) {
	stateStack := util.Stack[string]{Of: []string{lr.table.Initial()}}

	// we will use these to build our parse tree
	tokenBuffer := util.Stack[lex.Token]{}
	subTreeRoots := util.Stack[*Tree]{}

	// let a be the first symbol of w$;
	a := stream.Next()

	for { /* repeat forever */
		// let s be the state on top of the stack;
		s := stateStack.Peek()

		ACTION := lr.table.Action(s, a.Class().ID())

		switch ACTION.Type {
		case LRShift: // if ( ACTION[s, a] = shift t )
			// add token to our buffer
			tokenBuffer.Push(a)

			t := ACTION.State

			// push t onto the stack
			stateStack.Push(t)

			// let a be the next input symbol
			a = stream.Next()
		case LRReduce: // else if ( ACTION[s, a] = reduce A -> β )
			A := ACTION.Symbol
			beta := ACTION.Production

			// use the reduce to create a node in the parse tree
			node := &Tree{Value: A, Children: make([]*Tree, 0)}
			// we need to go from right to left of the production to pop things
			// from the stacks in the correct order
			for i := len(beta) - 1; i >= 0; i-- {
				sym := beta[i]
				if strings.ToLower(sym) == sym {
					// it is a terminal. read the source from the token buffer
					tok := tokenBuffer.Pop()
					subNode := &Tree{Terminal: true, Value: tok.Class().ID(), Source: tok}
					node.Children = append([]*Tree{subNode}, node.Children...)
				} else {
					// it is a non-terminal. it should be in our stack of
					// current tree roots.
					subNode := subTreeRoots.Pop()
					node.Children = append([]*Tree{subNode}, node.Children...)
				}
			}
			// remember it for next time
			subTreeRoots.Push(node)

			// pop |β| symbols off the stack;
			for i := 0; i < len(beta); i++ {
				stateStack.Pop()
			}

			// let state t now be on top of the stack
			t := stateStack.Peek()

			// push GOTO[t, A] onto the stack
			toPush, err := lr.table.Goto(t, A)
			if err != nil {
				return Tree{}, icterrors.NewSyntaxErrorFromToken("parsing failed", a)
			}
			stateStack.Push(toPush)

			// output the production A -> β
			// TODO: put it on the parse tree
		case LRAccept: // else if ( ACTION[s, a] = accept )
			// parsing is done. there should be at least one item on the stack
			pt := subTreeRoots.Pop()
			return *pt, nil
		case LRError:
			// call error-recovery routine
			// TODO: error recovery, for now, just report it
			return Tree{}, icterrors.NewSyntaxErrorFromToken("parsing failed", a)
		}
	}
}
