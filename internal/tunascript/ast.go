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
	nodes []astNode
}

type astNode struct {
	fn    *fnNode
	flag  *flagNode
	value *valueNode
	group *AST
}

type fnNode struct {
	name string
	args []*AST
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

type lexerState int

const (
	lexDefault lexerState = iota
	lexIdent
	lexStr
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

	exprNode, consumed, err := buildAST(sRunes[1:], true)
	if err != nil {
		return 0, nil, err
	}

	return consumed, exprNode, nil
}

// ParseText interprets the text in the abstract syntax tree and evaluates it.
func (inter Interpreter) evalExpr(ast *AST, queryOnly bool) ([]Value, error) {
	values := make([]Value, len(ast.nodes))

	for i := 0; i < len(ast.nodes); i++ {
		child := ast.nodes[i]

		// what kind of a node is it?
		if child.value != nil {
			// value node
			valueNode := child.value

			var v Value
			if valueNode.forceStr {
				v = NewStr(valueNode.source)
			} else {
				v = parseUntypedValString(valueNode.source)
			}
			values[i] = v
		} else if child.flag != nil {
			// flag node
			flagNode := child.flag
			flagName := strings.ToUpper(flagNode.name)

			var v Value
			flag, ok := inter.flags[flagName]
			if ok {
				v = flag.Value
			} else {
				v = NewStr("")
			}
			values[i] = v
		} else if child.fn != nil {
			// function call node, gather args and call it
			fnNode := child.fn

			funcArgNodes := fnNode.args
			funcName := strings.ToUpper(fnNode.name)

			fn, ok := inter.fn[funcName]
			if !ok {
				return nil, fmt.Errorf("function $%s() does not exist", funcName)
			}

			// restrict if requested
			if queryOnly && fn.SideEffects {
				return nil, fmt.Errorf("function $%s() will change game state and is not allowed here", funcName)
			}

			if len(funcArgNodes) < fn.RequiredArgs {
				s := "s"
				if fn.RequiredArgs == 1 {
					s = ""
				}
				return nil, fmt.Errorf("function $%s() requires at least %d parameter%s; %d given", fn.Name, fn.RequiredArgs, s, len(funcArgNodes))
			}

			maxArgs := fn.RequiredArgs + fn.OptionalArgs
			if len(funcArgNodes) > maxArgs {
				s := "s"
				if maxArgs == 1 {
					s = ""
				}
				return nil, fmt.Errorf("function $%s() takes at most %d parameter%s; %d given", fn.Name, maxArgs, s, len(funcArgNodes))
			}

			// evaluate args:
			args := make([]Value, len(funcArgNodes))
			for argIdx := range funcArgNodes {
				argResult, err := inter.evalExpr(funcArgNodes[argIdx], queryOnly)
				if err != nil {
					return nil, err
				}

				args[argIdx] = argResult[0]
			}

			// finally call the function

			// oh yeah. no error returned. you saw that right. the Call function is literally not allowed to fail.
			// int 100 bbyyyyyyyyyyyyyyyyyyy
			//
			// Oh my gog ::::/
			//
			// i AM ur gog now >38D
			//
			// This Is Later Than The Original Comment But I Must Say I Am Glad
			// That This Portion Retained Some Use With The Redesign. If Past
			// Info Is Needed Know That There Was A Prior Set Of Designs That
			// Also Functioned Correctly So The History Of This File May Be
			// Referred To.

			v := fn.Call(args)
			values[i] = v
		} else if child.group != nil {
			// group node, make shore there is exactly one node because more
			// than that in an unqualified group is not allowed (don't know how
			// to parse multi-value into single one).
			//
			// This grouping will make more sense when/if grouping parens and
			// operators are added
			groupNode := child.group

			if len(groupNode.nodes) > 1 {
				return nil, fmt.Errorf("multiple values between parenthesis but not in function call")
			}

			var v Value
			parsedVals, err := inter.evalExpr(groupNode, queryOnly)
			if err != nil {
				return nil, err
			}
			v = parsedVals[0]
			values[i] = v
		} else {
			return nil, fmt.Errorf("empty AST node (should never happen)")
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
// and whether an error was encountered. If the input is a comma-separated list
// of expressions, they will be returned as individual nodes of a single AST.
func buildAST(sRunes []rune, hasParent bool) (*AST, int, error) {
	tree := &AST{}

	escaping := false
	mode := lexDefault

	var buildingText string // TODO: should probs be a strings.Builder

	flushPendingImplicitValueNode := func() {
		if buildingText != "" {
			valNode := astNode{value: &valueNode{source: buildingText}}
			tree.nodes = append(tree.nodes, valNode)
			buildingText = ""
		}
	}

	for i := 0; i < len(sRunes); i++ {
		ch := sRunes[i]

		switch mode {
		case lexIdent:
			if ('A' <= ch && ch <= 'Z') || ('a' <= ch && ch <= 'z') || ('0' <= ch && ch <= '9') || ch == '_' {
				buildingText += string(ch)
			} else if ch == '(' {
				// immediately after identifier, this is a list of args
				if i+1 >= len(sRunes) {
					return nil, 0, fmt.Errorf("unexpected end of expression (unmatched left-parenthesis)")
				}

				// build sub-tree from args
				subTree, consumed, err := buildAST(sRunes[i+1:], true)
				if err != nil {
					return nil, 0, err
				}

				// create function node and add all subtree nodes to this one
				funcNode := astNode{
					fn: &fnNode{
						name: buildingText,
						args: make([]*AST, len(subTree.nodes)),
					},
				}
				buildingText = ""

				for subNodeIdx := range subTree.nodes {
					funcNode.fn.args[subNodeIdx] = &AST{
						nodes: []astNode{subTree.nodes[subNodeIdx]},
					}
				}

				tree.nodes = append(tree.nodes, funcNode)
				i += consumed
				mode = lexDefault
			} else {
				flNode := astNode{
					flag: &flagNode{
						name: buildingText,
					},
				}
				buildingText = ""

				tree.nodes = append(tree.nodes, flNode)
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
				// this is an EXPLICIT text node, not implicit, so use custom
				// behavior instead of func
				valNode := astNode{
					value: &valueNode{
						source:   buildingText,
						forceStr: true,
					},
				}

				tree.nodes = append(tree.nodes, valNode)
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
				flushPendingImplicitValueNode()
				mode = lexIdent
			} else if !escaping && ch == '(' {
				flushPendingImplicitValueNode()
				// enter grouped expression

				if i+1 >= len(sRunes) {
					return nil, 0, fmt.Errorf("unexpected end of expression (unmatched left-parenthesis)")
				}

				// build sub-tree from the group
				subTree, consumed, err := buildAST(sRunes[i+1:], true)
				if err != nil {
					return nil, 0, err
				}

				// unlike func args, this subtree entirely encapsulates the group
				// so we can add it directly
				groupNode := astNode{
					group: subTree,
				}

				tree.nodes = append(tree.nodes, groupNode)
				i += consumed
			} else if !escaping && ch == ',' {
				flushPendingImplicitValueNode()
				// comma breaks up values but doesn't actually need to be its own node
			} else if !escaping && ch == '|' {
				flushPendingImplicitValueNode()
				// explicit (quoted) string start
				mode = lexStr
			} else if !escaping && ch == ')' {
				flushPendingImplicitValueNode()

				// we have reached the end of our parsing. if we are the ROOT,
				// this is an error
				if !hasParent {
					return nil, 0, fmt.Errorf("unexpected end of expression (unmatched right-parenthesis)")
				}

				// don't add ")" as own node, it's not relevant and is inferred
				// by fact that parent is about to put this either into an arg
				// list or a group node
				return tree, i + 1, nil
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

	// if we get to the end but we are not the root, we have a paren mismatch
	if hasParent {
		return nil, 0, fmt.Errorf("unexpected end of expression (unmatched left-parenthesis)")
	}

	// check final pending text
	if mode == lexDefault {
		// in the middle of getting implicit value
		flushPendingImplicitValueNode()
	} else if mode == lexIdent {
		// in the middle of getting flag
		if buildingText != "" {
			flNode := astNode{
				flag: &flagNode{
					name: buildingText,
				},
			}
			buildingText = ""
			tree.nodes = append(tree.nodes, flNode)
		}
	} else if mode == lexStr {
		// in the middle of getting explicit text value, this is an error
		return nil, 0, fmt.Errorf("unexpected end of expression (unmatched string start \"|\")")
	}

	return tree, len(sRunes), nil
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

func (ast AST) MarshalBinary() ([]byte, error) {
	var data []byte

	// count
	data = append(data, encBinaryInt(len(ast.nodes))...)
	for i := range ast.nodes {
		data = append(data, encBinary(ast.nodes[i])...)
	}

	return data, nil
}

func (ast *AST) UnmarshalBinary(data []byte) error {
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

	// group ptr
	if node.group == nil {
		data = append(data, encBinaryBool(false)...)
	} else {
		data = append(data, encBinaryBool(true)...)
		data = append(data, encBinary(*node.group)...)
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
		readBytes, err := decBinary(data, &valVal)
		if err != nil {
			return err
		}
		data = data[readBytes:]

		node.value = &valVal
	}

	// group
	isNil, readBytes, err = decBinaryBool(data)
	if err != nil {
		return err
	}
	data = data[readBytes:]
	if isNil {
		node.fn = nil
	} else {
		var astVal AST
		_, err := decBinary(data, &astVal)
		if err != nil {
			return err
		}
		//data = data[readBytes:]

		node.group = &astVal
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
		var argNode AST
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
