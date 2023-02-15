package parse

import (
	"fmt"
	"strings"

	"github.com/dekarrin/tunaq/internal/ictiobus/grammar"
	"github.com/dekarrin/tunaq/internal/ictiobus/icterrors"
	"github.com/dekarrin/tunaq/internal/ictiobus/types"
	"github.com/dekarrin/tunaq/internal/util"
)

type ll1Parser struct {
	table grammar.LL1Table
	g     grammar.Grammar
}

// GenerateLL1Parser generates a parser for LL1 grammar g. The grammar must
// already be LL1 or convertible to an LL1 grammar.
//
// The returned parser parses the input using LL(k) parsing rules on the
// context-free Grammar g (k=1). The grammar must already be LL(1); it will not
// be forced to it.
func GenerateLL1Parser(g grammar.Grammar) (ll1Parser, error) {
	M, err := g.LLParseTable()
	if err != nil {
		return ll1Parser{}, err
	}
	return ll1Parser{table: M, g: g.Copy()}, nil
}

func (ll1 ll1Parser) Parse(stream types.TokenStream) (types.ParseTree, error) {
	stack := util.Stack[string]{Of: []string{ll1.g.StartSymbol(), "$"}}
	next := stream.Peek()
	X := stack.Peek()
	pt := types.ParseTree{Value: ll1.g.StartSymbol()}
	ptStack := util.Stack[*types.ParseTree]{Of: []*types.ParseTree{&pt}}

	node := ptStack.Peek()
	for X != "$" { /* stack is not empty */
		if strings.ToLower(X) == X {
			stream.Next()

			// is terminals
			t := ll1.g.Term(X)
			if next.Class().ID() == t.ID() {
				node.Terminal = true
				node.Source = next
				stack.Pop()
				X = stack.Peek()
				ptStack.Pop()
				node = ptStack.Peek()
			} else {
				return pt, icterrors.NewSyntaxErrorFromToken(fmt.Sprintf("There should be a %s here, but it was %q!", t.Human(), next.Lexeme()), next)
			}

			next = stream.Peek()
		} else {
			nextProd := ll1.table.Get(X, ll1.g.TermFor(next.Class()))
			if nextProd.Equal(grammar.Error) {
				return pt, icterrors.NewSyntaxErrorFromToken(fmt.Sprintf("It doesn't make any sense to put a %q here!", next.Class().Human()), next)
			}

			stack.Pop()
			ptStack.Pop()
			for i := len(nextProd) - 1; i >= 0; i-- {
				if nextProd[i] != grammar.Epsilon[0] {
					stack.Push(nextProd[i])
				}

				child := &types.ParseTree{Value: nextProd[i]}
				if nextProd[i] == grammar.Epsilon[0] {
					child.Terminal = true
				}
				node.Children = append([]*types.ParseTree{child}, node.Children...)

				if nextProd[i] != grammar.Epsilon[0] {
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
