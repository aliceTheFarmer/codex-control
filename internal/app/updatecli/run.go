package updatecli

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"codex-control/internal/cli"
	"codex-control/internal/codex"
	"codex-control/internal/config"
	"codex-control/internal/env"
	"codex-control/internal/logger"
	"codex-control/internal/output"
)

type updateConfig struct {
	Verbosity   int    `yaml:"verbosity"`
	GitHubToken string `yaml:"github-token"`
}

// Run executes the codex-update workflow.
func Run(args []string) int {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	command := "codex-update"
	synopsis := "codex-update [options]"

	log := logger.New()
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	defaults := updateConfig{Verbosity: 1}
	var cfg updateConfig
	if _, err := config.Load(command, defaults, &cfg); err != nil {
		log.Errorf(logger.PrefixCLI, "Failed to load config: %v", err)
		return 1
	}

	global := cli.GlobalFlags{}
	global.Register(fs, cfg.Verbosity)

	options := cli.GlobalUsageOptions()
	fs.Usage = func() {
		cli.UsagePrinter{Command: command, Synopsis: synopsis, Options: options}.Print()
	}

	leftovers, err := cli.Parse(fs, args, []cli.FlagAlias{{Canonical: "verbosity", Short: "v", HasValue: true}})
	if err != nil {
		log.Errorf(logger.PrefixCLI, "Flag parsing failed: %v", err)
		return 1
	}
	if len(leftovers) > 0 {
		log.Errorf(logger.PrefixCLI, "Unexpected positional arguments: %v", leftovers)
		return 1
	}
	if err := cli.ValidateVerbosity(global.Verbosity); err != nil {
		log.Errorf(logger.PrefixCLI, "Invalid verbosity: %v", err)
		return 1
	}

	workspace, err := env.PrepareWorkspace()
	if err != nil {
		log.Errorf(logger.PrefixCLI, "Failed to prepare workspace: %v", err)
		return 1
	}
	defer env.CleanupWorkspace(workspace)

	platform, err := codex.DetectPlatform()
	if err != nil {
		log.Errorf(logger.PrefixCLI, "Failed to resolve platform: %v", err)
		return 1
	}

	installer := codex.Installer{
		Client:     codex.NewClient(nil, cfg.GitHubToken),
		Log:        log,
		Workdir:    workspace,
		TargetPath: env.TargetBinaryPath(),
	}
	result, err := installer.InstallLatest(ctx, platform)
	if err != nil {
		log.Errorf(logger.PrefixInstall, "Installation failed: %v", err)
		return 1
	}

	printer := output.Printer{Verbosity: global.Verbosity}
	envDump := map[string]string{
		"workspace": workspace,
		"target":    result.Target,
		"archive":   result.Archive,
	}
	if err := printer.Print(envDump, result); err != nil {
		log.Errorf(logger.PrefixCLI, "Failed to render output: %v", err)
		return 1
	}
	return 0
}
