package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRefreshSkills(t *testing.T) {
	const hint = "No flux skill installed; run flux install claude, omp or codex\n"
	for _, tc := range []struct {
		name  string
		state map[string]string // agent → skill, folder, linkdir or linkfile
		fail  string            // agent whose install exits 1
		calls string
		out   string // {claude}, {omp} and {codex} stand for the agent roots
		warn  string
	}{
		{name: "nothing installed", out: hint},
		{name: "claude only", state: map[string]string{"claude": "skill"},
			calls: "install claude\n"},
		{name: "all three", state: map[string]string{"claude": "skill", "omp": "skill", "codex": "skill"},
			calls: "install claude\ninstall omp\ninstall codex\n"},
		{name: "folder without SKILL.md", state: map[string]string{"claude": "folder"}, out: hint},
		{name: "symlinked folder", state: map[string]string{"claude": "skill", "omp": "linkdir"},
			calls: "install claude\n", out: "Skipped omp: {omp}/skills/flux is a symlink\n"},
		{name: "symlinked SKILL.md only", state: map[string]string{"codex": "linkfile"},
			out: "Skipped codex: {codex}/skills/flux/SKILL.md is a symlink\n"},
		{name: "one install fails", state: map[string]string{"claude": "skill", "omp": "skill", "codex": "skill"},
			fail: "omp", calls: "install claude\ninstall omp\ninstall codex\n",
			warn: "Warning: the flux skill for omp was not updated (exit status 1)\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolateCommand(t)
			bin := t.TempDir()
			log := filepath.Join(bin, "calls")
			flux := filepath.Join(bin, "flux")
			script := "#!/bin/sh\necho \"$*\" >> \"$FAKE_LOG\"\n[ \"$2\" != \"$FAKE_FAIL\" ]\n"
			if err := os.WriteFile(flux, []byte(script), 0755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("FAKE_LOG", log)
			t.Setenv("FAKE_FAIL", tc.fail)
			t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
			t.Setenv("PI_CODING_AGENT_DIR", t.TempDir())

			roots := []string{}
			for _, a := range agents {
				root, err := a.root()
				if err != nil {
					t.Fatal(err)
				}
				roots = append(roots, "{"+a.name+"}", root)
				dir, path := skillPaths(root)
				switch tc.state[a.name] {
				case "skill":
					mustWrite(t, path)
				case "folder":
					mkdir(t, dir)
				case "linkdir":
					checkout := t.TempDir()
					mustWrite(t, filepath.Join(checkout, "SKILL.md"))
					mkdir(t, filepath.Dir(dir))
					symlink(t, checkout, dir)
				case "linkfile":
					target := filepath.Join(t.TempDir(), "SKILL.md")
					mustWrite(t, target)
					mkdir(t, dir)
					symlink(t, target, path)
				}
			}

			var stdout, stderr bytes.Buffer
			refreshSkills(&stdout, &stderr, flux)

			calls, _ := os.ReadFile(log)
			if string(calls) != tc.calls {
				t.Errorf("calls %q, want %q", calls, tc.calls)
			}
			if want := strings.NewReplacer(roots...).Replace(tc.out); stdout.String() != want {
				t.Errorf("stdout %q, want %q", stdout.String(), want)
			}
			if stderr.String() != tc.warn {
				t.Errorf("stderr %q, want %q", stderr.String(), tc.warn)
			}
		})
	}
}

func mustWrite(t *testing.T, path string) {
	t.Helper()
	mkdir(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte("skill"), 0644); err != nil {
		t.Fatal(err)
	}
}

func mkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
}

func symlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
}
