package cli

import "flag"

// GlobalFlags stores flags shared by every binary.
type GlobalFlags struct {
	Verbosity int
}

// Register binds the shared flags to the provided FlagSet.
func (g *GlobalFlags) Register(fs *flag.FlagSet, defaultVerbosity int) {
	g.Verbosity = defaultVerbosity
	fs.IntVar(&g.Verbosity, "verbosity", defaultVerbosity, "Set verbosity level (0-2).")
}
