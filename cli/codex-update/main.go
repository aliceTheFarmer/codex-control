package main

import (
	"os"

	"codex-control/internal/app/updatecli"
)

func main() {
	os.Exit(updatecli.Run(os.Args[1:]))
}
