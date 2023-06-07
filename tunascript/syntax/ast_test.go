package syntax

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func ref[T any](s T) *T {
	return &s
}

func Test_AST_String(t *testing.T) {
	testCases := []struct {
		name   string
		input  ASTNode
		expect string
	}{
		{
			name: "quoted string",
			input: LiteralNode{
				Quoted: true,
				Value:  TSValueOf("@hello@"),
			},
			expect: "(AST)\n" +
				`  \---: (QSTR_VALUE "@hello@")`,
		},
		{
			name: "unquoted string",
			input: LiteralNode{
				Value: TSValueOf("fishka"),
			},
			expect: "(AST)\n" +
				`  \---: (STR_VALUE "fishka")`,
		},
		{
			name: "bool true",
			input: LiteralNode{
				Value: TSValueOf(true),
			},
			expect: "(AST)\n" +
				`  \---: (BOOL_VALUE "true")`,
		},
		{
			name: "bool false",
			input: LiteralNode{
				Value: TSValueOf(false),
			},
			expect: "(AST)\n" +
				`  \---: (BOOL_VALUE "false")`,
		},
		{
			name: "num val",
			input: LiteralNode{
				Value: TSValueOf(28),
			},
			expect: "(AST)\n" +
				`  \---: (NUM_VALUE "28")`,
		},
		{
			name: "flag",
			input: FlagNode{
				Name: "GLUB_IS_GOOD",
			},
			expect: "(AST)\n" +
				`  \---: (FLAG "$GLUB_IS_GOOD")`,
		},
		{
			name: "group",
			input: &ASTNode{group: &groupNode{
				expr: &ASTNode{value: &valueNode{
					numVal: ref(413),
				}},
			}},
			expect: "(AST)\n" +
				`  \---: (GROUP)` + "\n" +
				`          \---: (NUM_VALUE "413")`,
		},
		{
			name: "fn",
			input: &ASTNode{fn: &fnNode{
				name: "$OUTPUT",
				args: []*ASTNode{
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
			input: &ASTNode{opGroup: &operatorGroupNode{infixOp: &binaryOperatorGroupNode{
				op: "+",
				left: &ASTNode{value: &valueNode{
					numVal: ref(612),
				}},
				right: &ASTNode{value: &valueNode{
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
			input: &ASTNode{opGroup: &operatorGroupNode{unaryOp: &unaryOperatorGroupNode{
				op: "--",
				operand: &ASTNode{flag: &flagNode{
					name: "$GLUB",
				}},
			}}},
			expect: "(AST)\n" +
				`  \---: (UNARY_OP "--")` + "\n" +
				`          \---: (FLAG "$GLUB")`,
		},
		{
			name: "complex function call",
			input: &ASTNode{fn: &fnNode{
				name: "$S_WAKE",
				args: []*ASTNode{
					{opGroup: &operatorGroupNode{infixOp: &binaryOperatorGroupNode{
						op: "+",
						left: &ASTNode{flag: &flagNode{
							name: "$ARADIA_PAIN",
						}},
						right: &ASTNode{value: &valueNode{numVal: ref(8)}},
					}}},
					{opGroup: &operatorGroupNode{unaryOp: &unaryOperatorGroupNode{
						op:      "!",
						operand: &ASTNode{flag: &flagNode{name: "$ANY_HELP"}},
					}}},
					{value: &valueNode{quotedStringVal: ref("@F8ck yeah!!!!!!!!@")}},
					{value: &valueNode{boolVal: ref(false)}},
					{fn: &fnNode{
						name: "$PAYBACK",
						args: []*ASTNode{
							{value: &valueNode{unquotedStringVal: ref("S_MAKE_HER_PAY")}},
							{value: &valueNode{boolVal: ref(false)}},
							{opGroup: &operatorGroupNode{infixOp: &binaryOperatorGroupNode{
								op: "*",
								left: &ASTNode{group: &groupNode{
									expr: &ASTNode{opGroup: &operatorGroupNode{infixOp: &binaryOperatorGroupNode{
										op:    "+",
										left:  &ASTNode{flag: &flagNode{name: "$VRISKA_PAIN"}},
										right: &ASTNode{value: &valueNode{numVal: ref(16)}},
									}}},
								}},
								right: &ASTNode{opGroup: &operatorGroupNode{unaryOp: &unaryOperatorGroupNode{
									op:      "-",
									operand: &ASTNode{value: &valueNode{numVal: ref(8)}},
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

			inputAST := AST{Nodes: []*ASTNode{tc.input}}

			actual := inputAST.String()

			assert.Equal(tc.expect, actual)
		})
	}
}
