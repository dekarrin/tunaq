package tunascript

import (
	"encoding"
	"encoding/binary"
	"fmt"
	"strings"
	"unicode/utf8"
)

// This file contains the format for binary encoding of ASTs.

func encBinaryBool(b bool) []byte {
	enc := make([]byte, 1)

	if b {
		enc[0] = 1
	} else {
		enc[0] = 0
	}

	return enc
}

func encBinaryString(s string) []byte {
	enc := make([]byte, 0)

	chCount := 0
	for _, ch := range s {
		chBuf := make([]byte, utf8.UTFMax)
		byteLen := utf8.EncodeRune(chBuf, ch)
		enc = append(enc, chBuf[:byteLen]...)
		chCount++
	}

	countBytes := encBinaryInt(chCount)
	enc = append(countBytes, enc...)

	return enc
}

func encBinaryInt(i int) []byte {
	enc := make([]byte, 8)
	enc = binary.AppendVarint(enc, int64(i))
	return enc
}

func encBinary(b encoding.BinaryMarshaler) []byte {
	enc, _ := b.MarshalBinary()

	enc = append(encBinaryInt(len(enc)), enc...)

	return enc
}

// always consumes 1 byte.
func decBinaryBool(data []byte) (bool, int, error) {
	if len(data) < 1 {
		return false, 0, fmt.Errorf("unexpected end of data")
	}

	if data[0] == 0 {
		return false, 0, nil
	} else if data[0] == 1 {
		return true, 0, nil
	} else {
		return false, 0, fmt.Errorf("unknown non-bool value")
	}
}

// returns the string followed by bytes consumed
func decBinaryString(data []byte) (string, int, error) {
	if len(data) < 8 {
		return "", 0, fmt.Errorf("unexpected end of data")
	}
	runeCount, _, err := decBinaryInt(data)
	if err != nil {
		return "", 0, fmt.Errorf("decoding string rune count: %w", err)
	}
	data = data[8:]

	if runeCount < 0 {
		return "", 0, fmt.Errorf("string rune count < 0")
	}

	readBytes := 8

	var sb strings.Builder

	for i := 0; i < runeCount; i++ {
		ch, bytesRead := utf8.DecodeRune(data)
		if ch == utf8.RuneError {
			if bytesRead == 0 {
				return "", 0, fmt.Errorf("unexpected end of data in string")
			} else if bytesRead == 1 {
				return "", 0, fmt.Errorf("invalid UTF-8 encoding in string")
			} else {
				return "", 0, fmt.Errorf("invalid unicode replacement character in rune")
			}
		}

		sb.WriteRune(ch)
		readBytes += bytesRead
		data = data[bytesRead:]
	}

	return sb.String(), readBytes, nil
}

// will always read 8 bytes but does return len
func decBinaryInt(data []byte) (int, int, error) {
	if len(data) < 8 {
		return 0, 0, fmt.Errorf("data does not contain 8 bytes")
	}

	val, read := binary.Varint(data[:8])
	if read == 0 {
		return 0, 0, fmt.Errorf("input buffer too small, should never happen")
	} else if read < 0 {
		return 0, 0, fmt.Errorf("input buffer contains value larger than 64 bits, should never happen")
	}
	return int(val), 8, nil
}

func decBinary(data []byte, b encoding.BinaryUnmarshaler) (int, error) {
	var readBytes int
	var byteLen int
	var err error

	byteLen, readBytes, err = decBinaryInt(data)
	if err != nil {
		return 0, err
	}
	data = data[readBytes:]

	if len(data) < byteLen {
		return 0, fmt.Errorf("unexpected end of data")
	}
	binData := data[:byteLen]

	err = b.UnmarshalBinary(binData)
	if err != nil {
		return 0, err
	}

	return byteLen + readBytes, nil
}

func (lex token) MarshalBinary() ([]byte, error) {
	var data []byte

	data = append(data, encBinaryString(lex.lexeme)...)
	data = append(data, encBinary(lex.class)...)
	data = append(data, encBinaryInt(lex.pos)...)
	data = append(data, encBinaryInt(lex.line)...)
	data = append(data, encBinaryString(lex.fullLine)...)

	return data, nil
}

func (lex *token) UnmarshalBinary(data []byte) error {
	var err error
	var bytesRead int

	lex.lexeme, bytesRead, err = decBinaryString(data)
	if err != nil {
		return err
	}
	data = data[bytesRead:]

	bytesRead, err = decBinary(data, &lex.class)
	if err != nil {
		return err
	}
	data = data[bytesRead:]

	lex.pos, bytesRead, err = decBinaryInt(data)
	if err != nil {
		return err
	}
	data = data[bytesRead:]

	lex.line, bytesRead, err = decBinaryInt(data)
	if err != nil {
		return err
	}
	data = data[bytesRead:]

	lex.fullLine, _, err = decBinaryString(data)
	if err != nil {
		return err
	}
	//data = data[bytesRead:]

	return nil
}

func (sym tokenClass) MarshalBinary() ([]byte, error) {
	var data []byte

	data = append(data, encBinaryString(sym.id)...)
	data = append(data, encBinaryString(sym.human)...)
	data = append(data, encBinaryInt(sym.lbp)...)

	return data, nil
}

func (sym *tokenClass) UnmarshalBinary(data []byte) error {
	var err error
	var bytesRead int

	sym.id, bytesRead, err = decBinaryString(data)
	if err != nil {
		return err
	}
	data = data[bytesRead:]

	sym.human, bytesRead, err = decBinaryString(data)
	if err != nil {
		return err
	}
	data = data[bytesRead:]

	sym.lbp, _, err = decBinaryInt(data)
	if err != nil {
		return err
	}
	// data = data[bytesRead:]

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
		var node expASTNode
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

func (etn expASTNode) MarshalBinary() ([]byte, error) {
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

	data = append(data, encBinary(etn.source)...)

	return data, nil
}

func (etn *expASTNode) UnmarshalBinary(data []byte) error {
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
		readBytes, err := decBinary(data, &flagVal)
		if err != nil {
			return err
		}
		data = data[readBytes:]

		etn.flag = &flagVal
	}

	_, err = decBinary(data, &etn.source)
	if err != nil {
		return err
	}
	// data = data[readBytes:]

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

func (es expSource) MarshalBinary() ([]byte, error) {
	var data []byte

	data = append(data, encBinaryString(es.text)...)
	data = append(data, encBinaryString(es.fullLine)...)
	data = append(data, encBinaryInt(es.line)...)
	data = append(data, encBinaryInt(es.pos)...)

	return data, nil
}

func (es *expSource) UnmarshalBinary(data []byte) error {
	var err error
	var bytesRead int

	es.text, bytesRead, err = decBinaryString(data)
	if err != nil {
		return err
	}
	data = data[bytesRead:]

	es.fullLine, bytesRead, err = decBinaryString(data)
	if err != nil {
		return err
	}
	data = data[bytesRead:]

	es.line, bytesRead, err = decBinaryInt(data)
	if err != nil {
		return err
	}
	data = data[bytesRead:]

	es.pos, _, err = decBinaryInt(data)
	if err != nil {
		return err
	}
	//data = data[bytesRead:]

	return nil
}

/* 	source expString
}

type expString struct {
	text       string
	sourceLine string
	line       int
	pos        int
}*/
