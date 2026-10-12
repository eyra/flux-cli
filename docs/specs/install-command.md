# Spec: `flux install claude` / `omp` / `codex`

- **Status:** implemented. Released in `v0.5.0` (#19); `install.sh` switched
  over in the follow-up PR.
- **Issue:** [Install the Flux skill with flux install claude, omp or codex](https://app.basecamp.com/5734045/buckets/45557517/todos/10394794625)
- **Date:** 2026-10-12
- **Base:** `eyra/flux-cli` `master` at `2aac1bb`; the latest release is `v0.4.0`
- **Checked against:**
  - omp 18.8.6: its bundled docs `skills.md`, `settings.md` and `context-files.md`
  - Claude Code 2.1.281: <https://docs.claude.com/en/docs/claude-code/skills>
  - Codex: <https://developers.openai.com/codex/skills>, and
    `openai/codex` `codex-rs/skills/src/parser.rs` at `d24507a`
  - GoReleaser v2.18 `homebrew_casks` docs, for `caveats`
  - `eyra/skills` at `db32d79`: `internal/installer/install.go`

## Problem

`brew install eyra/tap/flux-cli` and `go install` only install the binary. The
agent skill (`skill/SKILL.md`) is only installed by `install.sh`, which copies
it to `~/.claude/skills/flux`. People who install with brew or Go get no skill,
and an installed skill goes stale when the binary is upgraded: a skill installed
in September still documents personas, which #16 removed.

## Commands

```bash
flux install claude   # Claude Code
flux install omp      # oh-my-pi (omp)
flux install codex    # OpenAI Codex
```

| Command               | Writes                         | Default root   | Root override         |
| --------------------- | ------------------------------ | -------------- | --------------------- |
| `flux install claude` | `<root>/skills/flux/SKILL.md`  | `~/.claude`    | `CLAUDE_CONFIG_DIR`   |
| `flux install omp`    | `<root>/skills/flux/SKILL.md`  | `~/.omp/agent` | `PI_CODING_AGENT_DIR` |
| `flux install codex`  | `<root>/skills/flux/SKILL.md`  | `~/.agents`    | none                  |

The override variables are the ones Claude Code and omp read themselves, so the
skill lands where the tool looks. Codex has no override: its user skill
location is fixed at `$HOME/.agents/skills`
([Codex docs](https://developers.openai.com/codex/skills)).

### Why omp needs its own target

omp can read `~/.claude/skills`, but only when the user lists `claude` in
`enabledProviders` (or sets `CLAUDE_CONFIG_DIR`); that is off by default. Without
it, omp loads user skills only from its native `~/.omp/agent/skills` and from
`~/.agents/skills`, which Claude Code does not read. So one location does not
cover both tools.

`flux install omp` uses the native directory: it has omp's highest skill
priority and does not share `~/.agents/skills`, which the `eyra-skills`
installer manages. If a user has both copies and has enabled the `claude`
provider, omp silently drops a byte-identical duplicate. If the copies
differ, omp keeps the native skill as `flux` and loads the other under a
namespaced name (`<namespace>/flux`), with a collision warning. Running both
commands after an upgrade keeps them identical.

### Codex

Codex reads user skills only from `~/.agents/skills` (plus repository
`.agents/skills` folders and `/etc/codex/skills`), not from `~/.codex`. It
requires `name` and `description` in the frontmatter, which `SKILL.md` already
has. Codex picks up new or changed skills without a restart.

`~/.agents/skills` is the folder where the `eyra-skills` installer puts the
`eyra-*` skills. That installer replaces only its own skill folders, so it
leaves `flux` alone. omp also reads this folder: with both `omp` and `codex`
installed, the duplicate rules above apply, and the native omp copy wins.

## Behaviour

- The skill is embedded in the binary (`go:embed`), so the installed skill
  always matches the binary that installed it. The command does not download
  anything, and it needs no authentication: it ignores `--env`, `--project` and
  `--api-key`.
- It creates missing directories with mode 0755 and writes `SKILL.md` with mode
  0644.
- If an existing `SKILL.md` is byte-identical, nothing is written.
- If an existing `SKILL.md` differs, it is first renamed to
  `SKILL.md.bak-YYYYMMDD-HHMMSS` beside it (as `install.sh` does now), so local
  edits are not lost.
- Output: `Installed flux skill to <path>` (plus `Backed up previous skill to
  <path>` when a backup was made). With `--json`, `printOK("path", …)`, adding
  `"backup"` when a backup was made.
- If `<root>/skills/flux` or its `SKILL.md` is a symlink, nothing is written
  and the command exits nonzero with
  `<path> is a symlink to <target>; remove it to install`. Otherwise a
  developer's link to a `flux-cli/skill` checkout would be overwritten.
- If a write fails, the command exits nonzero with the OS error.
- `flux install` without a target shows help listing `claude`, `omp` and
  `codex`.
- Both commands stay. The help text says how they overlap, because omp also
  reads `~/.agents/skills`:
  - `flux install --help` lists the three targets and their paths, and says
    that `codex` also makes the skill available to omp.
  - `flux install omp --help` says it installs into omp's own skills folder,
    which takes priority, and that `flux install codex` already works for omp.
  - `flux install codex --help` says it also covers omp and other agents that
    read `~/.agents/skills`.
  - The README "Agent skill" subsection repeats this in one sentence.

## Implementation

- `skill/skill.go`: `package skill` with `//go:embed SKILL.md` into
  `var Markdown []byte`. `go:embed` cannot reach `../skill` from `cmd/`, so
  the embed lives beside the file.
- `cmd/install.go`: an `install` parent command. `installTarget` builds the
  `claude`, `omp` and `codex` subcommands, which resolve their root
  (environment variable where there is one, else `os.UserHomeDir()` plus the
  default) and call one `installSkill(root, data)` function.
- `cmd/install_test.go`:
  - each target, with and without its override variable, writes exactly
    `skill/SKILL.md` to the expected path (this also proves the embed)
  - an identical reinstall makes no backup; a locally edited skill is backed
    up byte-for-byte, then replaced
  - a symlinked folder or file is refused, and the link target is left
    untouched

## Switching over

`install.sh` is served from `master` but downloads the latest *release*
(`raw.githubusercontent.com/eyra/flux-cli/master/install.sh`). If it calls
`flux install claude` before a release has that command, every
`curl … | bash` install fails. CI does not catch this: `install-check.yml`
tests a locally built archive. So this ships in two steps.

1. First PR, followed by a release (`v0.5.0`). **Done: #19, released as
   `v0.5.0` on 2026-10-12; the cask in `eyra/homebrew-tap` has the caveats.**
   - add the command, `skill/skill.go`, the tests and the README "Agent skill"
     subsection
   - add cask `caveats` telling users to run `flux install` with `claude`,
     `omp` and/or `codex`, and to rerun it after `brew upgrade`
   - leave `install.sh` and the archive `files` unchanged
2. Second PR, after `v0.5.0` is released. **Done.**
   - `install.sh`: replace the copy in `install_skill` with
     `"$INSTALL_DIR/flux" install claude`, and drop `SKILL_DIR`
     (`CLAUDE_CONFIG_DIR` replaces it). If the skill install fails (for
     example on a symlink), `install.sh` prints a warning and keeps the
     installed binary.
   - `.goreleaser.yml`: replace `skill/SKILL.md` in the archive `files` with
     `none*`, so archives hold only the binary (without a `files` list,
     GoReleaser would add `README.md`). This must happen in the same PR as the
     `install.sh` change, because `install.sh` used to read the skill from the
     archive.
   - `scripts/test-install.sh`: drop the "archive contains skill/SKILL.md"
     check, and keep the install, no-backup and backup checks, now run through
     `flux install claude`. Unset `CLAUDE_CONFIG_DIR` instead of `SKILL_DIR`,
     so a developer's environment can't leak into the sandbox.

## Out of scope

- The `eyra-*` workflow skills: `eyra/skills` installs them into
  `~/.agents/skills` with its own `eyra-skills` installer.
- omp profiles (`~/.omp/profiles/<name>/agent`): use `PI_CODING_AGENT_DIR`.
- Project-level installs (`.claude/skills`, `.agents/skills`), uninstalling,
  and other agents (OpenCode, Gemini and others).
- Refreshing the skill automatically on upgrade: brew casks should not write to
  `$HOME`. Add a version check when stale skills cause real trouble.

## Decisions

- Keep both `flux install omp` and `flux install codex`, even though `codex`
  alone also reaches omp. The help text explains the overlap.

## Findings (2026-10-12)

The open questions from planning are resolved. Each check below ran against
the binary built from this branch.

1. **Symlinked skill folders.** The proposed rule is implemented and tested.
2. **Claude Code reads `CLAUDE_CONFIG_DIR/skills`.** Checked with Claude Code
   2.1.281 without logging in: the `system`/`init` event of
   `claude -p --output-format stream-json --verbose` lists the skills before
   any API call. With `CLAUDE_CONFIG_DIR` set to a folder that has the skill,
   `flux` is listed; with an empty folder it is not, even though
   `~/.claude/skills/flux` exists. So the override fully replaces `~/.claude`.
3. **Codex loads the skill.** Checked with `@openai/codex@0.162.1` under a
   temporary `HOME`, without logging in: `codex app-server` answers the
   `skills/list` request with `flux`, scope `user`, `enabled: true`, from
   `~/.agents/skills/flux/SKILL.md`. The multi-line `description` arrives as
   one line, and `triggers:` is ignored.
4. **omp loads one copy.** With identical copies in `~/.omp/agent/skills` and
   `~/.agents/skills`, `omp -p` (18.8.6) lists `flux` once, with no
   namespaced duplicate.
5. **The stale skill problem is real.** Running `flux install claude` on the
   machine this spec was written on backed up the September skill (which still
   documents personas) to `SKILL.md.bak-20261012-103300` and replaced it.
6. **`goreleaser check` (v2.18.3) accepts the cask `caveats`.**
7. **Not fixed here: errors are printed twice.** The whole CLI prints errors
   as `Error: <msg>` and then `<msg>` again: cobra prints the error, and
   `Execute` in `cmd/root.go` prints it once more. This is already tracked as
   [flux CLI prints every error twice](https://app.basecamp.com/5734045/buckets/45557517/todos/10390432942).
8. **The published `v0.5.0` works with the new `install.sh`.** Under a
   temporary `HOME`, the step 2 `install.sh` downloaded `v0.5.0` and installed
   the binary and a skill identical to `skill/SKILL.md`. With a symlinked
   `~/.claude/skills/flux`, it printed the warning, left the link target
   empty, kept the binary and exited 0.
9. **Related:** [flux update should also update the Flux skill](https://app.basecamp.com/5734045/buckets/45557517/todos/10385129256)
   (Specification) can now call the same `installSkill` with the embedded
   skill.
