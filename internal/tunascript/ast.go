package tunascript

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// AST is an AST of parsed (but not interpreted) block of
// Tunascript code. The zero-value of an AST is not suitable for
// use, and they should only be created by calls to ParseExpression.
type AST struct {
	children []*AST
	sym      symbol
	t        nodeType
}

type NewAST struct {
	nodes []astNode
}

type astNode struct {
	fn    *fnNode
	flag  *flagNode
	value *valueNode
}

type fnNode struct {
	name string
	args []*NewAST
}

type flagNode struct {
	name string
}

type valueNode struct {
	source   string
	forceStr bool
}

/*
// TunascriptString converts the AST into Tunascript source code. The returned
// source is guaranteed to be syntactically identical to the code that the AST
// was parsed from (i.e. parsing will return an identical AST), but it may not
// be exactly the same string; coding style is not preserved.
func (ast AST) TunascriptString() string {
	var tsCode strings.Builder
}*/

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

type symbol struct {
	sType  symbolType
	source string

	// only applies if sType is symbolValue
	forceStr bool
}

// MarshalBinary always returns a nil error.
func (sym symbol) MarshalBinary() ([]byte, error) {
	data := encBinaryInt(int(sym.sType))                // 8
	data = append(data, encBinaryString(sym.source)...) // 8+
	data = append(data, encBinaryBool(sym.forceStr)...) // 1
	return data, nil
}

func (sym *symbol) UnmarshalBinary(data []byte) error {
	var err error
	var iVal int
	var readBytes int

	iVal, readBytes, err = decBinaryInt(data)
	if err != nil {
		return err
	}
	sym.sType = symbolType(iVal)
	if sym.sType != symbolDollar && sym.sType != symbolIdentifier && sym.sType != symbolValue {
		return fmt.Errorf("bad symbol type")
	}
	data = data[readBytes:]

	sym.source, readBytes, err = decBinaryString(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]

	sym.forceStr, _, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	//data = data[readBytes:]

	return nil
}

type lexerState int

const (
	lexDefault lexerState = iota
	lexIdent
	lexStr
)

type nodeType int

const (
	nodeItem nodeType = iota
	nodeGroup
	nodeRoot
)

// indexOfMatchingTunascriptParen takes the given string, which must start with
// a parenthesis char "(", and returns the index of the ")" that matches it. Any
// text in between is analyzed for other parenthesis and if they are there, they
// must be matched as well.
//
// Index will be -1 if it does not have a match.
// Error is non-nil if there is malformed tunascript syntax between the parens,
// of if s cannot be operated on.
//
// Also returns the parsable AST of the analyzed expression as well.
func indexOfMatchingParen(sRunes []rune) (int, *AST, error) {
	// without a parent node on a paren scan, buildAST will produce an error.
	dummyNode := &AST{
		children: make([]*AST, 0),
	}

	if sRunes[0] != '(' {
		var errStr string
		if len(sRunes) > 50 {
			errStr = string(sRunes[:50]) + "..."
		} else {
			errStr = string(sRunes)
		}
		return 0, nil, fmt.Errorf("no opening paren at start of analysis string %q", errStr)
	}

	if len(sRunes) < 2 {
		return 0, nil, fmt.Errorf("unexpected end of expression (unmatched left-parenthesis)")
	}

	exprNode, consumed, err := buildAST(string(sRunes[1:]), dummyNode)
	if err != nil {
		return 0, nil, err
	}
	exprNode.t = nodeRoot

	return consumed, exprNode, nil
}

// ParseText interprets the text in the abstract syntax tree and evaluates it.
func (inter Interpreter) evalExpr(ast *AST, queryOnly bool) ([]Value, error) {
	if ast.t != nodeRoot && ast.t != nodeGroup {
		return nil, fmt.Errorf("cannot parse AST anywhere besides root of the tree")
	}

	values := []Value{}

	for i := 0; i < len(ast.children); i++ {
		child := ast.children[i]
		if child.t == nodeGroup {
			return nil, fmt.Errorf("unexpected parenthesis group\n(to use \"(\" and \")\" as Str values, put them in between two \"|\" chars or\nescape them)")
		}
		if child.sym.sType == symbolIdentifier {
			// TODO: move this into AST builder,
			// for now just convert these here
			// an identifier on its own is rly just a val.
			child.sym.sType = symbolValue
		}

		if child.sym.sType == symbolDollar {
			// okay, check ahead for an ident
			if i+1 >= len(ast.children) {
				return nil, fmt.Errorf("unexpected bare \"$\" character at end\n(to use \"$\" as a  Str value, put it in between two \"|\" chars or escape it)")
			}
			identNode := ast.children[i+1]
			if identNode.t == nodeGroup {
				return nil, fmt.Errorf("unexpected parenthesis group after \"$\" character, expected identifier")
			}
			if identNode.sym.sType != symbolIdentifier {
				return nil, fmt.Errorf("unexpected %s after \"$\" character, expected identifier", identNode.sym.sType.String())
			}

			// we now have an identifier, but is this a var or function?
			isFunc := false
			if i+2 < len(ast.children) {
				argsNode := ast.children[i+2]
				if argsNode.t == nodeGroup {
					isFunc = true
				}
			}

			if isFunc {
				// function call, gather args and call it
				argsNode := ast.children[i+2]
				funcName := strings.ToUpper(identNode.sym.source)

				fn, ok := inter.fn[funcName]
				if !ok {
					return nil, fmt.Errorf("function $%s() does not exist", funcName)
				}

				// restrict if requested
				if queryOnly && fn.SideEffects {
					return nil, fmt.Errorf("function $%s() will change game state", funcName)
				}

				args, err := inter.evalExpr(argsNode, queryOnly)
				if err != nil {
					return nil, err
				}

				if len(args) < fn.RequiredArgs {
					s := "s"
					if fn.RequiredArgs == 1 {
						s = ""
					}
					return nil, fmt.Errorf("function $%s() requires at least %d parameter%s; %d given", fn.Name, fn.RequiredArgs, s, len(args))
				}

				maxArgs := fn.RequiredArgs + fn.OptionalArgs
				if len(args) > maxArgs {
					s := "s"
					if maxArgs == 1 {
						s = ""
					}
					return nil, fmt.Errorf("function $%s() takes at most %d parameter%s; %d given", fn.Name, maxArgs, s, len(args))
				}

				// oh yeah. no error returned. you saw that right. the Call function is literally not allowed to fail.
				// int 100 bbyyyyyyyyyyyyyyyyyyy
				//
				// Oh my gog ::::/
				//
				// i AM ur gog now >38D
				v := fn.Call(args)
				values = append(values, v)
				i += 2
			} else {
				// flag substitution, read it in
				flagName := strings.ToUpper(identNode.sym.source)

				var v Value
				flag, ok := inter.flags[flagName]
				if ok {
					v = flag.Value
				} else {
					v = NewStr("")
				}
				values = append(values, v)
				i++
			}
		} else if child.sym.sType == symbolValue {
			src := child.sym.source
			var v Value
			if child.sym.forceStr {
				v = NewStr(src)
			} else {
				v = parseUntypedValString(src)
			}
			values = append(values, v)
		}
	}
	return values, nil
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

// LexText lexes the text. Returns the AST, number of runes consumed
// and whether an error was encountered.
func buildAST(s string, parent *AST) (*AST, int, error) {
	node := &AST{children: make([]*AST, 0)}

	escaping := false
	mode := lexDefault
	node.t = nodeItem

	s = strings.TrimSpace(s)
	sRunes := []rune(s)
	sBytes := make([]int, len(sRunes))
	sBytesIdx := 0
	for b := range s {
		sBytes[sBytesIdx] = b
		sBytesIdx++
	}

	var buildingText string
	for i := 0; i < len(sRunes); i++ {
		ch := sRunes[i]
		//chS := string(ch)
		//fmt.Println(chS)

		switch mode {
		case lexIdent:
			if ('A' <= ch && ch <= 'Z') || ('a' <= ch && ch <= 'z') || ('0' <= ch && ch <= '9') || ch == '_' {
				buildingText += string(ch)
			} else {
				idNode := &AST{
					sym: symbol{sType: symbolIdentifier, source: buildingText},
				}
				node.children = append(node.children, idNode)
				buildingText = ""
				i--
				mode = lexDefault
			}
		case lexStr:
			if !escaping && ch == '\\' {
				// do not add a node for this
				escaping = true
			} else if escaping && ch == 'n' {
				buildingText += "\n"
				escaping = false
			} else if escaping && ch == 't' {
				buildingText += "\t"
				escaping = false
			} else if !escaping && ch == '|' {
				symNode := &AST{
					sym: symbol{sType: symbolValue, source: buildingText, forceStr: true},
				}
				node.children = append(node.children, symNode)
				buildingText = ""
				mode = lexDefault
			} else {
				buildingText += string(ch)
				escaping = false
			}
		case lexDefault:
			if !escaping && ch == '\\' {
				// do not add a node for this
				escaping = true
			} else if !escaping && ch == '$' {
				if buildingText != "" {
					textNode := &AST{sym: symbol{sType: symbolValue, source: buildingText}}
					node.children = append(node.children, textNode)
					buildingText = ""
				}

				dNode := &AST{
					sym: symbol{sType: symbolDollar, source: "$"},
				}
				node.children = append(node.children, dNode)
				mode = lexIdent
			} else if !escaping && ch == '(' {
				if buildingText != "" {
					textNode := &AST{sym: symbol{sType: symbolValue, source: buildingText}}
					node.children = append(node.children, textNode)
					buildingText = ""
				}

				if i+1 >= len(sRunes) {
					return nil, 0, fmt.Errorf("unexpected end of expression (unmatched left-parenthesis)")
				}
				nextByteIdx := sBytes[i+1]

				subNode, consumed, err := buildAST(s[nextByteIdx:], node)
				if err != nil {
					return nil, 0, err
				}
				subNode.t = nodeGroup

				node.children = append(node.children, subNode)
				i += consumed
			} else if !escaping && ch == ',' {
				if buildingText != "" {
					textNode := &AST{sym: symbol{sType: symbolValue, source: buildingText}}
					node.children = append(node.children, textNode)
					buildingText = ""
				}

				// comma breaks up values but doesn't actually need to be its own node
			} else if !escaping && ch == '|' {
				if buildingText != "" {
					textNode := &AST{sym: symbol{sType: symbolValue, source: buildingText}}
					node.children = append(node.children, textNode)
					buildingText = ""
				}

				// string start
				mode = lexStr
			} else if !escaping && ch == ')' {
				// we have reached the end of our parsing. if we are the PARENT,
				// this is an error
				if parent == nil {
					return nil, 0, fmt.Errorf("unexpected end of expression (unmatched right-parenthesis)")
				}

				if buildingText != "" {
					textNode := &AST{sym: symbol{sType: symbolValue, source: buildingText}}
					node.children = append(node.children, textNode)
					buildingText = ""
				}

				// don't add it as a node, it's not relevant

				return node, i + 1, nil
			} else if escaping && ch == 'n' {
				buildingText += "\n"
				escaping = false
			} else if escaping && ch == 't' {
				buildingText += "\t"
				escaping = false
			} else {
				if escaping || !unicode.IsSpace(ch) {
					buildingText += string(ch)
				}
				escaping = false
			}
		}

	}

	// if we get to the end but we are not the parent, we have a paren mismatch
	if parent != nil {
		return nil, 0, fmt.Errorf("unexpected end of expression (unmatched left-parenthesis)")
	}

	node.t = nodeRoot

	if mode == lexDefault {
		if buildingText != "" {
			textNode := &AST{sym: symbol{sType: symbolValue, source: buildingText}}
			node.children = append(node.children, textNode)
			buildingText = ""
		}
	} else if mode == lexIdent {
		if buildingText != "" {
			textNode := &AST{sym: symbol{sType: symbolIdentifier, source: buildingText}}
			node.children = append(node.children, textNode)
			buildingText = ""
		}
	} else if mode == lexStr {
		return nil, 0, fmt.Errorf("unexpected end of expression (unmatched string start \"|\")")
	}

	// okay now go through and update
	// - make the values not have double quotes but force to str type

	return node, len(s), nil
}

// NOTE: does NOT set root, caller needs to set that themself since cannot know
func (ast AST) MarshalBinary() ([]byte, error) {
	var data []byte

	// children count
	data = append(data, encBinaryInt(len(ast.children))...)

	// each child
	for i := range ast.children {
		child := ast.children[i]
		data = append(data, encBinary(*child)...)
	}

	// the symbol
	data = append(data, encBinary(ast.sym)...)

	// node type
	data = append(data, encBinaryInt(int(ast.t))...)

	return data, nil
}

// NOTE: does NOT set root, caller needs to set that themself since cannot know
func (ast *AST) UnmarshalBinary(data []byte) error {
	var err error
	var readBytes int
	var childCount int
	var tVal int

	// children count
	childCount, readBytes, err = decBinaryInt(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]

	// each child
	for i := 0; i < childCount; i++ {
		var subAST *AST
		readBytes, err := decBinary(data, subAST)
		if err != nil {
			return err
		}
		data = data[readBytes:]

		// need to recursively tell all children who parent is

		ast.children = append(ast.children, subAST)
	}

	// the symbol
	readBytes, err = decBinary(data, &ast.sym)
	if err != nil {
		return err
	}
	data = data[readBytes:]

	// node type
	tVal, _, err = decBinaryInt(data)
	if err != nil {
		return err
	}
	ast.t = nodeType(tVal)

	if ast.t != nodeGroup && ast.t != nodeItem && ast.t != nodeRoot {
		return fmt.Errorf("unknown AST node type")
	}

	return nil
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

func (ast NewAST) MarshalBinary() ([]byte, error) {
	var data []byte

	// count
	data = append(data, encBinaryInt(len(ast.nodes))...)
	for i := range ast.nodes {
		data = append(data, encBinary(ast.nodes[i])...)
	}

	return data, nil
}

func (ast *NewAST) UnmarshalBinary(data []byte) error {
	var err error
	var readBytes int
	var nodeCount int

	// get count
	nodeCount, readBytes, err = decBinaryInt(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]

	// get each arg
	for i := 0; i < nodeCount; i++ {
		var n astNode
		readBytes, err = decBinary(data, &n)
		if err != nil {
			return err
		}
		data = data[readBytes:]

		ast.nodes = append(ast.nodes, n)
	}

	return nil
}

func (node astNode) MarshalBinary() ([]byte, error) {
	var data []byte

	// fn ptr
	if node.fn == nil {
		data = append(data, encBinaryBool(false)...)
	} else {
		data = append(data, encBinaryBool(true)...)
		data = append(data, encBinary(*node.fn)...)
	}

	// flag ptr
	if node.flag == nil {
		data = append(data, encBinaryBool(false)...)
	} else {
		data = append(data, encBinaryBool(true)...)
		data = append(data, encBinary(*node.flag)...)
	}

	// value ptr
	if node.value == nil {
		data = append(data, encBinaryBool(false)...)
	} else {
		data = append(data, encBinaryBool(true)...)
		data = append(data, encBinary(*node.value)...)
	}

	return data, nil
}

func (node *astNode) UnmarshalBinary(data []byte) error {
	var err error
	var readBytes int
	var isNil bool

	// fn
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		node.fn = nil
	} else {
		var fnVal fnNode
		readBytes, err := decBinary(data, &fnVal)
		if err != nil {
			return err
		}
		data = data[readBytes:]

		node.fn = &fnVal
	}

	// flag
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		node.fn = nil
	} else {
		var flagVal flagNode
		readBytes, err := decBinary(data, &flagVal)
		if err != nil {
			return err
		}
		data = data[readBytes:]

		node.flag = &flagVal
	}

	// value
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		node.fn = nil
	} else {
		var valVal valueNode
		_, err := decBinary(data, &valVal)
		if err != nil {
			return err
		}
		//data = data[readBytes:]

		node.value = &valVal
	}

	return nil
}

func (node fnNode) MarshalBinary() ([]byte, error) {
	var data []byte

	data = append(data, encBinaryString(node.name)...)

	// count
	data = append(data, encBinaryInt(len(node.args))...)
	for i := range node.args {
		data = append(data, encBinary(*node.args[i])...)
	}

	return data, nil
}

func (node *fnNode) UnmarshalBinary(data []byte) error {
	var err error
	var readBytes int
	var argCount int

	node.name, readBytes, err = decBinaryString(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]

	// get count
	argCount, readBytes, err = decBinaryInt(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]

	// get each arg
	for i := 0; i < argCount; i++ {
		var argNode NewAST
		readBytes, err = decBinary(data, &argNode)
		if err != nil {
			return err
		}
		data = data[readBytes:]

		node.args = append(node.args, &argNode)
	}

	return nil
}

func (node flagNode) MarshalBinary() ([]byte, error) {
	var data []byte

	data = append(data, encBinaryString(node.name)...)

	return data, nil
}

func (node *flagNode) UnmarshalBinary(data []byte) error {
	var err error
	//var readBytes int

	node.name, _, err = decBinaryString(data)
	if err != nil {
		return err
	}
	//data = data[readBytes:]

	return nil
}

func (node valueNode) MarshalBinary() ([]byte, error) {
	var data []byte

	data = append(data, encBinaryString(node.source)...)
	data = append(data, encBinaryBool(node.forceStr)...)

	return data, nil
}

func (node *valueNode) UnmarshalBinary(data []byte) error {
	var err error
	var readBytes int

	node.source, readBytes, err = decBinaryString(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]

	node.forceStr, _, err = decBinaryBool(data)
	if err != nil {
		return err
	}

	return nil
}
