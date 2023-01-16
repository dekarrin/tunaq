Context Free Grammar of Operator-Based Tunascript

Grammar is BNFish, todo: formalize

S -> expression
expression -> func_call | identifier | literal |  binary_op_group | unary_op_group | paren_group
paren_group -> "(" expression ")"
binary_op_group -> expression binary_op expression
unary_op_group -> unary_postfix_group | unary_prefix_group
unary_postfix_group -> expression unary_postfix_op
unary_prefix_group -> unary_prefix_op expression
identifier -> identifier_char+

identifier_char -> ["A"-"B"]
binary_op -> "+" | "-" | "*" | "/"
unary_postfix_op -> "++" | "--"
unary_prefix_op -> "-"