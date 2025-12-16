package main

import (
	"os"

	"codex-control/internal/app/updateselect"
)

func main() {
	os.Exit(updateselect.Run(os.Args[1:]))
}
