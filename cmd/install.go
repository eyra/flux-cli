package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/eyra/flux-cli/skill"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the flux skill for an AI coding agent",
	Long: `Install the flux agent skill, embedded in this binary, where an agent reads
its personal skills:

  claude  ~/.claude/skills/flux      (CLAUDE_CONFIG_DIR replaces ~/.claude)
  omp     ~/.omp/agent/skills/flux   (PI_CODING_AGENT_DIR replaces ~/.omp/agent)
  codex   ~/.agents/skills/flux

omp also reads ~/.agents/skills, so "flux install codex" makes the skill
available to omp too.

"flux update" refreshes the skills you installed; rerun after "brew upgrade" or
"go install". A SKILL.md that differs from this version is first backed up to
SKILL.md.bak-<time>.`,
}

// agent is a coding agent that reads personal skills from <root>/skills, where
// root is $envVar or else ~/defaultRoot.
type agent struct {
	name, envVar, defaultRoot, short, long string
}

var agents = []agent{
	{"claude", "CLAUDE_CONFIG_DIR", ".claude", "Install the flux skill for Claude Code",
		`Install the flux skill into ~/.claude/skills/flux, where Claude Code reads
personal skills. CLAUDE_CONFIG_DIR replaces ~/.claude.`},
	{"omp", "PI_CODING_AGENT_DIR", filepath.Join(".omp", "agent"), "Install the flux skill for omp (oh-my-pi)",
		`Install the flux skill into ~/.omp/agent/skills/flux, omp's own skills
folder, which takes priority over every other skill location.
PI_CODING_AGENT_DIR replaces ~/.omp/agent.

"flux install codex" already makes the skill available to omp, through
~/.agents/skills; use this command to give omp its own copy.`},
	{"codex", "", ".agents", "Install the flux skill for Codex",
		`Install the flux skill into ~/.agents/skills/flux, where Codex reads
personal skills. omp and other agents that read ~/.agents/skills pick it up
too.`},
}

func (a agent) root() (string, error) {
	if a.envVar != "" {
		if root := os.Getenv(a.envVar); root != "" {
			return root, nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, a.defaultRoot), nil
}

// skillPaths returns the skill folder and SKILL.md under root.
func skillPaths(root string) (dir, path string) {
	dir = filepath.Join(root, "skills", "flux")
	return dir, filepath.Join(dir, "SKILL.md")
}

// skillSymlink returns the first of dir and path that is a symlink, or "".
func skillSymlink(dir, path string) string {
	for _, p := range []string{dir, path} {
		if info, err := os.Lstat(p); err == nil && info.Mode()&fs.ModeSymlink != 0 {
			return p
		}
	}
	return ""
}

// installTarget returns the command that installs the skill for a.
func installTarget(a agent) *cobra.Command {
	return &cobra.Command{
		Use:          a.name,
		Short:        a.short,
		Long:         a.long,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := a.root()
			if err != nil {
				return err
			}
			path, backup, err := installSkill(root, skill.Markdown)
			if err != nil {
				return err
			}
			if jsonFlag {
				fields := []string{"path", path}
				if backup != "" {
					fields = append(fields, "backup", backup)
				}
				printOK(fields...)
				return nil
			}
			out := cmd.OutOrStdout()
			if backup != "" {
				fmt.Fprintf(out, "Backed up previous skill to %s\n", backup)
			}
			fmt.Fprintf(out, "Installed flux skill to %s\n", path)
			return nil
		},
	}
}

// installSkill writes data to <root>/skills/flux/SKILL.md. An identical file is
// left alone; a different one is renamed to SKILL.md.bak-<time> first. A
// symlinked skill folder or file is refused, so a link to a checkout of this
// repository is never overwritten.
func installSkill(root string, data []byte) (path, backup string, err error) {
	dir, path := skillPaths(root)
	if link := skillSymlink(dir, path); link != "" {
		target, _ := os.Readlink(link)
		return "", "", fmt.Errorf("%s is a symlink to %s; remove it to install", link, target)
	}
	existing, err := os.ReadFile(path)
	switch {
	case err == nil && bytes.Equal(existing, data):
		return path, "", nil
	case err == nil:
		backup = path + ".bak-" + time.Now().Format("20060102-150405")
		if err := os.Rename(path, backup); err != nil {
			return "", "", err
		}
	case !errors.Is(err, fs.ErrNotExist):
		return "", "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", "", err
	}
	return path, backup, nil
}

func init() {
	for _, a := range agents {
		installCmd.AddCommand(installTarget(a))
	}
	rootCmd.AddCommand(installCmd)
}
