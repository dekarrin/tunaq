package fe

import (
	"log"
	"strings"
	"testing"

	"github.com/dekarrin/ictiobus"
	"github.com/dekarrin/ictiobus/syntaxerr"
	"github.com/dekarrin/ictiobus/trans"
	"github.com/stretchr/testify/assert"
)

// Note: This frontend test file actually does not test the SDTS due to not
// having access to the final implementations located in the syntax package.
//
// It tests only against a modified Frontend with an entirely mocked SDTS.

func withMockedSDTS[E any](front ictiobus.Frontend[E]) ictiobus.Frontend[int] {
	newFront := ictiobus.Frontend[int]{
		Lexer:       front.Lexer,
		Parser:      front.Parser,
		IRAttribute: "value",
		Language:    front.Language,
		Version:     front.Version,
	}

	sdts := ictiobus.NewSDTS()
	sdts.Bind("TUNASCRIPT", []string{"EXPR"}, "value", "test_const", nil)
	sdts.SetHooks(fakeHooks)

	newFront.SDTS = sdts

	return newFront
}

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
		expect []string
	}{
		{
			name:   "bool",
			input:  "true",
			expect: []string{"bool"},
		},
		{
			name:   "int num",
			input:  "88",
			expect: []string{"num"},
		},
		{
			name:   "float num",
			input:  "88.3",
			expect: []string{"num"},
		},
		{
			name:   "exponentiated number",
			input:  "88.3e21",
			expect: []string{"num"},
		},
		{
			name:   "quoted string",
			input:  "@ this quoted string has a space@",
			expect: []string{"@str"},
		},
		{
			name:   "unquoted string",
			input:  "some input",
			expect: []string{"str"},
		},
		{
			name:  "long expression",
			input: "$FN(text, off, $FN($FLAG == (22.2 + num) * $FUNC() || bool && -2 / num), num, $text += @at text@)",
			expect: []string{
				"id", "lp", "str", "comma", "bool", "comma", "id", "lp", "id", "eq", "lp", "num", "+", "str", "rp",
				"*", "id", "lp", "rp", "or", "str", "and", "-", "num", "/", "str", "rp", "comma", "str", "comma",
				"id", "+=", "@str", "rp",
			},
		},
	}

	front := withMockedSDTS(Frontend(fakeHooks, nil))
	for _, tc := range testCases {

		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)

			r := strings.NewReader(tc.input)
			tokens, err := front.Lexer.Lex(r)
			if !assert.NoError(err) {
				return
			}

			var actual []string
			// lex them all:
			for tokens.HasNext() {
				actual = append(actual, tokens.Next().Class().ID())
			}
			if len(actual) > 0 {
				actual = actual[:len(actual)-1]
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
			name:  "bool",
			input: "true",
			expect: `( TUNASCRIPT )
  \---: ( EXPR )
          \---: ( BOOL-OP )
                  \---: ( EQUALITY )
                          \---: ( COMPARISON )
                                  \---: ( SUM )
                                          \---: ( PRODUCT )
                                                  \---: ( NEGATION )
                                                          \---: ( TERM )
                                                                  \---: ( VALUE )
                                                                          \---: (TERM "bool")`,
		},
		{
			name:  "int num",
			input: "88",
			expect: `( TUNASCRIPT )
  \---: ( EXPR )
          \---: ( BOOL-OP )
                  \---: ( EQUALITY )
                          \---: ( COMPARISON )
                                  \---: ( SUM )
                                          \---: ( PRODUCT )
                                                  \---: ( NEGATION )
                                                          \---: ( TERM )
                                                                  \---: ( VALUE )
                                                                          \---: (TERM "num")`,
		},
		{
			name:  "float num",
			input: "88.3",
			expect: `( TUNASCRIPT )
  \---: ( EXPR )
          \---: ( BOOL-OP )
                  \---: ( EQUALITY )
                          \---: ( COMPARISON )
                                  \---: ( SUM )
                                          \---: ( PRODUCT )
                                                  \---: ( NEGATION )
                                                          \---: ( TERM )
                                                                  \---: ( VALUE )
                                                                          \---: (TERM "num")`,
		},
		{
			name:  "exponentiated number",
			input: "88.3e21",
			expect: `( TUNASCRIPT )
  \---: ( EXPR )
          \---: ( BOOL-OP )
                  \---: ( EQUALITY )
                          \---: ( COMPARISON )
                                  \---: ( SUM )
                                          \---: ( PRODUCT )
                                                  \---: ( NEGATION )
                                                          \---: ( TERM )
                                                                  \---: ( VALUE )
                                                                          \---: (TERM "num")`,
		},
		{
			name:  "quoted string",
			input: "@ this quoted string has a space@",
			expect: `( TUNASCRIPT )
  \---: ( EXPR )
          \---: ( BOOL-OP )
                  \---: ( EQUALITY )
                          \---: ( COMPARISON )
                                  \---: ( SUM )
                                          \---: ( PRODUCT )
                                                  \---: ( NEGATION )
                                                          \---: ( TERM )
                                                                  \---: ( VALUE )
                                                                          \---: (TERM "@str")`,
		},
		{
			name:  "unquoted string",
			input: "some input",
			expect: `( TUNASCRIPT )
  \---: ( EXPR )
          \---: ( BOOL-OP )
                  \---: ( EQUALITY )
                          \---: ( COMPARISON )
                                  \---: ( SUM )
                                          \---: ( PRODUCT )
                                                  \---: ( NEGATION )
                                                          \---: ( TERM )
                                                                  \---: ( VALUE )
                                                                          \---: (TERM "str")`,
		},
	}

	front := withMockedSDTS(Frontend(fakeHooks, nil))
	for _, tc := range testCases {

		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)

			_, pt, err := front.AnalyzeString(tc.input)
			if !assert.NoError(err) {
				if sErr, ok := err.(*syntaxerr.Error); ok {
					log.Printf("\n%s", sErr.FullMessage())
				}
				return
			}

			actual := pt.String()

			assert.Equal(tc.expect, actual)
		})
	}
}
