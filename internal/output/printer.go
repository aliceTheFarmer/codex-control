package output

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// Printer renders results according to the shared verbosity contract.
type Printer struct {
	Verbosity int
}

// Print renders the environment dump (verbosity 2) and JSON payload.
func (p Printer) Print(env map[string]string, payload any) error {
	if p.Verbosity <= 0 {
		return nil
	}
	if p.Verbosity >= 2 {
		for _, key := range sortedKeys(env) {
			fmt.Fprintf(os.Stdout, "%s=%s\n", key, env[key])
		}
		fmt.Fprintln(os.Stdout, "----- / -----")
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, string(data))
	return nil
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
