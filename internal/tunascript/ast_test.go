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
				quotedStringVal: ref("@hello@"),
			}},
			expect: "(AST)\n" +
				`  \---: (QSTR_VALUE "@hello@")`,
		},
		{
			name: "unquoted string",
			input: &astNode{value: &valueNode{
				unquotedStringVal: ref("fishka"),
			}},
			expect: "(AST)\n" +
				`  \---: (STR_VALUE "fishka")`,
		},
		{
			name: "bool true",
			input: &astNode{value: &valueNode{
				boolVal: ref(true),
			}},
			expect: "(AST)\n" +
				`  \---: (BOOL_VALUE "true")`,
		},
		{
			name: "bool false",
			input: &astNode{value: &valueNode{
				boolVal: ref(false),
			}},
			expect: "(AST)\n" +
				`  \---: (BOOL_VALUE "false")`,
		},
		{
			name: "num val",
			input: &astNode{value: &valueNode{
				numVal: ref(28),
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
					numVal: ref(413),
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
							unquotedStringVal: ref("Hello, Sburb!"),
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
					numVal: ref(612),
				}},
				right: &astNode{value: &valueNode{
					numVal: ref(413),
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
		{
			name: "complex function call",
			input: &astNode{fn: &fnNode{
				name: "$S_WAKE",
				args: []*astNode{
					{opGroup: &operatorGroupNode{infixOp: &binaryOperatorGroupNode{
						op: "+",
						left: &astNode{flag: &flagNode{
							name: "$ARADIA_PAIN",
						}},
						right: &astNode{value: &valueNode{numVal: ref(8)}},
					}}},
					{opGroup: &operatorGroupNode{unaryOp: &unaryOperatorGroupNode{
						op:      "!",
						operand: &astNode{flag: &flagNode{name: "$ANY_HELP"}},
					}}},
					{value: &valueNode{quotedStringVal: ref("@F8ck yeah!!!!!!!!@")}},
					{value: &valueNode{boolVal: ref(false)}},
					{fn: &fnNode{
						name: "$PAYBACK",
						args: []*astNode{
							{value: &valueNode{unquotedStringVal: ref("S_MAKE_HER_PAY")}},
							{value: &valueNode{boolVal: ref(false)}},
							{opGroup: &operatorGroupNode{infixOp: &binaryOperatorGroupNode{
								op: "*",
								left: &astNode{group: &groupNode{
									expr: &astNode{opGroup: &operatorGroupNode{infixOp: &binaryOperatorGroupNode{
										op:    "+",
										left:  &astNode{flag: &flagNode{name: "$VRISKA_PAIN"}},
										right: &astNode{value: &valueNode{numVal: ref(16)}},
									}}},
								}},
								right: &astNode{opGroup: &operatorGroupNode{unaryOp: &unaryOperatorGroupNode{
									op:      "-",
									operand: &astNode{value: &valueNode{numVal: ref(8)}},
								}}},
							}}},
						},
					}},
					{value: &valueNode{numVal: ref(413)}},
				},
			}},
			// careful, ide's indent can sometimes think tabs are good in a raw
			// but they are inconsistent and only spaces should be used.
			expect: `(AST)
  \---: (FUNCTION "$S_WAKE")
          |-A0: (BINARY_OP "+")
          |       |--L: (FLAG "$ARADIA_PAIN")
          |       \--R: (NUM_VALUE "8")
          |-A1: (UNARY_OP "!")
          |       \---: (FLAG "$ANY_HELP")
          |-A2: (QSTR_VALUE "@F8ck yeah!!!!!!!!@")
          |-A3: (BOOL_VALUE "false")
          |-A4: (FUNCTION "$PAYBACK")
          |       |-A0: (STR_VALUE "S_MAKE_HER_PAY")
          |       |-A1: (BOOL_VALUE "false")
          |       \-A2: (BINARY_OP "*")
          |               |--L: (GROUP)
          |               |       \---: (BINARY_OP "+")
          |               |               |--L: (FLAG "$VRISKA_PAIN")
          |               |               \--R: (NUM_VALUE "16")
          |               \--R: (UNARY_OP "-")
          |                       \---: (NUM_VALUE "8")
          \-A5: (NUM_VALUE "413")`,
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
