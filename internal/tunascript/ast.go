package tunascript

import (
	"fmt"
	"strings"
)

type AST struct {
	nodes []*astNode
}

type astNode struct {
	value   *valueNode
	fn      *fnNode
	flag    *flagNode
	group   *groupNode
	opGroup *operatorGroupNode
	source  token
}

type flagNode struct {
	name string
}

type fnNode struct {
	name string
	args []*astNode
}

type valueNode struct {
	quotedStringVal   *string
	unquotedStringVal *string
	numVal            *int
	boolVal           *bool
}

type groupNode struct {
	expr *astNode
}
type operatorGroupNode struct {
	unaryOp *unaryOperatorGroupNode
	infixOp *binaryOperatorGroupNode
}

type unaryOperatorGroupNode struct {
	op      string
	operand *astNode
	prefix  bool
}

type binaryOperatorGroupNode struct {
	op    string
	left  *astNode
	right *astNode
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
