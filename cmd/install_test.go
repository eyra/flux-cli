package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestInstallTargets(t *testing.T) {
	want, err := os.ReadFile(filepath.Join("..", "skill", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		target, envVar, defaultDir string
	}{
		{"claude", "CLAUDE_CONFIG_DIR", ".claude"},
		{"omp", "PI_CODING_AGENT_DIR", ".omp/agent"},
		{"codex", "", ".agents"},
	} {
		for _, override := range []bool{false, true} {
			if override && tc.envVar == "" {
				continue
			}
			name := tc.target
			if override {
				name += " with " + tc.envVar
			}
			t.Run(name, func(t *testing.T) {
				isolateCommand(t)
				t.Setenv("CLAUDE_CONFIG_DIR", "")
				t.Setenv("PI_CODING_AGENT_DIR", "")
				root := filepath.Join(os.Getenv("HOME"), tc.defaultDir)
				if override {
					root = t.TempDir()
					t.Setenv(tc.envVar, root)
				}
				command, _, err := installCmd.Find([]string{tc.target})
				if err != nil {
					t.Fatal(err)
				}
				var output bytes.Buffer
				out := &cobra.Command{}
				out.SetOut(&output)
				if err := command.RunE(out, nil); err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(root, "skills", "flux", "SKILL.md")
				got, err := os.ReadFile(path)
				if err != nil || !bytes.Equal(got, want) {
					t.Fatalf("%s does not hold skill/SKILL.md: %v", path, err)
				}
				if output.String() != "Installed flux skill to "+path+"\n" {
					t.Fatalf("unexpected output: %q", output.String())
				}
			})
		}
	}
}

func TestInstallSkillBacksUpLocalEdits(t *testing.T) {
	root := t.TempDir()
	path, backup, err := installSkill(root, []byte("v1"))
	if err != nil || backup != "" {
		t.Fatalf("fresh install: backup %q, err %v", backup, err)
	}
	if _, backup, err = installSkill(root, []byte("v1")); err != nil || backup != "" {
		t.Fatalf("identical reinstall: backup %q, err %v", backup, err)
	}
	if err := os.WriteFile(path, []byte("v1 with local edits"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, backup, err = installSkill(root, []byte("v2")); err != nil || backup == "" {
		t.Fatalf("changed reinstall: backup %q, err %v", backup, err)
	}
	if got, _ := os.ReadFile(backup); string(got) != "v1 with local edits" {
		t.Fatalf("backup holds %q", got)
	}
	if got, _ := os.ReadFile(path); string(got) != "v2" {
		t.Fatalf("skill holds %q", got)
	}
	backups, _ := filepath.Glob(path + ".bak-*")
	if len(backups) != 1 {
		t.Fatalf("expected one backup, found %v", backups)
	}
}

func TestInstallSkillRefusesSymlinks(t *testing.T) {
	for _, link := range []string{"folder", "file"} {
		t.Run(link, func(t *testing.T) {
			root, checkout := t.TempDir(), t.TempDir()
			repoSkill := filepath.Join(checkout, "SKILL.md")
			if err := os.WriteFile(repoSkill, []byte("checkout"), 0644); err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(root, "skills", "flux")
			if link == "folder" {
				if err := os.MkdirAll(filepath.Dir(dir), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(checkout, dir); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(repoSkill, filepath.Join(dir, "SKILL.md")); err != nil {
					t.Fatal(err)
				}
			}
			_, _, err := installSkill(root, []byte("release"))
			if err == nil || !strings.Contains(err.Error(), "is a symlink to") {
				t.Fatalf("expected a symlink error, got %v", err)
			}
			entries, _ := os.ReadDir(checkout)
			if got, _ := os.ReadFile(repoSkill); string(got) != "checkout" || len(entries) != 1 {
				t.Fatalf("checkout was touched: %q, %d entries", got, len(entries))
			}
		})
	}
}
