package tunascript

// file eval.go has functions responsible for evaluation of AST trees.

import (
	"fmt"
	"strings"
)

// invoke takes the given AST and evaluates it to produce a value. If the given
// AST has multiple expression nodes, there will be multiple values returned.
func (inter Interpreter) invoke(ast AST, queryOnly bool) ([]Value, error) {
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

				// remove the quote-signs
				strVal = strVal[len(literalStrStringQuote) : len(strVal)-len(literalStrStringQuote)]

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
			funcName = funcName[len(literalStrIdentifierStart):]

			fn, ok := inter.fn[funcName]
			if !ok {
				return nil, syntaxErrorFromLexeme(fmt.Sprintf("function %s%s%s does not exist", literalStrIdentifierStart, funcName, literalStrGroupOpen+literalStrGroupClose), n.source)
			}

			// restrict if requested
			if queryOnly && fn.SideEffects {
				return nil, syntaxErrorFromLexeme(fmt.Sprintf("function %s%s%s will change the game state and is not allowed here", literalStrIdentifierStart, funcName, literalStrGroupOpen+literalStrGroupClose), n.source)
			}

			if len(funcArgNodes) < fn.RequiredArgs {
				s := "s"
				if fn.RequiredArgs == 1 {
					s = ""
				}
				return nil, syntaxErrorFromLexeme(fmt.Sprintf("function %s%s%s requires at least %d parameter%s; %d given", literalStrIdentifierStart, fn.Name, literalStrGroupOpen+literalStrGroupClose, fn.RequiredArgs, s, len(funcArgNodes)), n.source)
			}

			maxArgs := fn.RequiredArgs + fn.OptionalArgs
			if len(funcArgNodes) > maxArgs {
				s := "s"
				if maxArgs == 1 {
					s = ""
				}
				return nil, syntaxErrorFromLexeme(fmt.Sprintf("function %s%s%s takes at most %d parameter%s; %d given", literalStrIdentifierStart, fn.Name, literalStrGroupOpen+literalStrGroupClose, maxArgs, s, len(funcArgNodes)), n.source)
			}

			// evaluate args:
			args := make([]Value, len(funcArgNodes))
			for argIdx := range funcArgNodes {
				toEval := AST{
					nodes: []*astNode{funcArgNodes[argIdx]},
				}

				argResult, err := inter.invoke(toEval, queryOnly)
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
			flagName = flagName[len(literalStrIdentifierStart):]

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

			vals, err := inter.invoke(toEval, queryOnly)
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

				left, err := inter.invoke(leftExec, queryOnly)
				if err != nil {
					return nil, err
				}
				right, err := inter.invoke(rightExec, queryOnly)
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

				evaluated, err := inter.invoke(toExec, queryOnly)
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

// TranslateOperators returns the given tunascript re-written to have no
// operators and exclusively use functions to reproduce the same functionality.
// Every operator in the provided string s is converted into one or more
// built-in function calls that produce the same result.
func TranslateOperators(s string) (string, error) {
	lexed, err := Lex(s)
	if err != nil {
		return "", err
	}

	// TODO: need debug
	ast, err := parseExpression(&lexed, 0)
	if err != nil {
		return "", err
	}

	fullTree := AST{
		nodes: []*astNode{ast},
	}

	output := TranslateOperatorsInAST(fullTree)

	return output, nil
}

// TranslateOperatorsInAST turns the ast into a tunascript string containing only
// function calls and no operators. Originally this was for a 2-pass compiler
// but that is overkill; current ast already handles all cases.
func TranslateOperatorsInAST(ast AST) string {
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
			writeRuneSlice(&sb, literalGroupOpen)

			for i := range node.fn.args {
				toExec := AST{
					nodes: []*astNode{node.fn.args[i]},
				}
				insert := TranslateOperatorsInAST(toExec)
				sb.WriteString(insert)
				if i+1 < len(node.fn.args) {
					writeRuneSlice(&sb, literalSeparator)
					sb.WriteRune(' ')
				}
			}

			writeRuneSlice(&sb, literalGroupClose)
		} else if node.flag != nil {
			sb.WriteString(node.flag.name)
		} else if node.group != nil {
			writeRuneSlice(&sb, literalGroupOpen)
			toExec := AST{
				nodes: []*astNode{node.group.expr},
			}
			insert := TranslateOperatorsInAST(toExec)
			sb.WriteString(insert)
			writeRuneSlice(&sb, literalGroupClose)
		} else if node.opGroup != nil {
			if node.opGroup.infixOp != nil {
				op := node.opGroup.infixOp.op
				leftExec := AST{
					nodes: []*astNode{node.opGroup.infixOp.left},
				}
				rightExec := AST{
					nodes: []*astNode{node.opGroup.infixOp.right},
				}

				leftInsert := TranslateOperatorsInAST(leftExec)
				rightInsert := TranslateOperatorsInAST(rightExec)

				funcTemplate := binaryOpFuncTranslations[op]
				if funcTemplate == "" {
					// should never happen
					panic(fmt.Sprintf("unknown binary operator %q", op))
				}

				sb.WriteString(fmt.Sprintf(funcTemplate, literalStrIdentifierStart, literalStrGroupOpen, literalStrGroupClose, leftInsert, rightInsert, literalStrSeparator))
			} else if node.opGroup.unaryOp != nil {
				op := node.opGroup.unaryOp.op
				toExec := AST{
					nodes: []*astNode{node.opGroup.unaryOp.operand},
				}
				toInsert := TranslateOperatorsInAST(toExec)

				funcTemplate := unaryOpFuncTranslations[op]
				if funcTemplate == "" {
					// should never happen
					panic(fmt.Sprintf("unknown unary operator %q", op))
				}

				sb.WriteString(fmt.Sprintf(funcTemplate, literalStrIdentifierStart, literalStrGroupOpen, literalStrGroupClose, toInsert))
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

func writeRuneSlice(sb *strings.Builder, r []rune) {
	for i := range r {
		sb.WriteRune(r[i])
	}
}
