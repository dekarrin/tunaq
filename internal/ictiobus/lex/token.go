package lex

import (
	"fmt"
	"strings"

	"github.com/dekarrin/tunaq/internal/ictiobus/types"
)

// implementation of TokenClass interface for lex package use only.
type lexerClass struct {
	id   string
	name string
}

func (lc lexerClass) ID() string {
	return lc.id
}

func (lc lexerClass) Human() string {
	return lc.name
}

func (lc lexerClass) Equal(o any) bool {
	other, ok := o.(types.TokenClass)
	if !ok {
		otherPtr, ok := o.(*types.TokenClass)
		if !ok {
			return false
		}
		if otherPtr == nil {
			return false
		}
		other = *otherPtr
	}

	return other.ID() == lc.ID()
}

func NewTokenClass(id string, human string) lexerClass {
	return lexerClass{id: id, name: human}
}

// implementation of Token interface for lex package use only
type lexerToken struct {
	class   types.TokenClass
	lexed   string
	linePos int
	lineNum int
	line    string
}

func (lt lexerToken) Class() types.TokenClass {
	return lt.class
}

func (lt lexerToken) Lexeme() string {
	return lt.lexed
}

func (lt lexerToken) LinePos() int {
	return lt.linePos
}

func (lt lexerToken) Line() int {
	return lt.lineNum
}

func (lt lexerToken) FullLine() string {
	return lt.line
}

func (lt lexerToken) String() string {
	// turn all newline chars into \n because we dont want that in the output
	fmtStr := "(%s %d:%d \"%s\")"
	content := strings.ReplaceAll(lt.lexed, "\n", "\\n")
	return fmt.Sprintf(fmtStr, lt.class.ID(), lt.lineNum, lt.linePos, content)
}
