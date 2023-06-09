package syntax

import (
	"fmt"
	"strings"

	"github.com/dekarrin/ictiobus/lex"
)

type AST struct {
	Nodes []ASTNode
}

// Tunascript returns a string that contains tunascript code that if parsed
// would result in an equivalent ASTNode. It is not necessarily the source
// that produced this node as non-semantic elements are not included (such
// as extra whitespace not a part of an unquoted string).
//
// Each node is placed on its own line in the resulting string.
func (ast AST) Tunascript() string {
	var sb strings.Builder

	for i := range ast.Nodes {
		sb.WriteString(ast.Nodes[i].Tunascript())
		if i+1 < len(ast.Nodes) {
			sb.WriteRune('\n')
		}
	}

	return sb.String()
}

// String returns a prettified representation of the entire AST suitable for use
// in line-by-line comparisons of tree structure. Two ASTs are considered
// semantcally identical if they produce identical String() output. Each
// statement is shown on a new line.
func (ast AST) String() string {
	var sb strings.Builder

	sb.WriteString("AST\n")

	const stmtStart = " S: "
	for i := range ast.Nodes {
		stmtStr := spaceIndentNewlines(ast.Nodes[i].String(), len(stmtStart))

		sb.WriteString(stmtStart)
		sb.WriteString(stmtStr)
		if i+1 < len(ast.Nodes) {
			sb.WriteRune('\n')
		}
	}

	return sb.String()
}

// Equal checks if the AST is equal to another object. If the other object is
// not another AST, they are not considered equal. If the other object is, Equal
// returns whether the two trees, if invoked for any type of meaning, would
// return the same result. Note that this is distinct from whether they were
// created from the tunascript source code; to check that, use EqualSource().
func (ast AST) Equal(o any) bool {
	other, ok := o.(AST)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*AST)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if len(ast.Nodes) != len(other.Nodes) {
		return false
	}
	for i := range ast.Nodes {
		if !ast.Nodes[i].Equal(other.Nodes[i]) {
			return false
		}
	}

	return true
}

type NodeType int

const (
	ASTLiteral NodeType = iota
	ASTFunc
	ASTFlag
	ASTGroup
	ASTBinaryOp
	ASTUnaryOp
	ASTAssignment
)

type ASTNode interface {

	// Type returns the type of the ASTNode. This determines which of the As*()
	// functions may be called.
	Type() NodeType

	// Returns this node as a LiteralNode. Panics if Type() does not return
	// ASTLiteral.
	AsLiteralNode() LiteralNode

	// Returns this node as a FuncNode. Panics if Type() does not return
	// ASTFunc.
	AsFuncNode() FuncNode

	// Returns this node as a FlagNode. Panics if Type() does not return
	// ASTFlag.
	AsFlagNode() FlagNode

	// Returns this node as a GroupNode. Panics if Type() does not return
	// ASTGroup.
	AsGroupNode() GroupNode

	// Returns this node as a BinaryOpNode. Panics if Type() does not return
	// ASTBinaryOp.
	AsBinaryOpNode() BinaryOpNode

	// Returns this node as a UnaryOpNode. Panics if Type() does not return
	// ASTUnaryOp.
	AsUnaryOpNode() UnaryOpNode

	// Returns this node as an AssignmentNode. Panics if Type() does not return
	// ASTAssignment.
	AsAssignmentNode() AssignmentNode

	// Source is the token from source text that had the first token lexed as
	// part of this literal.
	Source() lex.Token

	// String returns a prettified representation of the node suitable for use
	// in line-by-line comparisons of tree structure. Two nodes are considered
	// semantcally identical if they produce identical String() output.
	String() string

	// Tunascript returns a string that contains tunascript code that if parsed
	// would result in an equivalent ASTNode. It is not necessarily the source
	// that produced this node as non-semantic elements are not included (such
	// as extra whitespace not a part of an unquoted string).
	Tunascript() string

	// Equal returns whether a node is equal to another. It will return false
	// if anything besides an ASTNode is passed in. ASTNodes do not consider
	// the result of Source() in their equality; ergo, this returns whether two
	// nodes have the same structure regardless of the exact source that
	// produced them.
	Equal(o any) bool
}

// LiteralNode is a node of the AST that represents a typed literal in code.
type LiteralNode struct {
	// Quoted can only be true if Value.Type() == String, and indicates whether
	// the value is wrapped in @-signs.
	Quoted bool

	// Value is the value of the literal.
	Value Value

	src lex.Token
}

func (n LiteralNode) Type() NodeType                   { return ASTLiteral }
func (n LiteralNode) AsLiteralNode() LiteralNode       { return n }
func (n LiteralNode) AsFuncNode() FuncNode             { panic("Type() is not ASTFunc") }
func (n LiteralNode) AsFlagNode() FlagNode             { panic("Type() is not ASTFlag") }
func (n LiteralNode) AsGroupNode() GroupNode           { panic("Type() is not ASTGroup") }
func (n LiteralNode) AsBinaryOpNode() BinaryOpNode     { panic("Type() is not ASTBinaryOp") }
func (n LiteralNode) AsUnaryOpNode() UnaryOpNode       { panic("Type() is not ASTUnaryOp") }
func (n LiteralNode) AsAssignmentNode() AssignmentNode { panic("Type() is not ASTAssignment") }
func (n LiteralNode) Source() lex.Token                { return n.src }

func (n LiteralNode) Tunascript() string {
	if n.Value.Type() == String {
		// then quoted applies
		if n.Quoted {
			return n.Value.Quoted()
		}
		return n.Value.Escaped()
	}
	return n.Value.String()
}

func (n LiteralNode) String() string {
	var typeName string
	switch n.Value.Type() {
	case Int:
		typeName = "NUMBER/INT"
	case Float:
		typeName = "NUMBER/FLOAT"
	case Bool:
		typeName = "BINARY/BOOL"
	case String:
		if n.Quoted {
			typeName = "TEXT/@STRING"
		} else {
			typeName = "TEXT/STRING"
		}
	}

	if n.Value.Type() == String {
		// add quotes to value if it's a string literal
		return fmt.Sprintf("[LITERAL %s \"%v\"]", typeName, n.Value.String())
	} else {
		return fmt.Sprintf("[LITERAL %s %v]", typeName, n.Value.String())
	}
}

// Does not consider Source.
func (n LiteralNode) Equal(o any) bool {
	other, ok := o.(LiteralNode)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*LiteralNode)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if n.Quoted != other.Quoted {
		return false
	}
	if !n.Value.Equal(other.Value) {
		return false
	}

	return true
}

type FuncNode struct {
	// Func is the name of the function being called, without the leading $.
	Func string

	// Args is all arguments to the function.
	Args []ASTNode

	src lex.Token
}

func (n FuncNode) Type() NodeType                   { return ASTFunc }
func (n FuncNode) AsLiteralNode() LiteralNode       { panic("Type() is not ASTLiteral") }
func (n FuncNode) AsFuncNode() FuncNode             { return n }
func (n FuncNode) AsFlagNode() FlagNode             { panic("Type() is not ASTFlag") }
func (n FuncNode) AsGroupNode() GroupNode           { panic("Type() is not ASTGroup") }
func (n FuncNode) AsBinaryOpNode() BinaryOpNode     { panic("Type() is not ASTBinaryOp") }
func (n FuncNode) AsUnaryOpNode() UnaryOpNode       { panic("Type() is not ASTUnaryOp") }
func (n FuncNode) AsAssignmentNode() AssignmentNode { panic("Type() is not ASTAssignment") }
func (n FuncNode) Source() lex.Token                { return n.src }

func (n FuncNode) Tunascript() string {
	s := "$" + n.Func + "("
	for i := range n.Args {
		argStr := n.Args[i].Tunascript()
		s += argStr
		if i+1 < len(n.Args) {
			s += ", "
		}
	}
	s += ")"
	return s
}

func (n FuncNode) String() string {
	const (
		argStart = " A: "
	)

	s := "[FUNC $" + n.Func

	if len(n.Args) == 0 {
		s += "]"
		return s
	}

	s += "\n"
	for i := range n.Args {
		s += argStart + spaceIndentNewlines(n.Args[i].String(), len(argStart)) + "\n"
	}
	s += "]"

	return s
}

// Does not consider Source.
func (n FuncNode) Equal(o any) bool {
	other, ok := o.(FuncNode)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*FuncNode)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if n.Func != other.Func {
		return false
	}
	if len(n.Args) != len(other.Args) {
		return false
	}
	for i := range n.Args {
		if !n.Args[i].Equal(other.Args[i]) {
			return false
		}
	}

	return true
}

type FlagNode struct {
	// Flag is the name of the flag, without the leading $.
	Flag string

	src lex.Token
}

func (n FlagNode) Type() NodeType                   { return ASTFlag }
func (n FlagNode) AsLiteralNode() LiteralNode       { panic("Type() is not ASTLiteral") }
func (n FlagNode) AsFuncNode() FuncNode             { panic("Type() is not ASTFunc") }
func (n FlagNode) AsFlagNode() FlagNode             { return n }
func (n FlagNode) AsGroupNode() GroupNode           { panic("Type() is not ASTGroup") }
func (n FlagNode) AsBinaryOpNode() BinaryOpNode     { panic("Type() is not ASTBinaryOp") }
func (n FlagNode) AsUnaryOpNode() UnaryOpNode       { panic("Type() is not ASTUnaryOp") }
func (n FlagNode) AsAssignmentNode() AssignmentNode { panic("Type() is not ASTAssignment") }

func (n FlagNode) Source() lex.Token { return n.src }

func (n FlagNode) Tunascript() string {
	return "$" + n.Flag
}

func (n FlagNode) String() string {
	return fmt.Sprintf("[FLAG $%s]", n.Flag)
}

// Does not consider Source.
func (n FlagNode) Equal(o any) bool {
	other, ok := o.(FlagNode)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*FlagNode)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if n.Flag != other.Flag {
		return false
	}

	return true
}

type GroupNode struct {
	Expr ASTNode

	src lex.Token
}

func (n GroupNode) Type() NodeType                   { return ASTGroup }
func (n GroupNode) AsLiteralNode() LiteralNode       { panic("Type() is not ASTLiteral") }
func (n GroupNode) AsFuncNode() FuncNode             { panic("Type() is not ASTFunc") }
func (n GroupNode) AsFlagNode() FlagNode             { panic("Type() is not ASTFlag") }
func (n GroupNode) AsGroupNode() GroupNode           { return n }
func (n GroupNode) AsBinaryOpNode() BinaryOpNode     { panic("Type() is not ASTBinaryOp") }
func (n GroupNode) AsUnaryOpNode() UnaryOpNode       { panic("Type() is not ASTUnaryOp") }
func (n GroupNode) AsAssignmentNode() AssignmentNode { panic("Type() is not ASTAssignment") }

func (n GroupNode) Source() lex.Token { return n.src }

func (n GroupNode) Tunascript() string {
	return "(" + n.Expr.Tunascript() + ")"
}

func (n GroupNode) String() string {
	const (
		exprStart = " E: "
	)

	s := "[GROUP\n"
	s += exprStart + spaceIndentNewlines(n.Expr.String(), len(exprStart)) + "\n"
	s += "]"

	return s
}

// Does not consider Source.
func (n GroupNode) Equal(o any) bool {
	other, ok := o.(GroupNode)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*GroupNode)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if !n.Expr.Equal(other.Expr) {
		return false
	}

	return true
}

type BinaryOpNode struct {
	Left  ASTNode
	Right ASTNode
	Op    BinaryOperation

	src lex.Token
}

func (n BinaryOpNode) Type() NodeType                   { return ASTBinaryOp }
func (n BinaryOpNode) AsLiteralNode() LiteralNode       { panic("Type() is not ASTLiteral") }
func (n BinaryOpNode) AsFuncNode() FuncNode             { panic("Type() is not ASTFunc") }
func (n BinaryOpNode) AsFlagNode() FlagNode             { panic("Type() is not ASTFlag") }
func (n BinaryOpNode) AsGroupNode() GroupNode           { panic("Type() is not ASTGroup") }
func (n BinaryOpNode) AsBinaryOpNode() BinaryOpNode     { return n }
func (n BinaryOpNode) AsUnaryOpNode() UnaryOpNode       { panic("Type() is not ASTUnaryOp") }
func (n BinaryOpNode) AsAssignmentNode() AssignmentNode { panic("Type() is not ASTAssignment") }

func (n BinaryOpNode) Source() lex.Token { return n.src }

func (n BinaryOpNode) Tunascript() string {
	return fmt.Sprintf("%s %s %s", n.Left.Tunascript(), n.Op.Symbol(), n.Right.Tunascript())
}

func (n BinaryOpNode) String() string {
	const (
		leftStart  = " L: "
		rightStart = " R: "
	)

	leftStr := spaceIndentNewlines(n.Left.String(), len(leftStart))
	rightStr := spaceIndentNewlines(n.Right.String(), len(rightStart))

	fmtStr := "[BINARY_OP %s\n%s%s\n%s%s\n]"
	return fmt.Sprintf(fmtStr, n.Op.String(), leftStart, leftStr, rightStart, rightStr)
}

// Does not consider Source.
func (n BinaryOpNode) Equal(o any) bool {
	other, ok := o.(BinaryOpNode)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*BinaryOpNode)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if !n.Left.Equal(other.Left) {
		return false
	}
	if !n.Right.Equal(other.Right) {
		return false
	}
	if n.Op != other.Op {
		return false
	}

	return true
}

type UnaryOpNode struct {
	Operand ASTNode
	Op      UnaryOperation
	PostFix bool

	src lex.Token
}

func (n UnaryOpNode) Type() NodeType                   { return ASTUnaryOp }
func (n UnaryOpNode) AsLiteralNode() LiteralNode       { panic("Type() is not ASTLiteral") }
func (n UnaryOpNode) AsFuncNode() FuncNode             { panic("Type() is not ASTFunc") }
func (n UnaryOpNode) AsFlagNode() FlagNode             { panic("Type() is not ASTFlag") }
func (n UnaryOpNode) AsGroupNode() GroupNode           { panic("Type() is not ASTGroup") }
func (n UnaryOpNode) AsBinaryOpNode() BinaryOpNode     { panic("Type() is not ASTBinaryOp") }
func (n UnaryOpNode) AsUnaryOpNode() UnaryOpNode       { return n }
func (n UnaryOpNode) AsAssignmentNode() AssignmentNode { panic("Type() is not ASTAssignment") }

func (n UnaryOpNode) Source() lex.Token { return n.src }

func (n UnaryOpNode) Tunascript() string {
	fmtStr := "%[1]s%[2]s"
	if n.PostFix {
		fmtStr = "%[2]s%[1]s"
	}

	return fmt.Sprintf(fmtStr, n.Op.Symbol(), n.Operand.Tunascript())
}

func (n UnaryOpNode) String() string {
	const (
		operandStart = " O: "
	)

	operandStr := spaceIndentNewlines(n.Operand.String(), len(operandStart))

	fmtStr := "[UNARY_OP %s\n%s%s\n]"
	return fmt.Sprintf(fmtStr, n.Op.String(), operandStart, operandStr)
}

// Does not consider Source.
func (n UnaryOpNode) Equal(o any) bool {
	other, ok := o.(UnaryOpNode)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*UnaryOpNode)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if !n.Operand.Equal(other.Operand) {
		return false
	}
	if n.Op != other.Op {
		return false
	}
	if n.PostFix != other.PostFix {
		return false
	}

	return true
}

// AssignmentNode represents assignment of a value to a flag. Strictly speaking,
// this is a type of binary operation or unary operation depedning on the type
// of assignment being done, but it is kept as a separate AST node to help with
// analysis.
type AssignmentNode struct {

	// Flag is the name of the flag being assigned to, without the leading $.
	Flag string

	// Value will be nil if Op is an operation which does not take an argument
	// (such as Increment or Decrement, which always have an implied argument of
	// 1).
	Value ASTNode

	// Op is the operation performed
	Op AssignmentOperation

	// PostFix can only be true when Expr is nil, otherwise it will always be
	// false.
	PostFix bool

	src lex.Token
}

func (n AssignmentNode) Type() NodeType                   { return ASTAssignment }
func (n AssignmentNode) AsLiteralNode() LiteralNode       { panic("Type() is not ASTLiteral") }
func (n AssignmentNode) AsFuncNode() FuncNode             { panic("Type() is not ASTFunc") }
func (n AssignmentNode) AsFlagNode() FlagNode             { panic("Type() is not ASTFlag") }
func (n AssignmentNode) AsGroupNode() GroupNode           { panic("Type() is not ASTGroup") }
func (n AssignmentNode) AsBinaryOpNode() BinaryOpNode     { panic("Type() is not ASTBinaryOp") }
func (n AssignmentNode) AsUnaryOpNode() UnaryOpNode       { panic("Type() is not ASTUnaryOp") }
func (n AssignmentNode) AsAssignmentNode() AssignmentNode { return n }

func (n AssignmentNode) Source() lex.Token { return n.src }

func (n AssignmentNode) Tunascript() string {
	if n.Value == nil {
		// then there is no argument and we are in "unary assignment" mode

		fmtStr := "%[1]s%[2]s"
		if n.PostFix {
			fmtStr = "%[2]s%[1]s"
		}

		return fmt.Sprintf(fmtStr, n.Op.Symbol(), n.Value.Tunascript())
	}

	// there is an argument; we are in "binary assignment" mode

	return fmt.Sprintf("$%s %s %s", n.Flag, n.Op.Symbol(), n.Value.Tunascript())
}

func (n AssignmentNode) String() string {
	const (
		valueStart = " V: "
	)

	s := fmt.Sprintf("[ASSIGNMENT %s $%s", n.Op.String(), n.Flag)

	if n.Value == nil {
		s += "]"
		return s
	}

	s += "\n"
	valueStr := spaceIndentNewlines(n.Value.String(), len(valueStart))
	s += valueStart + valueStr + "\n]"
	return s
}

// Does not consider Source.
func (n AssignmentNode) Equal(o any) bool {
	other, ok := o.(AssignmentNode)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*AssignmentNode)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if (n.Value == nil && other.Value != nil) || (n.Value != nil && other.Value == nil) {
		return false
	}
	if n.Value != nil && !n.Value.Equal(other.Value) {
		return false
	}
	if n.Op != other.Op {
		return false
	}
	if n.PostFix != other.PostFix {
		return false
	}
	if n.Flag != other.Flag {
		return false
	}

	return true
}
