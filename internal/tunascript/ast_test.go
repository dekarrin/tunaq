package tunascript

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_AST_String(t *testing.T) {
	testCases := []struct {
		name   string
		input  *astNode
		expect string
	}{
		{
			name: "",
			input: &astNode{value: &valueNode{
				quotedStringVal: sRef("hello"),
			}},
			expect: "(AST)\n" +
				` \---: (STR_VALUE "hello")`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)

			inputAST := AST{nodes: []*astNode{tc.input}}

			actual := inputAST.String()

			assert.Equal(tc.expect, actual)
		})
	}
}
