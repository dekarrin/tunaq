package tunascript

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
	nodes []expTreeNode
}

type expTreeNode struct {
	// can be a text node or a conditional node. Conditional nodes hold a series
	// of ifs
	text   *expTextNode   // if not nil its a text node
	branch *expBranchNode // if not nil its a branch node
	flag   *expFlagNode   // if not nil its a flag node
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
