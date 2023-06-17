/*
Tqi starts an interactive TunaQuest engine session.

It reads in a world file and starts the game in the designated starting
position, or in the previously saved position if loading a saved game. The
interpreter will then start printing what is happening in the game to stdout and
will read user input from stdin until the game is over or the "QUIT" command is
input.

Usage:

	tqi [flags]

The flags are:

	-v, --version
		Give the current version of TunaQuest and then exit.

	-w, --world FILE
		Use the provided TQW resource file for the world. Defaults to the file
		"world.tqw" in the current working directory.

	-d, --direct
	    Force reading directly from the console as opposed to using GNU readline
		based routines for reading command input even if launched in a tty with
		stdin and stdout.

	-c, --command COMMANDS
		Immediately run the given command(s) at start. Can be multiple commands
		separated by the ";" character.

Once a session has started, the user input will be parsed for TunaQuest
commands. For an explanation of the commands, type "HELP" once in a session. To
exit the interpreter, type "QUIT".
*/
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/dekarrin/tunaq"
	"github.com/dekarrin/tunaq/internal/version"
	"github.com/spf13/pflag"
)

const (

	// ExitSuccess indicates a successful program execution.
	ExitSuccess = iota

	// ExitGameError indicates an unsuccessful program execution due to a
	// problem during the game.
	ExitGameError

	// ExitInitError indicates an unsuccessful program execution due to an issue
	// initializing the engine.
	ExitInitError
)

var (
	returnCode   int     = ExitSuccess
	flagVersion  *bool   = pflag.BoolP("version", "v", false, "Gives the version info")
	worldFile    *string = pflag.StringP("world", "w", "world.tqw", "The TQW world data or manifest file that contains the definition of the world")
	forceDirect  *bool   = pflag.BoolP("direct", "d", false, "Force reading directly from stdin instead of going through GNU readline where possible")
	startCommand *string = pflag.StringP("command", "c", "", "Execute the given player commands immediately at start and leave the interpreter open")
)

func main() {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			// we are panicking, make sure we dont lose the panic just because
			// we checked
			panic(fmt.Sprintf("unrecoverable panic occured: %v", panicErr))
		} else {
			os.Exit(returnCode)
		}
	}()

	pflag.Parse()

	if *flagVersion {
		fmt.Printf("%s\n", version.Current)
		return
	}

	var startCommands []string
	if *startCommand != "" {
		startCommands = strings.Split(*startCommand, ";")
	}

	gameEng, initErr := tunaq.New(os.Stdin, os.Stdout, *worldFile, *forceDirect)
	if initErr != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", initErr.Error())
		returnCode = ExitInitError
		return
	}
	defer gameEng.Close()

	err := gameEng.RunUntilQuit(startCommands)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err.Error())
		returnCode = ExitGameError
		return
	}
}
