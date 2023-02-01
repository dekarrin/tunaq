package tunascript

import (
	"fmt"
	"strings"

	"github.com/dekarrin/tunaq/internal/util"
)

// Parse builds an abstract syntax tree by reading the tokens in the provided
// tokenStream. It returns the built up AST that is parsed from it. If any
// issues are encountered, an error is returned (likely a SyntaxError).
func Parse(tokens tokenStream) (AST, error) {
	ast, err := parseExpression(&tokens, 0)
	if err != nil {
		return AST{}, err
	}

	fullTree := AST{
		nodes: []*astNode{ast},
	}

	return fullTree, nil
}

func parseExpression(stream *tokenStream, rbp int) (*astNode, error) {
	// TODO: consider implementing panic mode to parse rest of the system

	var err error

	if stream.Remaining() < 1 {
		return nil, fmt.Errorf("no tokens to parse")
	}

	t := stream.Next()
	left, err := t.nud(stream)
	if err != nil {
		return nil, err
	}
	if left == nil {
		return nil, syntaxErrorFromLexeme(fmt.Sprintf("unexpected %[1]s\n(%[1]s cannot be at the start of an expression)", t.class.human), t)
	}

	for rbp < stream.Peek().class.lbp {
		t = stream.Next()
		left, err = t.led(left, stream)
		if err != nil {
			return nil, err
		}
	}
	return left, nil

}

type parseTree struct {
	terminal bool
	value    string
	children []*parseTree
}

// String returns a prettified representation of the entire parse tree suitable
// for use in line-by-line comparisons of tree structure. Two parse trees are
// considered semantcally identical if they produce identical String() output.
func (pt parseTree) String() string {
	return pt.leveledStr("", "")
}

func (pt parseTree) leveledStr(firstPrefix, contPrefix string) string {
	var sb strings.Builder

	sb.WriteString(firstPrefix)
	if pt.terminal {
		sb.WriteString(fmt.Sprintf("(TERM %q)", pt.value))
	} else {
		sb.WriteString(fmt.Sprintf("( %s )", pt.value))
	}

	for i := range pt.children {
		sb.WriteRune('\n')
		var leveledFirstPrefix string
		var leveledContPrefix string
		if i+1 < len(pt.children) {
			leveledFirstPrefix = contPrefix + makeASTTreeLevelPrefix("")
			leveledContPrefix = contPrefix + astTreeLevelOngoing
		} else {
			leveledFirstPrefix = contPrefix + makeASTTreeLevelPrefixLast("")
			leveledContPrefix = contPrefix + astTreeLevelEmpty
		}
		itemOut := pt.children[i].leveledStr(leveledFirstPrefix, leveledContPrefix)
		sb.WriteString(itemOut)
	}

	return sb.String()
}

// Equal returns whether the parseTree is equal to the given object. If the
// given object is not a parseTree, returns false, else returns whether the two
// parse trees have the exact same structure.
func (pt parseTree) Equal(o any) bool {
	other, ok := o.(parseTree)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*parseTree)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if pt.terminal != other.terminal {
		return false
	} else if pt.value != other.value {
		return false
	} else {
		// check every sub tree
		if len(pt.children) != len(other.children) {
			return false
		}

		for i := range pt.children {
			if !pt.children[i].Equal(other.children[i]) {
				return false
			}
		}
	}
	return true
}

// LL1PredictiveParse runse a parse of the input using LL(k) parsing rules on
// the context-free Grammar g (k=1). The grammar must be LL(1); it will not be
// forced to it.
func LL1PredictiveParse(g Grammar, stream tokenStream) (pt parseTree, err error) {
	M, err := g.LLParseTable()
	if err != nil {
		return pt, err
	}

	stack := util.Stack[string]{Of: []string{g.StartSymbol(), "$"}}
	next := stream.Peek()
	X := stack.Peek()
	pt = parseTree{value: g.StartSymbol()}
	ptStack := util.Stack[*parseTree]{Of: []*parseTree{&pt}}

	node := ptStack.Peek()
	for X != "$" { /* stack is not empty */
		if strings.ToLower(X) == X {
			stream.Next()

			// is terminals
			t := g.Term(X)
			if next.class.Equal(t) {
				node.terminal = true
				stack.Pop()
				X = stack.Peek()
				ptStack.Pop()
				node = ptStack.Peek()
			} else {
				return pt, syntaxErrorFromLexeme(fmt.Sprintf("There should be a %s here, but it was %q!", t.human, next.lexeme), next)
			}

			next = stream.Peek()
		} else {
			nextProd := M.Get(X, g.TermFor(next.class))
			if nextProd.Equal(Error) {
				return pt, syntaxErrorFromLexeme(fmt.Sprintf("It doesn't make any sense to put a %q here!", next.class.human), next)
			}

			stack.Pop()
			ptStack.Pop()
			for i := len(nextProd) - 1; i >= 0; i-- {
				if nextProd[i] != Epsilon[0] {
					stack.Push(nextProd[i])
				}

				child := &parseTree{value: nextProd[i]}
				if nextProd[i] == Epsilon[0] {
					child.terminal = true
				}
				node.children = append([]*parseTree{child}, node.children...)

				if nextProd[i] != Epsilon[0] {
					ptStack.Push(child)
				}
			}

			X = stack.Peek()

			// node stack will always be one smaller than symbol stack bc
			// glub, we dont put a node onto the stack for "$".
			if X != "$" {
				node = ptStack.Peek()
			}
		}
	}

	return pt, nil
}

// ShiftReduceParser is a parser that performs shift-reduce moves. These will be
// generated from a grammar for the purposes of performing bottom-up parsing
type ShiftReduceParser interface {
	// Parse parses the given tokenStream.
	Parse(stream tokenStream) parseTree

	// Shift reads one token of input. For SR parsers that are implemented with
	// a stack, this will push a terminal onto the stack.
	//
	// ABC|xyz => ABCx|yz
	Shift()

	// Reduce applies an inverse production at the right end of the left string.
	// For SR parsers that are implemented with a stack, this will pop 0 or more
	// terminals off of the stack (production rhs), then will push a
	// non-terminal onto the stack (production lhs).
	//
	// Given A -> xy is a production, then:
	// Cbxy|ijk => CbA|ijk
	Reduce()
}
