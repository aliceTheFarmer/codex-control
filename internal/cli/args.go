package cli

import (
	"errors"
	"flag"
	"fmt"
	"sort"
	"strings"
)

// FlagAlias maps a short flag to its canonical long name.
type FlagAlias struct {
	Canonical string
	Short     string
	HasValue  bool
}

// Parse expands aliases and parses args with the provided FlagSet.
func Parse(fs *flag.FlagSet, args []string, aliases []FlagAlias) ([]string, error) {
	expanded, err := expandAliases(args, aliases)
	if err != nil {
		return nil, err
	}
	if err := fs.Parse(expanded); err != nil {
		return nil, err
	}
	return fs.Args(), nil
}

func expandAliases(args []string, aliases []FlagAlias) ([]string, error) {
	if len(aliases) == 0 {
		return args, nil
	}
	aliasMap := map[string]FlagAlias{}
	for _, alias := range aliases {
		if alias.Short == "" {
			continue
		}
		aliasMap["-"+alias.Short] = alias
		aliasMap["--"+alias.Short] = alias
	}
	out := make([]string, 0, len(args))
	skipNext := map[int]struct{}{}
	for i := 0; i < len(args); i++ {
		if _, ok := skipNext[i]; ok {
			continue
		}
		arg := args[i]
		if arg == "--" {
			out = append(out, args[i:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || len(arg) < 2 {
			out = append(out, arg)
			continue
		}
		if alias, ok := aliasMap[arg]; ok {
			repl, remaining, err := normalizeAlias(arg, args, i, alias)
			if err != nil {
				return nil, err
			}
			out = append(out, repl...)
			for _, idx := range remaining {
				skipNext[idx] = struct{}{}
			}
			continue
		}
		out = append(out, arg)
	}
	return out, nil
}

func normalizeAlias(arg string, args []string, index int, alias FlagAlias) ([]string, []int, error) {
	var consumed []int
	key := fmt.Sprintf("--%s", alias.Canonical)
	if !alias.HasValue {
		return []string{key}, nil, nil
	}
	if strings.Contains(arg, "=") {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 || parts[1] == "" {
			return nil, nil, fmt.Errorf("flag %s requires a value", arg)
		}
		return []string{fmt.Sprintf("%s=%s", key, parts[1])}, nil, nil
	}
	next := index + 1
	if next >= len(args) {
		return nil, nil, fmt.Errorf("flag %s requires a value", arg)
	}
	consumed = append(consumed, next)
	return []string{fmt.Sprintf("%s=%s", key, args[next])}, consumed, nil
}

// UsageOption describes how to present a CLI flag in help output.
type UsageOption struct {
	Long        string
	Short       string
	Value       string
	Description string
}

// UsagePrinter prints help text following the shared conventions.
type UsagePrinter struct {
	Command  string
	Synopsis string
	Options  []UsageOption
}

// Print writes the usage information to stderr.
func (u UsagePrinter) Print() {
	if u.Command == "" {
		u.Command = "command"
	}
	fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s %s\n", u.Command, u.Synopsis)
	if len(u.Options) == 0 {
		return
	}
	fmt.Fprintln(flag.CommandLine.Output(), "\nOptions:")
	opts := make([]UsageOption, len(u.Options))
	copy(opts, u.Options)
	sort.Slice(opts, func(i, j int) bool {
		return opts[i].Long < opts[j].Long
	})
	for _, opt := range opts {
		line := formatOptionLabel(opt)
		fmt.Fprintf(flag.CommandLine.Output(), "  %s\n", line)
		fmt.Fprintf(flag.CommandLine.Output(), "      %s\n", opt.Description)
	}
}

func formatOptionLabel(opt UsageOption) string {
	parts := []string{}
	if opt.Short != "" {
		if opt.Value != "" {
			parts = append(parts, fmt.Sprintf("-%s %s", opt.Short, opt.Value))
		} else {
			parts = append(parts, fmt.Sprintf("-%s", opt.Short))
		}
	}
	long := fmt.Sprintf("--%s", opt.Long)
	if opt.Value != "" {
		long = fmt.Sprintf("%s=%s", long, opt.Value)
	}
	if opt.Short != "" {
		return fmt.Sprintf("%s, %s", parts[0], long)
	}
	return long
}

// GlobalUsageOptions returns help entries shared across binaries.
func GlobalUsageOptions() []UsageOption {
	return []UsageOption{
		{
			Long:        "help",
			Short:       "h",
			Description: "Show this help message and exit.",
		},
		{
			Long:        "verbosity",
			Short:       "v",
			Value:       "<0|1|2>",
			Description: "Set verbosity: 0 silent, 1 JSON only, 2 env dump + JSON.",
		},
	}
}

// ValidateVerbosity ensures verbosity values comply with shared rules.
func ValidateVerbosity(value int) error {
	if value < 0 || value > 2 {
		return errors.New("verbosity must be within 0-2")
	}
	return nil
}
