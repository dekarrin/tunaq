package syntax

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
				Value:  TSValueOf("hello"),
			},
			expect: "AST\n" +
				` S: [LITERAL TEXT/@STRING "hello"]`,
		},
		{
			name: "unquoted string",
			input: LiteralNode{
				Value: TSValueOf("fishka"),
			},
			expect: "AST\n" +
				` S: [LITERAL TEXT/STRING "fishka"]`,
		},
		{
			name: "bool true",
			input: LiteralNode{
				Value: TSValueOf(true),
			},
			expect: "AST\n" +
				` S: [LITERAL BINARY/BOOL ON]`,
		},
		{
			name: "bool false",
			input: LiteralNode{
				Value: TSValueOf(false),
			},
			expect: "AST\n" +
				` S: [LITERAL BINARY/BOOL OFF]`,
		},
		{
			name: "num val (int)",
			input: LiteralNode{
				Value: TSValueOf(28),
			},
			expect: "AST\n" +
				` S: [LITERAL NUMBER/INT 28]`,
		},
		{
			name: "num val (float)",
			input: LiteralNode{
				Value: TSValueOf(28.7),
			},
			expect: "AST\n" +
				` S: [LITERAL NUMBER/FLOAT 28.7]`,
		},
		{
			name: "flag",
			input: FlagNode{
				Flag: "GLUB_IS_GOOD",
			},
			expect: "AST\n" +
				` S: [FLAG $GLUB_IS_GOOD]`,
		},
		{
			name: "assignment (no value)",
			input: AssignmentNode{
				Op:      OpAssignIncrement,
				Flag:    "FEFERI",
				PostFix: true,
			},
			expect: "AST\n" +
				` S: [ASSIGNMENT INCREMENT_ONE $FEFERI]`,
		},
		{
			name: "assignment (with value)",
			input: AssignmentNode{
				Op:   OpAssignSet,
				Flag: "GLUB",
				Value: LiteralNode{
					Value: TSValueOf(true),
				},
			},
			expect: "AST\n" +
				` S: [ASSIGNMENT SET $GLUB` + "\n" +
				`     V: [LITERAL BINARY/BOOL ON]` + "\n" +
				`    ]`,
		},
		{
			name: "group",
			input: GroupNode{
				Expr: LiteralNode{
					Value: TSValueOf(413),
				},
			},
			expect: "AST\n" +
				` S: [GROUP` + "\n" +
				`     E: [LITERAL NUMBER/INT 413]` + "\n" +
				`    ]`,
		},
		{
			name: "fn (no args)",
			input: FuncNode{
				Func: "SOME_FUNC",
			},
			expect: "AST\n" +
				` S: [FUNC $SOME_FUNC]`,
		},
		{
			name: "fn (one arg)",
			input: FuncNode{
				Func: "OUTPUT",
				Args: []ASTNode{
					LiteralNode{
						Value: TSValueOf("Hello, Sburb!"),
					},
				},
			},
			expect: "AST\n" +
				` S: [FUNC $OUTPUT` + "\n" +
				`     A: [LITERAL TEXT/STRING "Hello, Sburb!"]` + "\n" +
				`    ]`,
		},
		{
			name: "fn (multiple args)",
			input: FuncNode{
				Func: "MULT_ARGS",
				Args: []ASTNode{
					LiteralNode{
						Value: TSValueOf("Hello, Sburb!"),
					},
					LiteralNode{
						Value: TSValueOf(41.3),
					},
				},
			},
			expect: "AST\n" +
				` S: [FUNC $MULT_ARGS` + "\n" +
				`     A: [LITERAL TEXT/STRING "Hello, Sburb!"]` + "\n" +
				`     A: [LITERAL NUMBER/FLOAT 41.3]` + "\n" +
				`    ]`,
		},
		{
			name: "binary operator",
			input: BinaryOpNode{
				Op: OpBinaryAdd,
				Left: LiteralNode{
					Value: TSValueOf(612),
				},
				Right: LiteralNode{
					Value: TSValueOf(413),
				},
			},
			expect: "AST\n" +
				` S: [BINARY_OP ADDITION` + "\n" +
				`     L: [LITERAL NUMBER/INT 612]` + "\n" +
				`     R: [LITERAL NUMBER/INT 413]` + "\n" +
				`    ]`,
		},
		{
			name: "unary operator",
			input: UnaryOpNode{
				Op: OpUnaryNegate,
				Operand: FlagNode{
					Flag: "GLUB",
				},
			},
			expect: "AST\n" +
				` S: [UNARY_OP NEGATION` + "\n" +
				`     O: [FLAG $GLUB]` + "\n" +
				`    ]`,
		},
		{
			name: "complex function call",
			input: FuncNode{
				Func: "S_WAKE",
				Args: []ASTNode{
					BinaryOpNode{Op: OpBinaryAdd,
						Left:  FlagNode{Flag: "ARADIA_PAIN"},
						Right: LiteralNode{Value: TSValueOf(8)},
					},
					UnaryOpNode{Op: OpUnaryNegate,
						Operand: FlagNode{Flag: "ANY_HELP"},
					},
					LiteralNode{Quoted: true, Value: TSValueOf("F8ck yeah!!!!!!!!")},
					LiteralNode{Value: TSValueOf(true)},
					FuncNode{Func: "PAYBACK",
						Args: []ASTNode{
							LiteralNode{Value: TSValueOf("S_MAKE_HER_PAY")},
							LiteralNode{Value: TSValueOf(false)},
							BinaryOpNode{Op: OpBinaryMultiply,
								Left: GroupNode{Expr: BinaryOpNode{Op: OpBinaryAdd,
									Left:  FlagNode{Flag: "VRISKA_PAIN"},
									Right: LiteralNode{Value: TSValueOf(16)},
								}},
								Right: UnaryOpNode{Op: OpUnaryNegate,
									Operand: LiteralNode{Value: TSValueOf(8.8)},
								},
							},
						},
					},
					LiteralNode{Value: TSValueOf(413)},
				},
			},
			expect: `AST
 S: [FUNC $S_WAKE
     A: [BINARY_OP ADDITION
         L: [FLAG $ARADIA_PAIN]
         R: [LITERAL NUMBER/INT 8]
        ]
     A: [UNARY_OP NEGATION
         O: [FLAG $ANY_HELP]
        ]
     A: [LITERAL TEXT/@STRING "F8ck yeah!!!!!!!!"]
     A: [LITERAL BINARY/BOOL ON]
     A: [FUNC $PAYBACK
         A: [LITERAL TEXT/STRING "S_MAKE_HER_PAY"]
         A: [LITERAL BINARY/BOOL OFF]
         A: [BINARY_OP MULTIPLICATION
             L: [GROUP
                 E: [BINARY_OP ADDITION
                     L: [FLAG $VRISKA_PAIN]
                     R: [LITERAL NUMBER/INT 16]
                    ]
                ]
             R: [UNARY_OP NEGATION
                 O: [LITERAL NUMBER/FLOAT 8.8]
                ]
            ]
        ]
     A: [LITERAL NUMBER/INT 413]
    ]`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)

			inputAST := AST{Nodes: []ASTNode{tc.input}}

			actual := inputAST.String()

			assert.Equal(tc.expect, actual)
		})
	}
}
