package tunascript

import (
	"fmt"
	"strings"
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

	fullTree := opAST{
		nodes: []*opASTNode{ast},
	}

	output := executeOpTree(fullTree)

	return output, nil
}

// just apply to the parse tree
func executeOpTree(ast opAST) string {
	var sb strings.Builder

	for i := 0; i < len(ast.nodes); i++ {
		node := ast.nodes[i]

		if node.value != nil {
			if node.value.quotedStringVal != nil {
				// should never happen
				// TODO: since lexer is designed for no distinction between unparsed text and quoted string
				// eliminate this branch from the AST entirely
				sb.WriteString(*node.value.quotedStringVal)
			} else if node.value.unquotedStringVal != nil {
				sb.WriteString(*node.value.unquotedStringVal)
			} else if node.value.boolVal != nil {
				sb.WriteString(fmt.Sprintf("%t", *node.value.boolVal))
			} else if node.value.numVal != nil {
				sb.WriteString(fmt.Sprintf("%d", *node.value.numVal))
			} else {
				panic("empty value node in AST")
			}
		} else if node.fn != nil {
			sb.WriteString(node.fn.name)
			sb.WriteRune('(')

			for i := range node.fn.args {
				toExec := opAST{
					nodes: []*opASTNode{node.fn.args[i]},
				}
				insert := executeOpTree(toExec)
				sb.WriteString(insert)
				if i+1 < len(node.fn.args) {
					sb.WriteRune(',')
					sb.WriteRune(' ')
				}
			}

			sb.WriteRune(')')
		} else if node.flag != nil {
			sb.WriteString(node.flag.name)
		} else if node.group != nil {
			sb.WriteRune('(')
			toExec := opAST{
				nodes: []*opASTNode{node.group.expr},
			}
			insert := executeOpTree(toExec)
			sb.WriteString(insert)
			sb.WriteRune(')')
		} else if node.opGroup != nil {
			if node.opGroup.infixOp != nil {
				op := node.opGroup.infixOp.op
				leftExec := opAST{
					nodes: []*opASTNode{node.opGroup.infixOp.left},
				}
				rightExec := opAST{
					nodes: []*opASTNode{node.opGroup.infixOp.right},
				}

				leftInsert := executeOpTree(leftExec)
				rightInsert := executeOpTree(rightExec)

				var opFunc string
				if op == "+" {
					opFunc = "ADD"
				} else if op == "-" {
					opFunc = "SUB"
				} else if op == "/" {
					opFunc = "DIV"
				} else if op == "*" {
					opFunc = "MULT"
				} else if op == "&&" {
					opFunc = "AND"
				} else if op == "::" {
					opFunc = "OR"
				} else {
					// should never happen
					panic(fmt.Sprintf("unknown binary operator %q", op))
				}

				sb.WriteString(opFunc)
				sb.WriteRune('(')
				sb.WriteString(strings.TrimSpace(leftInsert))
				sb.WriteRune(',')
				sb.WriteRune(' ')
				sb.WriteString(strings.TrimSpace(rightInsert))
				sb.WriteRune(')')
			} else if node.opGroup.unaryOp != nil {
				op := node.opGroup.unaryOp.op
				toExec := opAST{
					nodes: []*opASTNode{node.opGroup.unaryOp.operand},
				}
				toInsert := executeOpTree(toExec)
				var opFunc string
				if op == "!" {
					opFunc = "NOT"
				} else if op == "++" {
					opFunc = "INC"
				} else if op == "--" {
					opFunc = "DEC"
				} else {
					// should never happen
					panic(fmt.Sprintf("unknown unary operator %q", op))
				}

				sb.WriteString(opFunc)
				sb.WriteRune('(')
				sb.WriteString(toInsert)
				sb.WriteRune(')')
			} else {
				// should never happen
				panic("opGroup node in AST does not assign infix or unary")
			}
		} else {
			// should never happen
			panic("empty AST node")
		}
	}

	return sb.String()
}

func parseOpExpression(stream *tokenStream, rbp int) (*opASTNode, error) {
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
		return nil, fmt.Errorf("%s cannot appear at start of expression", t.token.String())
	}

	for rbp < stream.Peek().lbp() {
		t = stream.Next()
		left, err = t.led(left, stream)
		if err != nil {
			return nil, err
		}
	}
	return left, nil

}
