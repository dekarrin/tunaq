Context Free Grammar of Tunascript

Grammar is BNFish

e is our epsilon.

S                      ::= expr

expr                   ::= group-expr
                         | binary-expr
                         | single-value-expr

group-expr             ::= GROUP_OPEN expr GROUP_OPEN

single-value-expr      ::= prefix-op expr
                         | expr postfix-op
                         | literal
                         | func-call
                         | flag

arg-list               ::= expr separator-list

separator-list         ::= SEPARATOR expr separator-list
                         | e

literal                ::= BOOL_VAL
                         | NUM_VAL
                         | QUOTED_VAL
                         | UNQUOTED_VAL

func-call              ::= IDENTIFIER GROUP_OPEN arg-list GROUP_CLOSE

flag                   ::= IDENTIFIER

prefix-op               ::= OP_MINUS
                         |  OP_NOT

postfix-op             ::= OP_INC
                         | OP_DEC

binary-expr             ::= eq-expr
                          | arith-comp-expr
                          | product-expr
                          | sum-expr
                          | incdec-expr
                          | logical-expr
                        
eq-expr                 ::= expr OP_IS expr
                          | expr OP_IS_NOT expr

arith-comp-expr         ::= expr OP_LT expr
                          | expr OP_LE expr
                          | expr OP_GT expr
                          | expr OP_GE expr

product-expr            ::= expr OP_MULT expr
                          | expr OP_DIV expr

sum-expr                ::= expr OP_PLUS expr
                          | expr OP_MINUS expr

incdec-expr             ::= expr OP_INCSET expr
                          | expr OP_DECSET expr

logical-expr            ::= expr OP_AND expr
                          | expr OP_OR expr


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
