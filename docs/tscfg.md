Context Free Grammar of Operator-Based Tunascript Pre-Scan

Grammar is BNFish

S                       ::= <expression>

<expression>            ::= 
                          | <flag>
                          | <literal>
                          | <binary_op_group>
                          | <unary_op_group>
                          | <paren_group>

<paren_group>           ::= "(" <expression> ")"

<binary_op_group>       ::= <expression> <binary_op> <expression>

<unary_op_group>        ::= <unary_postfix_group>
                          | <unary_prefix_group>

<unary_postfix_group>   ::= <expression> <unary_postfix_op>

<unary_prefix_group>    ::= <unary_prefix_op> <expression>

<literal>               ::= <string>
                          | <quoted_string>
                          | <number>
                          | <bool>

<func_call>             ::= "$" <identifier_char>* "(" ( <expression> ( "," <expression> )* )? ")"

<flag>                  ::= "$" <identifier_char>*

<identifier_char>       ::= [ <alpha> <digit> "_" ]

<binary_op>             ::= "+"
                          | "-"
                          | "*"
                          | "/"

<unary_postfix_op>      ::= "++"
                          | "--"

<unary_prefix_op>       ::= "-"

<alpha>                 ::= [ "A" - "Z" "a" - "z" ]

<digit>                 ::= [ "0" - "9" ]

<bool>                  ::= [ "t" "T" ] [ "r" "R" ] [ "u" "U" ] [ "e" "E" ]
                          | [ "f" "F" ] [ "a" "A" ] [ "l" "L" ] [ "s" "S" ] [ "e" "E" ]

<string>                ::= E
                          | [ ]

<quoted_string>         ::= "|" <string> "|"