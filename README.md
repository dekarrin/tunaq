TunaQuest
=========

An engine for building text adventures and choose-your-own adventures. Set sail
for adventure! Glub glub glub.

TunaQuest allows text adventure creators to build worlds for their players using
a simple TOML-based world-file format, along with the help of a restricted
scripting language called 'tunascript'.

Some of its features:
* Full dungeon (world) definitions
* NPCs with dialog trees, who can move about
* Trigger scripted actions based on player actions (WIP)

This is an early pre-release of TunaQuest. You can find the latest releases at
the [TunaQuest GitHub Releases Page](https://github.com/dekarrin/tunaq/releases).
Please note that some features will be completely broken, some will be partially
implemented, and some will not work at all.

## Installation

### Releases
You can install it by grabbing one of the archives in the Releases section of
the TunaQuest repo; if you're reading this file in a decompressed archive,
congrats! You've downloaded it. From there, just take the provided file `tqi`
(or `tqi.exe` on Windows) and place it somewhere on your path.

### Go Install
As a Golang command, if you have a Go development environment set up, TunaQuest
can be installed with `go install github.com/dekarrin/tunaq/cmd/tqi`. This will
give you the TunaQuest interpreter.

## Usage

The TunaScript Interpreter is the main executable used to run TunaQuest worlds.
To use it, point it at a file containing world data:

```shell
./tqi -w myworld.tqw
```

If you don't give a world, it will try to run the file `./world.tqw`.

From there, the adventure starts! If you're not familiar with text adventure
games or just want a referesher, you can type "HELP" at the TQI prompt.

## Creating Worlds
To create a world, you create TunaQuest Worlds (TQW) format resource files.
These are simple text files, and can be edited with any text editor, including
Notepad.exe if you wish.

TQW is a flexible format that has a few subtypes:

* World data files - These contain world data. You could define an entire game
in one, or split it across several world data files by listing them in a
manifest file.
* Manifest files - These are TQW files whose sole purpose is to list other TQW
files to load. If you point `tqi` to a manifest-type TQW file, it will also load
the data in every file listed in the manifest. If there's a manifest file listed
in the manifest file, then all of THOSE listed files will be loaded as well.
It's done recursively, glub!

TQW files end in `.tqw` by convention, but it shorely not a requirement! End
them with .txt if you want to; `tqi` won't care as long as the contents of it
are readable.

A complete description of these files is beyond the scope of this guide; check
out the file `docs/tqwformat.md` for more information, or take a look at the
sample world data included in world.tqw.

## Tunascript
Sometimes, you may want an action in the world to cause something else to
happen; for instance, you may wish to make it so that reaching a point in an
NPC's dialog tree makes it so that a game flag called "$NPC_FRIEND" is enabled
for later reading.

This is accomplished by attaching pieces of a scripting language called
"tunascript" to certain points in the game. This is done with the use of `if`
attributes and `script` attributes. The variables can then be used in template
text (which most world text descriptions are treated as) by either directly
giving a variable starting with `$` or using template flow-control statements to
check variables and game state and show text based on their values.

Right now, that feature is still being developed. You can test it in the
meantime by using the `DEBUG EXPAND` or `DEBUG EXEC` commands. `EXPAND` will run
text expansion of variables and the special tunaquest template directives
`$[[ if TUNASCRIPT ]]` and `$[[ endif ]]`, along with their friends
`$[[else if TUNASCRIPT]]` and `$[[ else ]]`. You replace `TUNASCRIPT` in those
with the actual tunascript that is used to evaluate whether to expand that part
of the template. No tunascript that causes changes in the game state can be
executed here, but functions that check the state are okay! `EXEC` on the other
hand will evaluate any tunascript expression, and there are no restrictions on
what can be called.

A complete description of tunascript.md is beyond the scope of this guide; check
out the file `docs/tunascript.md` for more information.

## Showcase
todo: show some tunaquest logs

## File Format
a brief overview of the file format and tunascript

## Sample
link to source code to see sample directly, and link to live server to play,
(one day, glubglub!!!)

## Dev Info

### Requirements

* To build the distributions you need a Go build environment.
* To build the distributions you must have zip, 7z, or 7za installed. You can
install one via any method you choose; below are some instructions for getting
them using popular methods generally available.

#### Installing Build Requirements On Mac

* For zip-file creation support needed to run `scripts/create-dists.sh`, do at
least one of the following:
    * Install zip with brew by running `brew install zip`.

#### Installing Build Requirements On Linux (Ubuntu)

These steps should work with Ubuntu and most Debian-derived distributions
(although some may have different names for the packages).

* For zip-file creation support needed to run `scripts/create-dists.sh`, do at
least one of the following:
    * Install 7z: `sudo apt install p7zip-full`.
    * Install zip: `sudo apt install zip`.
#### Installing Build Requirements On Linux (CentOS)

These steps should work with CentOS and most RedHat-derived distributions
(although some may have different names for the packages).

* For zip-file creation support needed to run `scripts/create-dists.sh`, do at
least one of the following:
  * Install 7za only: `sudo yum install p7zip`
  * Install 7z: `sudo yum install p7zip p7zip-plugins`
  * Install zip: `sudo yum install zip`

#### Installing Build Requirements On Windows

* For zip-file creation support needed to run `scripts/create-dists.sh`, do at
least one of the following:
  * Install 7za: Get it from the
  [7-Zip downloads page](https://7-zip.org/download.html).
    1. First, grab the 7zr EXE. At the time of this writing, it is listed with
    the description "7zr.exe (x86): 7-Zip console executable" and the latest
    stable release is available at [7zr.exe v23.00 (x86)](https://7-zip.org/a/7zr.exe).
    2. Place 7zr on your PATH or other executable location if you wish, but as
    it will only be assisting in getting an executable that handles everything
    7zr does and more, you can simply elect to leave it in your Downloads folder
    for the moment.
    3. Next, download the "7-Zip Extra" archive. As of the time of this writing,
    it is listed with a type of .7z and a description that mentions the
    "standalone console version", and the latest stable release is available at
    [7-Zip Extra v22.01](https://7-zip.org/a/7z2201-extra.7z). Place it in the
    same location as the 7zr EXE.
    4. Use 7zr to extract the extras with Windows Command Prompt or a shell of
    your choice:
        * `cd Downloads` (or whatever directory you downloaded 7-Zip Extra to)
        * `mkdir 7zip-extra`
        * `move 7z2201-extra.7z 7zip-extra`
        * `move 7zr.exe 7zip-extra`
        * `cd 7zip-extra`
        * `7zr.exe x 7z2201-extra.7z`
    5. 7za.exe should now be present in the folder `7zip-extra` in your
    Downloads folder. Place it somewhere on your PATH, and then you can delete
    everything in the `7zip-extra` folder. If you are on a 64-bit system, you
    can grab the one in the `x64` sub-folder instead, but it's not required.

### Possible Issues
It's an in-dev engine and is very incomplete at the moment. Note the following
points if deving:
* Currently, the `tqi` bin will fail running if built for windows and executed
in a mintty terminal (such as `git-bash`). To get around this, the `build.sh`
script will detect if building for windows and if so, will produce a `tqi.sh`
along with the `tqi.exe` binary. This `tqi.sh` file will launch `tqi` using a
new Windows Command Prompt Shell if called from a mintty console, and should be
the preferred way of launching `tqi` in non-native windows shells (for native
shells such as Windows Command Prompt or Powershell it works fine).