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
