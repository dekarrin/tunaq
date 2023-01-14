package tunascript

import (
	"fmt"
	"sort"
	"strings"
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

// ExpansionAnalysis is a lexed (and somewhat parsed) block of text containing
// both tunascript expansion-legal expressions and regular text. The zero-value
// of a ParsedExpansion is not suitable for use and they should only be created
// by calls to AnalyzeExpansion.
type ExpansionAST struct {
	nodes []expTreeNode
}

type expTreeNode struct {
	// can be a text node or a conditional node. Conditional nodes hold a series
	// of ifs
	text   *string        // if not nil its a text node
	branch *expBranchNode // if not nil its a branch node
	flag   *string        // if not nil its a flag node
}

type expBranchNode struct {
	ifNode expCondNode
	/*elseIfNodes []expCondNode
	elseNode    *ExpansionAST*/
}

type expCondNode struct {
	cond    *AbstractSyntaxTree
	content *ExpansionAST
}

func (inter Interpreter) ExpandTree(ast *ExpansionAST) (string, error) {
	if ast == nil {
		return "", fmt.Errorf("nil ast")
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

	sb := strings.Builder{}

	for i := range ast.nodes {
		n := ast.nodes[i]

		if n.flag != nil {
			flagVal := expandFlag(*n.flag)
			sb.WriteString(flagVal)
		} else if n.text != nil {
			sb.WriteString(*n.text)
		} else if n.branch != nil {
			cond := n.branch.ifNode.cond
			contentExpansionAST := n.branch.ifNode.content

			conditionalValue, err := inter.evalExpr(cond, true)
			if err != nil {
				return "", fmt.Errorf("syntax error: %v", err)
			}
			if len(conditionalValue) != 1 {
				return "", fmt.Errorf("incorrect number of arguments to $IF; must be exactly 1")
			}

			if conditionalValue[0].Bool() {
				expandedContent, err := inter.ExpandTree(contentExpansionAST)
				if err != nil {
					return "", err
				}

				sb.WriteString(expandedContent)
			}
		}
	}

	return sb.String(), nil
}

// ParseExpansion applies expansion analysis to the given text.
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
func (inter Interpreter) ParseExpansion(s string) (*ExpansionAST, error) {
	sRunes := []rune{}
	sBytes := []int{}
	for b, ch := range s {
		sRunes = append(sRunes, ch)
		sBytes = append(sBytes, b)
	}

	ast, _, err := inter.parseExpansion(sRunes, sBytes, true)
	return ast, err
}

func (inter Interpreter) parseExpansion(sRunes []rune, sBytes []int, topLevel bool) (*ExpansionAST, int, error) {
	tree := &ExpansionAST{
		nodes: make([]expTreeNode, 0),
	}

	const (
		modeText = iota
		modeIdent
	)

	var ident strings.Builder

	var escaping bool

	curText := strings.Builder{}
	mode := modeText

	for i := 0; i < len(sRunes); i++ {
		ch := sRunes[i]
		switch mode {
		case modeText:
			if !escaping && ch == '\\' {
				escaping = true
			} else if !escaping && ch == '$' {
				if curText.Len() > 0 {
					lastText := curText.String()
					tree.nodes = append(tree.nodes, expTreeNode{
						text: &lastText,
					})
					curText.Reset()
				}

				ident.WriteRune('$')
				mode = modeText
			} else {
				curText.WriteRune(ch)
			}
		case modeIdent:
			if ('A' <= ch && ch <= 'Z') || ('a' <= ch && ch <= 'z') || ('0' <= ch && ch <= '9') || ch == '_' {
				ident.WriteRune(ch)
			} else if ch == '(' {
				fnName := ident.String()

				if fnName == "$IF" {
					// we've encountered an IF block, recurse.
					parenMatch, tsExpr, err := indexOfMatchingParen(sRunes[i:])
					if err != nil {
						return tree, 0, fmt.Errorf("at char %d: %w", i, err)
					}
					exprLen := parenMatch - 1

					if exprLen < 1 {
						return tree, 0, fmt.Errorf("at char %d: args cannot be empty", i)
					}

					branch := expBranchNode{
						ifNode: expCondNode{
							cond: tsExpr,
						},
					}

					i += parenMatch

					if i+1 >= len(sRunes) {
						return nil, 0, fmt.Errorf("unexpected end of text (unmatched $IF)")
					}

					ast, consumed, err := inter.parseExpansion(sRunes[i+1:], sBytes[i+1:], false)
					if err != nil {
						return nil, 0, err
					}

					branch.ifNode.content = ast

					tree.nodes = append(tree.nodes, expTreeNode{
						branch: &branch,
					})

					i += consumed

					ident.Reset()
					mode = modeText
				} else if fnName == "$ENDIF" {
					parenMatch, tsExpr, err := indexOfMatchingParen(sRunes[i:])
					if err != nil {
						return nil, 0, fmt.Errorf("at char %d: %w", i, err)
					}
					exprLen := parenMatch - 1

					if exprLen != 0 {
						return nil, 0, fmt.Errorf("at char %d: $ENDIF() takes zero arguments, received %d", i, len(tsExpr.children))
					}
					i += parenMatch

					if topLevel {
						return nil, 0, fmt.Errorf("unexpected end of text (unmatched $ENDIF)")
					}

					return tree, i, nil
				} else {
					return nil, 0, fmt.Errorf("at char %d: %s() is not a text function; only $IF() or $ENDIF() are allowed", i, ident.String())
				}
			} else {
				flagName := ident.String()

				tree.nodes = append(tree.nodes, expTreeNode{
					flag: &flagName,
				})

				mode = modeText
				i-- // reparse 'normally'
			}
		default:
			// should never happen
			return nil, 0, fmt.Errorf("unknown parser mode: %v", mode)
		}
	}

	if !topLevel {
		return nil, 0, fmt.Errorf("unexpected end of text (unmatched $IF)")
	}

	if curText.Len() > 0 {
		lastText := curText.String()
		tree.nodes = append(tree.nodes, expTreeNode{
			text: &lastText,
		})
		curText.Reset()
	}

	if ident.Len() > 0 {
		flagName := ident.String()

		tree.nodes = append(tree.nodes, expTreeNode{
			flag: &flagName,
		})
	}

	return tree, len(sRunes), nil
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
func (inter Interpreter) Expand(s string) (string, error) {
	expAST, err := inter.ParseExpansion(s)
	if err != nil {
		return "", err
	}

	expanded, err := inter.ExpandTree(expAST)
	if err != nil {
		return "", err
	}

	return expanded, nil
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
