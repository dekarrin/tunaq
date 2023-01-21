package tunascript

// file eval.go has functions responsible for evaluation of AST trees.

import (
	"fmt"
	"strings"
)

// eval takes the given AST and evaluates it to produce a value. If the given
// AST has multiple expression nodes, there will be multiple values returned.
func (inter Interpreter) eval(ast AST, queryOnly bool) ([]Value, error) {
	var values []Value = make([]Value, len(ast.nodes))

	// dont use range because that doesnt allow us to skip/backtrack i
	for i := 0; i < len(ast.nodes); i++ {
		n := ast.nodes[i]

		// what kind of a node is it?
		if n.value != nil {
			// node a literal value

			valNode := n.value
			if valNode.quotedStringVal != nil {
				// it is a quotedString
				strVal := (*valNode.quotedStringVal)

				// remove the @-signs
				strVal = strVal[1 : len(strVal)-1]

				v := NewStr(strVal)
				values[i] = v
			} else if valNode.unquotedStringVal != nil {
				// it is an unquoted string
				strVal := (*valNode.unquotedStringVal)
				v := NewStr(strVal)
				values[i] = v
			} else if valNode.boolVal != nil {
				// its a bool value

				boolVal := (*valNode.boolVal)
				v := NewBool(boolVal)
				values[i] = v
			} else if valNode.numVal != nil {
				// its a number value

				numVal := (*valNode.numVal)
				v := NewNum(numVal)
				values[i] = v
			} else {
				panic("empty value node in ast")
			}
		} else if n.fn != nil {
			// node is a func call

			fnNode := n.fn

			funcArgNodes := fnNode.args
			funcName := strings.ToUpper(fnNode.name)

			// remove leading identifier marker '$' from func name
			funcName = funcName[1:]

			fn, ok := inter.fn[funcName]
			if !ok {
				return nil, syntaxErrorFromLexeme(fmt.Sprintf("function $%s() does not exist", funcName), n.source)
			}

			// restrict if requested
			if queryOnly && fn.SideEffects {
				return nil, syntaxErrorFromLexeme(fmt.Sprintf("function $%s() will change the game state and is not allowed here", funcName), n.source)
			}

			if len(funcArgNodes) < fn.RequiredArgs {
				s := "s"
				if fn.RequiredArgs == 1 {
					s = ""
				}
				return nil, syntaxErrorFromLexeme(fmt.Sprintf("function $%s() requires at least %d parameter%s; %d given", fn.Name, fn.RequiredArgs, s, len(funcArgNodes)), n.source)
			}

			maxArgs := fn.RequiredArgs + fn.OptionalArgs
			if len(funcArgNodes) > maxArgs {
				s := "s"
				if maxArgs == 1 {
					s = ""
				}
				return nil, syntaxErrorFromLexeme(fmt.Sprintf("function $%s() takes at most %d parameter%s; %d given", fn.Name, maxArgs, s, len(funcArgNodes)), n.source)
			}

			// evaluate args:
			args := make([]Value, len(funcArgNodes))
			for argIdx := range funcArgNodes {
				toEval := AST{
					nodes: []*astNode{funcArgNodes[argIdx]},
				}

				argResult, err := inter.eval(toEval, queryOnly)
				if err != nil {
					return nil, err
				}

				args[argIdx] = argResult[0]
			}

			// finally call the function

			// oh yeah. no error returned. you saw that right. the Call function is literally not allowed to fail.
			// int 100 bbyyyyyyyyyyyyyyyyyyy
			//
			// Oh my gog ::::/
			//
			// i AM ur gog now >38D
			//
			// This Is Later Than The Original Comment But I Must Say I Am Glad
			// That This Portion Retained Some Use With The Redesign. If Past
			// Info Is Needed Know That There Was A Prior Set Of Designs That
			// Also Functioned Correctly So The History Of This File May Be
			// Referred To.
			//
			// uhhhhh why is this comment still here?
			//
			// Capitalism probably. Honk.
			//
			// No, lol.
			//
			// I Think It Has Simply Traveled From The Old Algorithm To This One
			// But You Are Right This Is Turning Into A Much Longer Discussion.

			v := fn.Call(args)
			values[i] = v
		} else if n.flag != nil {
			// node is a flag (variable) value

			flagNode := n.flag
			flagName := strings.ToUpper(flagNode.name)

			// remove identifier sign '$'
			flagName = flagName[1:]

			var v Value
			flag, ok := inter.flags[flagName]
			if ok {
				v = flag.Value
			} else {
				v = NewStr("")
			}
			values[i] = v
		} else if n.group != nil {
			// node is a parenthesized group

			toEval := AST{
				nodes: []*astNode{n.group.expr},
			}

			vals, err := inter.eval(toEval, queryOnly)
			if err != nil {
				return nil, err
			}

			v := vals[0]
			values[i] = v
		} else if n.opGroup != nil {
			// node is an operator applied to some operand(s)

			var opFn Function
			var opArgs []Value

			// which operator is it?
			if n.opGroup.infixOp != nil {
				opNode := n.opGroup.infixOp

				var ok bool
				opFn, ok = inter.opFn[opNode.op]
				if !ok {
					// should never happen
					panic(fmt.Sprintf("no implementation found for operator %q", opNode.op))
				}

				leftExec := AST{
					nodes: []*astNode{opNode.left},
				}
				rightExec := AST{
					nodes: []*astNode{opNode.left},
				}

				left, err := inter.eval(leftExec, queryOnly)
				if err != nil {
					return nil, err
				}
				right, err := inter.eval(rightExec, queryOnly)
				if err != nil {
					return nil, err
				}

				opArgs = []Value{left[0], right[0]}
			} else if n.opGroup.unaryOp != nil {
				opNode := n.opGroup.unaryOp

				var ok bool
				opFn, ok = inter.opFn[opNode.op]
				if !ok {
					// should never happen
					panic(fmt.Sprintf("no implementation found for operator %q", opNode.op))
				}

				toExec := AST{
					nodes: []*astNode{opNode.operand},
				}

				evaluated, err := inter.eval(toExec, queryOnly)
				if err != nil {
					return nil, err
				}

				opArgs = []Value{evaluated[0]}
			} else {
				// should never happen
				panic("empty op group node in ast")
			}

			v := opFn.Call(opArgs)
			values[i] = v
		} else {
			// should never happen
			panic("empty node in ast")
		}
	}

	return nil, nil
}

// translateOperators turns the ast into a tunascript string containing only
// function calls and no operators. Originally this was for a 2-pass compiler
// but that is overkill; current ast already handles all cases.
func translateOperators(ast AST) string {
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
				toExec := AST{
					nodes: []*astNode{node.fn.args[i]},
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
			toExec := AST{
				nodes: []*astNode{node.group.expr},
			}
			insert := translateOperators(toExec)
			sb.WriteString(insert)
			sb.WriteRune(')')
		} else if node.opGroup != nil {
			if node.opGroup.infixOp != nil {
				op := node.opGroup.infixOp.op
				leftExec := AST{
					nodes: []*astNode{node.opGroup.infixOp.left},
				}
				rightExec := AST{
					nodes: []*astNode{node.opGroup.infixOp.right},
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
				toExec := AST{
					nodes: []*astNode{node.opGroup.unaryOp.operand},
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
