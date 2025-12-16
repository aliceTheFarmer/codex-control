package codex

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"codex-control/internal/logger"
)

// Installer downloads and installs Codex binaries.
type Installer struct {
	Client     *Client
	Log        *logger.Logger
	Workdir    string
	TargetPath string
}

// InstallResult summarizes an installation run.
type InstallResult struct {
	Version string `json:"version"`
	Target  string `json:"target"`
	Archive string `json:"archive"`
	Bytes   int64  `json:"bytes"`
}

// InstallLatest fetches the newest release and installs it.
func (i *Installer) InstallLatest(ctx context.Context, platform Platform) (InstallResult, error) {
	if err := i.validate(); err != nil {
		return InstallResult{}, err
	}
	release, err := i.Client.Latest(ctx)
	if err != nil {
		return InstallResult{}, err
	}
	archive := platform.ArchiveName()
	asset, ok := release.FindAsset(archive)
	if !ok {
		return InstallResult{}, fmt.Errorf("asset %s not found in release %s", archive, release.Tag)
	}
	return i.install(ctx, release, asset)
}

// InstallRelease installs a specific release + asset pair.
func (i *Installer) InstallRelease(ctx context.Context, release Release, asset Asset) (InstallResult, error) {
	if err := i.validate(); err != nil {
		return InstallResult{}, err
	}
	return i.install(ctx, release, asset)
}

func (i *Installer) install(ctx context.Context, release Release, asset Asset) (InstallResult, error) {
	if i.Log != nil {
		i.Log.Printf(logger.PrefixDownload, "Downloading %s (%s)", asset.Name, release.Tag)
	}
	archiveFile, err := i.createTempFile("codex-archive-*.tar.gz")
	if err != nil {
		return InstallResult{}, err
	}
	defer os.Remove(archiveFile.Name())
	if err := i.download(ctx, asset.URL, archiveFile); err != nil {
		return InstallResult{}, err
	}
	if i.Log != nil {
		i.Log.Printf(logger.PrefixInstall, "Extracting Codex from %s", archiveFile.Name())
	}
	binaryPath, err := extractBinary(archiveFile.Name(), i.Workdir)
	if err != nil {
		return InstallResult{}, err
	}
	defer os.Remove(binaryPath)

	target := i.TargetPath
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return InstallResult{}, err
	}
	if i.Log != nil {
		i.Log.Printf(logger.PrefixInstall, "Installing Codex to %s", target)
	}
	cmd := exec.Command("sudo", "install", "-m", "0755", binaryPath, target)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return InstallResult{}, err
	}
	return InstallResult{Version: release.Tag, Target: target, Archive: asset.Name, Bytes: asset.Size}, nil
}

func (i *Installer) validate() error {
	if i.Client == nil {
		return errors.New("installer client is nil")
	}
	if i.Workdir == "" {
		return errors.New("installer workspace is empty")
	}
	if i.TargetPath == "" {
		return errors.New("installer target path is empty")
	}
	return nil
}

func (i *Installer) createTempFile(pattern string) (*os.File, error) {
	if err := os.MkdirAll(i.Workdir, 0o755); err != nil {
		return nil, err
	}
	return os.CreateTemp(i.Workdir, pattern)
}

func (i *Installer) download(ctx context.Context, url string, dest *os.File) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	if i.Client != nil {
		i.Client.decorateHeaders(req)
	}
	resp, err := i.Client.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	if _, err := io.Copy(dest, resp.Body); err != nil {
		return err
	}
	if err := dest.Sync(); err != nil {
		return err
	}
	if _, err := dest.Seek(0, io.SeekStart); err != nil {
		return err
	}
	return nil
}

func extractBinary(archivePath, destDir string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if !strings.HasPrefix(filepath.Base(hdr.Name), "codex") {
			continue
		}
		tmp, err := os.CreateTemp(destDir, "codex-bin-*")
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(tmp, tr); err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return "", err
		}
		if err := tmp.Chmod(0o755); err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return "", err
		}
		if err := tmp.Close(); err != nil {
			os.Remove(tmp.Name())
			return "", err
		}
		return tmp.Name(), nil
	}
	return "", errors.New("codex binary not found in archive")
}
