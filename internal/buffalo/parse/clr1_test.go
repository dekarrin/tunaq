package parse

import (
	"testing"

	"github.com/dekarrin/tunaq/internal/buffalo/grammar"
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
			expect: ``,
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
