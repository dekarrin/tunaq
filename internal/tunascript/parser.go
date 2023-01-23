package tunascript

import (
	"fmt"
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

//
