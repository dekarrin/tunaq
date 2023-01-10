package tunascript

import (
	"fmt"
	"sort"
	"strconv"
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

	// SideEffects tells whether the function has side-effects. Certain contexts
	// such as within an $IF() may restrict the execution of side-effect
	// functions.
	SideEffects bool
}

// Flag is a variable in the engine.
type Flag struct {
	Name string
	Value
}

// Interpreter should not be used directly, use NewInterpreter.
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
		flags: make(map[string]*Flag),
		world: w,
	}

	inter.fn["ADD"] = Function{Name: "ADD", RequiredArgs: 2, Call: builtIn_Add}
	inter.fn["SUB"] = Function{Name: "SUB", RequiredArgs: 2, Call: builtIn_Sub}
	inter.fn["MULT"] = Function{Name: "MULT", RequiredArgs: 2, Call: builtIn_Mult}
	inter.fn["DIV"] = Function{Name: "DIV", RequiredArgs: 2, Call: builtIn_Div}
	inter.fn["OR"] = Function{Name: "OR", RequiredArgs: 2, Call: builtIn_Or}
	inter.fn["AND"] = Function{Name: "AND", RequiredArgs: 2, Call: builtIn_And}
	inter.fn["NOT"] = Function{Name: "NOT", RequiredArgs: 1, Call: builtIn_Not}

	inter.fn["FLAG_ENABLED"] = Function{Name: "FLAG_ENABLED", RequiredArgs: 1, Call: inter.builtIn_FlagEnabled}
	inter.fn["FLAG_DISABLED"] = Function{Name: "FLAG_DISABLED", RequiredArgs: 1, Call: inter.builtIn_FlagDisabled}
	inter.fn["FLAG_IS"] = Function{Name: "FLAG_IS", RequiredArgs: 2, Call: inter.builtIn_FlagIs}
	inter.fn["FLAG_LESS_THAN"] = Function{Name: "FLAG_LESS_THAN", RequiredArgs: 2, Call: inter.builtIn_FlagLessThan}
	inter.fn["FLAG_GREATER_THAN"] = Function{Name: "FLAG_GREATER_THAN", RequiredArgs: 2, Call: inter.builtIn_FlagGreaterThan}
	inter.fn["IN_INVEN"] = Function{Name: "IN_INVEN", RequiredArgs: 1, Call: inter.builtIn_InInven}
	inter.fn["ENABLE"] = Function{Name: "ENABLE", RequiredArgs: 1, Call: inter.builtIn_Enable, SideEffects: true}
	inter.fn["DISABLE"] = Function{Name: "DISABLE", RequiredArgs: 1, Call: inter.builtIn_Disable, SideEffects: true}
	inter.fn["TOGGLE"] = Function{Name: "DISABLE", RequiredArgs: 1, Call: inter.builtIn_Toggle, SideEffects: true}
	inter.fn["INC"] = Function{Name: "INC", RequiredArgs: 1, OptionalArgs: 1, Call: inter.builtIn_Inc, SideEffects: true}
	inter.fn["DEC"] = Function{Name: "DEC", RequiredArgs: 1, OptionalArgs: 1, Call: inter.builtIn_Dec, SideEffects: true}
	inter.fn["SET"] = Function{Name: "SET", RequiredArgs: 2, Call: inter.builtIn_Set, SideEffects: true}
	inter.fn["MOVE"] = Function{Name: "MOVE", RequiredArgs: 2, Call: inter.builtIn_Move, SideEffects: true}
	inter.fn["OUTPUT"] = Function{Name: "OUTPUT", RequiredArgs: 1, Call: inter.builtIn_Output, SideEffects: true}

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

// ListFlags returns a list of all flags, sorted.
func (inter Interpreter) ListFlags() []string {
	flags := make([]string, len(inter.flags))
	curFlagIdx := 0
	for k := range inter.flags {
		flags[curFlagIdx] = k
	}

	sort.Strings(flags)
	return flags
}

// GetFlag gets the give flag's value. If it is unset, it will be "".
func (inter Interpreter) GetFlag(label string) string {
	label = strings.ToUpper(label)

	flag, ok := inter.flags[label]
	if !ok {
		return ""
	}
	return flag.String()
}

// Expand applies expansion on the given text. Expansion will expand the
// following constructs:
//
//   - any flag reference with the $ will be expanded to its full value.
//   - any $IF() ... $ENDIF() block will be evaluated and included in the output
//     text only if the tunaquest expression inside the $IF evaluates to true.
//   - function calls are not allowed outside of the tunascript expression in an
//     $IF. If they are there, they will be interpreted as a variable expansion,
//     and if there is no value matching that one, it will be expanded to an
//     empty string. E.g. "$ADD()" in the body text would evaluate to value of
//     flag called "ADD" (probably ""), followed by literal parenthesis.
//   - bare dollar signs are evaluated as literal. This will only happen if they
//     are not immediately followed by identifier chars.
//   - literal $ signs can be included with a backslash. Thus the escape
//     backslash will work.
//   - literal backslashes can be included by escaping them.
func (inter Interpreter) ExpandText(s string) (string, error) {
	sRunes := []rune{}
	sBytes := []int{}
	for b, ch := range s {
		sRunes = append(sRunes, ch)
		sBytes = append(sBytes, b)
	}

	expandFlag := func(fullFlagToken string) string {
		flagName := fullFlagToken[1:]
		if flagName == "" {
			// bare dollarsign, go add it to expanded
			return "$"
		} else {
			// it is a full var name
			flagName = strings.ToUpper(flagName)
			flag, ok := inter.flags[flagName]
			if !ok {
				return ""
			}
			return flag.Value.String()
		}
	}

	var contentStack []*strings.Builder
	var ifResultStack []bool
	var expanded *strings.Builder
	var ident strings.Builder

	var inIdent bool
	var escaping bool

	expanded = &strings.Builder{}
	contentStack = append(contentStack, expanded)
	for i := 0; i < len(sRunes); i++ {
		ch := sRunes[i]
		if !inIdent {
			if !escaping && ch == '\\' {
				escaping = true
			} else if !escaping && ch == '$' {
				ident.WriteRune('$')
				inIdent = true
			} else {
				expanded.WriteRune(ch)
				escaping = false
			}
		} else {
			if ('A' <= ch && ch <= 'Z') || ('a' <= ch && ch <= 'z') || ('0' <= ch && ch <= '9') || ch == '_' {
				ident.WriteRune(ch)
			} else if ch == '(' {
				// this is a func call, and the only allowed directly in text is $IF() and $ENDIF().
				if ident.String() == "$IF" {
					parenMatch, tsExpr, err := indexOfMatchingParen(s[sBytes[i]:])
					if err != nil {
						return "", fmt.Errorf("at char %d: %w", i, err)
					}
					exprLen := parenMatch - 1

					if exprLen < 1 {
						return "", fmt.Errorf("at char %d: args cannot be empty", i)
					}

					tsResult, err := inter.evalExpr(tsExpr, true)
					if err != nil {
						return "", fmt.Errorf("at char %d: %w", i, err)
					}
					if len(tsResult) > 1 {
						return "", fmt.Errorf("at char %d: $IF() takes one argument, received %d", i, len(tsResult))
					}
					ifResultStack = append(ifResultStack, tsResult[0].Bool())
					expanded = &strings.Builder{}
					contentStack = append(contentStack, expanded)
					ident.Reset()
					inIdent = false

					i += parenMatch
				} else if ident.String() == "$ENDIF" {
					parenMatch, tsExpr, err := indexOfMatchingParen(s[sBytes[i]:])
					if err != nil {
						return "", fmt.Errorf("at char %d: %w", i, err)
					}
					exprLen := parenMatch - 1

					if exprLen != 0 {
						return "", fmt.Errorf("at char %d: $ENDIF() takes zero arguments, received %d", i, len(tsExpr.children))
					}

					if len(ifResultStack) < 1 {
						return "", fmt.Errorf("at char %d: mismatched $ENDIF(); missing $IF() before it", i)
					}

					ifBlockContent := expanded.String()
					contentStack = contentStack[:len(contentStack)-1]
					expanded = contentStack[len(contentStack)-1]
					ifResult := ifResultStack[len(ifResultStack)-1]
					ifResultStack = ifResultStack[:len(ifResultStack)-1]
					if ifResult {
						// trim all spaces from both sides
						expanded.WriteString(strings.TrimSpace(ifBlockContent))
					} else {
						// remove the space prior to the if
						oldBeforeIf := expanded.String()
						// it appears iterating over the entire string is the
						// only way to do this.
						//
						// Hey, no8ody said we had to 8e efficient about it!
						//
						// efishient* 383

						finalCharByte := -1
						for b := range oldBeforeIf {
							finalCharByte = b
						}

						finalStr := []rune(oldBeforeIf[finalCharByte:])

						if unicode.IsSpace(finalStr[0]) {
							oldBeforeIf = oldBeforeIf[:finalCharByte]
						}

						expanded = &strings.Builder{}
						expanded.WriteString(oldBeforeIf)
						contentStack[len(contentStack)-1] = expanded
					}
					ident.Reset()
					inIdent = false

					i++ // for the extra paren
				} else {
					return "", fmt.Errorf("at char %d: %s() is not a text function; only $IF() or $ENDIF() are allowed", i, ident.String())
				}
			} else {
				varVal := expandFlag(ident.String())
				expanded.WriteString(varVal)
				ident = strings.Builder{}
				inIdent = false

				i-- // reparse 'normally'
			}
		}
	}

	// done looping, do we have any ident left over?
	// functions are look-ahaed analyzed as soon as opening paren is
	// encountered, so if still inIdent its deffo a flag
	if inIdent {
		varVal := expandFlag(ident.String())
		expanded.WriteString(varVal)
	}

	// check the stack, do we have mismatched ifs?
	if len(ifResultStack) > 0 {
		return "", fmt.Errorf("at end: mismatched $IF(); missing $ENDIF() after it")
	}

	return expanded.String(), nil
}

// Eval interprets the tunaquest expression in the given string. If there is an
// error in the code, it is returned as a non-nil error, otherwise the output of
// evaluating the expression is returned as a string.
func (inter Interpreter) Eval(s string) (string, error) {
	ast, _, err := buildAST(s, nil)
	if err != nil {
		return "", fmt.Errorf("syntax error: %w", err)
	}

	vals, err := inter.evalExpr(ast, false)
	if err != nil {
		return "", fmt.Errorf("syntax error: %w", err)
	}

	strVals := make([]string, len(vals))
	for i := range vals {
		strVals[i] = vals[i].Str()
	}

	return strings.Join(strVals, " "), nil
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
func indexOfMatchingParen(s string) (int, *astNode, error) {
	// without a parent node on a paren scan, buildAST will produce an error.
	dummyNode := &astNode{
		children: make([]*astNode, 0),
	}
	dummyNode.root = dummyNode

	sRunes := []rune(s)
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

	gotFirstByte := false
	nextByteIdx := -1
	for b := range s {
		if !gotFirstByte {
			gotFirstByte = true
		} else {
			nextByteIdx = b
			break
		}
	}

	if nextByteIdx == -1 {
		// should never happen
		return 0, nil, fmt.Errorf("byte analysis on string failed to produce a next-char byte")
	}

	exprNode, consumed, err := buildAST(s[nextByteIdx:], dummyNode)
	if err != nil {
		return 0, nil, err
	}
	exprNode.t = nodeRoot

	return consumed, exprNode, nil
}

// ParseText interprets the text in the abstract syntax tree and evaluates it.
func (inter Interpreter) evalExpr(ast *astNode, queryOnly bool) ([]Value, error) {
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
		//chS := string(ch)
		//fmt.Println(chS)

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
			} else if escaping && ch == 'n' {
				buildingText += "\n"
				escaping = false
			} else if escaping && ch == 't' {
				buildingText += "\t"
				escaping = false
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

				node.children = append(node.children, subNode)
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
			textNode := &astNode{root: node.root, sym: symbol{sType: symbolValue, source: buildingText}}
			node.children = append(node.children, textNode)
			buildingText = ""
		}
	} else if mode == lexIdent {
		if buildingText != "" {
			textNode := &astNode{root: node.root, sym: symbol{sType: symbolIdentifier, source: buildingText}}
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
