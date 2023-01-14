package tunascript

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// AbstractSyntaxTree is an AST of parsed (but not interpreted) block of
// Tunascript code. The zero-value of an AbstractSyntaxTree is not suitable for
// use, and they should only be created by calls to ParseExpression.
type AbstractSyntaxTree struct {
	root     *AbstractSyntaxTree
	children []*AbstractSyntaxTree
	sym      symbol
	t        nodeType
}

type symbolType int

const (
	symbolValue symbolType = iota
	symbolDollar
	symbolIdentifier
)

func (st symbolType) String() string {
	switch st {
	case symbolValue:
		return "value"
	case symbolDollar:
		return "\"$\""
	case symbolIdentifier:
		return "identifier"
	default:
		return "UNKNOWN SYMBOL TYPE"
	}
}

type symbol struct {
	sType  symbolType
	source string

	// only applies if sType is symbolValue
	forceStr bool
}

type lexerState int

const (
	lexDefault lexerState = iota
	lexIdent
	lexStr
)

type nodeType int

const (
	nodeItem nodeType = iota
	nodeGroup
	nodeRoot
)

// indexOfMatchingTunascriptParen takes the given string, which must start with
// a parenthesis char "(", and returns the index of the ")" that matches it. Any
// text in between is analyzed for other parenthesis and if they are there, they
// must be matched as well.
//
// Index will be -1 if it does not have a match.
// Error is non-nil if there is malformed tunascript syntax between the parens,
// of if s cannot be operated on.
//
// Also returns the parsable AST of the analyzed expression as well.
func indexOfMatchingParen(sRunes []rune) (int, *AbstractSyntaxTree, error) {
	// without a parent node on a paren scan, buildAST will produce an error.
	dummyNode := &AbstractSyntaxTree{
		children: make([]*AbstractSyntaxTree, 0),
	}
	dummyNode.root = dummyNode

	if sRunes[0] != '(' {
		var errStr string
		if len(sRunes) > 50 {
			errStr = string(sRunes[:50]) + "..."
		} else {
			errStr = string(sRunes)
		}
		return 0, nil, fmt.Errorf("no opening paren at start of analysis string %q", errStr)
	}

	if len(sRunes) < 2 {
		return 0, nil, fmt.Errorf("unexpected end of expression (unmatched left-parenthesis)")
	}

	exprNode, consumed, err := buildAST(string(sRunes[1:]), dummyNode)
	if err != nil {
		return 0, nil, err
	}
	exprNode.t = nodeRoot

	return consumed, exprNode, nil
}

// ParseText interprets the text in the abstract syntax tree and evaluates it.
func (inter Interpreter) evalExpr(ast *AbstractSyntaxTree, queryOnly bool) ([]Value, error) {
	if ast.t != nodeRoot && ast.t != nodeGroup {
		return nil, fmt.Errorf("cannot parse AST anywhere besides root of the tree")
	}

	values := []Value{}

	for i := 0; i < len(ast.children); i++ {
		child := ast.children[i]
		if child.t == nodeGroup {
			return nil, fmt.Errorf("unexpected parenthesis group\n(to use \"(\" and \")\" as Str values, put them in between two \"|\" chars or\nescape them)")
		}
		if child.sym.sType == symbolIdentifier {
			// TODO: move this into AST builder,
			// for now just convert these here
			// an identifier on its own is rly just a val.
			child.sym.sType = symbolValue
		}

		if child.sym.sType == symbolDollar {
			// okay, check ahead for an ident
			if i+1 >= len(ast.children) {
				return nil, fmt.Errorf("unexpected bare \"$\" character at end\n(to use \"$\" as a  Str value, put it in between two \"|\" chars or escape it)")
			}
			identNode := ast.children[i+1]
			if identNode.t == nodeGroup {
				return nil, fmt.Errorf("unexpected parenthesis group after \"$\" character, expected identifier")
			}
			if identNode.sym.sType != symbolIdentifier {
				return nil, fmt.Errorf("unexpected %s after \"$\" character, expected identifier", identNode.sym.sType.String())
			}

			// we now have an identifier, but is this a var or function?
			isFunc := false
			if i+2 < len(ast.children) {
				argsNode := ast.children[i+2]
				if argsNode.t == nodeGroup {
					isFunc = true
				}
			}

			if isFunc {
				// function call, gather args and call it
				argsNode := ast.children[i+2]
				funcName := strings.ToUpper(identNode.sym.source)

				fn, ok := inter.fn[funcName]
				if !ok {
					return nil, fmt.Errorf("function $%s() does not exist", funcName)
				}

				// restrict if requested
				if queryOnly && fn.SideEffects {
					return nil, fmt.Errorf("function $%s() will change game state", funcName)
				}

				args, err := inter.evalExpr(argsNode, queryOnly)
				if err != nil {
					return nil, err
				}

				if len(args) < fn.RequiredArgs {
					s := "s"
					if fn.RequiredArgs == 1 {
						s = ""
					}
					return nil, fmt.Errorf("function $%s() requires at least %d parameter%s; %d given", fn.Name, fn.RequiredArgs, s, len(args))
				}

				maxArgs := fn.RequiredArgs + fn.OptionalArgs
				if len(args) > maxArgs {
					s := "s"
					if maxArgs == 1 {
						s = ""
					}
					return nil, fmt.Errorf("function $%s() takes at most %d parameter%s; %d given", fn.Name, maxArgs, s, len(args))
				}

				// oh yeah. no error returned. you saw that right. the Call function is literally not allowed to fail.
				// int 100 bbyyyyyyyyyyyyyyyyyyy
				//
				// Oh my gog ::::/
				//
				// i AM ur gog now >38D
				v := fn.Call(args)
				values = append(values, v)
				i += 2
			} else {
				// flag substitution, read it in
				flagName := strings.ToUpper(identNode.sym.source)

				var v Value
				flag, ok := inter.flags[flagName]
				if ok {
					v = flag.Value
				} else {
					v = NewStr("")
				}
				values = append(values, v)
				i++
			}
		} else if child.sym.sType == symbolValue {
			src := child.sym.source
			var v Value
			if child.sym.forceStr {
				v = NewStr(src)
			} else {
				v = parseUntypedValString(src)
			}
			values = append(values, v)
		}
	}
	return values, nil
}

func parseUntypedValString(valStr string) Value {
	srcUpper := strings.ToUpper(valStr)
	if srcUpper == "TRUE" || srcUpper == "YES" || srcUpper == "ON" {
		return NewBool(true)
	} else if srcUpper == "FALSE" || srcUpper == "NO" || srcUpper == "OFF" {
		return NewBool(false)
	}

	intVal, err := strconv.Atoi(valStr)
	if err == nil {
		return NewNum(intVal)
	}

	return NewStr(valStr)
}

// LexText lexes the text. Returns the AST, number of runes consumed
// and whether an error was encountered.
func buildAST(s string, parent *AbstractSyntaxTree) (*AbstractSyntaxTree, int, error) {
	node := &AbstractSyntaxTree{children: make([]*AbstractSyntaxTree, 0)}
	if parent == nil {
		node.root = node
	} else {
		node.root = parent.root
	}

	escaping := false
	mode := lexDefault
	node.t = nodeItem

	s = strings.TrimSpace(s)
	sRunes := []rune(s)
	sBytes := make([]int, len(sRunes))
	sBytesIdx := 0
	for b := range s {
		sBytes[sBytesIdx] = b
		sBytesIdx++
	}

	var buildingText string
	for i := 0; i < len(sRunes); i++ {
		ch := sRunes[i]
		//chS := string(ch)
		//fmt.Println(chS)

		switch mode {
		case lexIdent:
			if ('A' <= ch && ch <= 'Z') || ('a' <= ch && ch <= 'z') || ('0' <= ch && ch <= '9') || ch == '_' {
				buildingText += string(ch)
			} else {
				idNode := &AbstractSyntaxTree{
					root: node.root,
					sym:  symbol{sType: symbolIdentifier, source: buildingText},
				}
				node.children = append(node.children, idNode)
				buildingText = ""
				i--
				mode = lexDefault
			}
		case lexStr:
			if !escaping && ch == '\\' {
				// do not add a node for this
				escaping = true
			} else if escaping && ch == 'n' {
				buildingText += "\n"
				escaping = false
			} else if escaping && ch == 't' {
				buildingText += "\t"
				escaping = false
			} else if !escaping && ch == '|' {
				symNode := &AbstractSyntaxTree{
					root: node.root,
					sym:  symbol{sType: symbolValue, source: buildingText, forceStr: true},
				}
				node.children = append(node.children, symNode)
				buildingText = ""
				mode = lexDefault
			} else {
				buildingText += string(ch)
				escaping = false
			}
		case lexDefault:
			if !escaping && ch == '\\' {
				// do not add a node for this
				escaping = true
			} else if !escaping && ch == '$' {
				if buildingText != "" {
					textNode := &AbstractSyntaxTree{root: node.root, sym: symbol{sType: symbolValue, source: buildingText}}
					node.children = append(node.children, textNode)
					buildingText = ""
				}

				dNode := &AbstractSyntaxTree{
					root: node.root,
					sym:  symbol{sType: symbolDollar, source: "$"},
				}
				node.children = append(node.children, dNode)
				mode = lexIdent
			} else if !escaping && ch == '(' {
				if buildingText != "" {
					textNode := &AbstractSyntaxTree{root: node.root, sym: symbol{sType: symbolValue, source: buildingText}}
					node.children = append(node.children, textNode)
					buildingText = ""
				}

				if i+1 >= len(sRunes) {
					return nil, 0, fmt.Errorf("unexpected end of expression (unmatched left-parenthesis)")
				}
				nextByteIdx := sBytes[i+1]

				subNode, consumed, err := buildAST(s[nextByteIdx:], node)
				if err != nil {
					return nil, 0, err
				}
				subNode.t = nodeGroup

				node.children = append(node.children, subNode)
				i += consumed
			} else if !escaping && ch == ',' {
				if buildingText != "" {
					textNode := &AbstractSyntaxTree{root: node.root, sym: symbol{sType: symbolValue, source: buildingText}}
					node.children = append(node.children, textNode)
					buildingText = ""
				}

				// comma breaks up values but doesn't actually need to be its own node
			} else if !escaping && ch == '|' {
				if buildingText != "" {
					textNode := &AbstractSyntaxTree{root: node.root, sym: symbol{sType: symbolValue, source: buildingText}}
					node.children = append(node.children, textNode)
					buildingText = ""
				}

				// string start
				mode = lexStr
			} else if !escaping && ch == ')' {
				// we have reached the end of our parsing. if we are the PARENT,
				// this is an error
				if parent == nil {
					return nil, 0, fmt.Errorf("unexpected end of expression (unmatched right-parenthesis)")
				}

				if buildingText != "" {
					textNode := &AbstractSyntaxTree{root: node.root, sym: symbol{sType: symbolValue, source: buildingText}}
					node.children = append(node.children, textNode)
					buildingText = ""
				}

				// don't add it as a node, it's not relevant

				return node, i + 1, nil
			} else if escaping && ch == 'n' {
				buildingText += "\n"
				escaping = false
			} else if escaping && ch == 't' {
				buildingText += "\t"
				escaping = false
			} else {
				if escaping || !unicode.IsSpace(ch) {
					buildingText += string(ch)
				}
				escaping = false
			}
		}

	}

	// if we get to the end but we are not the parent, we have a paren mismatch
	if parent != nil {
		return nil, 0, fmt.Errorf("unexpected end of expression (unmatched left-parenthesis)")
	}

	node.t = nodeRoot

	if mode == lexDefault {
		if buildingText != "" {
			textNode := &AbstractSyntaxTree{root: node.root, sym: symbol{sType: symbolValue, source: buildingText}}
			node.children = append(node.children, textNode)
			buildingText = ""
		}
	} else if mode == lexIdent {
		if buildingText != "" {
			textNode := &AbstractSyntaxTree{root: node.root, sym: symbol{sType: symbolIdentifier, source: buildingText}}
			node.children = append(node.children, textNode)
			buildingText = ""
		}
	} else if mode == lexStr {
		return nil, 0, fmt.Errorf("unexpected end of expression (unmatched string start \"|\")")
	}

	// okay now go through and update
	// - make the values not have double quotes but force to str type

	return node, len(s), nil
}
