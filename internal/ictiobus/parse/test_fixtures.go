package parse

import "github.com/dekarrin/tunaq/internal/ictiobus/lex"

// mockstream will just be a v simple token stream
type mockStream struct {
	tokens []lex.Token
	cur    int
}

func (ts *mockStream) Next() lex.Token {
	n := ts.tokens[ts.cur]
	ts.cur++
	return n
}

func (ts *mockStream) Peek() lex.Token {
	return ts.tokens[ts.cur]
}

func (ts *mockStream) HasNext() bool {
	return len(ts.tokens)-ts.cur > 0
}

type mockToken struct {
	c      lex.TokenClass
	l      int
	lp     int
	lexeme string
	f      string
}

func (tok mockToken) FullLine() string {
	return tok.f
}

func (tok mockToken) Class() lex.TokenClass {
	return tok.c
}

func (tok mockToken) Line() int {
	return tok.l
}

func (tok mockToken) LinePos() int {
	return tok.lp
}

func (tok mockToken) Lexeme() string {
	return tok.lexeme
}

func mockTokens(ofTerm ...string) lex.TokenStream {
	buildingLine := ""

	var lineTokens = make([]mockToken, 0)

	const lineEvery = 100

	curLine := 1
	curLinePos := 1
	var mocked []lex.Token
	for i := range ofTerm {
		tc := lex.MakeDefaultClass(ofTerm[i])
		m := mockToken{c: tc, l: curLine, lp: curLinePos, lexeme: tc.ID()}
		lineTokens = append(lineTokens, m)
		if tc.ID() != lex.TokenEndOfText.ID() && tc.ID() != lex.TokenUndefined.ID() {
			buildingLine += m.lexeme + " "
			curLinePos += len(m.lexeme) + 1 // for the space
		}
		if i > 0 && i%lineEvery == 0 {
			// this is a full line

			buildingLine = buildingLine[:len(buildingLine)-1] // trailing space
			for j := range lineTokens {
				m := lineTokens[j]
				m.f = buildingLine
				mocked = append(mocked, m)
			}
			buildingLine = ""
			curLine++
			curLinePos = 1
			lineTokens = make([]mockToken, 0)
		}
	}

	if len(lineTokens) > 0 {
		// this is a partial line
		if len(buildingLine) > 0 {
			buildingLine = buildingLine[:len(buildingLine)-1] // trailing space
		}
		for i := range lineTokens {
			m := lineTokens[i]
			m.f = buildingLine
			mocked = append(mocked, m)
		}
	}

	return &mockStream{tokens: mocked}
}
