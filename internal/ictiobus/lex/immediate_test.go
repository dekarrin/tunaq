package lex

import (
	"strings"
	"testing"

	"github.com/dekarrin/tunaq/internal/ictiobus/types"
	"github.com/stretchr/testify/assert"
)

func Test_ImmediateLex_singleStateLex(t *testing.T) {
	testCases := []struct {
		name       string
		classes    []types.TokenClass
		patterns   []string
		lexActions []Action
		input      string
		expect     []lexerToken
	}{
		{
			name:    "single-line lex",
			classes: allTestClasses,
			patterns: []string{
				`\+`,
				`\*`,
				`\(`,
				`\)`,
				`[A-Za-z_][A-Za-z_0-9]*`,
				`=`,
				`[0-9]+`,
				`\s+`,
			},
			lexActions: []Action{
				LexAs("plus"),
				LexAs("mult"),
				LexAs("lparen"),
				LexAs("rparen"),
				LexAs("id"),
				LexAs("equals"),
				LexAs("int"),
				{}, // do nothing for whitespace, drop it
			},
			input: "someVar = (8 + 1)* 2",
			expect: []lexerToken{
				{line: "someVar = (8 + 1)* 2", lineNum: 1, linePos: 1, class: testClassId, lexed: "someVar"},
				{line: "someVar = (8 + 1)* 2", lineNum: 1, linePos: 9, class: testClassEq, lexed: "="},
				{line: "someVar = (8 + 1)* 2", lineNum: 1, linePos: 11, class: testClassLParen, lexed: "("},
				{line: "someVar = (8 + 1)* 2", lineNum: 1, linePos: 12, class: testClassInt, lexed: "8"},
				{line: "someVar = (8 + 1)* 2", lineNum: 1, linePos: 14, class: testClassPlus, lexed: "+"},
				{line: "someVar = (8 + 1)* 2", lineNum: 1, linePos: 16, class: testClassInt, lexed: "1"},
				{line: "someVar = (8 + 1)* 2", lineNum: 1, linePos: 17, class: testClassRParen, lexed: ")"},
				{line: "someVar = (8 + 1)* 2", lineNum: 1, linePos: 18, class: testClassMult, lexed: "*"},
				{line: "someVar = (8 + 1)* 2", lineNum: 1, linePos: 20, class: testClassInt, lexed: "2"},
				{line: "someVar = (8 + 1)* 2", lineNum: 1, linePos: 21, class: types.TokenEndOfText},
			},
		},
		{
			name:    "no-space lex",
			classes: allTestClasses,
			patterns: []string{
				`\+`,
				`\*`,
				`\(`,
				`\)`,
				`[A-Za-z_][A-Za-z_0-9]*`,
				`=`,
				`[0-9]+`,
				`\s+`,
			},
			lexActions: []Action{
				LexAs("plus"),
				LexAs("mult"),
				LexAs("lparen"),
				LexAs("rparen"),
				LexAs("id"),
				LexAs("equals"),
				LexAs("int"),
				{}, // do nothing for whitespace, drop it
			},
			input: "someVar=(8+1)*2",
			expect: []lexerToken{
				{line: "someVar=(8+1)*2", lineNum: 1, linePos: 1, class: testClassId, lexed: "someVar"},
				{line: "someVar=(8+1)*2", lineNum: 1, linePos: 8, class: testClassEq, lexed: "="},
				{line: "someVar=(8+1)*2", lineNum: 1, linePos: 9, class: testClassLParen, lexed: "("},
				{line: "someVar=(8+1)*2", lineNum: 1, linePos: 10, class: testClassInt, lexed: "8"},
				{line: "someVar=(8+1)*2", lineNum: 1, linePos: 11, class: testClassPlus, lexed: "+"},
				{line: "someVar=(8+1)*2", lineNum: 1, linePos: 12, class: testClassInt, lexed: "1"},
				{line: "someVar=(8+1)*2", lineNum: 1, linePos: 13, class: testClassRParen, lexed: ")"},
				{line: "someVar=(8+1)*2", lineNum: 1, linePos: 14, class: testClassMult, lexed: "*"},
				{line: "someVar=(8+1)*2", lineNum: 1, linePos: 15, class: testClassInt, lexed: "2"},
				{line: "someVar=(8+1)*2", lineNum: 1, linePos: 16, class: types.TokenEndOfText},
			},
		},
		{
			name:    "multi-line lex",
			classes: allTestClasses,
			patterns: []string{
				`\+`,
				`\*`,
				`\(`,
				`\)`,
				`[A-Za-z_][A-Za-z_0-9]*`,
				`=`,
				`[0-9]+`,
				`\s+`,
			},
			lexActions: []Action{
				LexAs("plus"),
				LexAs("mult"),
				LexAs("lparen"),
				LexAs("rparen"),
				LexAs("id"),
				LexAs("equals"),
				LexAs("int"),
				{}, // do nothing for whitespace, drop it
			},
			input: "someVar =\n(8 + 1)* 2",
			expect: []lexerToken{
				{line: "someVar =", lineNum: 1, linePos: 1, class: testClassId, lexed: "someVar"},
				{line: "someVar =", lineNum: 1, linePos: 9, class: testClassEq, lexed: "="},
				{line: "(8 + 1)* 2", lineNum: 2, linePos: 1, class: testClassLParen, lexed: "("},
				{line: "(8 + 1)* 2", lineNum: 2, linePos: 2, class: testClassInt, lexed: "8"},
				{line: "(8 + 1)* 2", lineNum: 2, linePos: 4, class: testClassPlus, lexed: "+"},
				{line: "(8 + 1)* 2", lineNum: 2, linePos: 6, class: testClassInt, lexed: "1"},
				{line: "(8 + 1)* 2", lineNum: 2, linePos: 7, class: testClassRParen, lexed: ")"},
				{line: "(8 + 1)* 2", lineNum: 2, linePos: 8, class: testClassMult, lexed: "*"},
				{line: "(8 + 1)* 2", lineNum: 2, linePos: 10, class: testClassInt, lexed: "2"},
				{line: "(8 + 1)* 2", lineNum: 2, linePos: 11, class: types.TokenEndOfText},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// setup
			assert := assert.New(t)
			lx := NewLexer(false)
			for i := range tc.classes {
				lx.RegisterClass(tc.classes[i], "")
			}
			if len(tc.patterns) != len(tc.lexActions) {
				panic("bad test case: number of patterns doesnt match number of lex actions")
			}
			for i := range tc.patterns {
				pat := tc.patterns[i]
				act := tc.lexActions[i]
				err := lx.AddPattern(pat, act, "", 0)
				if !assert.NoErrorf(err, "adding pattern %d to lexer failed", i) {
					return
				}
			}
			inputReader := strings.NewReader(tc.input)

			// execute
			stream, err := lx.Lex(inputReader)
			if !assert.NoErrorf(err, "error while producing token stream") {
				return
			}

			// assert

			// go through each item in the stream and check that it matches
			// expected
			tokNum := 0
			for stream.HasNext() {
				if tokNum >= len(tc.expect) {
					assert.Failf("wrong number of produced tokens", "expected stream to produce %d tokens but got more", len(tc.expect))
					return
				}

				expectToken := tc.expect[tokNum]
				actualToken := stream.Next()

				if actualToken.Class().ID() == types.TokenError.ID() {
					assert.Fail("received error token", "error: %s", actualToken.Lexeme())
				}

				assert.Equal(expectToken.Class().ID(), actualToken.Class().ID(), "token #%d, class mismatch", tokNum)
				assert.Equal(expectToken.FullLine(), actualToken.FullLine(), "token #%d, full-line mismatch", tokNum)
				assert.Equal(expectToken.Line(), actualToken.Line(), "token #%d, line number mismatch", tokNum)
				assert.Equal(expectToken.LinePos(), actualToken.LinePos(), "token #%d, line position mismatch", tokNum)
				assert.Equal(expectToken.Lexeme(), actualToken.Lexeme(), "token #%d, lexeme mismatch", tokNum)

				tokNum++
			}
			if tokNum != len(tc.expect) {
				assert.Failf("wrong number of produced tokens", "expected stream to produce %d tokens but got %d", len(tc.expect), tokNum)
			}
		})
	}
}
