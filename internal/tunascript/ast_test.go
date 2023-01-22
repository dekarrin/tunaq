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
			name: "quoted string",
			input: &astNode{value: &valueNode{
				quotedStringVal: sRef("@hello@"),
			}},
			expect: "(AST)\n" +
				`  \---: (QSTR_VALUE "@hello@")`,
		},
		{
			name: "unquoted string",
			input: &astNode{value: &valueNode{
				unquotedStringVal: sRef("fishka"),
			}},
			expect: "(AST)\n" +
				`  \---: (STR_VALUE "fishka")`,
		},
		{
			name: "bool true",
			input: &astNode{value: &valueNode{
				boolVal: bRef(true),
			}},
			expect: "(AST)\n" +
				`  \---: (BOOL_VALUE "true")`,
		},
		{
			name: "bool false",
			input: &astNode{value: &valueNode{
				boolVal: bRef(false),
			}},
			expect: "(AST)\n" +
				`  \---: (BOOL_VALUE "false")`,
		},
		{
			name: "num val",
			input: &astNode{value: &valueNode{
				numVal: iRef(28),
			}},
			expect: "(AST)\n" +
				`  \---: (NUM_VALUE "28")`,
		},
		{
			name: "flag",
			input: &astNode{flag: &flagNode{
				name: "$GLUB_IS_GOOD",
			}},
			expect: "(AST)\n" +
				`  \---: (FLAG "$GLUB_IS_GOOD")`,
		},
		{
			name: "group",
			input: &astNode{group: &groupNode{
				expr: &astNode{value: &valueNode{
					numVal: iRef(413),
				}},
			}},
			expect: "(AST)\n" +
				`  \---: (GROUP)` + "\n" +
				`          \---: (NUM_VALUE "413")`,
		},
		{
			name: "fn",
			input: &astNode{fn: &fnNode{
				name: "$OUTPUT",
				args: []*astNode{
					{
						value: &valueNode{
							unquotedStringVal: sRef("Hello, Sburb!"),
						},
					},
				},
			}},
			expect: "(AST)\n" +
				`  \---: (FUNCTION "$OUTPUT")` + "\n" +
				`          \-A0: (STR_VALUE "Hello, Sburb!")`,
		},
		{
			name: "simple binary operator",
			input: &astNode{opGroup: &operatorGroupNode{infixOp: &binaryOperatorGroupNode{
				op: "+",
				left: &astNode{value: &valueNode{
					numVal: iRef(612),
				}},
				right: &astNode{value: &valueNode{
					numVal: iRef(413),
				}},
			}}},
			expect: "(AST)\n" +
				`  \---: (BINARY_OP "+")` + "\n" +
				`          |--L: (NUM_VALUE "612")` + "\n" +
				`          \--R: (NUM_VALUE "413")`,
		},
		{
			name: "simple unary operator",
			input: &astNode{opGroup: &operatorGroupNode{unaryOp: &unaryOperatorGroupNode{
				op: "--",
				operand: &astNode{flag: &flagNode{
					name: "$GLUB",
				}},
			}}},
			expect: "(AST)\n" +
				`  \---: (UNARY_OP "--")` + "\n" +
				`          \---: (FLAG "$GLUB")`,
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
