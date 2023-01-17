Context Free Grammar of Operator-Based Tunascript Pre-Scan

Grammar is BNFish

S                       ::= <expression>

<expression>            ::= <mult-expr> ( ( '+' | '-' ) <mult-expr> ) *
<mult-expr>             ::= <primary> ( ( '*' | '/' ) <primary> ) *
<primary>               ::= '(' <expression> ')'
                          | QUOTED_STRING
                          | UNPARSED_TUNASCRIPT
                          | <primary> '+' '+'
                          | <primary> '-' '-'

<quoted_string>         ::= "|" [^ "|" ]* "|"

<unparsed_tunascript>   ::= ANY_CHARACTER_NOT_ABOVE