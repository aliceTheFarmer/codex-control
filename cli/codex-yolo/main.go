package main

import (
	"os"

	"codex-control/internal/app/yolocli"
	"codex-control/internal/yolo"
)

func main() {
	synopsis := "codex-yolo [options] -- [codex arguments]"
	os.Exit(yolocli.Run(yolo.ModeDefault, "codex-yolo", synopsis))
}
