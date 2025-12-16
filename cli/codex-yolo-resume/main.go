package main

import (
	"os"

	"codex-control/internal/app/yolocli"
	"codex-control/internal/yolo"
)

func main() {
	synopsis := "codex-yolo-resume [options] -- [codex arguments]"
	os.Exit(yolocli.Run(yolo.ModeResume, "codex-yolo-resume", synopsis))
}
