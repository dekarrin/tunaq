package syntax

import (
	"fmt"
	"strings"

	"github.com/dekarrin/ictiobus/lex"
)

type AST struct {
	nodes []*astNode
}

const (
	astTreeLevelEmpty               = "        "
	astTreeLevelOngoing             = "  |     "
	astTreeLevelPrefix              = "  |%s: "
	astTreeLevelPrefixLast          = `  \%s: `
	astTreeLevelPrefixNamePadChar   = '-'
	astTreeLevelPrefixNamePadAmount = 3
)

func makeASTTreeLevelPrefix(msg string) string {
	for len([]rune(msg)) < astTreeLevelPrefixNamePadAmount {
		msg = string(astTreeLevelPrefixNamePadChar) + msg
	}
	return fmt.Sprintf(astTreeLevelPrefix, msg)
}

func makeASTTreeLevelPrefixLast(msg string) string {
	for len([]rune(msg)) < astTreeLevelPrefixNamePadAmount {
		msg = string(astTreeLevelPrefixNamePadChar) + msg
	}
	return fmt.Sprintf(astTreeLevelPrefixLast, msg)
}

// String returns a prettified representation of the entire AST suitable for use
// in line-by-line comparisons of tree structure. Two ASTs are considered
// semantcally identical if they produce identical String() output.
func (ast AST) String() string {
	var sb strings.Builder

	sb.WriteString("(AST)\n")

	for i := range ast.nodes {
		var firstPrefix string
		var contPrefix string
		if i+1 < len(ast.nodes) {
			firstPrefix = makeASTTreeLevelPrefix("")
			contPrefix = astTreeLevelOngoing
		} else {
			firstPrefix = makeASTTreeLevelPrefixLast("")
			contPrefix = astTreeLevelEmpty
		}
		itemOut := ast.nodes[i].leveledStr(firstPrefix, contPrefix)
		sb.WriteString(itemOut)
		if len(itemOut) > 0 && i+1 < len(ast.nodes) {
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

	if !equalSlices(ast.nodes, other.nodes) {
		return false
	}

	return true
}

// EqualSource returns whether the two ASTs were created from identical
// tunascript code.
func (ast AST) EqualSource(other AST) bool {
	if len(ast.nodes) != len(other.nodes) {
		return false
	}

	for i := range ast.nodes {
		if ast.nodes[i].source.Class().ID() != other.nodes[i].source.Class().ID() {
			return false
		}
	}

	return true
}

type astNode struct {
	value   *valueNode
	fn      *fnNode
	flag    *flagNode
	group   *groupNode
	opGroup *operatorGroupNode
	source  lex.Token
}

// Equal checks if the astNode is equal to the given parameter. If the parameter
// is not an astNode, it will not be equal. The source of an ASTNode is
// considered supplementary information and is not considered in the equality
// check.
func (n astNode) Equal(o any) bool {
	other, ok := o.(astNode)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*astNode)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if !equalNilness(n.value, other.value) {
		return false
	} else if n.value != nil && !n.value.Equal(*other.value) {
		return false
	} else if !equalNilness(n.fn, other.fn) {
		return false
	} else if n.fn != nil && !n.fn.Equal(*other.fn) {
		return false
	} else if !equalNilness(n.flag, other.flag) {
		return false
	} else if n.flag != nil && !n.flag.Equal(*other.flag) {
		return false
	} else if !equalNilness(n.group, other.group) {
		return false
	} else if n.group != nil && !n.group.Equal(*other.group) {
		return false
	} else if !equalNilness(n.opGroup, other.opGroup) {
		return false
	} else if n.opGroup != nil && !n.opGroup.Equal(*other.opGroup) {
		return false
	}

	// do not check source member; if all other things are the same, having
	// different source does not matter and is a consequence of the many-to-one
	// mapping of the meaning function L(x) from syntax to semantics.
	return true
}

func (n astNode) leveledStr(firstPrefix, contPrefix string) string {
	if n.value != nil {
		return n.value.leveledStr(firstPrefix)
	} else if n.flag != nil {
		return n.flag.leveledStr(firstPrefix)
	} else if n.fn != nil {
		return n.fn.leveledStr(firstPrefix, contPrefix)
	} else if n.group != nil {
		return n.group.leveledStr(firstPrefix, contPrefix)
	} else if n.opGroup != nil {
		return n.opGroup.leveledStr(firstPrefix, contPrefix)
	} else {
		// should never happen
		panic("empty ast node")
	}
}

type flagNode struct {
	name string
}

func (n flagNode) Equal(o any) bool {
	other, ok := o.(flagNode)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*flagNode)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if n.name != other.name {
		return false
	}

	return true
}

func (n flagNode) leveledStr(prefix string) string {
	return fmt.Sprintf("%s(FLAG \"%s\")", prefix, n.name)
}

type fnNode struct {
	name string
	args []*astNode
}

func (n fnNode) Equal(o any) bool {
	other, ok := o.(fnNode)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*fnNode)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if n.name != other.name {
		return false
	} else if !equalSlices(n.args, other.args) {
		return false
	}

	return true
}

func (n fnNode) leveledStr(firstPrefix, contPrefix string) string {
	var sb strings.Builder

	sb.WriteString(firstPrefix)
	sb.WriteString(fmt.Sprintf("(FUNCTION \"%s\")", n.name))

	for i := range n.args {
		sb.WriteRune('\n')
		var leveledFirstPrefix string
		var leveledContPrefix string
		if i+1 < len(n.args) {
			leveledFirstPrefix = contPrefix + makeASTTreeLevelPrefix(fmt.Sprintf("A%d", i))
			leveledContPrefix = contPrefix + astTreeLevelOngoing
		} else {
			leveledFirstPrefix = contPrefix + makeASTTreeLevelPrefixLast(fmt.Sprintf("A%d", i))
			leveledContPrefix = contPrefix + astTreeLevelEmpty
		}
		itemOut := n.args[i].leveledStr(leveledFirstPrefix, leveledContPrefix)
		sb.WriteString(itemOut)
	}

	return sb.String()
}

type valueNode struct {
	quotedStringVal   *string
	unquotedStringVal *string
	numVal            *int
	boolVal           *bool
}

func (n valueNode) Equal(o any) bool {
	other, ok := o.(valueNode)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*valueNode)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if !equalNilness(n.quotedStringVal, other.quotedStringVal) {
		return false
	} else if n.quotedStringVal != nil && *n.quotedStringVal != *other.quotedStringVal {
		return false
	} else if !equalNilness(n.unquotedStringVal, other.unquotedStringVal) {
		return false
	} else if n.unquotedStringVal != nil && *n.unquotedStringVal != *other.unquotedStringVal {
		return false
	} else if !equalNilness(n.numVal, other.numVal) {
		return false
	} else if n.numVal != nil && *n.numVal != *other.numVal {
		return false
	} else if !equalNilness(n.boolVal, other.boolVal) {
		return false
	} else if n.boolVal != nil && *n.boolVal != *other.boolVal {
		return false
	}

	return true
}

func (n valueNode) leveledStr(prefix string) string {
	if n.quotedStringVal != nil {
		return prefix + fmt.Sprintf("(QSTR_VALUE \"%s\")", *n.quotedStringVal)
	} else if n.unquotedStringVal != nil {
		return prefix + fmt.Sprintf("(STR_VALUE \"%s\")", *n.unquotedStringVal)
	} else if n.boolVal != nil {
		return prefix + fmt.Sprintf("(BOOL_VALUE \"%t\")", *n.boolVal)
	} else if n.numVal != nil {
		return prefix + fmt.Sprintf("(NUM_VALUE \"%d\")", *n.numVal)
	} else {
		// should never happen
		panic("empty ast node")
	}
}

type groupNode struct {
	expr *astNode
}

func (n groupNode) Equal(o any) bool {
	other, ok := o.(groupNode)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*groupNode)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if !equalNilness(n.expr, other.expr) {
		return false
	} else if n.expr != nil && !n.expr.Equal(*other.expr) {
		return false
	}

	return true
}

func (n groupNode) leveledStr(firstPrefix, contPrefix string) string {
	if n.expr == nil {
		panic("empty ast node")
	}

	fullStr := firstPrefix + "(GROUP)"

	leveledFirst := contPrefix + makeASTTreeLevelPrefixLast("")
	leveledCont := contPrefix + astTreeLevelEmpty

	groupOut := n.expr.leveledStr(leveledFirst, leveledCont)

	if len(groupOut) > 0 {
		fullStr += "\n" + groupOut
	}

	return fullStr
}

type operatorGroupNode struct {
	unaryOp *unaryOperatorGroupNode
	infixOp *binaryOperatorGroupNode
}

func (n operatorGroupNode) Equal(o any) bool {
	other, ok := o.(operatorGroupNode)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*operatorGroupNode)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if !equalNilness(n.unaryOp, other.unaryOp) {
		return false
	} else if n.unaryOp != nil && !n.unaryOp.Equal(*other.unaryOp) {
		return false
	} else if !equalNilness(n.infixOp, other.infixOp) {
		return false
	} else if n.infixOp != nil && !n.infixOp.Equal(*other.infixOp) {
		return false
	}

	return true
}

func (n operatorGroupNode) leveledStr(firstPrefix, contPrefix string) string {
	if n.infixOp != nil {
		return n.infixOp.leveledStr(firstPrefix, contPrefix)
	} else if n.unaryOp != nil {
		return n.unaryOp.leveledStr(firstPrefix, contPrefix)
	} else {
		panic("empty ast node")
	}
}

type unaryOperatorGroupNode struct {
	op      string
	operand *astNode
	prefix  bool
}

func (n unaryOperatorGroupNode) Equal(o any) bool {
	other, ok := o.(unaryOperatorGroupNode)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*unaryOperatorGroupNode)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if n.op != other.op {
		return false
	} else if !equalNilness(n.operand, other.operand) {
		return false
	} else if n.operand != nil && !n.operand.Equal(other.operand) {
		return false
	} else if n.prefix != other.prefix {
		return false
	}

	return true
}

func (n unaryOperatorGroupNode) leveledStr(firstPrefix, contPrefix string) string {
	if n.operand == nil {
		panic("empty ast node")
	}

	fullStr := firstPrefix + fmt.Sprintf("(UNARY_OP \"%s\")", n.op)

	leveledFirst := contPrefix + makeASTTreeLevelPrefixLast("")
	leveledCont := contPrefix + astTreeLevelEmpty

	operandOut := n.operand.leveledStr(leveledFirst, leveledCont)

	if len(operandOut) > 0 {
		fullStr += "\n" + operandOut
	}

	return fullStr
}

type binaryOperatorGroupNode struct {
	op    string
	left  *astNode
	right *astNode
}

func (n binaryOperatorGroupNode) Equal(o any) bool {
	other, ok := o.(binaryOperatorGroupNode)
	if !ok {
		// also okay if its the pointer value, as long as its non-nil
		otherPtr, ok := o.(*binaryOperatorGroupNode)
		if !ok {
			return false
		} else if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	if n.op != other.op {
		return false
	} else if !equalNilness(n.left, other.left) {
		return false
	} else if n.left != nil && !n.left.Equal(other.left) {
		return false
	} else if !equalNilness(n.right, other.right) {
		return false
	} else if n.right != nil && !n.right.Equal(other.right) {
		return false
	}

	return true
}

func (n binaryOperatorGroupNode) leveledStr(firstPrefix, contPrefix string) string {
	if n.left == nil || n.right == nil {
		panic("empty ast node")
	}

	fullStr := firstPrefix + fmt.Sprintf("(BINARY_OP \"%s\")", n.op)

	leftFirst := contPrefix + makeASTTreeLevelPrefix("L")
	leftCont := contPrefix + astTreeLevelOngoing
	rightFirst := contPrefix + makeASTTreeLevelPrefixLast("R")
	rightCont := contPrefix + astTreeLevelEmpty

	leftOut := n.left.leveledStr(leftFirst, leftCont)
	rightOut := n.right.leveledStr(rightFirst, rightCont)

	if len(leftOut) > 0 {
		fullStr += "\n" + leftOut
	}
	if len(rightOut) > 0 {
		fullStr += "\n" + rightOut
	}

	return fullStr
}

//

func (n binaryOperatorGroupNode) String() string {
	return "<BINARY OP>"
}

// ExpansionAnalysis is a lexed (and somewhat parsed) block of text containing
// both tunascript expansion-legal expressions and regular text. The zero-value
// of a ParsedExpansion is not suitable for use and they should only be created
// by calls to AnalyzeExpansion.
type ExpansionAST struct {
	nodes []expASTNode
}

type expASTNode struct {
	// can be a text node or a conditional node. Conditional nodes hold a series
	// of ifs
	text   *expTextNode   // if not nil its a text node
	branch *expBranchNode // if not nil its a branch node
	flag   *expFlagNode   // if not nil its a flag node
	source expSource
}

type expSource struct {
	text     string
	fullLine string
	line     int
	pos      int
}

type expFlagNode struct {
	name string
}

type expTextNode struct {
	t                string
	minusSpacePrefix *string
	minusSpaceSuffix *string
}

type expBranchNode struct {
	ifNode expCondNode
	/*elseIfNodes []expCondNode
	elseNode    *ExpansionAST*/
}
type expCondNode struct {
	cond    *AST
	content *ExpansionAST
}

// Tunascript returns a string that contains tunascript code that if parsed
// would result in an equivalent AST. It is not necessarily the source that
// produced this node as non-semantic elements are not included (such as extra
// whitespace not a part of an unquoted string), and any branches in it will be
// grouped if it is determined it is needed to preserve evaluation order of the
// statements.
//
// If the AST contains more than a single full tunascript expression, a space is
// inserted between each in the resulting string.
func (ast AST) Tunascript() string {
	var sb strings.Builder

	for i := range ast.nodes {
		sb.WriteString(ast.nodes[i].Tunascript())

		if i+1 < len(ast.nodes) {
			sb.WriteRune(' ')
		}
	}

	return sb.String()
}

// Tunascript returns a string that contains tunascript code that if parsed
// would result in this node. It is not necessarily the source that produced
// this node as non-semantic elements are not included (such as extra whitespace
// not a part of an unquoted string), and any branches in it will be grouped if
// it is determined it is needed to preserve evaluation order of the statements.
func (n astNode) Tunascript() string {
	if n.value != nil {
		return n.value.Tunascript()
	} else if n.fn != nil {
		return n.fn.Tunascript()
	} else if n.flag != nil {
		return n.fn.Tunascript()
	} else if n.group != nil {
		return n.group.Tunascript()
	} else if n.opGroup != nil {
		return n.opGroup.Tunascript()
	} else {
		// should never happen
		panic("empty node in ast")
	}
}

// Tunascript returns a string that contains tunascript code that if parsed
// would result in this node. It is not necessarily the source that produced
// this node as non-semantic elements are not included (such as extra whitespace
// not a part of an unquoted string), and any branches in it will be grouped if
// it is determined it is needed to preserve evaluation order of the statements.
func (n flagNode) Tunascript() string {
	return fmt.Sprintf("%s%s", literalStrIdentifierStart, n.name)
}

// Tunascript returns a string that contains tunascript code that if parsed
// would result in this node. It is not necessarily the source that produced
// this node as non-semantic elements are not included (such as extra whitespace
// not a part of an unquoted string), and any branches in it will be grouped if
// it is determined it is needed to preserve evaluation order of the statements.
func (n fnNode) Tunascript() string {
	var sb strings.Builder
	sb.WriteString(literalStrIdentifierStart)
	sb.WriteString(n.name)
	sb.WriteString(literalStrGroupOpen)

	for i := range n.args {
		sb.WriteString(n.args[i].Tunascript())
		if i+1 < len(n.args) {
			sb.WriteRune(',')
			sb.WriteRune(' ')
		}
	}

	sb.WriteString(literalStrGroupClose)

	return sb.String()
}

// Tunascript returns a string that contains tunascript code that if parsed
// would result in this node. It is not necessarily the source that produced
// this node as non-semantic elements are not included (such as extra whitespace
// not a part of an unquoted string), and any branches in it will be grouped if
// it is determined it is needed to preserve evaluation order of the statements.
func (n valueNode) Tunascript() string {
	if n.quotedStringVal != nil {
		return *n.quotedStringVal
	} else if n.unquotedStringVal != nil {
		return *n.unquotedStringVal
	} else if n.boolVal != nil {
		return fmt.Sprintf("%t", *n.boolVal)
	} else if n.numVal != nil {
		return fmt.Sprintf("%d", *n.numVal)
	} else {
		// should never happen
		panic("empty value node in ast")
	}
}

// Tunascript returns a string that contains tunascript code that if parsed
// would result in this node. It is not necessarily the source that produced
// this node as non-semantic elements are not included (such as extra whitespace
// not a part of an unquoted string), and any branches in it will be grouped if
// it is determined it is needed to preserve evaluation order of the statements.
func (n groupNode) Tunascript() string {
	if n.expr == nil {
		// should never happen
		panic("empty group node in ast")
	}
	return fmt.Sprintf("%s%s%s", literalStrGroupOpen, n.expr.Tunascript(), literalStrGroupClose)
}

// Tunascript returns a string that contains tunascript code that if parsed
// would result in this node. It is not necessarily the source that produced
// this node as non-semantic elements are not included (such as extra whitespace
// not a part of an unquoted string), and any branches in it will be grouped if
// it is determined it is needed to preserve evaluation order of the statements.
func (n operatorGroupNode) Tunascript() string {
	if n.unaryOp != nil {
		return n.unaryOp.Tunascript()
	} else if n.infixOp != nil {
		return n.infixOp.Tunascript()
	} else {
		// should never happen
		panic("empty operator group node in ast")
	}
}

// Tunascript returns a string that contains tunascript code that if parsed
// would result in this node. It is not necessarily the source that produced
// this node as non-semantic elements are not included (such as extra whitespace
// not a part of an unquoted string), and any branches in it will be grouped if
// it is determined it is needed to preserve evaluation order of the statements.
func (n unaryOperatorGroupNode) Tunascript() string {
	var fmtStr string
	if n.prefix {
		fmtStr = "%[2]s%[1]s"
	} else {
		fmtStr = "%[1]s%[2]s"
	}

	return fmt.Sprintf(fmtStr, n.operand.Tunascript(), n.op)
}

// Tunascript returns a string that contains tunascript code that if parsed
// would result in this node. It is not necessarily the source that produced
// this node as non-semantic elements are not included (such as extra whitespace
// not a part of an unquoted string).
func (n binaryOperatorGroupNode) Tunascript() string {
	return fmt.Sprintf("%s %s %s", n.left.Tunascript(), n.op, n.right.Tunascript())
}
