package tunascript

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Grammar_Validate(t *testing.T) {
	testCases := []struct {
		name      string
		rules     []Rule
		terminals []tokenClass
		expectErr bool
	}{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)

			// set up the grammar
			g := Grammar{}
			for _, term := range tc.terminals {
				g.AddTerm(term.id, term)
			}
			for _, r := range tc.rules {
				for _, alts := range r.Productions {
					g.AddRule(r.NonTerminal, alts)
				}
			}

			// checkActual
			actual := g.Validate()

			if tc.expectErr {
				assert.Error(actual)
			} else {
				assert.NoError(actual)
			}
		})
	}
}
