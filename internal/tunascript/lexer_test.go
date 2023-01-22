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
		{name: "blank string", input: "", expect: []tokenClass{tsEndOfText}},
		{name: "1 digit number", input: "1", expect: []tokenClass{tsNumber, tsEndOfText}},
		{name: "2 digit number", input: "39", expect: []tokenClass{tsNumber, tsEndOfText}},
		{name: "3 digit number", input: "026", expect: []tokenClass{tsNumber, tsEndOfText}},
		{name: "4 digit number", input: "4578", expect: []tokenClass{tsNumber, tsEndOfText}},
		{name: "2 numbers", input: "3284 1384", expect: []tokenClass{tsNumber, tsNumber, tsEndOfText}},
		{name: "we dont do decimals, thats a string", input: "13.4", expect: []tokenClass{tsUnquotedString, tsEndOfText}},
		{name: "bool on", input: "on", expect: []tokenClass{tsBool, tsEndOfText}},
		{name: "bool off", input: "OFF", expect: []tokenClass{tsBool, tsEndOfText}},
		{name: "bool true", input: "tRuE", expect: []tokenClass{tsBool, tsEndOfText}},
		{name: "bool false", input: "False", expect: []tokenClass{tsBool, tsEndOfText}},
		{name: "bool yes", input: "yeS", expect: []tokenClass{tsBool, tsEndOfText}},
		{name: "bool no", input: "no", expect: []tokenClass{tsBool, tsEndOfText}},
		{name: "some string", input: "fdksalfjaskldfj", expect: []tokenClass{tsUnquotedString, tsEndOfText}},
		{name: "a few random values", input: "no yes test string 3 eight", expect: []tokenClass{tsBool, tsBool, tsUnquotedString, tsUnquotedString, tsNumber, tsUnquotedString, tsEndOfText}},
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
