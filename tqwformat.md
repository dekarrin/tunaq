The TunaQuest Worlds Format v0.1
================================
This is a complete reference describing the data files that TunaQuest uses to
define games that it runs and techniques for building up a world.

World definitions are in a format of files called "TunaQuest Worlds Format", or
"TQW". TQW is a TOML-based text file format, with additional semantic
restrictions. A TQW file may be referred to by this library as a "Resource" file
as all of its world resources will be in that format.

All TQW files have some common keys at the top level, and from there the
contents vary based on the specific type of the file.

Case Sensitivity
----------------
In TQW files, the keys themselves are always case sensitive. The values are
mostly case-sensitive; however, `label` values, `aliases` values, and other
values that reference labels and aliases are explicitly case-insensitive. They
will be marked as such in their section.

The values for the common keys `format` and `type` are always case-insensitive.

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
contains keys that give information about the world itself.
* [[[room]]](#room-section) - Marks the start of a room definition.
* `[[npc]]` - Marks the start of an NPC definition. There is no requirement that
any NPCs be defined in a world; they are completely optional. See
[NPC Section](#npc-section) for a description of the keys in that section.
* `[[pronouns]]` - Marks the start of a custom pronoun definition that may be
referred to by an NPC for it to use them. Custom pronouns are not required and
basic ones ("HE/HIM", "SHE/HER", "THEY/THEM", "IT/ITS") are already available to
use out of the box. See [Pronouns Section](#pronouns-section) for a description
of the keys in that section.

For an example, see the [World Data File Example](#world-data-file-example) in
the appendix.

### World Section
- **Section Header:** `[world]`
- **Used In Section:** (top-level)

The world section defines information about the world itself. At least one world
data file in a world must include a `[world]` section with a `start` key.

The `[world]` section has the following keys:

* `start` - (Case-Insensitive) The label of the room that the player character
will begin the game in.

Example:

```toml
[world]
start = "YOUR_ROOM"
```

### Room Section
- **Section Header:** `[world]`
- **Used In Section:** (top-level)

A room section starts the definition of a room in the world. There must be at
least one room section across all world data files in a world.

A `[[room]]` section has the following keys:

* `label` - (Case-Insensitive) A unique identifier for the table. Must follow
the [Naming Rules] defined for TQW labels, and must be unique among all room
labels.
* `name` - The name of the room. This should be fairly short; it will be
used whenever the name of the room needs to be displayed to the player, such as
in EXITS listings.
* `description` - A long-form description of the room. This is shown when the
player runs LOOK on the room with no additional arguments.

Additionally, a `[[room]]` section can have the following sub-sections:

* `[[room.exit]]` - An exit from the room to another room. See
[Exit Section](#exit-section) for a description of the keys in that section.
* `[[room.item]]` - A definition for an item that is in that room. This is
subject to moving to its own top-level section along with the addition of a
`start` key, so it should not be relied on as always being a part of a room def.
See [Item Section](#item-section) for a description of the keys in that section.

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