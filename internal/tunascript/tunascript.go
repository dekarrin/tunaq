package tunascript

import (
	"fmt"
	"sort"
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

// ExpansionTree is a parsed (but not interpreted) block of text containing
// both tunascript expansion-legal expressions and regular text. The zero-value
// of a ParsedExpansion is not suitable for use and they should only be created
// by calls to ParseExpansion.
type ExpansionTree struct {
	nodes []expTreeNode
}

type expTreeNode struct {
	t   *string             // if not nil its a text node
	ast *AbstractSyntaxTree // if not nil its a code node
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
	tree := ExpansionTree{
		nodes: make([]expTreeNode, 0),
	}

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
