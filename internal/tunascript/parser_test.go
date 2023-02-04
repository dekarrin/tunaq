package tunascript

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	mockedBoolLexeme           = "true"
	mockedNumberLexeme         = "413"
	mockedQuotedStringLexeme   = "@ ROSE LALONDE @"
	mockedUnquotedStringLexeme = "JADE HARLEY"
	mockedIdentifierLexeme     = "$VRISKA"
)

func mockTokens(ofClass ...tokenClass) []token {
	buildingLine := ""

	var lineTokens = make([]token, 0)

	const lineEvery = 100

	curLine := 1
	curLinePos := 1
	var mocked []token
	for i := range ofClass {
		m := token{class: ofClass[i], line: curLine, pos: curLinePos}
		switch ofClass[i] {
		case tsBool:
			m.lexeme = mockedBoolLexeme
		case tsNumber:
			m.lexeme = mockedNumberLexeme
		case tsIdentifier:
			m.lexeme = mockedIdentifierLexeme
		case tsUnquotedString:
			m.lexeme = mockedUnquotedStringLexeme
		case tsQuotedString:
			m.lexeme = mockedQuotedStringLexeme
		case tsSeparator:
			m.lexeme = literalStrSeparator
		case tsGroupOpen:
			m.lexeme = literalStrGroupOpen
		case tsGroupClose:
			m.lexeme = literalStrGroupClose
		case tsOpAnd:
			m.lexeme = literalStrOpAnd
		case tsOpDec:
			m.lexeme = literalStrOpDec
		case tsOpDecset:
			m.lexeme = literalStrOpDecset
		case tsOpDivide:
			m.lexeme = literalStrOpDivide
		case tsOpGreaterThan:
			m.lexeme = literalStrOpGreaterThan
		case tsOpGreaterThanIs:
			m.lexeme = literalStrOpGreaterThanIs
		case tsOpInc:
			m.lexeme = literalStrOpInc
		case tsOpIncset:
			m.lexeme = literalStrOpIncset
		case tsOpIs:
			m.lexeme = literalStrOpIs
		case tsOpIsNot:
			m.lexeme = literalStrOpIsNot
		case tsOpLessThan:
			m.lexeme = literalStrOpLessThan
		case tsOpLessThanIs:
			m.lexeme = literalStrOpLessThanIs
		case tsOpMinus:
			m.lexeme = literalStrOpMinus
		case tsOpMultiply:
			m.lexeme = literalStrOpMultiply
		case tsOpNot:
			m.lexeme = literalStrOpNot
		case tsOpOr:
			m.lexeme = literalStrOpOr
		case tsOpPlus:
			m.lexeme = literalStrOpPlus
		case tsOpSet:
			m.lexeme = literalStrOpSet
		case tsEndOfText:
			fallthrough
		case tsWhitespace:
			fallthrough
		case tsUndefined:
			// deliberately blank
		}

		lineTokens = append(lineTokens, m)

		if ofClass[i] != tsEndOfText && ofClass[i] != tsWhitespace && ofClass[i] != tsUndefined {
			buildingLine += m.lexeme + " "
			curLinePos += len(m.lexeme) + 1 // for the space
		}
		if i > 0 && i%lineEvery == 0 {
			// this is a full line

			buildingLine = buildingLine[:len(buildingLine)-1] // trailing space
			for j := range lineTokens {
				m := lineTokens[j]
				m.fullLine = buildingLine
				mocked = append(mocked, m)
			}
			buildingLine = ""
			curLine++
			curLinePos = 1
			lineTokens = make([]token, 0)
		}
	}

	if len(lineTokens) > 0 {
		// this is a partial line
		if len(buildingLine) > 0 {
			buildingLine = buildingLine[:len(buildingLine)-1] // trailing space
		}
		for i := range lineTokens {
			m := lineTokens[i]
			m.fullLine = buildingLine
			mocked = append(mocked, m)
		}
	}

	return mocked
}

/*
func Test_Parse(t *testing.T) {
	testCases := []struct {
		name      string
		input     []tokenClass
		expect    string
		expectErr bool
	}{
		{
			name: "bool node",
			input: []tokenClass{
				tsBool, tsEndOfText,
			},
			expect: `(AST)
  \---: (BOOL_VALUE "` + mockedBoolLexeme + `")`,
		},
		{
			name: "num node",
			input: []tokenClass{
				tsNumber, tsEndOfText,
			},
			expect: `(AST)
  \---: (NUM_VALUE "` + mockedNumberLexeme + `")`,
		},
		{
			name: "quoted string node",
			input: []tokenClass{
				tsQuotedString, tsEndOfText,
			},
			expect: `(AST)
  \---: (QSTR_VALUE "` + mockedQuotedStringLexeme + `")`,
		},
		{
			name: "unquoted string node",
			input: []tokenClass{
				tsUnquotedString, tsEndOfText,
			},
			expect: `(AST)
  \---: (STR_VALUE "` + mockedUnquotedStringLexeme + `")`,
		},
		{
			name: "undefined fails to parse",
			input: []tokenClass{
				tsUndefined, tsEndOfText,
			},
			expectErr: true,
		}, commenting while we get parser stuff reworked
		{
			name: "$FN(text, bool, $FN($FLAG == (num + num) * $FUNC() || bool && -num / num), num, text += @at text@)",
			input: []tokenClass{
				tsIdentifier, tsGroupOpen, tsUnquotedString, tsSeparator, tsBool, tsSeparator, tsIdentifier,
				tsGroupOpen, tsIdentifier, tsOpIs, tsGroupOpen, tsNumber, tsOpPlus, tsNumber, tsGroupClose,
				tsOpMultiply, tsIdentifier, tsGroupOpen, tsGroupClose, tsOpOr, tsBool, tsOpAnd, tsOpMinus, tsNumber,
				tsOpDivide, tsNumber, tsGroupClose, tsSeparator, tsNumber, tsSeparator, tsUnquotedString, tsOpIncset,
				tsQuotedString, tsGroupClose, tsEndOfText,
			},
			expect: `(AST)
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)

			inputTokenStream := tokenStream{
				tokens: mockTokens(tc.input...),
			}

			actualAST, err := Parse(inputTokenStream)
			if tc.expectErr {
				assert.Error(err)
				return
			}
			assert.Nil(err)
			if err != nil {
				if se, ok := err.(SyntaxError); ok {
					fmt.Print(se.FullMessage() + "\n")
				}
			}

			// AST.String() is specifically defined to have a 1-to-1 relation
			// with semantic equality, which is the thing we care about.
			actual := actualAST.String()

			assert.Equal(tc.expect, actual)
		})
	}
}
*/

var (
	// the operations in the grammars used as examples from Prof. Aiken's
	// video series on compilers, represented as results of tunascript
	// tokenClasses so we can use them in test functions.
	termNumber = strings.ToLower(tsNumber.id)
	termPlus   = strings.ToLower(tsOpPlus.id)
	termMult   = strings.ToLower(tsOpMultiply.id)
	termLParen = strings.ToLower(tsGroupOpen.id)
	termRParen = strings.ToLower(tsGroupClose.id)
)

func Test_LL1PredictiveParse(t *testing.T) {
	testCases := []struct {
		name      string
		grammar   string
		input     []tokenClass
		expect    string
		expectErr bool
	}{
		{
			name: "aiken expression LL1 sample",
			grammar: `
				S -> T X ;

				T -> ` + termLParen + ` S ` + termRParen + `
				   | ` + termNumber + ` Y ;

				X -> ` + termPlus + ` S
				   | ε ;

				Y -> ` + termMult + ` T
				   | ε ;
			`,
			input: []tokenClass{
				tsNumber, tsOpMultiply, tsNumber, tsEndOfText,
			},
			expect: "( S )\n" +
				`  |---: ( T )` + "\n" +
				`  |       |---: (TERM "` + termNumber + `")` + "\n" +
				`  |       \---: ( Y )` + "\n" +
				`  |               |---: (TERM "` + termMult + `")` + "\n" +
				`  |               \---: ( T )` + "\n" +
				`  |                       |---: (TERM "` + termNumber + `")` + "\n" +
				`  |                       \---: ( Y )` + "\n" +
				`  |                               \---: (TERM "")` + "\n" +
				`  \---: ( X )` + "\n" +
				`          \---: (TERM "")`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// setup
			assert := assert.New(t)

			g := mustParseGrammar(tc.grammar)
			stream := tokenStream{
				tokens: mockTokens(tc.input...),
			}

			// execute
			actual, err := LL1PredictiveParse(g, stream)

			// assert
			if tc.expectErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)

			assert.Equal(tc.expect, actual.String())
		})
	}
}

func ref[T any](s T) *T {
	return &s
}

/*
type wi struct {
	inInven bool
	move    bool
	output  bool
}

func (w wi) InInventory(label string) bool {
	return w.inInven
}

func (w wi) Move(label string, dest string) bool {
	return w.move
}

func (w wi) Output(str string) bool {
	return w.output
}

func worldInterFixture() WorldInterface {
	return wi{}
}
*/
