The TunaQuest Worlds Format v0.1
================================
This is a complete reference describing the data files that TunaQuest uses to
define games that it runs and techniques for building up a world.

File Organization
-----------------
The entire world may be defined in a single TQW resource file. To do this, the
file must be a `data` type file, and all definitions are placed in that file:

Example directory structure:
```
myworld/
 |- world.tqw
```

Here, `myworld` is a directory containing a single file, `world.tqw`, which is
a world data file containing all data for the world. To use it, from directory
`myworld` you would call into the TunaQuest interpreter and pass the name of the
file to it:

```shell
tqi -w world.tqw
```

### Breaking Up World Files
Putting every definition in the world in one single file can quickly grow
cumbersome. Instead, the world data file can be split at the top-level header
level and every individual section can be placed in its own file.

To refer to multiple files in a world definition, create a manifest type file
and list all files you wish to include in it. You can even list another manifest
file if you want.

Example directory structure:
```
myworld/
 |- manifest.tqw
 |- world.tqw
 |- rooms
 |   |- bobs-house.tqw
 |   |- janes-house.tqw
 |   |- overworld.tqw
 |- npcs.tqw
```

As opposed to the prior example, `myworld` is now a directory containing
several files and a sub-folder.

To load all the files into a single world, the file `manifest.tqw` lists all of
them.

manifest.tqw:
```toml
format = "tuna"
type = "manifest"

files = [
    "world.tqw",
    "rooms/bobs-house.tqw",
    "rooms/janes-house.tqw",
    "rooms/overworld.tqw",
    "npcs.tqw"
]

Then, from directory `myworld`, the complete world can be loaded by passing the
manifest file to the TunaQuest interpreter:

```shell
tqi -w manifest.tqw
```

Case Sensitivity
----------------
Most game objects have case-sensitive values. There are certain values that will
always be case-insensitive: object labels, object aliases, and other values that
reference labels and/or aliases.

Generally, if it is used for programmatic reference and for the player to be
able to enter it in as part of input, it will be case-insensitive. If it's
intended solely to show output to the user (such as an object description, which
is output wen the LOOK command is called on it), it will be case-sensitive.

All data that is case-insensitive will be marked as such. The values for the
common TQW keys `format` and `type` are expliclty case-insensitive.

Naming Rules
------------
Most values in TQW format are free-form strings and can be anything the world
creator desires. However, object labels and object aliases have special rules
that are enforced in order to help prevent parser ambiguities.

Both labels and aliases are case-insensitive. Regardless of what is typed in in
the data files, they will be assumed to be upper case, always.

They may be made up of one or more of the following characters: the latin
alphabet characters `A` through `Z`, the digits `0` through `9`, or any of the
following special characters: `_!?#%^&*().,<>/+=[|{}:;-`. This is most
characters producable with a US QWERTY keyboard equipped with shift.

Additionally, an alias may have spaces in it as long as it does not start or end
with one. Labels are not allowed to have spaces in them; consider using the
underscore character `_` to separate words in a label if this is needed.

Notably, the characters `@` and `$` cannot be present in any label or alias.
These characters are reserved for command argument marking and variable
references respectively.

Finally, due to the nature of the TunaQuest command parser, there are a few
reserved words that cannot be present in any alias or label. They are the
following:

* TO
* THROUGH
* INTO
* FROM
* ON
* IN
* WITH
* AT

These may be a substring of labels and aliases as long as they don't contain
that whole word; e.g. an alias called "INNER DEPTHS" is valid, but an alias
called "IN DEPTHS" is not.

File Format
-----------
World definitions are in a format of files called "TunaQuest Worlds Format", or
"TQW". TQW is a TOML-based text file format, with additional semantic
restrictions. A TQW file may be referred to by this library as a "Resource" file
as all of its world resources will be in that format.

All TQW files have some common keys at the top level, and from there the
contents vary based on the specific type of the file.

Common Keys
-----------
The TQW format requires that there be two keys defined in every TQW file, in the
top-level TOML table (so, before the first "[]" or "[[]]" header). By convention
these are the first two keys in the first two lines of the file, but this is not
strictly required.

The common keys that every TQW file has are:
* `format` - (case-insensitive) This must always be set to "tuna"
* `type` - (case-insensitive) This must be set to one of "data" or "manifest",
based on what type of TQW file it is

Manifest File
-------------
This file contains a listing of files to be included in the TunaQuest world
definition. If the resource file given to `tqi` with `-w` is a manifest file, it
will load all files listed in the manifest, including other manifest files if
they are listed. This operation is performed recursively and ciruclar dependency
detection allows one manifest file to end up referencing itself without any
parsing errors.

A manifest file is made up of the following top-level keys:

* `format` - (case-insensitive) This must always be set to `"tuna"`
* `type` - (case-insensitive) For manifest files, this must be set to
`"manifest"`
* `files` - A list of files to include, relative to the manifest file. This
can be other manifest files.

Example:

manifest.tqw
```toml
format = "tuna"
type = "manifest"

files = [
    "metadata.tqw",
    "rooms/house.tqw",
    "rooms/city.tqw",
    "rooms/spaceship.tqw",
    "npcs/john.tqw",
    "npcs/rose.tqw",
    "npcs/dave.tqw",
    "npcs/jade.tqw",
]
```

World Data File
---------------
This file type contains definitions for the world. They may include NPC
definitions, room definitions, item defnitions, and pronoun set definitions. If
there are multiple world data files in a world, then together they must give a
complete definition of the world. If there is only a single world data file in a
world, the complete world definition must be within that one file.

World data files are able to split between entity definitions; for instance, it
is entirely possible to define an NPC in one file, but define the room it starts
in in another, but it is not possible to give the name of an NPC in one file and
a description of the NPC in another. See
(Breaking Up World Files)[#breaking-up-world-files] for more info.

World data files have the common top-level keys:

* `format` - (case-insensitive) This must be present in every world data file
and must always be set to `"tuna"`
* `type` - (case-insensitive) This must be present in every world data file and
must be set to `"data"`

Beyond that all data is given under header blocks. Keys included in a header
block must all be defined in lines under the header, in the same file, before
the next header. The following table and/or array-of-table header blocks may be
used at the top level of a world data file:

* [[world]](#world-section) - Marks the start of global world info. This
contains keys that give information about the world itself. This is a required
section that must be present somewhere in the loaded data files for a world.
* [[[room]]](#room-section) - Marks the start of a room definition.
* [[[npc]]](#npc-section) - Marks the start of an NPC definition. There is no
requirement that any NPCs be defined in a world; they are completely optional.
* [[[pronouns]]](#pronouns-section) - Marks the start of a custom pronoun
definition.

For an example of a complete standalone world data TQW file, see the
[World Data File Example](#world-data-file-example) in the appendix.

### World Section
- **Section Header:** `[world]`
- **Used In Section:** (top-level)

The world section defines information about the world itself. At least one world
data file in a world must include a `[world]` section with a `start` key.

Note that as opposed to other top-level sections, this one is named with just a
single set of brackets instead of two sets. This is a reminder that there should
be exactly one.

The `[world]` section has the following keys:

* `start` - (Case-Insensitive) The label of the room that the player character
will begin the game in.

Example:

```toml
format = "tuna"
type = "data"

[world]
start = "YOUR_ROOM"
```

### Room Section
- **Section Header:** `[[room]]`
- **Used In Section:** (top-level)

A room section starts the definition of a room in the world. There must be at
least one room defined across all world data files in a world.

A `[[room]]` section has the following keys:

* `label` - (Case-Insensitive) A unique identifier for the room that is used to
refer to it in other parts of the world definition. Must follow the
[Naming Rules](#naming-rules) defined for TQW labels, and must be unique among
all room labels.
* `name` - The name of the room. This should be fairly short; it will be
used whenever the name of the room needs to be displayed to the player, such as
in EXITS listings.
* `description` - A long-form description of the room. This is shown when the
player runs LOOK on the room with no additional arguments. By convention, it
would be a good idea to describe the exits here as well so that the player isn't
stuck continuously running the "EXITS" command.

Additionally, a `[[room]]` section can have the following sub-sections:

* [[[room.exit]]](#exit-section) - An exit from the room to another room. There
can be any number of exits defined on the room.
* [[[room.item]]](#item-section) - A definition for an item that is in that
room. This is subject to moving to its own top-level section along with the
addition of a `start` key to it to refer to the room, so it should not be relied
on as always being a sub-section of `[[room]]`. There can be any number of items
defined on the room.

Example:

```toml
format = "tuna"
type = "data"

[[room]]
label = "BACKYARD"
name = "Your house's backyard"
description = '''
This is the backyard of your house. It's full of greenery and gorgeous plants
that your aunt likes to maintain. That's not something that you are very
interested in helping with, but you sure enjoy the results.

You could go back inside from here, via the house's back door. Once you're done
enjoying the view, of course.
'''
```

### Exit Section
- **Section Header:** `[[room.exit]]`
- **Used In Section:** `[[room]]`

An exit section defines an egress from a room. It is required to be able to
leave a room. Exits are one-way only; to connect two rooms to each other so
that they may be freely traveled between, there must be an exit on the first
room to the second and an exit on the second room to the first.

There may be any number of exits defined on a room. A room with no exits cannot
be escaped using the GO command (although there may be other actions the user
can take to escape).

A `[[room.exit]]` section has the following keys:

* `aliases` - (Case-Insensitive) A list of phrases that the player may use to
refer to this exit in commands. Each item in the list is a string that must
follow the [Naming Rules](#naming-rules) defined for TQW aliases, and must be
unique among all exit aliases in the same room. Additionally, an exit alias
cannot conflict with any NPC or item label, as they may end up in the same room
and cause ambiguity in parsing.
* `dest` - (Case-Insensitive) The label of the room that this exit goes to.
* `description` - A description of the exit, shown when the player LOOKs at the
exit.
* `message` - The message shown to the player when they decide to take the exit
out of the room with the GO command.

Example:

```toml
format = "tuna"
type = "data"

[[room]]
label = "BACKYARD"
name = "Your house's backyard"
description = '''
Your backyard is full of greenery and pretty plants.

You could go back inside from here, via the house's back door.
'''

[[room.exit]]
aliases = ["BACK DOOR", "DOOR"]
dest = "FRONT_FOYER"
description = '''
The backdoor of your house. It has a screen in it that can be closed during the
colder months, but right now it's open to let in a nice breeze.
'''
message = "You swing open the door and walk into the house."
```

### Item Section
- **Section Header:** `[[room.item]]`
- **Used In Section:** `[[room]]`

An item section defines an item that can be picked up and placed in the player's
inventory. It will be located in the room that it is defined in (though note
that future versions will almost certainly decouple this, making `[[item]]` a
top-level section).

There may be any number of items defined in a room.

A `[[room.item]]` section has the following keys:

* `label` - (Case-Insensitive) A unique identifier for the item that is used to
refer to it in other parts of the world definition. Must follow the
[Naming Rules](#naming-rules) defined for TQW labels, and must be unique among
all item labels.
* `aliases` - (Case-Insensitive) A list of phrases that the player may use to
refer to this item in commands. Each element of the list is a string that must
follow the [Naming Rules](#naming-rules) defined for TQW aliases.
* `name` - The name of the item used when displaying the name of the item to the
player.
* `description` - A more long-form description of the item, used when the player
uses LOOK on the item.

Example:

```toml
toml
format = "tuna"
type = "data"

[[room]]
label = "BACKYARD"
name = "Your house's backyard"
description = '''
Your backyard is full of greenery and pretty plants.

You could go back inside from here, via the house's back door.
'''

[[room.item]]
label = "POGO_HAMMER"
aliases = ["HAMMER", "POGOHAMMER", "POGO", "POGO HAMMER"]
name = "A pogo hammer"
description = "A hammer combined with a pogo-stick. What could go wrong?"
```
### NPC Section
- **Section Header:** `[[npc]]`
- **Used In Section:** (top-level)

An NPC section starts the definition of an NPC in the world. Once defined, the
NPC will act somewhat independently and go to the next step on its movement
route every time the player changes rooms.

There may be any number of NPCs defined in the world.

NPCs need to have a pronoun set defined for use with them. This can be done by
refering to an existing set's label with the `pronouns` key of an NPC section,
or by defining a `[npc.custom_pronoun_set]` sub-section in it to directly give
the pronouns. One of the two options must be done with every NPC section, and
the two options are mutually exclusive.

An `[[npc]]` section has the following keys:

* `label` - (Case-Insensitive) A unique identifier for the NPC that is used to
refer to it in other parts of the world definition. Must follow the
[Naming Rules](#naming-rules) defined for TQW labels, and must be unique among
all NPC labels.
* `aliases` - (Case-Insensitive) A list of phrases that the player may use to
refer to this NPC in commands. Each element of the list is a string that must
follow the [Naming Rules](#naming-rules) defined for TQW aliases.
* `name` - The canonical name of the NPC as shown to the player. This will be
used in dialog headings for their lines and other places their name is shown.
Note that unless this value is manually added to `aliases` as well, the player
cannot refer to NPCs by their `name`.
* `description` - A more long-form description of the NPC, shown when the player
uses the LOOK command on the NPC.
* `start` - (Case-Insensitive) The label of the room that this NPC starts the
game in.
* `pronouns` - (Case-Insensitive) (Optional) The label of the pronoun set to use
with this NPC. This can either be one of the built-in pronoun set labels,
(`"he/him"`, `"she/her"`, `"they/them"`, and `"it/its"`), or it can be the label
of a top-level `[[pronouns]]` section defined elsewhere in the world. If this
key is not set, the NPC must have an `[npc.custom_pronoun_set]` sub-section
defined in it.

Additionally, an `[[npc]]` section can have the following sub-sections:

* [[npc.custom_pronoun_set]](#custom-pronoun-set-section) - (Optional) - A set
of pronouns to use for the NPC. If this sub-section does not exist, the NPC
section must have a `pronouns` key that refers to an existing set of pronouns.
There can be at most one single `[npc.custom_pronoun_set]` sub-section in an
`[[npc]]` section.
* [[npc.movement]](#movement-section) - (Optional) The type of movement that the
NPC does in the world. If not present, the NPC defaults to "STATIC" movement
type and does not move. There can be at most one single `[npc.movement]`
sub-section in an `[[npc]]` section.
* [[[npc.line]]](#line-section) - (Optional) A line of dialog (or step) in the
NPC's dialog tree. There may be any number of `[[npc.line]]` sub-sections in an
`[[npc]]` section.

Appendix
--------

### World Data File Example

This is an example of a complete, runnable world with four rooms, one item, and
one NPC.

```toml
format = "tuna"
type = "data"

# tabbing in the sub-sections is not required
# but it can help make it clear a sub-section is a child of its parent

[world]
start = "KITCHEN"


[[npc]]
label = "IMP"
aliases = ["IMP", "PATROLING IMP", "PATROLLER"]
name = "Patroller, the patroling imp"
pronouns = "they/them"
start = "BACKYARD"
description = '''
A weird imp covered in strange ink. It's wearing a jester's hat and has a surprisingly cat-like face.
'''

  [[npc.line]]
  content = "Grrrr :3"


[[room]]
label = "KITCHEN"
name = "The kitchen"
description = '''
You are standing in a big magical kitchen where food is made from. The remnant scents of lovely baked goods waft through
the air and make you think of simpler times.

There's a long hallway leading north.
'''

  [[room.exit]]
  aliases = ["NORTH", "HALLWAY", "LONG HALLWAY", "HALL", "LONG HALL"]
  dest = "HALLWAY"
  description = "A long hallway connecting the kitchen to the bathroom and the bedroom"
  message = "You step into the hallway"


[[room]]
label = "HALLWAY"
name = "a hallway"
desc = '''
You're in a hallway connecting your bedroom to the kitchen. Never one to let things stay bare, you've decorated the
walls with dozens of posters and photos of things relevant to your hobbies. You can barely see them, though, because
recently you had a light go out and while the kitchen light makes it possible to see in here, it's hard to make out the
details of the pictures.

The kitchen is south of here, and at the north end of the hallway is your bedroom door. There's also another door on the
east wall.
'''

  [[room.exit]]
  aliases = ["SOUTH", "KITCHEN"]
  dest = "KITCHEN"
  description = "It's the south end of the hall. Bright light from the kitchen streams in."
  message = "You walk into the kitchen, letting your eyes adjust to the light."

  [[room.exit]]
  aliases = ["NORTH", "BEDROOM", "BEDROOM DOOR", "NORTH DOOR"]
  dest = "BEDROOM"
  description = "A plain wooden door leading to your room."
  message = "You open the door and step into your room."

  [[room.exit]]
  aliases = ["EAST", "EAST DOOR", "BATHROOM", "BATHROOM DOOR"]
  dest = "BATHROOM"
  description = "A plain wooden door leading to the bathroom."
  message = "You open the door to reveal the bathroom and walk in."


[[room]]
label = "BEDROOM"
name = "your bedroom"
desc = '''
This is your bedroom! You know and love this room as it's where you've done most of your growing up. You've made so many
memories here.

You can leave the room via the door south to get back to the hallway, or go into the bathroom via a door to the east.
'''

  [[room.exit]]
  aliases = ["SOUTH", "SOUTH DOOR", "DOOR", "HALLWAY", "BEDROOM DOOR", "HALL"]
  dest = "HALLWAY"
  description = "A plain wooden door leading into the hall."
  message = "You walk into the hall."

  [[room.exit]]
  aliases = ["EAST", "EAST DOOR", "BATHROOM"]
  dest = "BATHROOM"
  description = '''
  A plain door with a deadbolt lock on it you added to keep people from barging in via the bathroom. The lock has long
  since broken.
  '''
  message = '''
  You try and throw open a deadbolt on the door out of habit before remembering it doesn't work. You step
  into the bathroom.
  '''

  [[room.item]]
  label = "POGO_HAMMER"
  aliases = ["HAMMER", "POGOHAMMER", "POGO", "POGO_HAMMER"]
  name = "a pogo hammer"
  description = "A hammer combined with a pogo-stick. What could go wrong?"


[[room]]
label = "BATHROOM"
name = "your ensuite bathroom"
desc = '''
You are in the facilities of your house. The bathroom. The throne. The comode. Keeping it spotless has been a matter of
pride for you throughout the years.

This restroom has an additional door besides the one you came through, to let you come in from either the hall or your
bedroom directly. Fancy. You can go southwest through the door leading to the hallway, or through the northwest door to
your room.
'''

  [[room.exit]]
  aliases = ["SOUTHWEST", "HALL", "HALLWAY", "SOUTHWEST DOOR", "HALLWAY DOOR", "HALL DOOR"]
  dest = "HALLWAY"
  description = "A plain-looking door, scrubbed clean of all dirt."
  message = "You go through the door into the hall."

  [[room.exit]]
  aliases = ["NORTHWEST", "BEDROOM", "BEDROOM DOOR", "NORTHWEST DOOR"]
  dest = "BEDROOM"
  description = "A door with a very modern looking handle, leading to your bedroom."
  message = "You open the door and walk into your bedroom, shutting it behind you."

```