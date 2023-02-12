package parse

import (
	"testing"

	"github.com/dekarrin/tunaq/internal/ictiobus/grammar"
	"github.com/dekarrin/tunaq/internal/ictiobus/lex"
	"github.com/stretchr/testify/assert"
)

func Test_ConstructLALR1ParseTable(t *testing.T) {
	testCases := []struct {
		name      string
		grammar   string
		expect    string
		expectErr bool
	}{
		{
			name: "purple dragon LALR(1) example grammar 4.55",
			grammar: `
				S -> C C ;
				C -> c C | d ;
			`,
			expect: `S  |  A:C        A:D        A:$        |  G:C  G:S
--------------------------------------------------
0  |  s2         s4                    |  1    6  
1  |  s2         s4                    |  5       
2  |  s2         s4                    |  3       
3  |  rC -> c C  rC -> c C  rC -> c C  |          
4  |  rC -> d    rC -> d    rC -> d    |          
5  |                        rS -> C C  |          
6  |                        acc        |          `,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// setup
			assert := assert.New(t)
			g := grammar.MustParse(tc.grammar)

			// execute
			actual, err := constructLALR1ParseTable(g)

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

func Test_LALR1Parse(t *testing.T) {
	testCases := []struct {
		name      string
		grammar   string
		input     []string
		expect    string
		expectErr bool
	}{
		{
			name: "purple dragon example 4.45",
			grammar: `
				E -> E + T | T ;
				T -> T * F | F ;
				F -> ( E ) | id ;
				`,
			input: []string{"(", "id", "+", "id", ")", "*", "id", lex.TokenEndOfText.ID()},
			expect: `( E )
  \---: ( T )
          |---: ( T )
          |       \---: ( F )
          |               |---: (TERM "(")
          |               |---: ( E )
          |               |       |---: ( E )
          |               |       |       \---: ( T )
          |               |       |               \---: ( F )
          |               |       |                       \---: (TERM "id")
          |               |       |---: (TERM "+")
          |               |       \---: ( T )
          |               |               \---: ( F )
          |               |                       \---: (TERM "id")
          |               \---: (TERM ")")
          |---: (TERM "*")
          \---: ( F )
                  \---: (TERM "id")`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// setup
			assert := assert.New(t)
			g := grammar.MustParse(tc.grammar)
			stream := mockTokens(tc.input...)

			// execute
			parser, err := GenerateLALR1Parser(g)
			assert.NoError(err, "generating LALR parser failed")
			actual, err := parser.Parse(stream)

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
