package logger

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// Prefix identifies a log category.
type Prefix string

const (
	PrefixCLI      Prefix = "[CLI]"
	PrefixCodex    Prefix = "[Codex]"
	PrefixDownload Prefix = "[Download]"
	PrefixInstall  Prefix = "[Install]"
	PrefixMenu     Prefix = "[Menu]"
	PrefixAuth     Prefix = "[Auth]"
	PrefixError    Prefix = "[Error]"
)

// Logger renders prefixed log messages with optional ANSI colors.
type Logger struct {
	colors      map[Prefix]string
	reset       string
	colorOutput bool
}

// New creates a Logger whose colors depend on stdout being a TTY.
func New() *Logger {
	colorOutput := term.IsTerminal(int(os.Stdout.Fd()))
	reset := ""
	colors := map[Prefix]string{}
	if colorOutput {
		reset = "\u001b[0m"
		colors = map[Prefix]string{
			PrefixCLI:      "\u001b[36m",
			PrefixCodex:    "\u001b[35m",
			PrefixDownload: "\u001b[34m",
			PrefixInstall:  "\u001b[32m",
			PrefixMenu:     "\u001b[33m",
			PrefixAuth:     "\u001b[38;5;204m",
			PrefixError:    "\u001b[31m",
		}
	}
	return &Logger{colors: colors, reset: reset, colorOutput: colorOutput}
}

func (l *Logger) formatPrefix(prefix Prefix) string {
	if code, ok := l.colors[prefix]; ok && l.colorOutput {
		return fmt.Sprintf("%s%s%s", code, prefix, l.reset)
	}
	return string(prefix)
}

// Printf writes a formatted line to stdout.
func (l *Logger) Printf(prefix Prefix, format string, args ...any) {
	fmt.Fprintf(os.Stdout, "%s %s\n", l.formatPrefix(prefix), fmt.Sprintf(format, args...))
}

// Errorf writes a formatted line to stderr.
func (l *Logger) Errorf(prefix Prefix, format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s %s\n", l.formatPrefix(prefix), fmt.Sprintf(format, args...))
}
