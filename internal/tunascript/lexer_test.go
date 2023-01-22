package tunascript

import (
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
		{
			name:   "blank string",
			input:  "",
			expect: []tokenClass{tsEndOfText},
		},
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

			assert.Equal(tc.expect, actual)
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
