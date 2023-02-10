package parse

import (
	"testing"

	"github.com/dekarrin/tunaq/internal/ictiobus/grammar"
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

func Test_kernels(t *testing.T) {
	testCases := []struct {
		name    string
		grammar string
		expect  string
	}{
		{
			name: "2-rule ex from https://www.cs.york.ac.uk/fp/lsa/lectures/lalr.pdf",
			grammar: `
				S -> C C ;
				C -> c C | d ;
			`,
			expect: `glub`,
		},
		/*{
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
			actual, err := constructLALR1ParseTable(g)
			assert.NoError(err)

			// assert)
			assert.Equal(tc.expect, actual.String())
		})
	}

}
