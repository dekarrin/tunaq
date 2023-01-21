package tunascript

import (
	"fmt"
	"strconv"
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

// indexOfMatchingParen takes the given string, which must start with
// a parenthesis char "(", and returns the index of the ")" that matches it. Any
// text in between is analyzed for other parenthesis and if they are there, they
// must be matched as well.
//
// Index will be -1 if it does not have a match.
// Error is non-nil if there is unlex-able tunascript syntax between the parens,
// of if s cannot be operated on.
func indexOfMatchingParen(sRunes []rune) (int, AST, error) {
	if sRunes[0] != '(' {
		var errStr string
		if len(sRunes) > 50 {
			errStr = string(sRunes[:50]) + "..."
		} else {
			errStr = string(sRunes)
		}
		return 0, AST{}, SyntaxError{
			message: fmt.Sprintf("no opening paren at start of analysis string %q", errStr),
		}
	}

	if len(sRunes) < 2 {
		return 0, AST{}, SyntaxError{
			message: "unexpected end of expression (unmatched left-parenthesis)",
		}
	}

	tokenStr, consumed, err := lexRunes(sRunes, true)
	if err != nil {
		return 0, AST{}, err
	}

	// check that we got minimum tokens
	if tokenStr.Len() < 3 {
		// not enough tokens; at minimum we require lparen, rparen, and EOT.
		return 0, AST{}, SyntaxError{
			message: "unexpected end of expression (unmatched left-parenthesis)",
		}
	}
	// check that we ended on a right paren (will be second-to-last bc last is EOT)
	if tokenStr.tokens[len(tokenStr.tokens)-2].class.id != tsGroupClose.id {
		// in this case, lexing got to the end of the string but did not finish
		// on a right paren. This is a syntax error.
		return 0, AST{}, SyntaxError{
			message: "unexpected end of expression (unmatched left-parenthesis)",
		}
	}

	// modify returned list of tokens to not include the start and end parens
	// before parsing
	eotLexeme := tokenStr.tokens[len(tokenStr.tokens)-1]          // preserve EOT
	tokenStr.tokens = tokenStr.tokens[1 : len(tokenStr.tokens)-2] // chop off ends
	tokenStr.tokens = append(tokenStr.tokens, eotLexeme)          // add EOT back in

	// now parse it to get back the actual AST
	ast, err := Parse(tokenStr)
	if err != nil {
		return 0, ast, err
	}

	return consumed, ast, nil
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

func (east ExpansionAST) MarshalBinary() ([]byte, error) {
	var data []byte

	// node count
	data = append(data, encBinaryInt(len(east.nodes))...)

	// each node
	for i := range east.nodes {
		data = append(data, encBinary(east.nodes[i])...)
	}

	return data, nil
}

func (east *ExpansionAST) UnmarshalBinary(data []byte) error {
	var err error
	var readBytes int
	var nodeCount int

	// node count
	nodeCount, readBytes, err = decBinaryInt(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]

	// each node
	for i := 0; i < nodeCount; i++ {
		var node expTreeNode
		readBytes, err := decBinary(data, &node)
		if err != nil {
			return err
		}
		data = data[readBytes:]

		east.nodes = append(east.nodes, node)
	}

	return nil
}

func (ecn expCondNode) MarshalBinary() ([]byte, error) {
	var data []byte

	// cond ptr
	if ecn.cond == nil {
		data = append(data, encBinaryBool(false)...)
	} else {
		data = append(data, encBinaryBool(true)...)
		data = append(data, encBinary(*ecn.cond)...)
	}

	// content ptr
	if ecn.content == nil {
		data = append(data, encBinaryBool(false)...)
	} else {
		data = append(data, encBinaryBool(true)...)
		data = append(data, encBinary(*ecn.content)...)
	}

	return data, nil
}

func (ecn *expCondNode) UnmarshalBinary(data []byte) error {
	var err error
	var readBytes int
	var isNil bool

	// cond ptr
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		ecn.cond = nil
	} else {
		var condVal *AST
		readBytes, err := decBinary(data, condVal)
		if err != nil {
			return err
		}
		data = data[readBytes:]
		ecn.cond = condVal
	}

	// content ptr
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		ecn.content = nil
	} else {
		var contentVal *ExpansionAST
		_, err := decBinary(data, contentVal)
		if err != nil {
			return err
		}
		//data = data[readBytes:]
		ecn.content = contentVal
	}

	return nil
}

func (etn expTreeNode) MarshalBinary() ([]byte, error) {
	var data []byte

	// text ptr
	if etn.text == nil {
		data = append(data, encBinaryBool(false)...)
	} else {
		data = append(data, encBinaryBool(true)...)
		data = append(data, encBinary(*etn.text)...)
	}

	// branch ptr
	if etn.branch == nil {
		data = append(data, encBinaryBool(false)...)
	} else {
		data = append(data, encBinaryBool(true)...)
		data = append(data, encBinary(*etn.branch)...)
	}

	// flag ptr
	if etn.flag == nil {
		data = append(data, encBinaryBool(false)...)
	} else {
		data = append(data, encBinaryBool(true)...)
		data = append(data, encBinary(*etn.flag)...)
	}

	return data, nil
}

func (etn *expTreeNode) UnmarshalBinary(data []byte) error {
	var err error
	var readBytes int
	var isNil bool

	// text ptr
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		etn.text = nil
	} else {
		var textVal expTextNode
		readBytes, err := decBinary(data, &textVal)
		if err != nil {
			return err
		}
		data = data[readBytes:]

		etn.text = &textVal
	}

	// branch ptr
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		etn.branch = nil
	} else {
		var branchVal expBranchNode
		readBytes, err := decBinary(data, &branchVal)
		if err != nil {
			return err
		}
		data = data[readBytes:]

		etn.branch = &branchVal
	}

	// flag ptr
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		etn.flag = nil
	} else {
		var flagVal expFlagNode
		_, err := decBinary(data, &flagVal)
		if err != nil {
			return err
		}
		//data = data[readBytes:]

		etn.flag = &flagVal
	}

	return nil
}

func (efn expFlagNode) MarshalBinary() ([]byte, error) {
	var data []byte

	data = append(data, encBinaryString(efn.name)...)

	return data, nil
}

func (efn *expFlagNode) UnmarshalBinary(data []byte) error {
	var err error
	//var readBytes int

	efn.name, _, err = decBinaryString(data)
	if err != nil {
		return err
	}
	//data = data[readBytes:]

	return nil
}

func (etn expTextNode) MarshalBinary() ([]byte, error) {
	var data []byte

	// main string
	data = append(data, encBinaryString(etn.t)...)

	// minus space prefix
	if etn.minusSpacePrefix == nil {
		data = append(data, encBinaryBool(false)...)
	} else {
		data = append(data, encBinaryBool(true)...)
		data = append(data, encBinaryString(*etn.minusSpacePrefix)...)
	}

	// minus space suffix
	if etn.minusSpaceSuffix == nil {
		data = append(data, encBinaryBool(false)...)
	} else {
		data = append(data, encBinaryBool(true)...)
		data = append(data, encBinaryString(*etn.minusSpaceSuffix)...)
	}

	return data, nil
}

func (etn *expTextNode) UnmarshalBinary(data []byte) error {
	var err error
	var readBytes int
	var isNil bool

	// main string
	etn.t, readBytes, err = decBinaryString(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]

	// minus space prefix
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		etn.minusSpacePrefix = nil
	} else {
		mspVal, readBytes, err := decBinaryString(data)
		if err != nil {
			return err
		}
		data = data[readBytes:]

		etn.minusSpacePrefix = &mspVal
	}

	// minus space suffix
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		etn.minusSpaceSuffix = nil
	} else {
		mssVal, _, err := decBinaryString(data)
		if err != nil {
			return err
		}
		//data = data[readBytes:]

		etn.minusSpaceSuffix = &mssVal
	}

	return nil
}

func (ebn expBranchNode) MarshalBinary() ([]byte, error) {
	var data []byte

	data = append(data, encBinary(ebn.ifNode)...)

	return data, nil
}

func (ebn *expBranchNode) UnmarshalBinary(data []byte) error {
	var err error
	//var readBytes int

	_, err = decBinary(data, &ebn.ifNode)
	if err != nil {
		return err
	}
	//data = data[readBytes:]

	return nil
}

func (n AST) MarshalBinary() ([]byte, error) {
	var data []byte

	// node count
	data = append(data, encBinaryInt(len(n.nodes))...)

	// each arg (skip using leading bool pointer validatity for space, if they
	// aren't valid, panic)
	for i := range n.nodes {
		if n.nodes[i] == nil {
			// should never happen
			panic("empty node in ast")
		}
		data = append(data, encBinary(*n.nodes[i])...)
	}

	return data, nil
}

func (n *AST) UnmarshalBinary(data []byte) error {
	var err error
	var readBytes int
	var nodeCount int

	// arg count
	nodeCount, readBytes, err = decBinaryInt(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]

	// each arg
	n.nodes = make([]*astNode, nodeCount)
	for i := 0; i < nodeCount; i++ {
		var argVal astNode
		readBytes, err = decBinary(data, &argVal)
		if err != nil {
			return err
		}
		data = data[readBytes:]
		n.nodes[i] = &argVal
	}

	return nil
}

func (n astNode) MarshalBinary() ([]byte, error) {
	var data []byte

	// value
	data = append(data, encBinaryBool(n.value != nil)...)
	if n.value != nil {
		data = append(data, encBinary(*n.value)...)
	}

	// fn
	data = append(data, encBinaryBool(n.fn != nil)...)
	if n.fn != nil {
		data = append(data, encBinary(*n.fn)...)
	}

	// flag
	data = append(data, encBinaryBool(n.flag != nil)...)
	if n.flag != nil {
		data = append(data, encBinary(*n.flag)...)
	}

	// group
	data = append(data, encBinaryBool(n.group != nil)...)
	if n.group != nil {
		data = append(data, encBinary(*n.group)...)
	}

	// opGroup
	data = append(data, encBinaryBool(n.opGroup != nil)...)
	if n.opGroup != nil {
		data = append(data, encBinary(*n.opGroup)...)
	}

	// source
	data = append(data, encBinary(n.source)...)

	return data, nil
}

func (n *astNode) UnmarshalBinary(data []byte) error {
	var err error
	var readBytes int
	var isNil bool

	// value
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		n.value = nil
	} else {
		readBytes, err = decBinary(data, n.value)
		if err != nil {
			return err
		}
		data = data[readBytes:]
	}

	// fn
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		n.fn = nil
	} else {
		readBytes, err = decBinary(data, n.fn)
		if err != nil {
			return err
		}
		data = data[readBytes:]
	}

	// flag
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		n.flag = nil
	} else {
		readBytes, err = decBinary(data, n.flag)
		if err != nil {
			return err
		}
		data = data[readBytes:]
	}

	// group
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		n.group = nil
	} else {
		readBytes, err = decBinary(data, n.group)
		if err != nil {
			return err
		}
		data = data[readBytes:]
	}

	// opGroup
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		n.opGroup = nil
	} else {
		readBytes, err = decBinary(data, n.opGroup)
		if err != nil {
			return err
		}
		data = data[readBytes:]
	}

	// source
	_, err = decBinary(data, &n.source)
	if err != nil {
		return err
	}
	// data = data[readBytes:]

	return nil
}

func (n flagNode) MarshalBinary() ([]byte, error) {
	var data []byte

	data = append(data, encBinaryString(n.name)...)

	return data, nil
}

func (n *flagNode) UnmarshalBinary(data []byte) error {
	var err error
	//var readBytes int

	// func name
	n.name, _, err = decBinaryString(data)
	if err != nil {
		return err
	}
	//data = data[readBytes:]

	return nil
}

func (n fnNode) MarshalBinary() ([]byte, error) {
	var data []byte

	// func name
	data = append(data, encBinaryString(n.name)...)

	// arg count
	data = append(data, encBinaryInt(len(n.args))...)

	// each arg (skip using leading bool pointer validatity for space, if they
	// aren't valid, panic)
	for i := range n.args {
		if n.args[i] == nil {
			// should never happen
			panic("empty node in func arg list ast")
		}
		data = append(data, encBinary(*n.args[i])...)
	}

	return data, nil
}

func (n *fnNode) UnmarshalBinary(data []byte) error {
	var err error
	var readBytes int
	var argCount int

	// func name
	n.name, readBytes, err = decBinaryString(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]

	// arg count
	argCount, readBytes, err = decBinaryInt(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]

	// each arg
	n.args = make([]*astNode, argCount)
	for i := 0; i < argCount; i++ {
		var argVal astNode
		readBytes, err = decBinary(data, &argVal)
		if err != nil {
			return err
		}
		data = data[readBytes:]
		n.args[i] = &argVal
	}

	return nil
}

func (n valueNode) MarshalBinary() ([]byte, error) {
	var data []byte

	// quoted str
	data = append(data, encBinaryBool(n.quotedStringVal != nil)...)
	if n.quotedStringVal != nil {
		data = append(data, encBinaryString(*n.quotedStringVal)...)
	}

	// unquoted str
	data = append(data, encBinaryBool(n.unquotedStringVal != nil)...)
	if n.quotedStringVal != nil {
		data = append(data, encBinaryString(*n.unquotedStringVal)...)
	}

	// num
	data = append(data, encBinaryBool(n.numVal != nil)...)
	if n.quotedStringVal != nil {
		data = append(data, encBinaryInt(*n.numVal)...)
	}

	// bool
	data = append(data, encBinaryBool(n.boolVal != nil)...)
	if n.quotedStringVal != nil {
		data = append(data, encBinaryBool(*n.boolVal)...)
	}

	return data, nil
}

func (n *valueNode) UnmarshalBinary(data []byte) error {
	var err error
	var readBytes int
	var isNil bool

	// quoted str
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		n.quotedStringVal = nil
	} else {
		var strVal string
		strVal, readBytes, err = decBinaryString(data)
		if err != nil {
			return err
		}
		data = data[readBytes:]

		n.quotedStringVal = &strVal
	}

	// unquoted str
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		n.unquotedStringVal = nil
	} else {
		var strVal string
		strVal, readBytes, err = decBinaryString(data)
		if err != nil {
			return err
		}
		data = data[readBytes:]

		n.unquotedStringVal = &strVal
	}

	// num
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		n.numVal = nil
	} else {
		var iVal int
		iVal, readBytes, err = decBinaryInt(data)
		if err != nil {
			return err
		}
		data = data[readBytes:]

		n.numVal = &iVal
	}

	// bool
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		n.boolVal = nil
	} else {
		var bVal bool
		bVal, _, err = decBinaryBool(data)
		if err != nil {
			return err
		}
		//data = data[readBytes:]

		n.boolVal = &bVal
	}

	return nil
}

func (n groupNode) MarshalBinary() ([]byte, error) {
	var data []byte

	data = append(data, encBinaryBool(n.expr != nil)...)
	if n.expr != nil {
		data = append(data, encBinary(*n.expr)...)
	}

	return data, nil
}

func (n *groupNode) UnmarshalBinary(data []byte) error {
	var err error
	var readBytes int
	var isNil bool

	// expr
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		n.expr = nil
	} else {
		_, err = decBinary(data, n.expr)
		if err != nil {
			return err
		}
		//data = data[readBytes:]
	}

	return nil
}

func (n operatorGroupNode) MarshalBinary() ([]byte, error) {
	var data []byte

	data = append(data, encBinaryBool(n.unaryOp != nil)...)
	if n.unaryOp != nil {
		data = append(data, encBinary(*n.unaryOp)...)
	}

	data = append(data, encBinaryBool(n.infixOp != nil)...)
	if n.infixOp != nil {
		data = append(data, encBinary(*n.infixOp)...)
	}

	return data, nil
}

func (n *operatorGroupNode) UnmarshalBinary(data []byte) error {
	var err error
	var readBytes int
	var isNil bool

	// unaryOp
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		n.unaryOp = nil
	} else {
		readBytes, err = decBinary(data, n.unaryOp)
		if err != nil {
			return err
		}
		data = data[readBytes:]
	}

	// infix
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		n.infixOp = nil
	} else {
		_, err = decBinary(data, n.infixOp)
		if err != nil {
			return err
		}
		//data = data[readBytes:]
	}

	return nil
}

func (n unaryOperatorGroupNode) MarshalBinary() ([]byte, error) {
	var data []byte

	data = append(data, encBinaryString(n.op)...)

	data = append(data, encBinaryBool(n.operand != nil)...)
	if n.operand != nil {
		data = append(data, encBinary(*n.operand)...)
	}

	return data, nil
}

func (n *unaryOperatorGroupNode) UnmarshalBinary(data []byte) error {
	var err error
	var readBytes int
	var isNil bool

	n.op, readBytes, err = decBinaryString(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]

	// operand
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		n.operand = nil
	} else {
		_, err = decBinary(data, n.operand)
		if err != nil {
			return err
		}
		//data = data[readBytes:]
	}

	return nil
}

func (n binaryOperatorGroupNode) MarshalBinary() ([]byte, error) {
	var data []byte

	data = append(data, encBinaryString(n.op)...)

	data = append(data, encBinaryBool(n.left != nil)...)
	if n.left != nil {
		data = append(data, encBinary(*n.left)...)
	}

	data = append(data, encBinaryBool(n.right != nil)...)
	if n.right != nil {
		data = append(data, encBinary(*n.right)...)
	}

	return data, nil
}

func (n *binaryOperatorGroupNode) UnmarshalBinary(data []byte) error {
	var err error
	var readBytes int
	var isNil bool

	n.op, readBytes, err = decBinaryString(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]

	// left
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		n.left = nil
	} else {
		readBytes, err = decBinary(data, n.left)
		if err != nil {
			return err
		}
		data = data[readBytes:]
	}

	// right
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		n.right = nil
	} else {
		_, err = decBinary(data, n.right)
		if err != nil {
			return err
		}
		//data = data[readBytes:]
	}

	return nil
}
