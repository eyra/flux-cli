package cmd

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

const githubRepo = "eyra/flux-cli"

var currentVersion = "dev"

var noSkill bool

func SetVersion(v string) {
	currentVersion = v
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Flux CLI to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Current version: %s\n", currentVersion)
		fmt.Println("Checking for updates...")

		latest, downloadURL, err := fetchLatestRelease()
		if err != nil {
			return fmt.Errorf("could not fetch latest release: %w", err)
		}

		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("could not determine executable path: %w", err)
		}
		execPath, err = filepath.EvalSymlinks(execPath)
		if err != nil {
			return fmt.Errorf("could not resolve symlinks: %w", err)
		}

		if latest == currentVersion {
			fmt.Printf("Already up to date (%s)\n", currentVersion)
		} else {
			fmt.Printf("Updating to %s...\n", latest)
			if err := downloadAndReplace(downloadURL, execPath); err != nil {
				return fmt.Errorf("update failed: %w", err)
			}
			fmt.Printf("✓ Updated to %s\n", latest)
		}

		if !noSkill {
			refreshSkills(cmd.OutOrStdout(), cmd.ErrOrStderr(), execPath)
		}
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show Flux CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(currentVersion)
	},
}

func fetchLatestRelease() (tag, downloadURL string, err error) {
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", err
	}

	os_ := runtime.GOOS
	arch := runtime.GOARCH
	want := fmt.Sprintf("flux_%s_%s.tar.gz", os_, arch)

	for _, asset := range release.Assets {
		if asset.Name == want {
			return release.TagName, asset.BrowserDownloadURL, nil
		}
	}

	return "", "", fmt.Errorf("no release asset found for %s/%s", os_, arch)
}

// refreshSkills runs "<flux> install <agent>" for every agent that already has
// the skill, so the skill comes from the binary at flux, not this process.
// Symlinked skills are skipped; a failed install is only a warning, because
// the binary itself was updated.
func refreshSkills(stdout, stderr io.Writer, flux string) {
	found := false
	for _, a := range agents {
		root, err := a.root()
		if err != nil {
			continue
		}
		dir, path := skillPaths(root)
		if link := skillSymlink(dir, path); link != "" {
			fmt.Fprintf(stdout, "Skipped %s: %s is a symlink\n", a.name, link)
			found = true
			continue
		}
		if _, err := os.Lstat(path); err != nil {
			continue
		}
		found = true
		install := exec.Command(flux, "install", a.name)
		install.Stdout, install.Stderr = stdout, stderr
		if err := install.Run(); err != nil {
			fmt.Fprintf(stderr, "Warning: the flux skill for %s was not updated (%v)\n", a.name, err)
		}
	}
	if !found {
		fmt.Fprintln(stdout, "No flux skill installed; run flux install claude, omp or codex")
	}
}

func downloadAndReplace(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Name != "flux" && !strings.HasSuffix(hdr.Name, "/flux") {
			continue
		}

		tmp, err := os.CreateTemp(filepath.Dir(dest), ".flux-update-*")
		if err != nil {
			return err
		}
		if _, err := io.Copy(tmp, tr); err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return err
		}
		tmp.Close()

		if err := os.Chmod(tmp.Name(), 0755); err != nil {
			os.Remove(tmp.Name())
			return err
		}

		return os.Rename(tmp.Name(), dest)
	}

	return fmt.Errorf("flux binary not found in archive")
}

func init() {
	updateCmd.Flags().BoolVar(&noSkill, "no-skill", false, "Don't refresh the installed flux skills")
	rootCmd.AddCommand(updateCmd, versionCmd)
}
