Tunascript Expression Language
==============================

The tunascript expression language is the language used to update and read flags
during particular points of gameplay to make the game more dynamic. It can be
used to set whether someone has talked to a person, to record the number of
times they've done something, or to record a string value for later use.

This is not a turing-complete language; it lacks control flow structures in the
general case, and is more intended to do variable reading and setting which is
used by other elements of the game to affect action using the `if` keyword in
their world data.

Lanugage Constructs
-------------------

There are functions, which start with "$" and end with "()". These will be
evaluated and their return value will substitute their place in the result.

There are values, which are either quoted or unquoted, it does not matter. All
values are interpreted to whatever type they can be; e.g. `"true"` is the same
as `true` and both will evaluate to the boolean-typed `true` value.
* If a value appears to be "true" or "false", it will be interpreted as a bool
* If a value appears to be an integer, it will be interpreted as a number.
* If a value does not match the above rules, it will be interpreted as a string.
* Specific interpretation can be forced with the $STR(x), $BOOL(x), or $NUM(x)
functions, which always returns that type of its arguments.

FOR FUTURE:
```
There are operations, which result in built-in funcs being called:
* `x + y` is the same as `$ADD(x, y)`. Works for both concatenation and
addition. Y will be cast to x's type.
* `x - y` is the same as `$SUB(x, y)`. Works for number types only.
* `x * y` is the same as `$MULT(x, y)`. Works for number types and string * num
types.
* `x / y` is the same as `$DIV(x, y)`. Works for number types only.
* `x || y` is the same as `$OR(x, y)`. Works for bool types only.
* `x && y` is the same as `$AND(x, y)`. Works for bool types only.
* `!x` is the same as `$NOT(x)`. Works for bool types only.
```

(for now those will only be implemented as functions to make parser easier)

Parenthesis may be used for grouping.

Finally, there are variable references, which will be replaced by their value
during expansions.

Note that the naming rules for variables are considerably more stringent than
the typical labeling rules. `[A-Z0-9_]+`.

Names of variables and functions are not case-sensitive. They must always start
with a dollar sign.

Built-in Functions
------------------
The following built-in functions are in tunascript:

* `$add(x str, y str) str`
* `$add(x num, y num) num`
* `$add(x any, y any) -> $add(num(x), y)`
* `$sub(x num, y num) num`
* `$mult(x num, y num) num`
* `$mult(x str, y num) str`
* `$div(x num, y num) num`
* `$or(x any, y any) bool`
* `$and(x any, y any) bool`
* `$not(x any) bool`
* `$enable(flag str) bool`
* `$disable(flag str) bool`
* `$toggle(flag str) bool`
* `$inc(flag str[, amt int]) int`
* `$dec(flag str[, amt int]) int`
* `$set(flag str, val T) T`
* `$flagEnabled(flag str) bool`
* `$flagDisabled(flag str) bool`
* `$flagIs(flag str, val str) bool`
* `$flagIsLessThan(flag str, val num) bool`
* `$flagIsGreaterThan(flag str, val num) bool`
* `$value(flag str) any`
* `$ininven(item str) bool`
* `$bool(val any) bool`
* `$num(val any) num`
* `$str(val any) str`


The following functions are used for side-effects only. They may not be used in
`if` clauses.
* `$crush/$kill/$destroy(label str)`
* `$move(label str, to str)`
* `$output(x any) empty-str`

### `$ADD(x (num | str), y -> type(x)) ) type(x)`
Add two numbers or two strings together.

Parameters:
* `x`, `y` - what to add together. If `x` is a number, `y` will be forced to
a number and added to its value. If `x` is a string, `y` will be forced to a
string and concatenated to it. If `x` is a bool, it will be forced to a number.
If `x` is untyped, it will be forced to a number.

Returns string-type if x is a string, and number-type if x is a number.

### `$SUB(x num, y num) num`
Subtract y from x. Both will be interpreted as a number.

Returns a num-typed value.

### `$MULT(x (num | str), y num) type(x)
Multiply two numbers together, or a string by a number (to get python-style
multiplied strings).

Returns string-type is x is string, or num type if x is num. num takes
precedence over str.

### `