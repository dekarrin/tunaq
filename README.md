TunaQuest
=========

An engine for building text adventures and choose-your-own adventures. Set sail
for adventure!

TunaQuest allows text adventure creators to build worlds for their players using
a simple TOML-based world-file format, along with the help of a restricted
scripting language called 'tunascript'.

This is an early release of TunaQuest. You can find the latest releases at
the [TunaQuest GitHub Releases Page](https://github.com/dekarrin/tunaq/releases).
Please note that some features will be completely broken, some will be partially
implemented, and some will not work at all.

## Usage

## Showcase
todo: show some tunaquest logs

## File Format
a brief overview of the file format and tunascript

## Sample
link to source code to see sample directly, and link to live server to play,
(one day, glubglub!!!)


Possible Issues For Devs
------------------------
It's an in-dev engine and is very incomplete at the moment. Note the following
points if deving:
* Currently, the `tqi` bin will fail running if built for windows and executed
in a mintty terminal (such as `git-bash`). To get around this, the `build.sh`
script will detect if building for windows and if so, will produce a `tqi.sh`
along with the `tqi.exe` binary. This `tqi.sh` file will launch `tqi` using a
new Windows Command Prompt Shell if called from a mintty console, and should be
the preferred way of launching `tqi` in non-native windows shells (for native
shells such as Windows Command Prompt or Powershell it works fine).