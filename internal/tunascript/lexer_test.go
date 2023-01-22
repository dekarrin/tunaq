package tunascript

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Lex_tokenClassSequence(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		expect    []tokenClass
		expectErr bool
	}{
		{name: "blank string", input: "", expect: []tokenClass{
			tsEndOfText,
		}},
		{name: "1 digit number", input: "1", expect: []tokenClass{
			tsNumber, tsEndOfText,
		}},
		{name: "2 digit number", input: "39", expect: []tokenClass{
			tsNumber, tsEndOfText,
		}},
		{name: "3 digit number", input: "026", expect: []tokenClass{
			tsNumber, tsEndOfText,
		}},
		{name: "4 digit number", input: "4578", expect: []tokenClass{
			tsNumber, tsEndOfText,
		}},
		{name: "2 numbers", input: "3284 1384", expect: []tokenClass{
			tsUnquotedString, tsEndOfText,
		}},
		{name: "we dont do decimals, thats a string", input: "13.4", expect: []tokenClass{
			tsUnquotedString, tsEndOfText,
		}},
		{name: "negative number is actually 2 tokens", input: "-12", expect: []tokenClass{
			tsOpMinus, tsNumber, tsEndOfText,
		}},
		{name: "bool on", input: "on", expect: []tokenClass{
			tsBool, tsEndOfText,
		}},
		{name: "bool off", input: "OFF", expect: []tokenClass{
			tsBool, tsEndOfText,
		}},
		{name: "bool true", input: "tRuE", expect: []tokenClass{
			tsBool, tsEndOfText,
		}},
		{name: "bool false", input: "False", expect: []tokenClass{
			tsBool, tsEndOfText,
		}},
		{name: "bool yes", input: "yeS", expect: []tokenClass{
			tsBool, tsEndOfText,
		}},
		{name: "bool no", input: "no", expect: []tokenClass{
			tsBool, tsEndOfText,
		}},
		{name: "some string", input: "fdksalfjaskldfj", expect: []tokenClass{
			tsUnquotedString, tsEndOfText,
		}},
		{name: "a few random values", input: "no yes test string 3 eight", expect: []tokenClass{
			tsUnquotedString, tsEndOfText,
		}},
		{name: "flag", input: "$SOME_FLAG", expect: []tokenClass{
			tsIdentifier, tsEndOfText,
		}},
		{name: "quoted string", input: "@quoted@", expect: []tokenClass{
			tsQuotedString, tsEndOfText,
		}},
		{name: "quoted string with spaces", input: "@quoted   string@", expect: []tokenClass{
			tsQuotedString, tsEndOfText,
		}},
		{name: "quoted string with leading and trailing spaces", input: "@  quoted   string @", expect: []tokenClass{
			tsQuotedString, tsEndOfText,
		}},
		{name: "quoted string with escaped quote", input: `@include an \@ with escape chars!@`, expect: []tokenClass{
			tsQuotedString, tsEndOfText,
		}},
		{name: "quoted string with two escaped quotes", input: `@put a \@ then \@ again@`, expect: []tokenClass{
			tsQuotedString, tsEndOfText,
		}},
		{name: "function call, empty args", input: "$SOME_FLAG()", expect: []tokenClass{
			tsIdentifier, tsGroupOpen, tsGroupClose, tsEndOfText,
		}},
		{name: "function call, 1 arg", input: "$SOME_FUNC(2)", expect: []tokenClass{
			tsIdentifier, tsGroupOpen, tsNumber, tsGroupClose, tsEndOfText,
		}},
		{name: "function call, 2 args", input: "$000_FUNC_STARTING_WITH_NUM(2, on)", expect: []tokenClass{
			tsIdentifier, tsGroupOpen, tsNumber, tsSeparator, tsBool, tsGroupClose, tsEndOfText,
		}},
		{name: "function call, 3 args", input: "$SOME_FUNC(@a quoted string@, 8293, terezi)", expect: []tokenClass{
			tsIdentifier, tsGroupOpen, tsQuotedString, tsSeparator, tsNumber, tsSeparator, tsUnquotedString,
			tsGroupClose, tsEndOfText,
		}},
		{name: "addition", input: "1 + hello", expect: []tokenClass{
			tsNumber, tsOpPlus, tsUnquotedString, tsEndOfText,
		}},
		{name: "subtraction", input: "3 -   8", expect: []tokenClass{
			tsNumber, tsOpMinus, tsNumber, tsEndOfText,
		}},
		{name: "add negative", input: "3 +-8", expect: []tokenClass{
			tsNumber, tsOpPlus, tsOpMinus, tsNumber, tsEndOfText,
		}},
		{name: "add two strings", input: "@hello glub@ + string 2", expect: []tokenClass{
			tsQuotedString, tsOpPlus, tsUnquotedString, tsEndOfText,
		}},
		{name: "multiply numbers", input: " 2  *    8", expect: []tokenClass{
			tsNumber, tsOpMultiply, tsNumber, tsEndOfText,
		}},
		{name: "multiply string", input: "some unquoted  string * 2", expect: []tokenClass{
			tsUnquotedString, tsOpMultiply, tsNumber, tsEndOfText,
		}},
		{name: "divide numbers", input: "3 / 4", expect: []tokenClass{
			tsNumber, tsOpDivide, tsNumber, tsEndOfText,
		}},
		{name: "increment flag", input: "$A_FLAG++", expect: []tokenClass{
			tsIdentifier, tsOpInc, tsEndOfText,
		}},
		{name: "increment flag, ignore space", input: "$A_FLAG ++", expect: []tokenClass{
			tsIdentifier, tsOpInc, tsEndOfText,
		}},
		{name: "decrement flag", input: "$X--", expect: []tokenClass{
			tsIdentifier, tsOpDec, tsEndOfText,
		}},
		{name: "decrement flag, ignore space", input: "$X --", expect: []tokenClass{
			tsIdentifier, tsOpDec, tsEndOfText,
		}},
		{name: "incset flag", input: "$X += 5", expect: []tokenClass{
			tsIdentifier, tsOpIncset, tsNumber, tsEndOfText,
		}},
		{name: "incset flag, ignore space", input: "$X+=5", expect: []tokenClass{
			tsIdentifier, tsOpIncset, tsNumber, tsEndOfText,
		}},
		{name: "decset flag", input: "$X -= 5", expect: []tokenClass{
			tsIdentifier, tsOpDecset, tsNumber, tsEndOfText,
		}},
		{name: "decset flag, ignore space", input: "$X-=5", expect: []tokenClass{
			tsIdentifier, tsOpDecset, tsNumber, tsEndOfText,
		}},
		{name: "set flag", input: "$X = 249", expect: []tokenClass{
			tsIdentifier, tsOpSet, tsNumber, tsEndOfText,
		}},
		{name: "set flag, ignore space", input: "$X=test string", expect: []tokenClass{
			tsIdentifier, tsOpSet, tsUnquotedString, tsEndOfText,
		}},
		{name: "AND-ing flags", input: "$X && $Y", expect: []tokenClass{
			tsIdentifier, tsOpAnd, tsIdentifier, tsEndOfText,
		}},
		{name: "AND-ing flags, ignore space", input: "$X&&$Y", expect: []tokenClass{
			tsIdentifier, tsOpAnd, tsIdentifier, tsEndOfText,
		}},
		{name: "AND sequence", input: "$X && Off && 3 && @this is truthy@", expect: []tokenClass{
			tsIdentifier, tsOpAnd, tsBool, tsOpAnd, tsNumber, tsOpAnd, tsQuotedString, tsEndOfText,
		}},
		{name: "OR-ing flags", input: "$X || $Y", expect: []tokenClass{
			tsIdentifier, tsOpOr, tsIdentifier, tsEndOfText,
		}},
		{name: "OR-ing flags, ignore space", input: "$X||$Y", expect: []tokenClass{
			tsIdentifier, tsOpOr, tsIdentifier, tsEndOfText,
		}},
		{name: "OR sequence", input: "$X || Off || 3 || @this is truthy@", expect: []tokenClass{
			tsIdentifier, tsOpOr, tsBool, tsOpOr, tsNumber, tsOpOr, tsQuotedString, tsEndOfText,
		}},
		{name: "NOT flag", input: "!$X", expect: []tokenClass{
			tsOpNot, tsIdentifier, tsEndOfText,
		}},
		{name: "NOT flag, ignore space", input: "! $X", expect: []tokenClass{
			tsOpNot, tsIdentifier, tsEndOfText,
		}},
		{name: "mixed boolean expression", input: "($FLAG + 3 || false) && !$x || $y", expect: []tokenClass{
			tsGroupOpen, tsIdentifier, tsOpPlus, tsNumber, tsOpOr, tsBool, tsGroupClose, tsOpAnd, tsOpNot, tsIdentifier,
			tsOpOr, tsIdentifier, tsEndOfText,
		}},
		{name: "expr 1", input: "@glubglub@ - exit * 600 /($FLAG_VAR+3)+$ADD(3, 4)", expect: []tokenClass{
			tsQuotedString, tsOpMinus, tsUnquotedString, tsOpMultiply, tsNumber, tsOpDivide, tsGroupOpen, tsIdentifier,
			tsOpPlus, tsNumber, tsGroupClose, tsOpPlus, tsIdentifier, tsGroupOpen, tsNumber, tsSeparator, tsNumber,
			tsGroupClose, tsEndOfText,
		}},
		{name: "expr 2", input: "@some fin@ + 243 * b - $SOME_FUNC(glubin) * 3", expect: []tokenClass{
			tsQuotedString, tsOpPlus, tsNumber, tsOpMultiply, tsUnquotedString, tsOpMinus, tsIdentifier, tsGroupOpen,
			tsUnquotedString, tsGroupClose, tsOpMultiply, tsNumber, tsEndOfText,
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)

			actualStream, err := Lex(tc.input)
			if tc.expectErr {
				assert.Error(err)
			}
			assert.Nil(err)

			actual := make([]tokenClass, len(actualStream.tokens))
			for i := range actualStream.tokens {
				actual[i] = actualStream.tokens[i].class
			}

			expectStrings := make([]string, len(tc.expect))
			for i := range tc.expect {
				expectStrings[i] = tc.expect[i].id
			}
			actualStrings := make([]string, len(actual))
			for i := range actual {
				actualStrings[i] = actual[i].id
			}

			assert.Equal(strings.Join(expectStrings, " "), strings.Join(actualStrings, " "))
		})
	}
}

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
