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
* Specific interpretation can be forced with the $TYPE(x, y) function, which always
returns the x-type of its arguments (or string if x is not one of "str", "bool",
or "num")

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

Parenthesis may be used for grouping.

Finally, there are variable references, which will be replaced by their value
during expansions.

Note that the naming rules for variables are considerably more stringent than
the typical labeling rules. `[A-Z0-9_]+`.

Built-in Functions
------------------
The following built-in functions are in tunascript:

### `$ADD(x, y)`
Parameters:
* `x`, `y` - what to add together. If `x` is a number, `y` will be forced to
a number and added to its value. If `x` is a string, `y` will be forced to a
string and concatenated to it. If `x` is a bool, it will be forced to a number.
If `x` is untyped, it will be forced to a number.

Returns 