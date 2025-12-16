package codex

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Platform describes the host architecture and OS suffix used by Codex releases.
type Platform struct {
	Arch string
	OS   string
}

// DetectPlatform resolves the Codex archive suffix for the current machine.
func DetectPlatform() (Platform, error) {
	arch, err := detectArch()
	if err != nil {
		return Platform{}, err
	}
	osPart, err := detectOS()
	if err != nil {
		return Platform{}, err
	}
	return Platform{Arch: arch, OS: osPart}, nil
}

// ArchiveName returns the tarball name that matches the platform.
func (p Platform) ArchiveName() string {
	return fmt.Sprintf("codex-%s-%s.tar.gz", p.Arch, p.OS)
}

func detectArch() (string, error) {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64", nil
	case "arm64":
		return "aarch64", nil
	default:
		return "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}
}

func detectOS() (string, error) {
	switch runtime.GOOS {
	case "linux":
		if isMusl() {
			return "unknown-linux-musl", nil
		}
		return "unknown-linux-gnu", nil
	case "darwin":
		return "apple-darwin", nil
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func isMusl() bool {
	cmd := exec.Command("ldd", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(output)), "musl")
}
