// Package tunascript is an interpretation engine for reading tunascript code
// and applying it to a running world.
package tunascript

import (
	"fmt"
	"io"

	"github.com/dekarrin/ictiobus"
	"github.com/dekarrin/ictiobus/syntaxerr"
	"github.com/dekarrin/tunaq/tunascript/fe"
	"github.com/dekarrin/tunaq/tunascript/syntax"
)

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

// Interpreter reads tunascript code and applies it to a target. The zero-value
// is ready for use, but Target needs to be assigned to before calling Exec or
// Eval.
type Interpreter struct {
	InitialFlags map[string]syntax.Value
	Flags        map[string]syntax.Value

	// Target is where world mutations are applied to. Must be set before
	// calling Exec or Eval.
	Target WorldInterface

	// LastResult is the result of the last statement that was successfully
	// executed.
	LastResult syntax.Value

	// File is the name of the file currently being executed by the engine. This
	// is used in error reporting and is optional to set.
	File string

	fe ictiobus.Frontend[syntax.AST]
}

// Init initializes the interpreter environment. All defined symbols
// and variables are removed and reset to those defined in InitialFlags, and
// LastResult is reset. interp.File is not modified.
func (interp *Interpreter) Init() {
	interp.Flags = map[string]syntax.Value{}
	interp.LastResult = syntax.Value{}

	if interp.InitialFlags != nil {
		for k := range interp.InitialFlags {
			interp.Flags[k] = interp.InitialFlags[k]
		}
	}
}

// Exec executes all statements contained in the AST and returns the result of
// the last statement. Additionally, interp.LastResult is set to that result. If
// no statements are in the AST, the returned TSValue will be the zero value and
// interp.LastResult will not be altered.
//
// This function requires Target to have been set on the interpreter. If it is
// not set, this function will panic.
func (interp *Interpreter) Exec(ast syntax.AST) syntax.Value {
	if interp.Target == nil {
		panic("Exec() called on Interpreter with nil Target")
	}

	if interp.Flags == nil {
		interp.Flags = map[string]syntax.Value{}
	}

	if len(ast.Nodes) < 1 {
		return syntax.Value{}
	}

	var lastResult syntax.Value
	for i := range ast.Nodes {
		stmt := ast.Nodes[i]
		lastResult = interp.execNode(stmt)
	}

	return lastResult
}

// Parse parses (but does not execute) TunaScript code. The code is converted
// into an AST for further examination.
func (interp *Interpreter) Parse(code string) (ast syntax.AST, err error) {
	interp.initFrontend()

	ast, _, err = interp.fe.AnalyzeString(code)
	if err != nil {

		// wrap syntax errors so user of the Interpreter doesn't have to check
		// for a special syntax error just to get the detailed syntax err info
		if synErr, ok := err.(*syntaxerr.Error); ok {
			return ast, fmt.Errorf("%s", synErr.MessageForFile(interp.File))
		}
	}

	return ast, err
}

// ParseReader parses (but does not execute) TunaScript code in the given
// reader. The entire contents of the Reader are read as TS code, which is
// returned as an AST for further examination.
func (interp *Interpreter) ParseReader(r io.Reader) (ast syntax.AST, err error) {
	interp.initFrontend()

	ast, _, err = interp.fe.Analyze(r)
	if err != nil {

		// wrap syntax errors so user of the Interpreter doesn't have to check
		// for a special syntax error just to get the detailed syntax err info
		if synErr, ok := err.(*syntaxerr.Error); ok {
			return ast, fmt.Errorf("%s", synErr.MessageForFile(interp.File))
		}
	}

	return ast, err
}

// execNode executes the mathematical expression contained in the AST node and
// returns the result of the final one. This will also set interp.LastResult to
// that value.
func (interp *Interpreter) execNode(n syntax.ASTNode) (result syntax.Value) {
	defer func() {
		interp.LastResult = result
	}()

	switch n.Type() {
	case syntax.ASTAssignment:
		result = interp.execAssignmentNode(n.AsAssignmentNode())
	case syntax.ASTBinaryOp:
		result = interp.execBinaryOpNode(n.AsBinaryOpNode())
	case syntax.ASTFlag:
		result = interp.execFlagNode(n.AsFlagNode())
	case syntax.ASTFunc:
		result = interp.execFuncNode(n.AsFuncNode())
	case syntax.ASTGroup:
		result = interp.execGroupNode(n.AsGroupNode())
	case syntax.ASTLiteral:
		result = interp.execLiteralNode(n.AsLiteralNode())
	case syntax.ASTUnaryOp:
		result = interp.execUnaryOpNode(n.AsUnaryOpNode())
	default:
		panic(fmt.Sprintf("unknown AST node type: %v", n.Type()))
	}

	return result
}

func (interp *Interpreter) execFlagNode(n syntax.FlagNode) syntax.Value {

}

func (interp *Interpreter) execLiteralNode(n syntax.LiteralNode) syntax.Value {
	return n.Value
}

// initializes the frontend in member fe so that it can be used. If frontend is
// already initialized, this function does nothing. interp.fe can be safely used
// after calling this function.
func (interp *Interpreter) initFrontend() {
	// if IR attribute is blank, fe is by-extension not yet set, because
	// Ictiobus-generated frontends will never have an empty IRAttribute.
	if interp.fe.IRAttribute == "" {
		interp.fe = fe.Frontend(syntax.HooksTable, nil)
	}
}
