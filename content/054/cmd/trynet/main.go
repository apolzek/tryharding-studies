// trynet is the CLI. It does no heavy lifting — it just talks to the local
// trynetd daemon over a UNIX socket.
package main

import (
	"os"

	"github.com/apolzek/trynet/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
