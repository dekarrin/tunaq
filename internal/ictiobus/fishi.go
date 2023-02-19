package ictiobus

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dekarrin/tunaq/internal/ictiobus/lex"
	"github.com/gomarkdown/markdown"
	mkast "github.com/gomarkdown/markdown/ast"
	mkparser "github.com/gomarkdown/markdown/parser"
)

type fishiScanner bool

func (fs fishiScanner) RenderNode(w io.Writer, node mkast.Node, entering bool) mkast.WalkStatus {
	if !entering {
		return mkast.GoToNext
	}

	codeBlock, ok := node.(*mkast.CodeBlock)
	if !ok || codeBlock == nil {
		return mkast.GoToNext
	}

	if strings.ToLower(strings.TrimSpace(string(codeBlock.Info))) == "fishi" {
		w.Write(codeBlock.Literal)
	}
	return mkast.GoToNext
}

func (fs fishiScanner) RenderHeader(w io.Writer, ast mkast.Node) {}
func (fs fishiScanner) RenderFooter(w io.Writer, ast mkast.Node) {}

func ReadFishiMdFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	err = ProcessFishiMd(data)
	if err != nil {
		return err
	}

	return nil
}

func ProcessFishiMd(mdText []byte) error {
	fishiSource := GetFishiFromMarkdown(mdText)
	fishiSource = Preprocess(fishiSource)
	fishi := bytes.NewBuffer(fishiSource)

	lx := CreateBootstrapLexer()
	stream, err := lx.Lex(fishi)
	if err != nil {
		return err
	}

	for stream.HasNext() {
		fmt.Printf("%s\n", stream.Next().String())
	}

	return nil
}

func GetFishiFromMarkdown(mdText []byte) []byte {
	doc := markdown.Parse(mdText, mkparser.New())
	var scanner fishiScanner
	fishi := markdown.Render(doc, scanner)
	return fishi
}

// Preprocess does a preprocess step on the source, which as of now includes
// stripping comments and normalizing end of lines to \n.
func Preprocess(source []byte) []byte {
	toBuf := make([]byte, len(source))
	copy(toBuf, source)
	scanner := bufio.NewScanner(bytes.NewBuffer(toBuf))
	var preprocessed strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		line, _, _ = strings.Cut(line, "#")
		preprocessed.WriteString(line)
		preprocessed.WriteRune('\n')
	}

	return []byte(preprocessed.String())
}

func CreateBootstrapLexer() Lexer {
	bootLx := NewLexer()

	tcHeaderTokens := lex.NewTokenClass("tokens_header", "'tokens' header")
	tcHeaderGrammar := lex.NewTokenClass("grammar_header", "'grammar' header")
	tcHeaderActions := lex.NewTokenClass("actions_header", "'actions' header")
	tcDirAction := lex.NewTokenClass("action_dir", "'action' directive")
	tcDirDefault := lex.NewTokenClass("default_dir", "'default' directive")
	tcDirHook := lex.NewTokenClass("hook_dir", "'hook' directive")
	tcDirHuman := lex.NewTokenClass("human_dir", "'human' directive")
	tcDirIndex := lex.NewTokenClass("index_dir", "'index' directive")
	tcDirProd := lex.NewTokenClass("prod_dir", "'prod' directive")
	tcDirShift := lex.NewTokenClass("shift_dir", "'stateshift' directive")
	tcDirStart := lex.NewTokenClass("start_dir", "'start' directive")
	tcDirState := lex.NewTokenClass("state_dir", "'state' directive")
	tcDirSymbol := lex.NewTokenClass("symbol_dir", "'symbol' directive")
	tcDirToken := lex.NewTokenClass("token_dir", "'token' directive")
	tcDirWith := lex.NewTokenClass("with_dir", "'with' directive")
	tcFreeformText := lex.NewTokenClass("freeform_text", "freeform text value")
	tcNewline := lex.NewTokenClass("newline", "'\n'")
	tcTerminal := lex.NewTokenClass("terminal", "terminal symbol")
	tcNonterminal := lex.NewTokenClass("nonterminal", "non-terminal symbol")
	tcEq := lex.NewTokenClass("eq", "'='")
	tcAlt := lex.NewTokenClass("alt", "'|'")
	tcAttrRef := lex.NewTokenClass("attr_ref", "attribute reference")
	tcInt := lex.NewTokenClass("int", "integer value")
	tcId := lex.NewTokenClass("id", "identifier")
	tcEscseq := lex.NewTokenClass("escseq", "escape sequence")

	// default state, shared by all
	bootLx.RegisterClass(tcEscseq, "")
	bootLx.RegisterClass(tcHeaderTokens, "")
	bootLx.RegisterClass(tcHeaderGrammar, "")
	bootLx.RegisterClass(tcHeaderActions, "")
	bootLx.RegisterClass(tcDirStart, "")
	bootLx.RegisterClass(tcDirState, "")

	// default patterns and defs
	bootLx.AddPattern(`(?:%!.)`, lex.LexAs(tcEscseq.ID()), "")
	bootLx.AddPattern(`%%[Tt][Oo][Kk][Ee][Nn][Ss]`, lex.LexAndSwapState(tcHeaderTokens.ID(), "tokens"), "")
	bootLx.AddPattern(`%%[Gg][Rr][Aa][Mm][Mm][Aa][Rr]`, lex.LexAndSwapState(tcHeaderGrammar.ID(), "grammar"), "")
	bootLx.AddPattern(`%%[Aa][Cc][Tt][Ii][Oo][Nn][Ss]`, lex.LexAndSwapState(tcHeaderActions.ID(), "actions"), "")
	bootLx.AddPattern(`%[Ss][Tt][Aa][Rr][Tt]`, lex.LexAs(tcDirStart.ID()), "")
	bootLx.AddPattern(`%[Ss][Tt][Aa][Tt][Ee]`, lex.LexAs(tcDirState.ID()), "")

	// tokens classes
	bootLx.RegisterClass(tcFreeformText, "tokens")
	bootLx.RegisterClass(tcDirShift, "tokens")
	bootLx.RegisterClass(tcDirHuman, "tokens")
	bootLx.RegisterClass(tcDirToken, "tokens")
	bootLx.RegisterClass(tcDirDefault, "tokens")
	bootLx.RegisterClass(tcNewline, "tokens")

	// tokens patterns
	bootLx.AddPattern(`%[Sa][Tt][Aa][Tt][Ee][Ss][Hh][Ii][Ff][Tt]`, lex.LexAs(tcDirShift.ID()), "tokens")
	bootLx.AddPattern(`%[Hh][Uu][Mm][Aa][Nn]`, lex.LexAs(tcDirHuman.ID()), "tokens")
	bootLx.AddPattern(`%[Tt][Oo][Kk][Ee][Nn]`, lex.LexAs(tcDirToken.ID()), "tokens")
	bootLx.AddPattern(`%[Dd][Ee][Ff][Aa][Uu][Ll][Tt]`, lex.LexAs(tcDirDefault.ID()), "tokens")
	bootLx.AddPattern(`\n`, lex.LexAs(tcNewline.ID()), "tokens")
	bootLx.AddPattern(".+", lex.LexAs(tcFreeformText.ID()), "tokens")

	// grammar classes
	bootLx.RegisterClass(tcNewline, "grammar")
	bootLx.RegisterClass(tcEq, "grammar")
	bootLx.RegisterClass(tcAlt, "grammar")
	bootLx.RegisterClass(tcNonterminal, "grammar")
	bootLx.RegisterClass(tcTerminal, "grammar")

	// gramamr patterns
	bootLx.AddPattern(`\n`, lex.LexAs(tcNewline.ID()), "grammar")
	bootLx.AddPattern(`\s+`, lex.Discard(), "grammar")
	bootLx.AddPattern(`=`, lex.LexAs(tcEq.ID()), "grammar")
	bootLx.AddPattern(`\|`, lex.LexAs(tcAlt.ID()), "grammar")
	bootLx.AddPattern(`{[A-Za-z].*}`, lex.LexAs(tcNonterminal.ID()), "grammar")
	bootLx.AddPattern(`.+`, lex.LexAs(tcTerminal.ID()), "grammar")

	// actions classes
	bootLx.RegisterClass(tcAttrRef, "actions")
	bootLx.RegisterClass(tcInt, "actions")
	bootLx.RegisterClass(tcNonterminal, "actions")
	bootLx.RegisterClass(tcDirSymbol, "actions")
	bootLx.RegisterClass(tcDirProd, "actions")
	bootLx.RegisterClass(tcDirWith, "actions")
	bootLx.RegisterClass(tcDirHook, "actions")
	bootLx.RegisterClass(tcDirAction, "actions")
	bootLx.RegisterClass(tcDirIndex, "actions")
	bootLx.RegisterClass(tcId, "actions")
	bootLx.RegisterClass(tcTerminal, "actions")

	// actions patterns
	bootLx.AddPattern(`\s+`, lex.Discard(), "actions")
	bootLx.AddPattern(`[A-Za-z][A-Za-z0-9_-]*(?:\$\d+)?\.[\$A-Za-z][$A-Za-z0-9_-]*`, lex.LexAs(tcAttrRef.ID()), "actions")
	bootLx.AddPattern(`[0-9]+`, lex.LexAs(tcInt.ID()), "actions")
	bootLx.AddPattern(`{[A-Za-z].*}`, lex.LexAs(tcNonterminal.ID()), "actions")
	bootLx.AddPattern(`%[Ss][Yy][Mm][Bb][Oo][Ll]`, lex.LexAs(tcDirSymbol.ID()), "actions")
	bootLx.AddPattern(`%[Pp][Rr][Oo][Dd]`, lex.LexAs(tcDirProd.ID()), "actions")
	bootLx.AddPattern(`%[Ww][Ii][Tt][Hh]`, lex.LexAs(tcDirWith.ID()), "actions")
	bootLx.AddPattern(`%[Hh][Oo][Oo][Kk]`, lex.LexAs(tcDirHook.ID()), "actions")
	bootLx.AddPattern(`%[Aa][Cc][Ti][Ii][Oo][Nn]`, lex.LexAs(tcDirAction.ID()), "actions")
	bootLx.AddPattern(`%[Ii][Nn][Dd][Ee][Xx]`, lex.LexAs(tcDirIndex.ID()), "actions")
	bootLx.AddPattern(`[A-Za-z][A-Za-z0-9_-]*`, lex.LexAs(tcId.ID()), "actions")
	bootLx.AddPattern(`.+`, lex.LexAs(tcTerminal.ID()), "grammar")

	return bootLx
}
