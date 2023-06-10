// Package tunascript is an interpretation engine for reading tunascript code
// and applying it to a running world.
package tunascript

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/dekarrin/ictiobus"
	"github.com/dekarrin/ictiobus/lex"
	"github.com/dekarrin/ictiobus/syntaxerr"
	"github.com/dekarrin/tunaq/tunascript/expfe"
	"github.com/dekarrin/tunaq/tunascript/fe"
	"github.com/dekarrin/tunaq/tunascript/syntax"
)

// TODO: these should not be aliased like this, aliasing is not intended for
// use outside of conversion. (Yeah, 8ut we're kind of doing exactly that!) true.
type AST = syntax.AST
type ExpansionAST = syntax.Template
type Value = syntax.Value

// TranslateOperators converts the operators in the given TunaScript string to
// function calls.
func TranslateOperators(s string) (string, error) {
	var tqi Interpreter
	ast, err := tqi.Parse(s)
	if err != nil {
		return "", err
	}

	// now go and convert all operations into their equivalents
	for i := range ast.Nodes {
		ast.Nodes[i] = convertNodeToBuiltIn(ast.Nodes[i])
	}

	return ast.Tunascript(), nil
}

func convertNodeToBuiltIn(n syntax.ASTNode) syntax.ASTNode {
	switch n.Type() {
	case syntax.ASTAssignment:
		an := n.AsAssignmentNode()

		args := []syntax.ASTNode{
			syntax.LiteralNode{Value: syntax.ValueOf(an.Flag)},
		}
		if an.Value != nil {
			args = append(args, convertNodeToBuiltIn(an.Value))
		}

		return syntax.FuncNode{Func: an.Op.BuiltInFunc(), Args: args}
	case syntax.ASTBinaryOp:
		bn := n.AsBinaryOpNode()
		return syntax.FuncNode{Func: bn.Op.BuiltInFunc(), Args: []syntax.ASTNode{
			convertNodeToBuiltIn(bn.Left), convertNodeToBuiltIn(bn.Right),
		}}
	case syntax.ASTFlag:
		return n
	case syntax.ASTFunc:
		fn := n.AsFuncNode()
		newF := syntax.FuncNode{
			Func: fn.Func,
			Args: make([]syntax.ASTNode, len(fn.Args)),
		}

		for i := range fn.Args {
			newF.Args[i] = convertNodeToBuiltIn(fn.Args[i])
		}
		return newF
	case syntax.ASTGroup:
		return syntax.GroupNode{
			Expr: convertNodeToBuiltIn(n.AsGroupNode().Expr),
		}
	case syntax.ASTLiteral:
		return n
	case syntax.ASTUnaryOp:
		un := n.AsUnaryOpNode()
		return syntax.FuncNode{Func: un.Op.BuiltInFunc(), Args: []syntax.ASTNode{
			convertNodeToBuiltIn(un.Operand),
		}}
	default:
		panic(fmt.Sprintf("unknown AST node type: %v", n.Type()))
	}
}

func ParseValue(s string) Value {
	srcUpper := strings.ToUpper(s)
	if srcUpper == "TRUE" || srcUpper == "YES" || srcUpper == "ON" {
		return syntax.ValueOf(true)
	} else if srcUpper == "FALSE" || srcUpper == "NO" || srcUpper == "OFF" {
		return syntax.ValueOf(false)
	}

	if strings.Contains(s, ".") {
		fVal, err := strconv.ParseFloat(s, 64)
		if err == nil {
			return syntax.ValueOf(fVal)
		}
	}

	iVal, err := strconv.Atoi(s)
	if err == nil {
		return syntax.ValueOf(iVal)
	}

	return syntax.ValueOf(s)
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

// Interpreter reads tunascript code and applies it to a target. The zero-value
// is ready for use, but Target needs to be assigned to before calling Exec or
// Eval.
type Interpreter struct {
	InitialFlags map[string]Value

	// Target is where world mutations are applied to. Must be set before
	// calling Exec or Eval.
	Target WorldInterface

	// LastResult is the result of the last statement that was successfully
	// executed.
	LastResult Value

	// File is the name of the file currently being executed by the engine. This
	// is used in error reporting and is optional to set.
	File string

	flags map[string]Value
	fn    map[string]funcInfo
	fe    ictiobus.Frontend[AST]
	exp   ictiobus.Frontend[ExpansionAST]
}

// Init initializes the interpreter environment. All defined symbols
// and variables are removed and reset to those defined in InitialFlags, and
// LastResult is reset. interp.File is not modified.
func (interp *Interpreter) Init() {
	interp.flags = map[string]Value{}
	interp.LastResult = Value{}

	if interp.InitialFlags != nil {
		for k := range interp.InitialFlags {
			interp.flags[k] = interp.InitialFlags[k]
		}
	}
}

// Expand parses the given string as a TunaQuest template and expands it into
// the full contents immediately. Returns a non-nil error if there is a syntax
// error in the template, in TunaScript within template flow-control statements,
// or if a non-pure function from TunaScript is within the template.
func (interp *Interpreter) Expand(tmpl string) (string, error) {
	ast, err := interp.ParseTemplate(tmpl)
	if err != nil {
		return "", err
	}

	return interp.ExecTemplate(ast), nil
}

// Expand parses the given contents of a Reader as a TunaQuest template and
// expands it into the full contents immediately. Returns a non-nil error if
// there is a syntax error in the template, in TunaScript within template
// flow-control statements, or if a non-pure function from TunaScript is within
// the template.
func (interp *Interpreter) ExpandReader(r io.Reader) (string, error) {
	ast, err := interp.ParseTemplateReader(r)
	if err != nil {
		return "", err
	}

	return interp.ExecTemplate(ast), nil
}

// Eval parses the given string as TunaScript code and applies it immediately.
// Returns a non-nil error if there is a syntax error in the text. The value of
// the last valid statement will be in interp.LastResult after Eval returns.
func (interp *Interpreter) Eval(code string) (Value, error) {
	ast, err := interp.Parse(code)
	if err != nil {
		return Value{}, err
	}

	return interp.Exec(ast), nil
}

// EvalReader parses the contents of a Reader as TunaScript code and applies it
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

// ExecTemplate executes the template represented by the given ExpansionAST and
// returns the result of expanding it. Additionally, interp.LastResult is set to
// the last pure TunaScript result executed within the template.
//
// This function does not require Target to have been set on the interpreter.
func (interp *Interpreter) ExecTemplate(ast ExpansionAST) string {
	if interp.flags == nil {
		interp.flags = map[string]Value{}
	}

	if interp.fn == nil {
		interp.initFuncs()
	}

	if len(ast.Blocks) < 1 {
		return ""
	}

	var sb strings.Builder
	for i := range ast.Blocks {
		s := interp.templateExecNode(ast.Blocks[i])
		sb.WriteString(s)
	}

	return sb.String()
}

// Exec executes all statements contained in the AST and returns the result of
// the last statement. Additionally, interp.LastResult is set to that result. If
// no statements are in the AST, the returned TSValue will be the zero value and
// interp.LastResult will not be altered.
//
// This function requires Target to have been set on the interpreter. If it is
// not set, this function will panic.
func (interp *Interpreter) Exec(ast AST) Value {
	if interp.Target == nil {
		panic("Exec() called on Interpreter with nil Target")
	}

	if interp.flags == nil {
		interp.flags = map[string]Value{}
	}

	if interp.fn == nil {
		interp.initFuncs()
	}

	if len(ast.Nodes) < 1 {
		return Value{}
	}

	var lastResult Value
	for i := range ast.Nodes {
		stmt := ast.Nodes[i]
		lastResult = interp.execNode(stmt)
	}

	return lastResult
}

// ParseTemplate parses (but does not execute) a block of expandable TunaScript
// templated text. Any TunaScript within template flow control blocks is also
// parsed and checked for proper call semantics (i.e. they are checked to make
// sure only query functions are used, and not ones with side effects).
func (interp *Interpreter) ParseTemplate(code string) (ast ExpansionAST, err error) {
	interp.initFrontend()

	if interp.fn == nil {
		interp.initFuncs()
	}

	ast, _, err = interp.exp.AnalyzeString(code)
	if err != nil {

		// wrap syntax errors so user of the Interpreter doesn't have to check
		// for a special syntax error just to get the detailed syntax err info
		if synErr, ok := err.(*syntaxerr.Error); ok {
			return ast, fmt.Errorf("%s", synErr.MessageForFile(interp.File))
		}
	}

	// okay, we got the template AST, now go through and recursively translate
	// the RawCond of ExpCondNodes to TunaScript ASTs.
	for i := range ast.Blocks {
		newNode, err := interp.translateTemplateTunascript(ast.Blocks[i])
		if err != nil {
			return ast, err
		}
		ast.Blocks[i] = newNode
	}

	return ast, nil
}

// ParseTemplateReader parses (but does not execute) a block of expandable
// TunaScript templated text from the given reader. Any TunaScript within
// template flow control blocks is also parsed and checked for proper call
// semantics (i.e. they are checked to make sure only query functions are used,
// and not ones with side effects).
func (interp *Interpreter) ParseTemplateReader(r io.Reader) (ast ExpansionAST, err error) {
	interp.initFrontend()

	if interp.fn == nil {
		interp.initFuncs()
	}

	ast, _, err = interp.exp.Analyze(r)
	if err != nil {

		// wrap syntax errors so user of the Interpreter doesn't have to check
		// for a special syntax error just to get the detailed syntax err info
		if synErr, ok := err.(*syntaxerr.Error); ok {
			return ast, fmt.Errorf("%s", synErr.MessageForFile(interp.File))
		}
	}

	// okay, we got the template AST, now go through and recursively translate
	// the RawCond of ExpCondNodes to TunaScript ASTs.
	for i := range ast.Blocks {
		newNode, err := interp.translateTemplateTunascript(ast.Blocks[i])
		if err != nil {
			return ast, err
		}
		ast.Blocks[i] = newNode
	}

	return ast, nil
}

// Parse parses (but does not execute) TunaScript code. The code is converted
// into an AST for further examination.
func (interp *Interpreter) Parse(code string) (ast AST, err error) {
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
func (interp *Interpreter) ParseReader(r io.Reader) (ast AST, err error) {
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

// AddFlag adds a flag to the interpreter's flag store, with an initial value.
func (interp *Interpreter) AddFlag(label string, val string) error {
	if interp.flags == nil {
		interp.flags = make(map[string]Value)
	}

	label = strings.ToUpper(label)

	if len(label) < 1 {
		return fmt.Errorf("label %q does not match pattern /[A-Z0-9_]+/", label)
	}

	for _, ch := range label {
		if !('A' <= ch && ch <= 'Z') && !('0' <= ch && ch <= '9') && ch != '_' {
			return fmt.Errorf("label %q does not match pattern /[A-Z0-9_]+/", label)
		}
	}

	interp.flags[label] = ParseValue(val)

	return nil
}

// ListFlags returns a list of all flags, sorted.
func (interp *Interpreter) ListFlags() []string {
	flags := make([]string, len(interp.flags))
	curFlagIdx := 0
	for k := range interp.flags {
		flags[curFlagIdx] = k
	}

	sort.Strings(flags)
	return flags
}

// GetFlag gets the give flag's value. If it is unset, it will be "".
func (interp *Interpreter) GetFlag(label string) string {
	if interp.flags == nil {
		interp.flags = make(map[string]Value)
	}

	label = strings.ToUpper(label)

	flag, ok := interp.flags[label]
	if !ok {
		return ""
	}
	return flag.String()
}

// templateExecNode executes a single template node and converts it to the
// completed text.
func (interp *Interpreter) templateExecNode(n syntax.Block) string {
	switch n.Type() {
	case syntax.TmplText:
		return n.AsTextNode().Text
	case syntax.TmplFlag:
		fl, ok := interp.flags[n.AsFlagNode().Flag]
		// in this case, we *do* care about it being defined, and cannot simply
		// use the value. if it's not defined, we explicitly want to return an
		// empty string. The zero value for Value will not do this; it
		// would be the default type (Int) converted to string ("0").
		if !ok {
			return ""
		}
		return fl.String()
	case syntax.TmplBranch:
		nb := n.AsBranchNode()

		ifResult := interp.Exec(nb.If.Cond)
		if ifResult.Bool() {
			var sb strings.Builder
			for i := range nb.If.Content {
				contentStr := interp.templateExecNode(nb.If.Content[i])
				sb.WriteString(contentStr)
			}
			return sb.String()
		}

		// are there any else-ifs? if so, check them now
		for _, elif := range nb.ElseIf {
			elifResult := interp.Exec(elif.Cond)
			if elifResult.Bool() {
				var sb strings.Builder
				for i := range elif.Content {
					contentStr := interp.templateExecNode(elif.Content[i])
					sb.WriteString(contentStr)
				}
				return sb.String()
			}
		}

		// finally, is there an else?
		if len(nb.Else) > 0 {
			var sb strings.Builder
			for i := range nb.Else {
				contentStr := interp.templateExecNode(nb.Else[i])
				sb.WriteString(contentStr)
			}
			return sb.String()
		}

		// we hit none of the branch conditions and there return none of its
		// content. return an empty string
		return ""
	case syntax.TmplCond:
		// should never happen
		panic("ExpCondNode passed to Interpreter.templateExecNode")
	default:
		panic(fmt.Sprintf("unknown ExpNode type: %v", n.Type()))
	}
}

// execNode executes the mathematical expression contained in the AST node and
// returns the result of the final one. This will also set interp.LastResult to
// that value. Make sure initFuncs is called at least once before calling
// execNode.
func (interp *Interpreter) execNode(n syntax.ASTNode) (result Value) {
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

func (interp *Interpreter) execFuncNode(n syntax.FuncNode) Value {
	// existence and arity should already be validated by the translation layer
	// of the frontend, so no need to check here.

	var args []Value

	for i := range n.Args {
		argVal := interp.execNode(n.Args[i])
		args = append(args, argVal)
	}

	result := interp.fn[n.Func].call(args)

	return result
}

func (interp *Interpreter) execGroupNode(n syntax.GroupNode) Value {
	return interp.execNode(n.Expr)
}

func (interp *Interpreter) execAssignmentNode(n syntax.AssignmentNode) Value {
	var newVal Value
	oldVal := interp.flags[n.Flag]

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

	interp.flags[n.Flag] = newVal
	return newVal
}

func (interp *Interpreter) execBinaryOpNode(n syntax.BinaryOpNode) Value {
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

func (interp *Interpreter) execUnaryOpNode(n syntax.UnaryOpNode) Value {
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

func (interp *Interpreter) execFlagNode(n syntax.FlagNode) Value {
	return interp.flags[n.Flag]
}

func (interp *Interpreter) execLiteralNode(n syntax.LiteralNode) Value {
	return n.Value
}

// initializes the frontends in members fe and expfe so that they can be used.
// If frontends are already initialized, this function does nothing. interp.fe
// and interp.exp can be safely used after calling this function.
func (interp *Interpreter) initFrontend() {
	// if IR attribute is blank, fe is by-extension not yet set, because
	// Ictiobus-generated frontends will never have an empty IRAttribute.
	if interp.fe.IRAttribute == "" {
		interp.fe = fe.Frontend(syntax.HooksTable, nil)
	}
	if interp.exp.IRAttribute == "" {
		interp.exp = expfe.Frontend(syntax.ExpHooksTable, nil)
	}
}

func (interp *Interpreter) translateTemplateTunascript(n syntax.Block) (syntax.Block, error) {
	switch n.Type() {
	case syntax.TmplFlag:
		return n, nil
	case syntax.TmplText:
		return n, nil
	case syntax.TmplBranch:
		nb := n.AsBranchNode()
		newIf, err := interp.translateTemplateTunascript(nb.If)
		if err != nil {
			return n, err
		}

		newBranch := syntax.ExpBranchNode{
			If:     newIf.AsCondNode(),
			ElseIf: make([]syntax.ExpCondNode, len(nb.ElseIf)),
			Else:   nb.Else,
		}
		for i := range nb.ElseIf {
			newElseIf, err := interp.translateTemplateTunascript(nb.ElseIf[i])
			if err != nil {
				return n, err
			}
			newBranch.ElseIf[i] = newElseIf.AsCondNode()
		}
		return newBranch, nil
	case syntax.TmplCond:
		nc := n.AsCondNode()

		// feed the text into the tunascript frontend and validate only query
		// funcs were called.
		ast, err := interp.Parse(nc.RawCond)
		if err != nil {
			// provide some context
			synErr, ok := err.(*syntaxerr.Error)
			if !ok {
				return n, err
			}

			curErr := lex.NewSyntaxErrorFromToken("syntax error encountered while parsing TunaScript in template", nc.Source)
			contextualizedErr := fmt.Errorf("%s:\n%s", curErr.MessageForFile(interp.File), synErr.FullMessage())

			return n, contextualizedErr
		}

		// no errors! great, double-check that all the TS is legal
		queryOnly, badNode := interp.validateQueryOnly(ast)
		if !queryOnly {
			// Goodness It Appears The User Is Attempting To Perform Mutations In A Template. This Is Disallowed.
			// 4ND TH1S S1N SH4LL B3 D34LT W1TH SW1FTLY BY 1SSU1NG TH3 WORST OF PUN1SHM3NTS >:]
			// No. But It Will Be Dealt With By Returning An Error.
			// CLOS3 3NOUGH.
			var tsSynErr *syntaxerr.Error
			if badNode.Type() == syntax.ASTFunc {
				fNode := badNode.AsFuncNode()
				tsSynErr = lex.NewSyntaxErrorFromToken(fmt.Sprintf("$%s() changes things, so it can't be used in TQ templates", fNode.Func), badNode.Source())
			} else if badNode.Type() == syntax.ASTAssignment {
				aNode := badNode.AsAssignmentNode()
				tsSynErr = lex.NewSyntaxErrorFromToken(fmt.Sprintf("%s changes things, so it can't be used in TQ templates", aNode.Op.Symbol()), badNode.Source())
			} else {
				panic("badNode is not assignment or func node")
			}
			curErr := lex.NewSyntaxErrorFromToken("syntax error encountered while parsing TunaScript in template", nc.Source)

			contextualizedErr := fmt.Errorf("%s:\n%s", curErr.MessageForFile(interp.File), tsSynErr.FullMessage())

			return n, contextualizedErr
		}

		// otherwise, build the new node and it's good to go
		newCondNode := syntax.ExpCondNode{
			RawCond: nc.RawCond,
			Cond:    ast,
			Content: nc.Content,
			Source:  nc.Source,
		}
		return newCondNode, nil
	default:
		panic("unknown ExpNode type")
	}
}

func (interp *Interpreter) validateQueryOnly(ast AST) (queryOnly bool, badNode syntax.ASTNode) {
	for i := range ast.Nodes {
		bn := interp.findFirstWithSideEffects(ast.Nodes[i])
		if bn != nil {
			return false, bn
		}
	}
	return true, syntax.FuncNode{}
}

// Will only return non-nil if it finds a FuncNode with a non-compliant func
// node in a left-first, depth-first visit through all nodes.
func (interp *Interpreter) findFirstWithSideEffects(n syntax.ASTNode) syntax.ASTNode {
	switch n.Type() {
	case syntax.ASTAssignment:
		// this has side-effects
		return n
	case syntax.ASTBinaryOp:
		leftBad := interp.findFirstWithSideEffects(n.AsBinaryOpNode().Left)
		if leftBad != nil {
			return leftBad
		}
		rightBad := interp.findFirstWithSideEffects(n.AsBinaryOpNode().Right)
		if rightBad != nil {
			return rightBad
		}
		return nil
	case syntax.ASTFlag:
		return nil
	case syntax.ASTGroup:
		return interp.findFirstWithSideEffects(n.AsGroupNode().Expr)
	case syntax.ASTLiteral:
		return nil
	case syntax.ASTUnaryOp:
		return interp.findFirstWithSideEffects(n.AsUnaryOpNode().Operand)
	case syntax.ASTFunc:
		// now this is the good stuff, the actual validation
		fnode := n.AsFuncNode()
		info := interp.fn[fnode.Func]
		if info.def.SideEffects {
			return &fnode
		}

		// ...but if it didnt have side effects, be shore to check its args 38O
		for i := range fnode.Args {
			badArg := interp.findFirstWithSideEffects(fnode.Args[i])
			if badArg != nil {
				return badArg
			}
		}
		return nil
	default:
		panic(fmt.Sprintf("unknown AST node type: %v", n.Type()))
	}
}
