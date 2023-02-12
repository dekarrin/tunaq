Context Free Grammar of Tunascript

Grammar is BNFish

'' (the empty string) is our epsilon.

# Productions

// NOTE: this is currently left-recursive i think; see any left-assoc bin operator
// and youll see a possible looping derivation rule.
//
// left->right recursion rewrite for recursive-descent parsing algo is described
// at https://www.youtube.com/watch?v=D6uMYa41Ank @ 49:30ish
//
// episilon elimination covered well in https://www.youtube.com/watch?v=j9cNTlGkyZM
// doing ^ plus single production elimination effectively makes a CFG without any
// cycles (for A | Non-terminals: there exists no sequence of rules s.t. A -*> A is possible.)


TS-EXPR -> BINARY-EXPR
{ value: BINARY-EXPR.value }

BINARY-EXPR -> BINARY-SET-EXPR
{ value: BINARY-SET-EXPR.value }






expr                     ::= binary-expr

binary-expr              ::= binary-set-expr

// afaik explicitly including the comma within the op precedence as opposed to
// within arg list will make it so wherever an expression is expected, comma
// list is allowed. This is simply untrue semantically, so further analysis
// may be needed to ensure we dont do something silly like "2 * 4, 5" which is
// technically permitted by the grammar as-written.
//
// Couldn't we just remove 8inary-separ9or from the hierarchy and add something
// that dips into it, 8ut with separ8or at lowest precedence *specifically* for
// arg lists? C only keeps it as such 8ecause commas are used frequently there;
// for us, we only want it in exactly one place: arg lists!
//
// TODO: yeah we did the above, ensure still works then delete this and the
// rule bc its all in arg-list now.
//
//binary-separator-expr    ::= binary-set-expr SEPARATOR binary-separator-expr
//                           | binary-set-expr


// TODO: we appear to have confused right/left associativity here;
// see p48 of purple dragon book (71 of pdf)

// right associativity
binary-set-expr          ::= binary-set-expr OP_SET binary-incset-expr
                           | binary-incset-expr

// right associativity
binary-incset-expr       ::= binary-incset-expr OP_INCSET binary-decset-expr
                           | binary-decset-expr

// right associativity
binary-decset-expr       ::= binary-decset-expr OP_DECSET binary-or-expr
                           | binary-or-expr

// and back to left
binary-or-expr           ::= binary-and-expr OP_OR binary-or-expr
                           |  binary-and-expr

binary-and-expr          ::= binary-eq-expr OP_AND binary-and-expr
                           |  binary-eq-expr

binary-eq-expr           ::= binary-ne-expr OP_IS binary-eq-expr
                           | binary-ne-expr

binary-ne-expr           ::= binary-lt-expr OP_IS_NOT binary-ne-expr
                           | binary-lt-expr

binary-lt-expr           ::= binary-le-expr OP_LT binary-lt-expr
                           | binary-le-expr

binary-le-expr           ::= binary-gt-expr OP_LE binary-le-expr
                           | binary-gt-expr

binary-gt-expr           ::= binary-ge-expr OP_GT binary-gt-expr
                           | binary-ge-expr

binary-ge-expr           ::= binary-add-expr OP_GE binary-ge-expr
                           | binary-add-expr

binary-add-expr          ::= binary-subtract-expr OP_PLUS binary-add-expr
                           | binary-subtract-expr

binary-subtract-expr     ::= binary-mult-expr OP_MINUS binary-subtract-expr
                           | binary-mult-expr

binary-mult-epxr         ::= binary-div-expr OP_MULT binary-mult-expr
                           | binary-div-expr

binary-div-expr          ::= unary-expr OP_DIV binary-div-expr
                           | unary-expr

unary-expr               ::= unary-not-expr

unary-not-expr           ::= OP_NOT unary-negate-expr
                           | unary-negate-expr
                         
unary-negate-expr        ::= OP_MINUS unary-inc-expr
                           | unary-inc-expr

unary-inc-expr           ::= unary-dec-expr OP_INC
                           | unary-dec-expr

unary-dec-expr           ::= expr-group OP_DEC
                           | expr-group

expr-group               ::= GROUP_OPEN expr GROUP_CLOSE
                           | identified-obj
                           | literal

identified-obj           ::= IDENTIFIER GROUP_OPEN arg-list GROUP_CLOSE
                           | IDENTIFIER
                        
arg-list                 ::= expr SEPARATOR arg-list
                           | expr
                           | ''

literal                  ::= BOOL_VAL
                           | NUM_VAL
                           | QUOTED_VAL
                           | UNQUOTED_VAL


# Terminals
Weird notation bc technically lexer operates at slightly higher level than the
items described here.

OP_DECSET                  ::= '-='

OP_INCSET                  ::= '+='

OP_LT                      ::= '<'

OP_LE                      ::= '<='

OP_GT                      ::= '>'

OP_GE                      ::= '>='

OP_IS_NOT                  ::= '!='

OP_IS                      ::= '=='

OP_MINUS                   ::= '-'

OP_PLUS                    ::= '+'

OP_MULT                    ::= '*'

OP_DIV                     ::= '/'

OP_NOT                     ::= '!'

OP_AND                     ::= '&&'

OP_OR                      ::= '||'

SEPARATOR                  ::= ','

GROUP_OPEN                 ::= '('

GROUP_CLOSE                ::= ')'

IDENTIFIER                 ::= '$' ['A-Za-z0-9_']*

QUOTED_VAL                 ::= '@' NONENDING_OR_ESCAPE_SEQ* '@'

NONENDING_OR_ESCAPE_SEQ    ::= NONENDING_CHAR
                             | ESCAPE_SEQ

NONENDING_CHAR             ::= [^'@\']

CHAR                       ::= [ (any character) ]

ESCAPE_SEQ                 ::= '\' CHAR

BOOL_VAL                   ::= ['Tt']['Rr']['Uu']['Ee']
                             | ['Ff']['Aa']['Ll']['Ss']['Ee']
                             | ['Oo']['Nn']
                             | ['Oo']['Ff']['Ff']
                             | ['Yy']['Ee']['Ss']
                             | ['Nn']['Oo']

UNQUOTED_VAL               ::= [^whitespace]+ ( .* [^whitespace] )?
