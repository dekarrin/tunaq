package tunascript

import (
	"fmt"
	"strings"
	"unicode"
)

// tunascript execution engine

// FuncCall is the implementation of a function in tunaquest. It receives some
// number of values defined externally and returns a Value of the appropriate
// type.
type FuncCall func(val []Value) Value

// Function defines a function that can be executed in tunascript.
type Function struct {
	// Name is the name of the function. It would be called with $Name(). Name
	// is case-insensitive and must follow identifier naming rules ([A-Z0-9_]+)
	Name string

	// RequiredArgs is the number of required arguments. This many arguments is
	// gauratneed to be passed to Call.
	RequiredArgs int

	// OptionalArgs is the number of optional arguments. Up to the
	OptionalArgs int

	// Call is a function point to the golang implementation of the function. It
	// is guaranteed to receive RequiredArgs, and may receive up to OptionalArgs
	// additional args.
	Call FuncCall
}

// Flag is a variable in the engine.
type Flag struct {
	Name string
	Value
}

type Interpreter struct {
	fn    map[string]Function
	flags map[string]*Flag
	world WorldInterface
}

type WorldInterface interface {

	// InInventory returns whether the given label Item is in the player
	// inventory.
	InInventory(label string) bool

	// Move moves the label to the dest. The label can be an NPC or an Item. If
	// label is "@PLAYER", the player will be moved. Returns whether the thing
	// moved.
	Move(label string, dest string) bool

	// Output prints the given string. Returns whether it did successfully.
	Output(s string) bool
}

func NewInterpreter(w WorldInterface) Interpreter {
	inter := Interpreter{
		fn:    make(map[string]Function),
		world: w,
	}

	inter.fn["ADD"] = Function{Name: "ADD", RequiredArgs: 2, Call: builtIn_Add}
	inter.fn["SUB"] = Function{Name: "SUB", RequiredArgs: 2, Call: builtIn_Sub}
	inter.fn["MULT"] = Function{Name: "MULT", RequiredArgs: 2, Call: builtIn_Mult}
	inter.fn["DIV"] = Function{Name: "DIV", RequiredArgs: 2, Call: builtIn_Add}
	inter.fn["OR"] = Function{Name: "OR", RequiredArgs: 2, Call: builtIn_Or}
	inter.fn["AND"] = Function{Name: "AND", RequiredArgs: 2, Call: builtIn_And}
	inter.fn["NOT"] = Function{Name: "NOT", RequiredArgs: 1, Call: builtIn_Not}

	inter.fn["FLAG_ENABLED"] = Function{Name: "FLAG_ENABLED", RequiredArgs: 1, Call: inter.builtIn_FlagEnabled}
	inter.fn["FLAG_DISABLED"] = Function{Name: "FLAG_DISABLED", RequiredArgs: 1, Call: inter.builtIn_FlagDisabled}
	inter.fn["FLAG_IS"] = Function{Name: "FLAG_IS", RequiredArgs: 2, Call: inter.builtIn_FlagIs}
	inter.fn["FLAG_LESS_THAN"] = Function{Name: "FLAG_LESS_THAN", RequiredArgs: 2, Call: inter.builtIn_FlagLessThan}
	inter.fn["FLAG_GREATER_THAN"] = Function{Name: "FLAG_GREATER_THAN", RequiredArgs: 2, Call: inter.builtIn_FlagGreaterThan}
	inter.fn["IN_INVEN"] = Function{Name: "IN_INVEN", RequiredArgs: 1, Call: inter.builtIn_InInven}
	inter.fn["ENABLE"] = Function{Name: "ENABLE", RequiredArgs: 1, Call: inter.builtIn_Enable}
	inter.fn["DISABLE"] = Function{Name: "DISABLE", RequiredArgs: 1, Call: inter.builtIn_Disable}
	inter.fn["INC"] = Function{Name: "INC", RequiredArgs: 1, OptionalArgs: 1, Call: inter.builtIn_Inc}
	inter.fn["DEC"] = Function{Name: "DEC", RequiredArgs: 1, OptionalArgs: 1, Call: inter.builtIn_Dec}
	inter.fn["SET"] = Function{Name: "SET", RequiredArgs: 2, Call: inter.builtIn_Set}
	inter.fn["MOVE"] = Function{Name: "MOVE", RequiredArgs: 2, Call: inter.builtIn_Move}
	inter.fn["OUTPUT"] = Function{Name: "OUTPUT", RequiredArgs: 1, Call: inter.builtIn_Output}

	return inter
}

// AddFlag adds a flag to the interpreter's flag store, with an initial value.
func (inter Interpreter) AddFlag(label string, val string) error {
	label = strings.ToUpper(label)

	if len(label) < 1 {
		return fmt.Errorf("label %q does not match pattern /[A-Z0-9_]+/", label)
	}

	for _, ch := range label {
		if !('A' <= ch && ch <= 'Z') && !('0' <= ch && ch <= '9') && ch != '_' {
			return fmt.Errorf("label %q does not match pattern /[A-Z0-9_]+/", label)
		}
	}

	inter.flags[label] = &Flag{
		Name:  label,
		Value: parseUntypedValString(val),
	}

	return nil
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

type astNode struct {
	root     *astNode
	children []*astNode
	sym      symbol
	t        nodeType

	forceType ValueType
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

type evalState int

const (
	evalDefault evalState = iota
	evalDollar
)

func (inter Interpreter) Eval(s string) (string, error) {
	ast, _, err := buildAST(s, nil)
	if err != nil {
		return "", fmt.Errorf("syntax error: %w", err)
	}

	vals, err := inter.evalExpr(ast)
	if err != nil {
		return "", fmt.Errorf("syntax error: %w", err)
	}

	strVals := make([]string, len(vals))
	for i := range vals {
		strVals[i] = vals[i].Str()
	}

	return strings.Join(strVals, " "), nil
}

// ParseText interprets the text in the abstract syntax tree and evaluates it.
func (inter Interpreter) evalExpr(ast *astNode) ([]Value, error) {
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

				args, err := inter.evalExpr(argsNode)
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
					v = NewStr(flag.Str())
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
	var v Value
	if srcUpper == "TRUE" || srcUpper == "YES" || srcUpper == "ON" {
		v = NewBool(true)
	}
	return v
}

// LexText lexes the text. Returns the AST, whether exiting on right paren, how
// many bytes were consumed, and whether an error was encountered.
func buildAST(s string, parent *astNode) (*astNode, int, error) {
	node := &astNode{children: make([]*astNode, 0)}
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

		switch mode {
		case lexIdent:
			if ('A' <= ch && ch <= 'Z') || ('a' <= ch && ch <= 'z') || ('0' <= ch && ch <= '9') || ch == '_' {
				buildingText += string(ch)
			} else {
				idNode := &astNode{
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
			} else if !escaping && ch == '|' {
				symNode := &astNode{
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
					textNode := &astNode{root: node.root, sym: symbol{sType: symbolValue, source: buildingText}}
					node.children = append(node.children, textNode)
					buildingText = ""
				}

				dNode := &astNode{
					root: node.root,
					sym:  symbol{sType: symbolDollar, source: "$"},
				}
				node.children = append(node.children, dNode)
				mode = lexIdent
			} else if !escaping && ch == '(' {
				if buildingText != "" {
					textNode := &astNode{root: node.root, sym: symbol{sType: symbolValue, source: buildingText}}
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

				node.children = append(node.children[:len(node.children)-1], subNode)
				i += consumed
			} else if !escaping && ch == ',' {
				if buildingText != "" {
					textNode := &astNode{root: node.root, sym: symbol{sType: symbolValue, source: buildingText}}
					node.children = append(node.children, textNode)
					buildingText = ""
				}

				// comma breaks up values but doesn't actually need to be its own node
			} else if !escaping && ch == '|' {
				if buildingText != "" {
					textNode := &astNode{root: node.root, sym: symbol{sType: symbolValue, source: buildingText}}
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
					textNode := &astNode{root: node.root, sym: symbol{sType: symbolValue, source: buildingText}}
					node.children = append(node.children, textNode)
					buildingText = ""
				}

				// don't add it bc parent will
				return node, sBytes[i], nil
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
			textNode := &astNode{root: node.root, sym: symbol{sType: symbolValue, source: buildingText}}
			node.children = append(node.children, textNode)
			buildingText = ""
		}
	}
	if mode == lexStr {
		return nil, 0, fmt.Errorf("unexpected end of expression (unmatched string start \"|\")")
	}

	// okay now go through and update
	// - make the values not have double quotes but force to str type

	return node, len(s), nil
}
