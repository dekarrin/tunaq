package lex

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_LazyLex_singleStateLex(t *testing.T) {
	testCases := []struct {
		name       string
		classes    []string
		patterns   []string
		lexActions []Action
		input      string
		expect     []lexerToken
	}{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// setup
			assert := assert.New(t)
			lx := NewLexer(true)
			for i := range tc.classes {
				cl := NewTokenClass(strings.ToLower(tc.classes[i]), tc.classes[i])
				lx.RegisterClass(cl, "")
			}
			if len(tc.patterns) != len(tc.lexActions) {
				panic("bad test case: number of patterns doesnt match number of lex actions")
			}
			for i := range tc.patterns {
				pat := tc.patterns[i]
				act := tc.lexActions[i]
				err := lx.AddPattern(pat, act, "")
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
				}

				expectToken := tc.expect[tokNum]
				actualToken := stream.Next()

				assert.Equal(expectToken.Class().ID(), actualToken.Class().ID(), "token #%d, class mismatch", tokNum)
				assert.Equal(expectToken.FullLine(), actualToken.FullLine(), "token #%d, full-line mismatch", tokNum)
				assert.Equal(expectToken.Line(), actualToken.Line(), "token #%d, line number mismatch", tokNum)
				assert.Equal(expectToken.LinePos(), actualToken.LinePos(), "token #%d, line position mismatch", tokNum)
				assert.Equal(expectToken.Lexeme(), actualToken.Lexeme(), "token #%d, lexeme mismatch", tokNum)

				tokNum++
			}
		})
	}
}
