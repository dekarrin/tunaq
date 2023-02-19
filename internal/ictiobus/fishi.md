# Frontend Instruction Specification for self-Hosting Ictiobus (FISHI), v1.0
This is a grammar for the Frontend Instruction Specification for
(self-)Hosting Ictiobus (FISHI). It is for version 1.0 and this version was
started on 2/18/23 by Jello! Glubglu8glub 38D

## Escape Sequence
To escape something that would otherwise have special meaning in FISHI, use the
escape sequence directly before it, `%!`.

## Format Use
Languages that describe themselves in the FISHI language are taken from
definitions described with FISHI for the frontend of Ictiobus and used to
produce a compiler frontend.

These definitions are to be embedded in Markdown-formatted text in special code
blocks delimited with the triple-tick that are marked with the special syntax
tag `fishi`, as in the following:

    ```fishi
    (FISHI directives would go here)
    ```

Multiple consecutive `fishi` code blocks in the same file are appended together
to create the full source that is parsed.

## Parser
This is the context-free grammar for FISHI, glub.

```fishi
%%grammar

{fishispec}            = {blocks}

{blocks}               = {blocks} {block}
                       | {block}

{block}                = {tokens-block} | {grammar-block} | {actions-block}

{actions-block}        = ACTIONS_HEADER {actions-content}

{actions-content}      = {actions-state-block}
                       | {actions-content} {actions-state-block}

# actions does not include NEWLINES because that mode of lexer does not lex
# newlines as they have no semantic meaning in %%action blocks

{actions-state-block}  = {state-ins} {symbol-actions}
                       | {symbol-actions}

{symbol-actions}       = SYMBOL_DIR NONTERMINAL {prod-actions}

{prod-actions}         = {prod-actions} {prod-action}
                       | {prod-action}

{prod-action}          = {prod-specifier} {semantic-actions}

{prod-specifier}       = PROD_DIR {prod-addr}

{prod-addr}            = INDEX_DIR INT
                       | {actions-production}
                       | # empty; note: only way to specify epsilon production is to use index

{actions-production}   = {actions-production} {symbol}
                       | {symbol}

{semantic-actions}     = {semantic-actions} {semantic-action}
                       | {semantic-action}

{semantic-action}      = ACTION_DIR ATTR_REF HOOK_DIR ID {with-clause}

{with-clause}          = {explicit-with} {implicit-withs}
                       | {explicit-with}
                       | # empty string

{explicit-with}        = WITH_DIR ATTR_REF

{implicit-withs}       = {implicit-withs} {implicit-with}
                       | {implicit-with}

{implicit-with}        = {explicit-with}
                       | ATTR_REF

{grammar-block}        = GRAMMAR_HEADER {grammar-content}

{grammar-content}      = {grammar-state-block}
                       | {grammar-content} {grammar-state-block}

{grammar-state-block}  = {state-ins} {newlines} {grammar-rules}
                       | {grammar-rules}

{grammar-rules}        = {grammar-rules} {newlines} {grammar-rule}
                       | {grammar-rule}

{grammar-rule}         = NONTERMINAL EQ {opt-newlines} {alternations}

{alternations}         = {production}
                       | {alternations} {opt-newlines} ALT {production}

{production}           = {production} {symbol}
                       | {symbol}
                       | # empty string for epsilon

{symbol}               = NONTERMINAL
                       | TERMINAL
                       | EQ

                       # need to include EQ above bc lexer cannot distinguish
                       # between 'eq operator at start' and "=" used as
                       # production symbol. we will treat it as a normal
                       # non-term if it's in the productions.

{tokens-block}         = TOKENS_HEADER {tokens-content}

{state-ins}            = STATE_DIR {state-expr}

{state-expr}           = {id-expr}
                       | {newlines} {id-expr}

# any of these COULD be an ID, the lexer's weird multi-state thing makes this
# difficult atm:
{id-expr}              = ID | TERMINAL | FREEFORM_TEXT

{opt-newlines}         = {newlines}
                       | #empty string

{newlines}             = NEWLINE
                       | {newlines} NEWLINE

{tokens-content}       = {tokens-state-block}
                       | {tokens-content} {tokens-state-block}

{tokens-state-block}   = {state-ins} {newlines} {tokens-entries}
                       | {tokens-entries}

{tokens-entries}       = {tokens-entries} {newlines} {tokens-entry}
                       | {tokens-entry}

{tokens-entry}         = {pattern} {opt-newlines} {token-entry-opts}

{token-entry-opts}     = {token-entry-opts} {opt-newlines} {token-option}
                       | {token-option}

{token-option}         = {stateshift}
                       | {token}
                       | {human}

{stateshift}           = SHIFT_DIR FREEFORM_TEXT
{token}                = TOKEN_DIR FREEFORM_TEXT
{human}                = HUMAN_DIR FREEFORM_TEXT
```

## Lexer
The following gives the lexical specification for the FISHI language.

```fishi
%%tokens

%%[Tt][Oo][Kk][Ee][Nn][Ss]        %token tokens_header
%human Token header mark          %stateshift tokens

%%[Gg][Rr][Aa][Mm][Mm][Aa][Rr]    %token grammar_header  
%human Grammar header mark        %stateshift grammar

%%[Aa][Cc][Tt][Ii][Oo][Nn][Ss]    %token actions_header
%human Action header mark         %stateshift actions

%[Ss][Tt][Aa][Rr][Tt]             %token start_dir
%human start directive
```

For tokens state:

```fishi
%state tokens

# escapes need to be handled here
((?:%!%!.|.)+)(?:{token_dir}|{human_dir}|{state_dir}|{shift_dir}|{:start_dir}|{default_dir}\n)
%token freeform_text
%human freeform-text value

%[Sa][Tt][Aa][Tt][Ee][Ss][Hh][Ii][Ff][Tt]
                                %token shift_dir    %human state-shift directive
%[Ss][Tt][Aa][Tt][Ee]           %token state_dir    %human state directive
%[Hh][Uu][Mm][Aa][Nn]           %token human_dir    %human human directive
%[Tt][Oo][Kk][Ee][Nn]           %token token_dir    %human token directive
%[Dd][Ee][Ff][Aa][Uu][Ll][Tt]   %token default_dir  %human default directive
\n                              %token newline      %human new line
```

For grammar state:

```fishi
%state grammar

\n                          %token newline       %human new line
\s+                         # discard other whitespace

((?:%!%!.|.)+)(?:{eq}|{alt}|{newline}|\s+|{non-term}|{state_dir}|{:start_dir})
                            %token terminal      %human terminal

# this will result in needing to escape equals which may be used. oh well.
# somefin to fix in later versions

=                           %token eq            %human '='
\|                          %token alt           %human '|'
%[Ss][Tt][Aa][Tt][Ee]       %token state_dir     %human state directive
%!{[A-Za-z].*%!}            %token nonterminal   %human non-terminal
```

For actions state:
```fishi
%state action

\s+                         # discard all whitespace

[A-Za-z][A-Za-z0-9_-]*(?:\$\d+)?\.[\$A-Za-z][$A-Za-z0-9_-]*
                             %token attr_ref      %human attribute reference
[0-9]+                       %token int           %human integer
%!{[A-Za-z].*%!}             %token nonterminal   %human non-terminal
%[Ss][Yy][Mm][Bb][Oo][Ll]    %token symbol_dir    %human symbol directive
%[Pp][Rr][Oo][Dd]            %token prod_dir      %human prod directive
%[Ww][Ii][Tt][Hh]            %token with_dir      %human with directive
%[Hh][Oo][Oo][Kk]            %token hook_dir      %human hook directive
%[Ss][Tt][Aa][Tt][Ee]        %token state_dir     %human state directive
%[Aa][Cc][Ti][Ii][Oo][Nn]    %token action_dir    %human action directive
%[Ii][Nn][Dd][Ee][Xx]        %token index_dir     %human index directive
[A-Za-z][A-Za-z0-9_-]*       %token id            %human identifier
(?:%!%!.|.)                  %token terminal      %human terminal

```
