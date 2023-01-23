Context Free Grammar of Tunascript

Grammar is BNFish

e is our epsilon.

S                      ::= sum-expr

expr                   ::= p20-expr

p20-expr               ::= p19-expr SEPARATOR p20-expr
                         | p19-expr

// right associativity
p19-expr                ::= p19-expr OP_SET p18-expr
                          | p18-expr

p18-expr                ::= p18-expr OP_INCSET p17-expr
                          | p17-expr

p17-expr                ::= p17-expr OP_DECSET p16-expr
                          | p16-expr

// and back to left
p16-expr                ::= p15-expr OP_OR p16-expr
                         |  p15-expr

p15-expr                ::= p14-expr OP_AND p15-expr
                         |  p14-expr

p14-expr                ::= p13-expr OP_IS p14-expr
                          | p13-expr

p13-expr               ::= p12-expr OP_IS_NOT p13-expr
                          | p12-expr

p12-expr                ::= p11-expr OP_LT p12-expr
                          | p11-expr

p11-expr                ::= p10-expr OP_LE p11-expr
                          | p10-expr

p10-expr                ::= p9-expr OP_GT p10-expr
                          | p9-expr

p9-expr                 ::= p8-expr OP_GE p9-expr
                          | p8-expr

p8-expr                ::= p7-expr OP_PLUS p8-expr
                          | p7-expr

p7-expr                ::= p6-expr OP_MINUS p7-expr
                         | p6-expr

p6-epxr                ::= p5-expr OP_MULT p6-expr
                         | p5-expr

p5-expr                ::= unary-expr OP_DIV p5-expr
                        | unary-expr

unary-expr             ::= OP_NOT unary-p0
                         | unary-p0
                         
unary-p0               ::= OP_MINUS unary-p1
                         | unary-p1

unary-p1               ::= unary-p2 OP_INC
                         | unary-p2

unary-p2               ::= unary-p3 OP_DEC
                         | unary-p3

unary-p3               ::= unit-val
                         | GROUP_OPEN expr GROUP_CLOSE

unit-val               ::= func-call
                         | flag
                         | literal

arg-list               ::= expr separator-list

separator-list         ::= SEPARATOR expr separator-list
                         | ''

literal                ::= BOOL_VAL
                         | NUM_VAL
                         | QUOTED_VAL
                         | UNQUOTED_VAL

func-call              ::= IDENTIFIER GROUP_OPEN arg-list GROUP_CLOSE

flag                   ::= IDENTIFIER


OP_DECSET       ::= '-='
OP_INCSET       ::= '+='
OP_LT           ::= '<'
OP_LE           ::= '<='
OP_GT           ::= '>'
OP_GE           ::= '>='
OP_IS_NOT       ::= '!='
OP_IS           ::= '=='
OP_MINUS        ::= '-'
OP_PLUS         ::= '+'
OP_MULT         ::= '*'
OP_DIV          ::= '/'
OP_NOT          ::= '!'
OP_AND          ::= '&&'
OP_OR           ::= '||'
SEPARATOR       ::= ','
GROUP_OPEN      ::= '('
GROUP_CLOSE     ::= ')'
IDENTIFIER      ::= '$' ['A-Za-z0-9_']*

QUOTED_VAL      ::= '@' NONENDING_OR_ESCAPE_SEQ* '@'

NONENDING_OR_ESCAPE_SEQ  ::= NONENDING_CHAR
                           | ESCAPE_SEQ

NONENDING_CHAR ::= [^'@\']

CHAR            ::= [ (any character) ]

ESCAPE_SEQ      ::= '\' CHAR
BOOL_VAL        ::= ['Tt']['Rr']['Uu']['Ee']
                  | ['Ff']['Aa']['Ll']['Ss']['Ee']
                  | ['Oo']['Nn']
                  | ['Oo']['Ff']['Ff']
                  | ['Yy']['Ee']['Ss']
                  | ['Nn']['Oo']

UNQUOTED_VAL    ::= [^whitespace]+ ( .* [^whitespace] )?
