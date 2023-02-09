package parse

import (
	"testing"

	"github.com/dekarrin/tunaq/internal/ictiobus/grammar"
	"github.com/stretchr/testify/assert"
)

func Test_kernels(t *testing.T) {
	testCases := []struct {
		name    string
		grammar string
		expect  string
	}{
		{
			name: "purple dragon LALR(1) example grammar",
			grammar: `
				S -> L = R | R ;
				L -> * R | id ;
				R -> L ;
			`,
			expect: `glub`,
		}, /*
			{
				name: "quick2",
				grammar: `
					E -> E + T | T ;
					T -> T * F | F ;
					F -> ( E ) | id ;
				`,
				expect: `glub`,
			},*/
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// setup
			assert := assert.New(t)
			g := grammar.MustParse(tc.grammar)

			// execute
			actual := computeLALR1Kernels(g)

			// assert)
			assert.Equal(tc.expect, actual.String())
		})
	}

}
