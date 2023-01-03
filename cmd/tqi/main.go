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

	-version
		Give the current version of TunaQuest and then exit.

	-w/-world [FILE]
		Use the provided JSON world file. Defaults to the file "world.json" in
		the current working directory.

Once a session has started, the user input will be parsed for TunaQuest
commands. For an explanation of the commands, type "HELP" once in a session. To
exit the interpreter, type "QUIT".
*/
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dekarrin/tunaq"
	"github.com/dekarrin/tunaq/internal/version"
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
	returnCode  int   = ExitSuccess
	flagVersion *bool = flag.Bool("version", false, "Gives the version info")
	worldFile   string
)

func init() {
	const (
		defaultWorldFile = "world.json"
		worldUsage       = "the JSON file that contains the definition of the world"
	)
	flag.StringVar(&worldFile, "world", defaultWorldFile, worldUsage)
	flag.StringVar(&worldFile, "w", defaultWorldFile, worldUsage+" (shorthand)")
}

func main() {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			// we are panicking, make sure we dont lose the panic just because
			// we checked
			panic("unrecoverable panic occured")
		} else {
			os.Exit(returnCode)
		}
	}()

	flag.Parse()

	if *flagVersion {
		fmt.Printf("%s\n", version.Current)
		return
	}

	gameEng, initErr := tunaq.New(os.Stdin, os.Stdout, worldFile)
	if initErr != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", initErr.Error())
		returnCode = ExitInitError
		return
	}
	defer gameEng.Close()

	err := gameEng.RunUntilQuit()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err.Error())
		returnCode = ExitGameError
		return
	}
}
