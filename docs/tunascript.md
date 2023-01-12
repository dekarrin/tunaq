Tunascript Expression Language
==============================

UPDATE: there is no untyped; it is always string by default

this file is a major work in prog and should rly not be relied on at all.

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

(for now those will only be implemented as functions to make parser easier)

Parenthesis may be used for grouping.

Finally, there are variable references, which will be replaced by their value
during expansions.

Note that the naming rules for variables are considerably more stringent than
the typical labeling rules. `[A-Z0-9_]+`.

Names of variables and functions are not case-sensitive. They must always start
with a dollar sign.

pipe char may be used to delimit strings. As a result, the pipe char must be
escaped if referenced literally.

Pipe char in particular is used instead of quotes because all scripts at this
time MUST be enclosed within toml strings, which means double quotes would be
far, far worse, and it means that all escapes must be doubled.

Built-in Functions
------------------
The following built-in functions are in tunascript:

These are expression functions and have no side-effects. They may be used in any
context that a tunascript expression is required.

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
* `$flag_enabled(flag str) bool`
* `$flag_disabled(flag str) bool`
* `$flag_is(flag str, val str) bool`
* `$flag_less_than(flag str, val num) bool`
* `$flag_greater_than(flag str, val num) bool`
* `$in_inven(item str) bool`

The following functions have side-effects, and may not be used in `if` clauses
in text to be expanded:
* `$enable(flag str) bool`
* `$disable(flag str) bool`
* `$toggle(flag str) bool`
* `$inc(flag str[, amt int]) int`
* `$dec(flag str[, amt int]) int`
* `$set(flag str, val T) T`
* `$move(label str, to str)`
* `$output(x any) empty-str`

### Expression Functions

#### `$ADD(x (num | str), y -> type(x)) ) type(x)`
Add two numbers or two strings together.

Parameters:
* `x`, `y` - what to add together. If `x` is a number, `y` will be forced to
a number and added to its value. If `x` is a string, `y` will be forced to a
string and concatenated to it. If `x` is a bool, it will be forced to a number.
If `x` is untyped, it will be forced to a number.

Returns string-type if x is a string, and number-type if x is a number.

#### `$SUB(x num, y num) num`
Subtract y from x. Both will be interpreted as a number.

Returns a num-typed value.

#### `$MULT(x (num | str), y num) type(x)
Multiply two numbers together, or a string by a number (to get python-style
multiplied strings).

Returns string-type is x is string, or num type if x is num. num takes
precedence over str.

#### `$DIV(x num, y num) num`
Divides x by y. Does rounding when result would not be a whole number. Rounding
type is half-up, not truncation.

Returns a num-typed value.

#### `$OR(x bool, y bool) bool`
Returns x logically OR'd with y.

#### `$AND(x bool, y bool) bool`
Returns x logically AND'd with y.

#### `$NOT(x bool) bool`
Returns the logical negation of x.

#### `$FLAGENABLED(flag str) bool`
Checks whether flag is enabled. Note that if flag starts with a $, it will be
EXPANDED and that will be used as the flag name.

Returns whether the flag is enabled (set to anything other than false, 0, the
string "false", or an empty string).

#### `$FLAGDISABLED(flag str) bool`
Checks whether flag is enabled. Note that if flag starts with a $, it will be
EXPANDED and that will be used as the flag name.

Returns whether the flag is disabled (set to false, 0, the string "false", or an
empty string).

#### `$FLAGIS(flag str, val (str | num)) bool`
Checks whether the flag is the given value. val will be interpreted as num if it
can be, or str if not. Can be used for bool checking but ENABLED/DISABLED is
probably betta for that.

Returns whether the flag is equal to the given value, or coercable to the given
value.

#### `$FLAGLESSTHAN(flag str, val num) bool`
Checks whether the flag is less than the given value. val will be interpreted as
num always.

Returns whether the flag value is less than the given value, or once coerced is
less than.

#### `$FLAGGREATERTHAN(flag str, val num) bool`
Checks whether the flag is greater than the given value. val will be interpreted
as num always.

Returns whether the flag value is less than the given value, or once coerced is
greater than.

#### `$ININVEN(label str) bool`
Checks whether the item with the given label is currently in the player
inventory.

### Side-Effect Functions

#### `$ENABLE(flag str) bool`
Sets the value of flag to true. Returns the new value of the flag (which will be
true)

#### `$DISABLE(flag str) bool`
Sets the value of flag to false. Returns the new value of the flag (which will
be false)

#### `$TOGGLE(flag str) bool`
Sets the value of flag to whatever its opposite is. If flag is not already bool
typed, its current value is coerced to bool and that is negated.

Returns the new value of the flag.

#### `$INC(flag str[, amt=1 num]) num`
Increment the flag by amt. If flag is not already num type, its current value is
coerced to num and that is incremented.

Returns the new value of the flag.

#### `$DEC(flag str[, amt=1 num]) num`
Decrement the flag by amt. If flag is not already num type, its current value is
coerced to num and that is decremented.

Returns the new value of the flag.

#### `$SET(flag str, val (num | str | bool)) -> type(val)`
Sets the value of flag to val. If val can be interpreted as a num, it is set to
num value, otherwise it will be interpreted as a str.

Returns the new value of the flag.

#### `$MOVE(label str, roomLabel str) bool`
Moves the thing with label to the given roomLabel. A turn move is not taken. If
label is "@PLAYER", it is the player that is teleported.

Returns whether the thing is in a new place after the move.

#### `$OUTPUT(value str) bool`
Prints the given value to the screen. If it isnt string type, it is converted to
it.

Returns whether it successfully printed.

### Low-Priority: Operators

either this would make it so parser needs to consider associativity instead of
mandatory paren matching, complicate parser. maybe somefin like, an external
first-pass parser that converts these all to their func calls, then parse w the
tunascript internal parser

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

### Low-Priority Expressions
may be implemented in future:

#### `$BOOL(x any) bool`
Forces intepretation of x as a bool value.

If x is a bool:
    x is returned as-is
If x is a num:
    0 is interpreted as `false`
    anything other than 0 is interpreted as `true`
If x is a str:
    An empty string is interpreted as `false`
    A non-empty string is interpreted as `true`... except for any string that
        would strings.ToUpper(x) to "FALSE", that will be false.
If x is untyped:
    It is treated as an empty string

Returns x intepreted as a bool.

#### `$NUM(x any) num`
Forces interpretation of x as a num value.

If x is a bool:
    true is interpreted as 1
    false is interpreted as 0
If x is a num:
    x is returned as-is
If x is a str:
    x is parsed to a number. if not parsable or empty, it is 0.
If x is untyped:
    It is treated as an empty string.

Returns x interpreted as an int.

#### `$STR(x any) str`
Forces interpretation of x as a str value.

If x is a bool:
    true is interpreted as "true"
    false is intpereted as "false"
If x is a num:
    x is converted to a string with the number
If x is a str:
    x is returned as-is
If x is untyped:
    x is treated as an empty string.

Returns x interpreted as a str.

### Low-Priority Mutation Funcs

Note: destroy functions make it so that static checking of input is WAY harder
since we can no longer pre-analyze labels for existence; it might NOT exist
later.

#### `$DESTROY(label str) bool`
Remove the ITEM/NPC with label from the game completely.

Returns whether the game object was destroyed. Will be false if the label did
not exist.

Can also be referred to with `$CRUSH` or `$KILL`.

#### `$DESTROY_EXIT(roomlabel str, exitAlias str)`
Remove the exit with the given alias from the game completely.

Returns whether the game object was destroyed. Will be false if the exit did not
exist.