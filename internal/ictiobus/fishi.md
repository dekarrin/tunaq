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

{fishispec}     = {blocks}

{blocks}        = {blocks} {block}
                | {block}

{block}         = {token-block} | HEADER {directives}

{token-block}   = 

{directives}    = {}


```


## Lexer
The following gives the lexical specification for the FISHI language.

```fishi
%%tokens

%%[Tt][Oo][Kk][Ee][Nn][Ss]                   %token tokens_header
%human Token header mark                     %stateshift tokens

%%[Gg][Rr][Aa][Mm][Mm][Aa][Rr]               %token grammar_header
%human Grammar header mark                   %stateshift grammar

%%[Aa][Cc][Tt][Ii][Oo][Nn][Ss]               %token actions_header
%human Action header mark                    %stateshift actions

%[Tt][Oo][Kk][Ee][Nn]                        %token token_dir
%human token directive

%[Sa][Tt][Aa][Tt][Ee][Ss][Hh][Ii][Ff][Tt]    %token shift_dir
%human state-shift directive

%[Ss][Tt][Aa][Tt][Ee]                        %token state_dir
%human state directive

%[Hh][Uu][Mm][Aa][Nn]                        %token human_dir
%human human directive

%[Ss][Tt][Aa][Rr][Tt]                        %token start_dir
%human start directive

%[Ss][Yy][Mm][Bb][Oo][Ll]                    %token symbol_dir
%human symbol directive

%[Pp][Rr][Oo][Dd]                            %token prod_dir
%human prod directive

%[Ww][Ii][Tt][Hh]                            %token with_dir
%human with directive

%[Hh][Oo][Oo][Kk]                            %token hook_dir
%human hook directive

%[Aa][Cc][Ti][Ii][Oo][Nn]                    %token action_dir
%human action directive

%[Ii][Nn][Dd][Ee][Xx]                        %token index_dir
%human index directive

%[Dd][Ee][Ff][Aa][Uu][Ll][Tt]                %token default_dir
%human default directive



```

For tokens state:

```fishi
%state tokens

# escapes need to be handled here
((?:%!.|.)+)(?:{token_dir}|{human_dir}|{state_dir}|{shift_dir}|\n)
%token freeform_text
%human freeform-text value

%[Sa][Tt][Aa][Tt][Ee][Ss][Hh][Ii][Ff][Tt]    %token shift_dir
%human state-shift directive

%[Ss][Tt][Aa][Tt][Ee]                        %token state_dir
%human state directive

%[Hh][Uu][Mm][Aa][Nn]                        %token human_dir
%human human directive

%[Tt][Oo][Kk][Ee][Nn]                        %token token_dir
%human token directive

\n                                           %token newline

```

For grammar state:

```fishi
%state grammar

%!{[A-Za-z].*%!}                            %token non-term
%human non-terminal

# this will result in 
=                                           %token eq




(?:(?:\s*(?:\S+\s+)+)|(?:\s+\S+)+|\S+)    %token text






# this horrible syntax with the escapes everywhere is glubbin due to use of '}'
# there will always be at least ONE that is a massive pita and for this it is }.
# esp w the egregious escape char made as such so as not to conflict w backslash
# which is used in so many glubbin places due to regex

%!{%!{(?:|%!%%!!%!%%!!|%!%%!!}|%!}[^%!}]|[^%!}]|)*%!}%!}
%token str %human "string"

[]

\n                                      %token newline         %human "new line"


\s+                                     # (no action; discard other whitespace)
                           


```
