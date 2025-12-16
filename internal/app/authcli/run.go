package authcli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"codex-control/internal/auth"
	"codex-control/internal/cli"
	"codex-control/internal/logger"
	"codex-control/internal/output"
	"codex-control/internal/tui/menu"
)

// Run executes the codex-auth workflow.
func Run(args []string) int {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	const command = "codex-auth"
	const synopsis = "codex-auth [options]"

	log := logger.New()
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	global := cli.GlobalFlags{}
	global.Register(fs)

	var authPathFlag string
	fs.StringVar(&authPathFlag, "auths-path", "", "Folder containing Codex auth profiles.")

	options := append(cli.GlobalUsageOptions(), cli.UsageOption{
		Long:        "auths-path",
		Short:       "a",
		Value:       "<path>",
		Description: "Override CODEX_AUTHS_PATH for this run.",
	})
	fs.Usage = func() {
		cli.UsagePrinter{Command: command, Synopsis: synopsis, Options: options}.Print()
	}

	leftovers, err := cli.Parse(fs, args, []cli.FlagAlias{
		{Canonical: "verbosity", Short: "v", HasValue: true},
		{Canonical: "auths-path", Short: "a", HasValue: true},
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

	authPath := authPathFlag
	if authPath == "" {
		authPath = os.Getenv("CODEX_AUTHS_PATH")
	}
	authPath, err = auth.ValidateRoot(authPath)
	if err != nil {
		log.Errorf(logger.PrefixAuth, "Invalid auth directory: %v", err)
		return 1
	}

	tracker, err := auth.LoadUsageTracker(authPath)
	if err != nil {
		log.Errorf(logger.PrefixAuth, "Failed to load auth usage data: %v", err)
		return 1
	}
	loader := &authLoader{root: authPath, tracker: tracker}
	cfg := menu.Config{
		Context:          ctx,
		ListTitle:        "Codex auth profiles",
		ListHelp:         []string{"Use ↑/↓ or digits + Enter to highlight a profile.", "Press R to rescan the folder, Ctrl+C to abort."},
		ActionsTitle:     "Auth actions",
		ActionsHelp:      []string{"Enter copies the highlighted profile to ~/.codex/auth.json."},
		PanelPlaceholder: "Selections show copy results here.",
		Loader:           loader.Load,
		DisablePanel:     true,
	}
	cfg.Actions = []menu.Action{
		{
			Label: "Use auth file",
			Exec: func(entry menu.Entry) tea.Cmd {
				authFile, ok := entry.Payload.(auth.File)
				if !ok {
					return func() tea.Msg {
						return menu.PanelUpdate("Copy auth", "Invalid selection payload", nil, fmt.Errorf("invalid payload"))
					}
				}
				return tea.Sequence(runCopyCmd(authFile, tracker), tea.Quit)
			},
		},
	}

	result, err := menu.Start(cfg)
	if err != nil {
		log.Errorf(logger.PrefixMenu, "Menu failed: %v", err)
		return 1
	}
	if !result.Success {
		log.Errorf(logger.PrefixMenu, "Operation cancelled before copying an auth file")
		return 1
	}
	copyResult, ok := result.ActionPayload.(auth.CopyResult)
	if !ok {
		log.Errorf(logger.PrefixMenu, "Unexpected action payload type")
		return 1
	}

	printer := output.Printer{Verbosity: global.Verbosity}
	envDump := map[string]string{
		"auths_path": authPath,
	}
	if err := printer.Print(envDump, copyResult); err != nil {
		log.Errorf(logger.PrefixCLI, "Failed to render output: %v", err)
		return 1
	}
	return 0
}

type authLoader struct {
	root    string
	tracker *auth.UsageTracker
}

func (a *authLoader) Load(_ context.Context) ([]menu.Entry, error) {
	files, err := auth.ListFiles(a.root)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("%s contains no files", a.root)
	}
	lastUsed := func(name string) time.Time {
		if a.tracker == nil {
			return time.Time{}
		}
		return a.tracker.LastUsed(name)
	}
	sort.SliceStable(files, func(i, j int) bool {
		ti := lastUsed(files[i].Name)
		tj := lastUsed(files[j].Name)
		if ti.IsZero() && tj.IsZero() {
			return files[i].Name < files[j].Name
		}
		if ti.IsZero() {
			return true
		}
		if tj.IsZero() {
			return false
		}
		if !ti.Equal(tj) {
			return ti.Before(tj)
		}
		return files[i].Name < files[j].Name
	})
	entries := make([]menu.Entry, 0, len(files))
	for _, file := range files {
		entries = append(entries, menu.Entry{
			Title:       file.Name,
			Description: fmt.Sprintf("Last used %s", formatAuthTimestamp(lastUsed(file.Name))),
			Payload:     file,
		})
	}
	return entries, nil
}

func runCopyCmd(file auth.File, tracker *auth.UsageTracker) tea.Cmd {
	return func() tea.Msg {
		result, err := auth.Install(file.Path)
		if err != nil {
			return menu.PanelUpdate("Copy auth", err.Error(), result, err)
		}
		if tracker != nil {
			_ = tracker.Touch(file.Name, time.Now())
		}
		content := fmt.Sprintf("Copied %s to %s", file.Name, result.Destination)
		return menu.PanelUpdate("Copy auth", content, result, nil)
	}
}

func formatAuthTimestamp(t time.Time) string {
	if t.IsZero() {
		return "never used"
	}
	return t.Local().Format("Mon, 02 Jan 2006 15:04")
}
