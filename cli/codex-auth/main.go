package main

import (
	"os"

	"codex-control/internal/app/authcli"
)

func main() {
	os.Exit(authcli.Run(os.Args[1:]))
}
