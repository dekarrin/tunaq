/*
Tqserver starts a TunaQuest server and begins listening for new connections.

Usage:

	tqserver [flags]
	tqserver [flags] [[BIND_ADDRESS]:PORT]

Once started, the TunaQuest server will listen for HTTP requests and respond to
them using REST protocol. By default, it will listen on localhost:8080. This can
be changed by passing in either a full address with port, such as
"192.168.0.2:6001", or just the IP address preceeded by a colon, such as
":6001".

The flags are:

	--version/-v
		Give the current version of the TunaQuest server and then exit.
*/
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/dekarrin/tunaq/internal/version"
	"github.com/dekarrin/tunaq/server"
	"github.com/spf13/pflag"
)

var (
	flagVersion = pflag.BoolP("version", "v", false, "Give the current version of TunaQuest server and then exit")
)

func main() {
	pflag.Parse()

	if *flagVersion {
		fmt.Printf("%s (TunaQuest v%s)\n", version.ServerCurrent, version.Current)
		return
	}

	// get address info
	args := pflag.Args()

	if len(args) > 1 {
		fmt.Fprintf(os.Stderr, "Too many arguments\nDo -h for help\n")
		os.Exit(1)
	}

	port := 0
	addr := ""

	if len(args) >= 1 {
		listenAddr := args[0]

		bindParts := strings.SplitN(listenAddr, ":", 2)
		if len(bindParts) != 2 {
			fmt.Fprintf(os.Stderr, "Listen address is not in ADDRESS:PORT or :PORT format.\nDo -h for help\n")
			os.Exit(1)
		}

		var err error

		addr = bindParts[0]
		port, err = strconv.Atoi(bindParts[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%q is not a valid port number.\nDo -h for help\n", bindParts[1])
			os.Exit(1)
		}
	}

	tqs := server.New(addr, port)

	tqs.ServeForever()
}
