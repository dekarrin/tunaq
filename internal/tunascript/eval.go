package tunascript

// file eval.go has functions responsible for evaluation of AST trees.

import (
	"fmt"
	"strings"
)

// eval takes the given AST and evaluates it to produce a value. If the given
// AST has multiple expression nodes, there will be multiple values returned.
func eval(ast opAST, queryOnly bool) ([]Value, error) {
	var values []Value
	var err error

	// dont use range because that doesnt allow us to skip/backtrack i
	for i := 0; i < len(ast.nodes); i++ {
		n := ast.nodes[i]

		// what kind of a node is it?
		if n.value != nil {
			// node is a value
		} else if n.fn != nil {
			// node is a func call
		} else if n.flag != nil {
			// node is a flag (variable) value
		} else if n.group != nil {
			// node is a parenthesized group
		} else if n.opGroup != nil {
			// node is an operator applied to some operand(s)
		} else {
			// should never happen
		}
	}

	return nil, nil
}

// translateOperators turns the ast into a tunascript string containing only
// function calls and no operators. Originally this was for a 2-pass compiler
// but that is overkill; current ast already handles all cases.
func translateOperators(ast opAST) string {
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
				insert := translateOperators(toExec)
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
			insert := translateOperators(toExec)
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

				leftInsert := translateOperators(leftExec)
				rightInsert := translateOperators(rightExec)

				var funcTemplate string
				if op == "+" {
					funcTemplate = "$ADD(%s, %s)"
				} else if op == "-" {
					funcTemplate = "$SUB(%s, %s)"
				} else if op == "/" {
					funcTemplate = "$DIV(%s, %s)"
				} else if op == "*" {
					funcTemplate = "$MULT(%s, %s)"
				} else if op == "&&" {
					funcTemplate = "$AND(%s, %s)"
				} else if op == "||" {
					funcTemplate = "$OR(%s, %s)"
				} else if op == "+=" {
					funcTemplate = "$INC(%s, %s)"
				} else if op == "-=" {
					funcTemplate = "$DEC(%s, %s)"
				} else if op == "!=" {
					funcTemplate = "$NOT(FLAG_IS(%s, %s))"
				} else if op == "==" {
					funcTemplate = "$FLAG_IS(%s, %s)"
				} else if op == "<" {
					funcTemplate = "$FLAG_LESS_THAN(%s, %s)"
				} else if op == ">" {
					funcTemplate = "$FLAG_GREATER_THAN(%s, %s)"
				} else if op == ">=" {
					funcTemplate = "$OR($FLAG_GREATER_THAN(%[1]s, %[2]s), $FLAG_IS(%[1]s, %[2]s))"
				} else if op == "<=" {
					funcTemplate = "$OR($FLAG_LESS_THAN(%[1]s, %[2]s), $FLAG_IS(%[1]s, %[2]s))"
				} else if op == "=" {
					funcTemplate = "$SET(%[1]s, %[2]s)"
				} else {
					// should never happen
					panic(fmt.Sprintf("unknown binary operator %q", op))
				}

				sb.WriteString(fmt.Sprintf(funcTemplate, leftInsert, rightInsert))
			} else if node.opGroup.unaryOp != nil {
				op := node.opGroup.unaryOp.op
				toExec := opAST{
					nodes: []*opASTNode{node.opGroup.unaryOp.operand},
				}
				toInsert := translateOperators(toExec)
				var funcTemplate string
				if op == "!" {
					funcTemplate = "$NOT(%s)"
				} else if op == "++" {
					funcTemplate = "$INC(%s)"
				} else if op == "--" {
					funcTemplate = "$DEC(%s)"
				} else if op == "-" {
					funcTemplate = "$NEG(%s)"
				} else {
					// should never happen
					panic(fmt.Sprintf("unknown unary operator %q", op))
				}

				sb.WriteString(fmt.Sprintf(funcTemplate, toInsert))
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
