# TunaQuest Expansion Mini-Language

This is the specification for the expansion mini-language of TunaQuest used
inside expandable blocks of text. This parses flags out of the text and parses
templating IF-statements. This specification is written in FISHI and can be
parsed with [Ictiobus](github.com/dekarrin/ictiobus) to produce the expansion
frontend.

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
%token else
```

## SDTS

Minimal SDTS for the moment while we get the rest of things in order.

```fishi
%%actions

%symbol {EXPANSION}
-> {BLOCKS}:              {^}.ast = ast({BLOCKS}.nodes)

%symbol {BLOCKS}
-> {BLOCKS} {BLOCK}:    {^}.nodes = node_list({BLOCK}.node, {BLOCKS}.nodes)
-> {BLOCK}:             {^}.nodes = node_list({BLOCK}.node)

%symbol {BLOCK}
-> text     :   {^}.node = text( text.$text)
-> flag     :   {^}.node = flag( flag.$text)
-> {BRANCH} :   {^}.node = identity({BRANCH}.node)


%symbol {BRANCH}
-> if {BLOCKS} endif                         :
{^}.node = branch( if.$text, {BLOCKS}.nodes)

-> if {BLOCKS} {ELSEIFS} endif               :
{^}.node = branch( if.$text, {BLOCKS}.nodes, {ELSEIFS}.conds)

-> if {BLOCKS} else {BLOCKS} endif           :
{^}.node = branch_with_else( if.$text, {&0}.nodes, {&1}.nodes)

-> if {BLOCKS} {ELSEIFS} else {BLOCKS} endif :
{^}.node = branch_with_else( if.$text, {&0}.nodes, {&2}.nodes, {ELSEIFS}.conds)


%symbol {ELSEIFS}
-> {ELSEIFS} elseif {BLOCKS}:   {^}.conds = cond_list( elseif.$text, {BLOCKS}.nodes, {ELSEIFS}.conds)
-> elseif {BLOCKS}:             {^}.conds = cond_list( elseif.$text, {BLOCKS}.nodes)

```
