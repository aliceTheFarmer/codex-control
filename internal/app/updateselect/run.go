package updateselect

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"codex-control/internal/cli"
	"codex-control/internal/codex"
	"codex-control/internal/env"
	"codex-control/internal/logger"
	"codex-control/internal/output"
	"codex-control/internal/tui/menu"
)

// Run executes the codex-update-select workflow.
func Run(args []string) int {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	const command = "codex-update-select"
	const synopsis = "codex-update-select [options]"

	log := logger.New()
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	global := cli.GlobalFlags{}
	global.Register(fs)

	var releaseLimit int
	fs.IntVar(&releaseLimit, "release-limit", 200, "Maximum number of releases to display.")

	options := append(cli.GlobalUsageOptions(), cli.UsageOption{
		Long:        "release-limit",
		Short:       "l",
		Value:       "<count>",
		Description: "Limit the number of releases fetched from GitHub.",
	})
	fs.Usage = func() {
		cli.UsagePrinter{Command: command, Synopsis: synopsis, Options: options}.Print()
	}

	leftovers, err := cli.Parse(fs, args, []cli.FlagAlias{
		{Canonical: "verbosity", Short: "v", HasValue: true},
		{Canonical: "release-limit", Short: "l", HasValue: true},
	})
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
	if releaseLimit <= 0 {
		releaseLimit = 200
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

	client := codex.NewClient(nil)
	installer := codex.Installer{Client: client, Log: log, Workdir: workspace, TargetPath: env.TargetBinaryPath()}
	loader := &releaseLoader{client: client, platform: platform, limit: releaseLimit}

	cfg := menu.Config{
		Context:          ctx,
		ListTitle:        "Available Codex releases",
		ListHelp:         []string{"Use ↑/↓ or digits + Enter to highlight a release.", "Press R to refresh, Ctrl+C to abort."},
		ActionsTitle:     "Release actions",
		ActionsHelp:      []string{"Enter installs the highlighted release.", "Esc returns to the release list."},
		PanelPlaceholder: "Action output appears here.",
		Loader:           loader.Load,
		DisablePanel:     true,
	}
	cfg.Actions = []menu.Action{
		{
			Label: "Install release",
			Exec: func(entry menu.Entry) tea.Cmd {
				choice, ok := entry.Payload.(releaseChoice)
				if !ok {
					return func() tea.Msg {
						return menu.PanelUpdate("Install release", "Invalid choice payload", nil, fmt.Errorf("invalid payload"))
					}
				}
				return tea.Sequence(runInstallCmd(ctx, &installer, choice), tea.Quit)
			},
		},
	}

	result, err := menu.Start(cfg)
	if err != nil {
		log.Errorf(logger.PrefixMenu, "Menu failed: %v", err)
		return 1
	}
	if !result.Success {
		log.Errorf(logger.PrefixMenu, "Operation cancelled before installation")
		return 1
	}
	installResult, ok := result.ActionPayload.(codex.InstallResult)
	if !ok {
		log.Errorf(logger.PrefixMenu, "Unexpected action payload type")
		return 1
	}

	printer := output.Printer{Verbosity: global.Verbosity}
	envDump := map[string]string{
		"workspace": workspace,
		"target":    installResult.Target,
		"archive":   installResult.Archive,
	}
	if err := printer.Print(envDump, installResult); err != nil {
		log.Errorf(logger.PrefixCLI, "Failed to render output: %v", err)
		return 1
	}
	return 0
}

type releaseChoice struct {
	Release codex.Release
	Asset   codex.Asset
}

type releaseLoader struct {
	client   *codex.Client
	platform codex.Platform
	limit    int
}

func (r *releaseLoader) Load(ctx context.Context) ([]menu.Entry, error) {
	releases, err := r.client.List(ctx, r.limit)
	if err != nil {
		return nil, err
	}
	entries := make([]menu.Entry, 0, len(releases))
	archive := r.platform.ArchiveName()
	for _, rel := range releases {
		asset, ok := rel.FindAsset(archive)
		if !ok {
			continue
		}
		entries = append(entries, menu.Entry{
			Title:       rel.Tag,
			Description: fmt.Sprintf("%s • %s", humanSize(asset.Size), formatPublished(rel.PublishedAt)),
			Badges:      []string{"ready"},
			Payload:     releaseChoice{Release: rel, Asset: asset},
		})
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no releases provide %s", archive)
	}
	return entries, nil
}

func formatPublished(t time.Time) string {
	if t.IsZero() {
		return "unknown release time"
	}
	return t.UTC().Format("Mon, 02 Jan 2006 15:04 MST")
}

func humanSize(size int64) string {
	if size <= 0 {
		return "unknown size"
	}
	mb := float64(size) / (1024 * 1024)
	return fmt.Sprintf("%.1f MiB", mb)
}

func runInstallCmd(ctx context.Context, installer *codex.Installer, choice releaseChoice) tea.Cmd {
	return func() tea.Msg {
		installCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		result, err := installer.InstallRelease(installCtx, choice.Release, choice.Asset)
		if err != nil {
			return menu.PanelUpdate("Install release", err.Error(), result, err)
		}
		content := fmt.Sprintf("Version %s installed at %s", result.Version, result.Target)
		return menu.PanelUpdate("Install release", content, result, nil)
	}
}
