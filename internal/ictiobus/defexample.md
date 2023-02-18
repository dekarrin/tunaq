Example Language
################
This is a complete example of a language specified in the special ictiobus
format. It is a markdown based format that uses specially named sections to
allow both specificiation of a language and freeform text to co-exist.

It will process the sections in order, and parse only those sections in code
fences with the `ictio` syntax specifier.

Specifying Terminals
--------------------
One of the first steps in any language specification is to give the patterns
that the lexer needs to identify tokens in source text. Ictiobus allows this to
be done programmatically if desired, or in a markdown file code block that
starts with the line `%%tokens`.

```ictio
# Comments can be included in any line in an ictio block with the '#' character.
# All content from the first # encountered until the end of the line are
# considered a comment.
#
# If a literal '#' is needed at any point in the ictio source, it can be escaped
# with a backslash.

%%tokens

# if a %state X sub-section is found in a %%token section, then all entries will
# be considered to apply only to state X of the lexer. The first state given
# is considered to be the starting state; otherwise, %start Y can be given at
# any point in a %%token section (but before the first %state) to say that the
# lexer should begin in that state.
#
# if no states are explicitly given, all rules are assumed to apply to the same,
# default lexer state.
#
# State names are case-insensitive.

%state normal

[A-Za-z_][A-Za-z0-9_]+    %token identifier    %human "identifier sequence"
"(?:\\\\|\\"|[^"])+"      %token str           %human "string"
\d+                       %token int           %human "integer"
\+                        %token +             %human "'+'"
\*                        %token *             %human "'*'"
\(                        %token (             %human "'('"
\)                        %token )             %human "')'"
<                         %token <             %human "'<'"    %stateshift angled

# this state shift sequence is very contrived and could easily be avoided by
# simply not state-shifting on <, but it makes for a good example and test.

%state angled

# multiple %human definitions are allowed if the others are in another state
# we'll use this state to create some of the same tokens with slightly different
# human-readable names to reflect that they were in another state, but by using
# the same token class names, we can treat them all the same regardless of which
# state they were produced in in the cfg.

>                         %token >             %human "'>'"    %stateshift normal
"(?:\\\\|\\"|[^"])+"      %token str           %human "angly string"
\d+                       %token int           %human "angly int"
,                         %token ,             %human ","
```

All lexer specifications begin with a pattern in regular expression RE2 syntax.
As the lexer uses the built-in RE2 engine in Go to analyze input and identify
tokens and state shifts, these must follow the allowed syntax given in the
[RE2 Specification](https://github.com/google/re2/wiki/Syntax).

As lookaheads and lookbehinds are not supported at this time, a single capture
group may be given within the regex that gives the lexeme of interest; whatever
is captured in this group will be what becomes the body of a parsed token. Note
that if this method is used, when advancing token input, ONLY the characters
that come before the subexpression and those captured in the subexpression
itself will be considered 'processed'; any that match after the capturing group
will be processed during the next attempt to read a token.

A subexpression is optional; if one is not provided, the entire pattern is
considered to be of interest. If there is more than 1 subexpression in a
pattern, it is an error, and the specification will not be accepted.
Non-capturing groups are not subject to this limitation and may be given as many
times as desired.

For example, if the input being processed is `int3alpha double`, then the
pattern `[A-Za-z]+(\d)alpha` would match the `int3alpha` and the `3` would be
set as the lexeme parsed for the associated token. The `[A-Za-z]+` in the
pattern that comes directly before the capturing group would match the `int`,
and the `alpha` part of the pattern would match the literal `alpha` in the
input. `int` comes before the capture group that received `3`, and so the input
would be advanced past `int3` and the next lexical analysis pass would attempt
to match against `alpha double`.

Patterns can embed previously-defined patterns within them. Just wrap the ID of
the pattern within '{' and '}' in the regex. The ID is automatically generated
for each pattern as simply the order that it occurs within an ictiobus
specification, starting with 1. So the first pattern is {1}, the second is {2},
etc. To use a literal '{' or '}' within a pattern, escape them with a backslash.

After the pattern, a sequence of lexer directives are given that tell the
lexer what to do on matching the pattern.

They each begin with a keyword starting with %, and can be specified in any
order after the pattern. To use a % literally in a pattern, escape it with a
backslash.

The most common directive is `%token`, which tells the lexer to take the input
that the pattern matched (or the input that is in the capturing group in the
regex, if one is specified), and create a new token. The name of the class of
that token is given after `%token`; it can contain any characters (but a %
must be escaped) but is case-insensitive; conventionally, they are given as
lower-case, as internally they will be converted to lowercase.

* `%token CLASS` - Specifies to take the matched input and use it to create a
new token of the given class. `CLASS` may be any characters but note that they
are case-insenseitve and will be represented internally as lower-case. The same
token class may be given for different patterns; this means that a match against
any of those patterns will result in a token of that class.
* `%human "HUMAN-READABLE NAME" - Specifies that the token class given should
have the human-readable name given between the two quotes. This is generally
used for reporting errors related to that token. This human readable name is
associated with the token named by the `%token` directive on the same pattern;
if no `%token` is given, it is an error. The `%human` part of the token is
shared by all uses of that token class; this means that `%human` may only be
specified one time for a given `%token` class. It is an error to have multiple
`%human` directives across patterns with the same token class name. If desired
this can be handled entirely in a `%tokendef class "human"` directive outside
of any pattern.
* `%changestate STATE` - Specifies that the lexer should swap state to STATE
and then continue lexing. If in the same rule as a `%token`, the token will be
lexed in the prior state before swapping to the new one.

Specifying Grammar
------------------
The grammar of the language accepted by the parsing phase is given in a
`%%grammar` section in an `ictio` code block. In that section is a context-free
grammar that uses a "BNF-ish" notation. In this notation, a non-terminal is
defined by a symbol name on the left side enclosed in angle brackes `<` and `>`,
followed by space and the sequence `::=`, followed by one or more productions
for the non-terminal separated by a `|` character.

Terminals are specified by giving the name of the token class that makes the
terminal. This can be any `%token` class name defined in a `%%tokens` section.

The start production of the language is the first non-terminal that is defined
in a `%%grammar` section.

```ictio
%%grammar

<S>     :=   <S>


%%tokens

%state normal

[A-Za-z_][A-Za-z0-9_]+    %token identifier    %human "identifier sequence"
"(?:\\\\|\\"|[^"])+"      %token str           %human "string"
\d+                       %token int           %human "integer"
\+                        %token +             %human "'+'"
\*                        %token *             %human "'*'"
\(                        %token (             %human "'('"
\)                        %token )             %human "')'"
<                         %token <             %human "'<'"    %stateshift angled

# this state shift sequence is very contrived and could easily be avoided by
# simply not state-shifting on <, but it makes for a good example and test.

%state angled

# multiple %human definitions are allowed if the others are in another state
# we'll use this state to create some of the same tokens with slightly different
# human-readable names to reflect that they were in another state, but by using
# the same token class names, we can treat them all the same regardless of which
# state they were produced in in the cfg.

>                         %token >             %human "'>'"    %stateshift normal
"(?:\\\\|\\"|[^"])+"      %token str           %human "angly string"
\d+                       %token int           %human "angly int"
,                         %token ,             %human ","
```

```



There can even be multiple `%%tokens` blocks if
desired.