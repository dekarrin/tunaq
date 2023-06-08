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

	fn map[string]funcInfo
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

// Eval parses the given string as FISHIMath code and applies it immediately.
// Returns a non-nil error if there is a syntax error in the text. The value of
// the last valid statement will be in interp.LastResult after Eval returns.
func (interp *Interpreter) Eval(code string) error {
	ast, err := interp.Parse(code)
	if err != nil {
		return err
	}

	interp.Exec(ast)
	return nil
}

// EvalReader parses the contents of a Reader as FISHIMath code and applies it
// immediately. Returns a non-nil error if there is a syntax error in the text
// or if there is an error reading bytes from the Reader. The value of the last
// valid statement will be in interp.LastResult after EvalReader returns.
func (interp *Interpreter) EvalReader(r io.Reader) error {
	ast, err := interp.ParseReader(r)
	if err != nil {
		return err
	}

	interp.Exec(ast)
	return nil
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

	if interp.fn == nil {
		interp.initFuncs()
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
// that value. Make sure initFuncs is called at least once before calling
// execNode.
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

func (interp *Interpreter) execFuncNode(n syntax.FuncNode) syntax.Value {
	// existence and arity should already be validated by the translation layer
	// of the frontend, so no need to check here.

	var args []syntax.Value

	for i := range n.Args {
		argVal := interp.execNode(n.Args[i])
		args = append(args, argVal)
	}

	result := interp.fn[n.Func].call(args)

	return result
}

func (interp *Interpreter) execGroupNode(n syntax.GroupNode) syntax.Value {
	return interp.execNode(n.Expr)
}

func (interp *Interpreter) execAssignmentNode(n syntax.AssignmentNode) syntax.Value {
	var newVal syntax.Value
	oldVal := interp.Flags[n.Flag]

	switch n.Op {
	case syntax.OpAssignDecrement:
		newVal = oldVal.Subtract(syntax.ValueOf(1))
	case syntax.OpAssignDecrementBy:
		amt := interp.execNode(n.Value)
		newVal = oldVal.Subtract(amt)
	case syntax.OpAssignIncrement:
		newVal = oldVal.Add(syntax.ValueOf(1))
	case syntax.OpAssignIncrementBy:
		amt := interp.execNode(n.Value)
		newVal = oldVal.Add(amt)
	case syntax.OpAssignSet:
		newVal = interp.execNode(n.Value)
	default:
		panic(fmt.Sprintf("unrecognized AssignmentOperation: %v", n.Op))
	}

	interp.Flags[n.Flag] = newVal
	return newVal
}

func (interp *Interpreter) execBinaryOpNode(n syntax.BinaryOpNode) syntax.Value {
	left := interp.execNode(n.Left)
	right := interp.execNode(n.Right)

	switch n.Op {
	case syntax.OpBinaryAdd:
		return left.Add(right)
	case syntax.OpBinaryDivide:
		return left.Divide(right)
	case syntax.OpBinaryEqual:
		return left.EqualTo(right)
	case syntax.OpBinaryGreaterThan:
		return left.GreaterThan(right)
	case syntax.OpBinaryGreaterThanEqual:
		return left.GreaterThanEqualTo(right)
	case syntax.OpBinaryLessThan:
		return left.LessThan(right)
	case syntax.OpBinaryLessThanEqual:
		return left.LessThanEqualTo(right)
	case syntax.OpBinaryLogicalAnd:
		return left.And(right)
	case syntax.OpBinaryLogicalOr:
		return left.Or(right)
	case syntax.OpBinaryMultiply:
		return left.Multiply(right)
	case syntax.OpBinaryNotEqual:
		return left.EqualTo(right).Not()
	case syntax.OpBinarySubtract:
		return left.Subtract(right)
	default:
		panic(fmt.Sprintf("unrecognized BinaryOperation: %v", n.Op))
	}
}

func (interp *Interpreter) execUnaryOpNode(n syntax.UnaryOpNode) syntax.Value {
	operand := interp.execNode(n.Operand)

	switch n.Op {
	case syntax.OpUnaryLogicalNot:
		return operand.Not()
	case syntax.OpUnaryNegate:
		return operand.Negate()
	default:
		panic(fmt.Sprintf("unrecognized UnaryOperation: %v", n.Op))
	}
}

func (interp *Interpreter) execFlagNode(n syntax.FlagNode) syntax.Value {
	return interp.Flags[n.Flag]
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
