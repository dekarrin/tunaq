package parse

import (
	"testing"

	"github.com/dekarrin/tunaq/internal/ictiobus/grammar"
	"github.com/dekarrin/tunaq/internal/ictiobus/types"
	"github.com/stretchr/testify/assert"
)

func Test_LL1PredictiveParse(t *testing.T) {
	testCases := []struct {
		name      string
		grammar   string
		input     []string
		expect    string
		expectErr bool
	}{
		{
			name: "aiken expression LL1 sample",
			grammar: `
				S -> T X ;

				T -> ( S )
				   | int Y ;

				X -> + S
				   | ε ;

				Y -> * T
				   | ε ;
			`,
			input: []string{
				"int", "*", "int", types.TokenEndOfText.ID(),
			},
			expect: "( S )\n" +
				`  |---: ( T )` + "\n" +
				`  |       |---: (TERM "int")` + "\n" +
				`  |       \---: ( Y )` + "\n" +
				`  |               |---: (TERM "*")` + "\n" +
				`  |               \---: ( T )` + "\n" +
				`  |                       |---: (TERM "int")` + "\n" +
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
			g := grammar.MustParse(tc.grammar)
			stream := mockTokens(tc.input...)
			ll1, err := GenerateLL1Parser(g)
			if !assert.NoError(err) {
				return
			}

			// execute
			actual, err := ll1.Parse(stream)

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
