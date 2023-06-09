# Tunascript Expansion Mini-Language

This is the specification for the subset of TunaScript allowed inside expandable
blocks of text. This parses flags out of the text and parses templating
IF-statements. It is intended to be a super-set of sorts of TunaScript, though
This specification is written in FISHI and can be parsed with
[Ictiobus](github.com/dekarrin/ictiobus) to produce the expansion frontend.

This includes only the parts that have embedded tunascript, and not tunascript
itself (that is in tunascript.md). To use it with Ictiobus, pass this file as
the first argument to ictcc, and pass tunascript.md as the second argument.

## Parser

```fishi
%%grammar

{EXPANSION}         =   {BLOCKS}

{BLOCKS}            =   {BLOCKS} {BLOCK} | {BLOCK}

{BLOCK}             =   text | flag | {BRANCH}

{BRANCH}            =   if {BLOCKS} endif
                    |   if {BLOCKS} {ELSEIFS} endif
                    |   if {BLOCKS} else {BLOCKS} endif
                    |   if {BLOCKS} {ELSEIFS} else {BLOCKS} endif

{ELSEIFS}           =   {ELSEIFS} elseif {BLOCKS}
                    |   elseif {BLOCKS}
```

## Lexer

```fishi
%%tokens

(?:[^\\\$]|\\.)+
%token text

\$[A-Za-z0-9_]+
%token flag

\$\[\[\s*[Ii][Ff](?:\s+(?:[^\\\]]|\][^\]]|\\.)*)?\]\]
%token if

\$\[\[\s*[Ee][Ll](?:[Ss][Ee]\s*)?[Ii][Ff](?:\s+(?:[^\\\]]|\][^\]]|\\.)*)?\]\]
%token elseif

\$\[\[\s*[Ee][Nn][Dd]\s*[Ii][Ff]\s*\]\]
%token endif

\$\[\[\s*[Ee][Ll][Ss][Ee]\s*\]\]
```

## SDTS

Minimal SDTS for the moment while we get the rest of things in order.

```fishi
%%actions

%symbol {EXPANSION}
-> {BLOCKS}:              {^}.ast = test_const()

```
