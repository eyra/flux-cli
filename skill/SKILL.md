---
name: flux
description: |
  Manage Flux project issues, epics, milestones, and AppSignal incidents via the Flux CLI.
  Use for ANY question or action about Flux issues, epics, milestones, personas, or incidents.
triggers:
  - flux issue
  - flux epic
  - flux milestone
  - flux issues
  - flux epics
  - flux milestones
  - /flux
  - list issues
  - create issue
  - update issue
  - advance issue
  - link issue
  - list epics
  - create epic
  - list milestones
  - create milestone
  - flux incident
  - appsignal incident
invocable: true
argument-hint: "[action] [args...]"
---

# /flux - Flux Project Management

CLI for managing issues, epics, milestones, and AppSignal incidents in Flux.

## Agent Invariants

**MUST follow these rules:**

1. **Always use `--json`** for data extraction and confirmation of mutations. Only omit it when presenting results directly to a human in prose.
2. **Two environments** — `prod` (default) manages the Next Platform project; `--env test` manages the Flux Platform project itself. When the user asks about Flux's own issues/epics, always add `--env test`.
3. **Two projects** — `--project flux` for Flux Platform issues; `--project next` for Next Platform issues. Default is `next` on prod, `flux` on `--env test`. Pass `--project` explicitly when it differs from the default.
4. **Check auth first** — if a command fails with "unauthorized", run `flux auth login [--env test]` and retry.
5. **Stage emojis belong in titles** — when advancing beyond Specification, the title must include the stage emoji at the end: ✏️ Design, 💻 Development, 🧪 Testing, ✅ Done. Use `--title` on the advance or update command to set it.
6. **IDs are Basecamp recording IDs** — long integers like `9958752901`. Always pass the exact ID.
7. **Persona attribution** — use `--persona <name>` on create/update/comment/advance commands when acting on behalf of an AI persona (e.g. `--persona sam`).

## Quick Reference

| Task | Command |
|------|---------|
| Sign in | `flux auth login [--env test]` |
| Auth status | `flux auth status [--env test] --json` |
| List projects | `flux projects list --json` |
| List people | `flux people list --json` |
| List issues | `flux issues list --json [--stage testing]` |
| Get issue | `flux issues get <id> --json` |
| Create issue | `flux issues create --title "..." [--stage specification] [--program dev] [--size M] --json` |
| Update issue | `flux issues update <id> --title "..." --json` |
| Advance issue | `flux issues advance <id> --stage testing --comment "..." --json` |
| Assign issue | `flux issues assign <id> --assignees <person_id,...> --json` |
| Link issue to epic | `flux issues link <id> --target-type epic --target-id <epic_id> --json` |
| Add comment | `flux issues comment <id> --content "..." --json` |
| Update comment | `flux comments update <comment_id> --content "..." --json` |
| Delete comment | `flux comments delete <comment_id> --json` |
| Delete issue | `flux issues delete <id> --json` |
| List epics | `flux epics list --json` |
| Get epic | `flux epics get <id> --json` |
| Create epic | `flux epics create --title "..." --json` |
| List epic issues | `flux epics issues <id> --json` |
| Resync epic linked issues | `flux epics resync <id> --json` |
| List milestones | `flux milestones list --json` |
| Get milestone | `flux milestones get <id> --json` |
| Resync milestone linked issues | `flux milestones resync <id> --json` |
| List personas | `flux personas list --json` |
| Upload image | `flux images upload --file <path> [--caption "..."] --json` |
| Render diagram | `flux diagrams render --file <path.mmd> --json` |
| Render diagram (inline) | `flux diagrams render --mermaid "graph TD; A-->B" --json` |
| AppSignal apps | `flux appsignal apps --json` |
| AppSignal incidents | `flux appsignal incidents list --app <app> --json` |

## Environment & Project Selection

```
Flux Platform issues (our own backlog):
  flux issues list --env test --json        (flux is default project on test)
  flux issues list --project flux --json    (explicit, works on prod too)

Next Platform issues (what Flux manages):
  flux issues list --json                   (next is default project on prod)
```

## Issue Stages

Stages in order: `triage` → `specification` → `design` → `development` → `testing` → (done)

Stage emojis (add to title when advancing beyond specification):
- Design: ✏️
- Development: 💻
- Testing: 🧪
- Done: ✅ (also mark complete in Basecamp)

```bash
# Advance to development (add emoji to title)
flux issues advance <id> --stage development --comment "Starting implementation" --json
flux issues update <id> --title "[Dev] Fix the thing 💻" --json
```

## Common Workflows

### Create and link an issue to an epic

```bash
# Create issue in Development stage
flux issues create \
  --title "[Dev] Implement feature X 💻" \
  --stage development \
  --program dev \
  --size M \
  --json

# Link to epic (use ID from create response)
flux issues link <issue_id> --target-type epic --target-id <epic_id> --json
```

### Advance an issue through stages

```bash
# Specification → Design
flux issues advance <id> --stage design --json
flux issues update <id> --title "[Web] Fix the thing ✏️" --json

# Design → Development
flux issues advance <id> --stage development --comment "Design approved" --json
flux issues update <id> --title "[Web] Fix the thing 💻" --json

# Development → Testing
flux issues advance <id> --stage testing --comment "PR #42 merged" --json
flux issues update <id> --title "[Web] Fix the thing 🧪" --json
```

### Add a comment with persona attribution

```bash
flux issues comment <id> \
  --content "Investigated root cause: the session store is evicting tokens too early." \
  --persona sam \
  --json
```

### Check AppSignal incidents

```bash
# List available apps
flux appsignal apps --json

# List open incidents
flux appsignal incidents list --app <app_name> --state open --json

# Get incident details
flux appsignal incidents get --app <app_name> --number <N> --json
```

## Auth

```bash
flux auth login              # Sign in to prod (Next project)
flux auth login --env test   # Sign in to test (Flux project)
flux auth logout             # Sign out
flux auth status --json      # Check status
```

Credentials stored in `~/.config/flux/credentials.json`, one entry per environment. The old `FLUX_API_KEY` env var and `--api-key` flag still work for CI/CD.

## Comment Formatting

The server processes all comment content through the same pipeline as the MCP tools:

1. **Plain text** is automatically converted — newlines become `<br>`, blank lines become paragraph breaks, and common Markdown syntax is converted to HTML:
   - `**bold**` → `<strong>bold</strong>`
   - `_italic_` → `<em>italic</em>`
   - Lines starting with `- ` or `* ` become `<ul><li>` bullet lists
2. **HTML** — if your content contains `<`, it is sent as-is. Use only tags Trix supports: `<strong>`, `<em>`, `<s>`, `<a href>`, `<ul>/<li>`, `<ol>/<li>`, `<blockquote>`, `<pre>`, `<br>`, `<p>`.
3. **@mentions** — write `@Name` or `@First Last` anywhere in the content. The server looks up the person in the project and converts to a proper Basecamp mention (notifies them). Works in both plain text and HTML content.

**Recommended:** write plain text with Markdown syntax and let the server handle the conversion.

## JSON Output

All commands support `--json`. Reads return the full resource object. Mutations return:

```json
{"ok": "true", "id": "<id>"}
```

Use `--json` output to chain commands: extract the `id` field from create responses to use in subsequent link or advance calls.

## Error Handling

| Error | Fix |
|-------|-----|
| `unauthorized: run 'flux auth login'` | Run `flux auth login [--env test]` |
| `not found` | Verify the ID exists for the current env/project |
| Non-zero exit | Check stderr for the error message |

Exit codes: 0 = success, non-zero = error (check stderr).
