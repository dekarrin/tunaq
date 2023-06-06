package fe

import (
	"strings"
	"testing"

	"github.com/dekarrin/ictiobus/trans"
	"github.com/stretchr/testify/assert"
)

func fakeHook(retVal interface{}) trans.Hook {
	return func(info trans.SetterInfo, args []interface{}) (interface{}, error) {
		return retVal, nil
	}
}

var (
	fakeHooks = trans.HookMap{
		"test_const": fakeHook(0),
	}
)

func Test_Lex(t *testing.T) {
	testCases := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "bool",
			input:  "true",
			expect: ``,
		},
		{
			name:   "int num",
			input:  "88",
			expect: ``,
		},
		{
			name:   "float num",
			input:  "88.3",
			expect: ``,
		},
		{
			name:   "exponentiated number",
			input:  "88.3e21",
			expect: ``,
		},
		{
			name:   "quoted string",
			input:  "@ this quoted string has a space@",
			expect: ``,
		},
		{
			name:   "unquoted string",
			input:  "some input",
			expect: ``,
		},
		{
			name:   "long expression",
			input:  "$FN(text, bool, $FN($FLAG == (num + num) * $FUNC() || bool && -num / num), num, text += @at text@)",
			expect: ``,
		},
	}

	front := Frontend(fakeHooks, nil)
	for _, tc := range testCases {

		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)

			r := strings.NewReader(tc.input)
			tokens, err := front.Lexer.Lex(r)
			if !assert.NoError(err) {
				return
			}

			// lex them all:
			for tokens.HasNext() {

			}

			assert.Equal(tc.expect, actual)
		})
	}
}

func Test_Parse(t *testing.T) {
	testCases := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "bool",
			input:  "true",
			expect: ``,
		},
		{
			name:   "int num",
			input:  "88",
			expect: ``,
		},
		{
			name:   "float num",
			input:  "88.3",
			expect: ``,
		},
		{
			name:   "exponentiated number",
			input:  "88.3e21",
			expect: ``,
		},
		{
			name:   "quoted string",
			input:  "@ this quoted string has a space@",
			expect: ``,
		},
		{
			name:   "unquoted string",
			input:  "some input",
			expect: ``,
		},
		{
			name:   "long expression",
			input:  "$FN(text, bool, $FN($FLAG == (num + num) * $FUNC() || bool && -num / num), num, text += @at text@)",
			expect: ``,
		},
	}

	front := Frontend(fakeHooks, nil)
	for _, tc := range testCases {

		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)

			_, pt, err := front.AnalyzeString(tc.input)
			if !assert.NoError(err) {
				return
			}

			actual := pt.String()

			assert.Equal(tc.expect, actual)
		})
	}
}
