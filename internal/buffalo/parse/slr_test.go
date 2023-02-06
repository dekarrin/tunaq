package parse

import (
	"testing"

	"github.com/dekarrin/tunaq/internal/buffalo/grammar"
	"github.com/stretchr/testify/assert"
)

func Test_ConstructSimpleLRParseTable(t *testing.T) {
	testCases := []struct {
		name      string
		grammar   string
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
			expect: `S   |  A:(  A:)          A:*          A:+          A:ID  A:$          |  G:E  G:F  G:T
--------------------------------------------------------------------------------------
0   |  s1                                          s9                 |  4    10   6  
1   |  s1                                          s9                 |  5    10   6  
2   |  s1                                          s9                 |       10   3  
3   |       rE -> E + T  s8           rE -> E + T        rE -> E + T  |               
4   |                                 s2                 acc          |               
5   |       s7                        s2                              |               
6   |       rE -> T      s8           rE -> T            rE -> T      |               
7   |       rF -> ( E )  rF -> ( E )  rF -> ( E )        rF -> ( E )  |               
8   |  s1                                          s9                 |       11      
9   |       rF -> id     rF -> id     rF -> id           rF -> id     |               
10  |       rT -> F      rT -> F      rT -> F            rT -> F      |               
11  |       rT -> T * F  rT -> T * F  rT -> T * F        rT -> T * F  |               `,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// setup
			assert := assert.New(t)
			g := grammar.MustParse(tc.grammar)

			// execute
			actual, err := constructSimpleLRParseTable(g)

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

func Test_SLR1Parse(t *testing.T) {
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
			parser, err := GenerateSimpleLRParser(g)
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
