package tunascript

import (
	"fmt"
)

// InterpretOpText returns the interpreted TS op text.
func InterpretOpText(s string) (string, error) {
	lexed, err := LexOperationText(s)
	if err != nil {
		return "", err
	}

	// TODO: need debug
	ast, err := parseOpExpression(&lexed, 0)
	if err != nil {
		return "", err
	}

	fullTree := AST{
		nodes: []*astNode{ast},
	}

	output := translateOperators(fullTree)

	return output, nil
}

func Parse(tokens tokenStream) (AST, error) {
	ast, err := parseOpExpression(&tokens, 0)
	if err != nil {
		return AST{}, err
	}

	fullTree := AST{
		nodes: []*astNode{ast},
	}

	return fullTree, nil
}

func parseOpExpression(stream *tokenStream, rbp int) (*astNode, error) {

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
