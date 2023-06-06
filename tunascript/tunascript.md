# Tunascript Language

This is the specification for the Tunascript language. It is written in FISHI
and intended to be consumed by the ictcc command of
[Ictiobus](github.com/dekarrin/ictiobus) to produce the Tunascript frontend.

## Parser

```fishi
%%grammar

{TUNASCRIPT}        =   {EXPR}

# right assiociativity puts the recursion on the right, left does the left.

{EXPR}              =   id set {EXPR}
                    |   id += {EXPR}
                    |   id -= {EXPR}
                    |   {BOOL-OP}

{BOOL-OP}           =   {BOOL-OP} or {EQUALITY}
                    |   {BOOL-OP} and {EQUALITY}
                    |   {EQUALITY}

{EQUALITY}          =   {EQUALITY} eq {COMPARISON}
                    |   {EQUALITY} ne {COMPARISON}
                    |   {COMPARISON}

{COMPARISON}        =   {COMPARISON} < {SUM}
                    |   {COMPARISON} > {SUM}
                    |   {COMPARISON} <= {SUM}
                    |   {COMPARISON} >= {SUM}
                    |   {SUM}

{SUM}               =   {SUM} + {PRODUCT}
                    |   {SUM} - {PRODUCT}
                    |   {PRODUCT}

{PRODUCT}           =   {PRODUCT} * {NEGATION}
                    |   {PRODUCT} / {NEGATION}
                    |   {NEGATION}

{NEGATION}          =   ! {NEGATION}
                    |   - {NEGATION}
                    |   {TERM}

{TERM}              =   lp {EXPR} rp
                    |   id {ARG-LIST}
                    |   {VALUE}

{ARG-LIST}          =   lp {ARGS} rp
                    |   lp rp

{ARGS}              =   {ARGS} comma {EXPR}
                    |   {EXPR}

{VALUE}             =   id ++ | id -- | id | num | @str | str | bool
```

## Lexer

```fishi
%%tokens

-=                      %token -=       %human minus-equals sign "-="
\+=                     %token +=       %human plus-equals sign "+="
<                       %token <        %human less-than sign "<"
<=                      %token <=       %human less-than-equals sign "<="
>                       %token >        %human greater-than sign ">"
>=                      %token >=       %human greater-than-equals sign ">="
!=                      %token ne       %human not-equals sign "!="
=                       %token set      %human set operator "="
==                      %token eq       %human double-equals sign "=="
-                       %token -        %human minus sign "-"
\+                      %token +        %human plus sign "+"
\+\+                    %token ++       %human increment operator "++"
--                      %token --       %human decrement operator "--"
\*                      %token *        %human multiplication sign "*"
/                       %token /        %human divide sign "/"
!                       %token !        %human logical-not operator "!"
&&                      %token and      %human logical-and operator "&&"
\|\|                    %token or       %human logical-or operator "||"
,                       %token comma    %human ","
\(                      %token lp       %human "("
\)                      %token rp       %human ")"
\$[A-Za-z0-9_]+         %token id       %human identifier

[Tt][Rr][Uu][Ee]|[Ff][Aa][Ll][Ss][Ee]   %token bool
[Oo][Nn]|[Oo][Ff][Ff]                   %token bool
[Yy][Ee][Ss]|[Nn][Oo]                   %token bool
%human boolean (true/false) value

(?:\d+(?:\.\d*)?|\.\d+)(?:[Ee]-?\d+)?   %token num
%human number value

@(?:\\.|[^@\\])*@                       %token @str
%human @-text value

# Gonna go ahead and make the call that no, we're not doing unquoted values that
# include whitespace. As cool as that is, it adds the concept of semantic
# whitespace which vastly complicates the lexer and will probably confuse the
# heck out of human operators. If they want to include whitespace in a value,
# they can quote it or else make an escape sequence.

(?:\\.|\S)(?:\.*(?:\\.|\S))?            %token str
%human text value


# finally, discard any and all whitespace
\s+                                     %discard
```

## SDTS

Minimal SDTS for the moment while we get the rest of things in order.

```fishi
%%actions

%symbol {TUNASCRIPT}
->:                     {^}.value = test_const()
```
