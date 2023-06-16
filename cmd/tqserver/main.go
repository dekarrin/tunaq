/*
Tqserver starts a TunaQuest server and begins listening for new connections.

Usage:

	tqserver [flags]
*/
package main

import "github.com/dekarrin/tunaq/server"

func main() {
	tqs := server.New()

	tqs.ServeForever()
}
