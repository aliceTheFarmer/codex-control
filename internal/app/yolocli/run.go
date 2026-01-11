package yolocli

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"codex-control/internal/cli"
	"codex-control/internal/config"
	"codex-control/internal/logger"
	"codex-control/internal/output"
	"codex-control/internal/yolo"
)

type yoloConfig struct {
	Verbosity   int    `yaml:"verbosity"`
	CodexBinary string `yaml:"codex-binary"`
}

// Run executes the codex-yolo style CLI for the provided mode.
func Run(mode yolo.Mode, command string, synopsis string) int {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log := logger.New()

	defaults := yoloConfig{Verbosity: 1, CodexBinary: "codex"}
	var cfg yoloConfig
	if _, err := config.Load(command, defaults, &cfg); err != nil {
		log.Errorf(logger.PrefixCLI, "Failed to load config: %v", err)
		return 1
	}

	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	global := cli.GlobalFlags{}
	global.Register(fs, cfg.Verbosity)

	var codexBinary string
	fs.StringVar(&codexBinary, "codex-binary", cfg.CodexBinary, "Path to the codex binary.")

	options := append(cli.GlobalUsageOptions(), cli.UsageOption{
		Long:        "codex-binary",
		Short:       "c",
		Value:       "<path>",
		Description: "Override the codex executable path.",
	})
	fs.Usage = func() {
		cli.UsagePrinter{Command: command, Synopsis: synopsis, Options: options}.Print()
	}

	args, err := cli.Parse(fs, os.Args[1:], []cli.FlagAlias{
		{Canonical: "verbosity", Short: "v", HasValue: true},
		{Canonical: "codex-binary", Short: "c", HasValue: true},
	})
	if err != nil {
		log.Errorf(logger.PrefixCLI, "Flag parsing failed: %v", err)
		return 1
	}
	if err := cli.ValidateVerbosity(global.Verbosity); err != nil {
		log.Errorf(logger.PrefixCLI, "Invalid verbosity: %v", err)
		return 1
	}
	if codexBinary == "" {
		codexBinary = defaults.CodexBinary
	}

	runner := yolo.Runner{Binary: codexBinary, Mode: mode, Log: log}
	result, runErr := runner.Run(ctx, args)
	if runErr != nil {
		log.Errorf(logger.PrefixCodex, "Codex failed: %v", runErr)
		return result.ExitCode
	}
	printer := output.Printer{Verbosity: global.Verbosity}
	env := map[string]string{
		"binary": codexBinary,
		"mode":   string(mode),
	}
	payload := struct {
		Command  []string `json:"command"`
		ExitCode int      `json:"exit_code"`
	}{Command: result.Command, ExitCode: result.ExitCode}
	if err := printer.Print(env, payload); err != nil {
		log.Errorf(logger.PrefixCLI, "Failed to render output: %v", err)
	}
	return 0
}
