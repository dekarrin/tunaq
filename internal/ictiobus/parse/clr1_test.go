package parse

import (
	"testing"

	"github.com/dekarrin/tunaq/internal/ictiobus/grammar"
	"github.com/stretchr/testify/assert"
)

func Test_ConstructCanonicalLR1ParseTable(t *testing.T) {
	testCases := []struct {
		name      string
		grammar   string
		expect    string
		expectErr bool
	}{
		{
			name: "purple dragon example 4.45",
			grammar: `
				S -> C C ;
				C -> c C | d ;
			`,
			expect: `S  |  A:C        A:D        A:$        |  G:C  G:S
--------------------------------------------------
0  |  s2         s7                    |  1    9  
1  |  s3         s6                    |  8       
2  |  s2         s7                    |  5       
3  |  s3         s6                    |  4       
4  |                        rC -> c C  |          
5  |  rC -> c C  rC -> c C             |          
6  |                        rC -> d    |          
7  |  rC -> d    rC -> d               |          
8  |                        rS -> C C  |          
9  |                        acc        |          `,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// setup
			assert := assert.New(t)
			g := grammar.MustParse(tc.grammar)

			// execute
			actual, err := constructCanonicalLR1ParseTable(g)

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

func Test_CanonicalLR1Parse(t *testing.T) {
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
			input: []string{"id", "*", "id", "+", "id", "$"},
			expect: `( E )
  |---: ( E )
  |       \---: ( T )
  |               |---: ( T )
  |               |       \---: ( F )
  |               |               \---: (TERM "id")
  |               |---: (TERM "*")
  |               \---: ( F )
  |                       \---: (TERM "id")
  |---: (TERM "+")
  \---: ( T )
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
			parser, err := GenerateCanonicalLR1Parser(g)
			assert.NoError(err, "generating SLR parser failed")
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
