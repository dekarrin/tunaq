TunaQuest
=========

An engine for building text adventures and choose-your-own adventures. Set sail
for adventure!

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