/*
Tqserver starts a TunaQuest server and begins listening for new connections.

Usage:

	tqserver [flags]
	tqserver [flags] -l [[ADDRESS]:PORT]

Once started, the TunaQuest server will listen for HTTP requests and respond to
them using REST protocol. By default, it will listen on localhost:8080. This can
be changed with the --listen/-l flag (or config via environment var). The flag
argument must be either a full address with port, such as "192.168.0.2:6001", or
just the IP address preceeded by a colon, such as ":6001".

If a JWT token secret is not given, one will be automatically generated and
seeded with the current system time. As a consequence, in this mode of operation
all tokens are rendered invalid as soon as the server shuts down. This is
suitable for testing, but must be given via either CLI flags or environment
variable if running in production.

The flags are:

	-v, --version
		Give the current version of the TunaQuest server and then exit.

	-l, --listen LISTEN_ADDRESS
		Listen on the given address. Must be in BIND_ADDRESS:PORT or :PORT
		format. If not given, will default to the value of environment variable
		TUNAQUEST_LISTEN_ADDRESS, and if that is not given, will default to
		localhost:8080.

	-s, --secret TOKEN_SECRET
		Use the provided secret for signing JWT tokens. If there are less than
		32 bytes in the secret, it will be repeated until it is. The maximum
		size is 64 bytes. If not given, will default to the value of environment
		variable TUNAQUEST_TOKEN_SECRET. If no secret is specified or an emty
		secret is given, a random secret will be automatically generated. Note
		that any tokens issued with a random secret will become invalid as soon
		as the server shuts down.
*/
package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/dekarrin/tunaq/internal/version"
	"github.com/dekarrin/tunaq/server"
	"github.com/dekarrin/tunaq/server/dao"
	"github.com/spf13/pflag"
)

const (
	EnvListen = "TUNAQUEST_LISTEN_ADDRESS"
	EnvSecret = "TUNAQUEST_TOKEN_SECRET"
)

var (
	flagVersion = pflag.BoolP("version", "v", false, "Give the current version of TunaQuest server and then exit.")
	flagListen  = pflag.StringP("listen", "l", "", "Listen on the given address.")
	flagSecret  = pflag.StringP("secret", "s", "", "Use the given secret for token generation.")
)

func main() {
	pflag.Parse()

	if *flagVersion {
		fmt.Printf("%s (TunaQuest v%s)\n", version.ServerCurrent, version.Current)
		return
	}

	args := pflag.Args()

	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "Too many arguments\nDo -h for help.\n")
		os.Exit(1)
	}

	// get address info
	port := 0
	addr := ""
	listenAddr := os.Getenv(EnvListen)
	if pflag.Lookup("listen").Changed {
		listenAddr = *flagListen
	}
	if listenAddr != "" {
		bindParts := strings.SplitN(listenAddr, ":", 2)
		if len(bindParts) != 2 {
			fmt.Fprintf(os.Stderr, "Listen address is not in ADDRESS:PORT or :PORT format.\nDo -h for help.\n")
			os.Exit(1)
		}

		var err error

		addr = bindParts[0]
		port, err = strconv.Atoi(bindParts[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%q is not a valid port number.\nDo -h for help.\n", bindParts[1])
			os.Exit(1)
		}
	}

	// get token secret
	var tokSecret []byte
	tokSecStr := os.Getenv(EnvSecret)
	if pflag.Lookup("secret").Changed {
		tokSecStr = *flagSecret
	}
	// was the secret given?
	if tokSecStr != "" {
		// if so, validate it
		tokSecret = []byte(tokSecStr)

		for len(tokSecret) < 32 {
			doubledTokSecret := make([]byte, len(tokSecret)*2)
			copy(doubledTokSecret, tokSecret)
			copy(doubledTokSecret[len(tokSecret):], tokSecret)
			tokSecret = doubledTokSecret
		}

		if len(tokSecret) > 64 {
			// keys would be chopped at 64, so rather than the user thinking
			// they have more security by giving a longer key, refuse to start.
			fmt.Fprintf(os.Stderr, "Token secret is %d bytes, but it must be <= 64 bytes\nDo -h for help.\n", len(tokSecret))
		}
	} else {
		// generate a new one

		// use all 64 possible bytes if doing a generated secret
		tokSecret = make([]byte, 64)
		_, err := rand.Read(tokSecret)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not generate token secret: %s\n", err.Error())
		}

		// yell at the user bc they should know their secret might be bad
		log.Printf("WARN : Using generated token secret; all tokens issued will become invalid at shutdown")
	}

	// configuration complete, initialize the server
	tqs := server.New(tokSecret)
	log.Printf("DEBUG: Server initialized")

	// immediately create the admin user so we have someone we can log in as.
	_, err := tqs.CreateUser(context.Background(), "admin", "password", "bogus@example.com", dao.Admin)
	if err != nil {
		log.Printf("ERROR: could not create initial admin user: %v", err)
		os.Exit(2)
	}
	log.Printf("INFO : Added initial admin user with password 'password'...")

	// okay, now actually launch it
	log.Printf("INFO : Starting TunaQuest server %s...", version.ServerCurrent)
	tqs.ServeForever(addr, port)
}
