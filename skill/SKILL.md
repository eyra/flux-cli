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

CLI for managing issues, epics, milestones, scenes, use cases, and AppSignal incidents in Flux.

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
| List issues | `flux issues list --json [--stage testing] [--app web]` |
| Get issue | `flux issues get <id> --json` |
| Create issue | `flux issues create --title "..." --app web [--stage specification] [--program dev] [--size M] --json` |
| Update issue | `flux issues update <id> --title "..." [--app ios] --json` |
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
| List milestones | `flux milestones list --json [--app ios]` |
| Get milestone | `flux milestones get <id> --json` |
| Resync milestone linked issues | `flux milestones resync <id> --json` |
| List personas | `flux personas list --json` |
| Upload image | `flux images upload --file <path> [--caption "..."] --json` |
| Render diagram | `flux diagrams render --file <path.mmd> --json` |
| Render diagram (inline) | `flux diagrams render --mermaid "graph TD; A-->B" --json` |
| List scenes | `flux scenes list --json [--completed]` |
| Get scene | `flux scenes get <id-or-code> --json` |
| Create scene | `flux scenes create --title "..." --area Next [--status ...] --json` |
| Use cases of a scene | `flux scenes usecases <id-or-code> --json` |
| List use cases | `flux usecases list --json [--scene <id-or-code>]` |
| Get use case | `flux usecases get <id-or-code> --json` |
| Create use case | `flux usecases create --title "..." --area NEXT [--scene <id-or-code>] --json` |
| Link use case to scene | `flux usecases link <id-or-code> --target-type scene --target-id <scene> --json` |
| Unlink use case | `flux usecases unlink <id-or-code> --target-type scene --target-id <scene> --json` |
| Issues of a use case | `flux usecases issues <id-or-code> --json` |
| Link issue to use case | `flux issues link <id> --target-type usecase --target-id <use-case> --json` |
| Resync linked titles | `flux scenes resync <id-or-code> --json` / `flux usecases resync <id-or-code> --json` |
| AppSignal apps | `flux appsignal apps --json` |
| AppSignal incidents | `flux appsignal incidents list --app <app> --json` |

## Scenes and Use Cases

Scene → Use Case → Issue, from the project's Product to-do set. Each child has
at most one parent; linking to another parent moves it. Scenes and use cases
take Basecamp IDs or codes (`SCN-Next-02`, legacy `UJ-Next-02`, `UC-NEXT-01`).
`create`, `update` and `comment` take `--ai-model`; `update --status none`
moves an item out of its group. Use cases are completed by hand in Basecamp.
A project without Product configured fails with `product_not_configured`; an
older server fails with "not supported by this server".

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
  --app web \
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

`auth status` verifies the active credentials through the selected server's
`GET /api/dev/identity` endpoint. Credential precedence is `--api-key`, then
`FLUX_API_KEY`, then saved personal credentials for the selected environment.
Missing, invalid, or unverifiable authentication exits nonzero without success
JSON; a local credential file alone is not proof of authentication.

An explicit `--env` overrides `FLUX_ENV`, even `--env prod` when `FLUX_ENV=test`.
Otherwise `FLUX_ENV` applies, then the default `prod`.

## Comment Formatting

The server processes all comment content through the same pipeline as the MCP tools:

1. **Plain text** is automatically converted — newlines become `<br>`, blank lines become paragraph breaks, and common Markdown syntax is converted to HTML:
   - `**bold**` → `<strong>bold</strong>`
   - `_italic_` → `<em>italic</em>`
   - Lines starting with `- ` or `* ` become `<ul><li>` bullet lists
2. **HTML** — if your content contains `<`, it is sent as-is. Use only tags Trix supports: `<strong>`, `<em>`, `<s>`, `<a href>`, `<ul>/<li>`, `<ol>/<li>`, `<blockquote>`, `<pre>`, `<br>`, `<p>`.
3. **@mentions** — write `@Name` or `@First Last` anywhere in the content. The server looks up the person in the project and converts to a proper Basecamp mention (notifies them). Works in both plain text and HTML content.

**Recommended:** write plain text with Markdown syntax and let the server handle the conversion.

## AI Model Attribution

Pass `--ai-model "<model>"` only when declaring the model that generated the
description or comment you are supplying. It is optional, caller-declared display
metadata, not verified identity. Never infer it from persona attribution,
credentials, or environment. It does not invoke/select a model or change
authentication, environment, or project selection.

Supported content writes:

- `flux issues create` / `update` with `--description`
- `flux epics create` / `update` with `--description`
- `flux milestones create` / `update` with `--description`
- `flux issues comment`, `flux epics comment`, `flux milestones comment`
- `flux comments update`
- `flux issues advance` for its optional `--comment` only, not its automatic stage comment

```bash
flux issues comment <id> --content "Investigated the failure." --persona sam --ai-model "openai/gpt-5" --json
flux comments update <comment_id> --content "Revised findings." --ai-model "anthropic/claude-sonnet-4" --json
```

The flag is local to these commands, not available on reads, deletes, links,
assignments, resyncs, or other operations. The exact string is forwarded as JSON
`ai_model`; provider/model IDs and HTML-looking strings are not interpreted by
the CLI. Omission leaves the JSON field absent.

The server adds one code-block footer, with only Flux and the model name bold and no italics:

- Absent, empty, or whitespace-only model: `<pre>Assisted by <strong>Flux</strong></pre>`
- Model `openai/gpt-5`: `<pre>Assisted by <strong>Flux</strong> · Generated with <strong>openai/gpt-5</strong></pre>`

Model text is HTML-escaped by the server and never expanded into `@mentions`.
When editing content, the existing generated footer is replaced, not duplicated;
omitting the model on a content edit removes the old model suffix. Independent
AI-disclosure text remains. A metadata-only update without `--description` leaves
the historical description and model footer untouched even when `--ai-model` is
given. An advance without `--comment` does not attribute any user text.

## JSON Output

All commands support `--json`. Reads return the server's full resource object,
with every field it sends (issues include `ref`, `epic`, `milestone`, `use_case`
and `url`). Scene and use case writes, links and resyncs return the server's
response. Other mutations return:

```json
{"ok": "true", "id": "<id>"}
```

Use `--json` output to chain commands: extract the `id` field from create responses to use in subsequent link or advance calls.

`flux auth status --json` returns only verified identity and environment fields:

```json
{"signed_in":true,"env":"prod","basecamp_account_id":"123","basecamp_person_id":"456","display_name":"Alex"}
```

Account and person IDs are strings. For API keys the identity is the actual bot
principal, not a persona. No provider credentials or tokens are included.

`flux issues get <id> --project <key> --json` scopes the issue lookup to that
project; issues outside it return an error. Without `--project`, the default is
`next` on prod and `flux` on test. Each `thread` comment preserves `author` and adds
`author_id`, the Basecamp creator's string person ID. Missing creator IDs are
omitted or `null` as the server sends them, never inferred from a name:

```json
{"id":"789","author":"Alex","author_id":"456","date":"2026-10-06","content":"Comment text"}
```

Names can be shared. Within the same Basecamp account, compare a comment's
`author_id` to the verified `basecamp_person_id` to determine whether it belongs
to the signed-in principal; do not compare display names or persona labels.

## Error Handling

| Error | Fix |
|-------|-----|
| `unauthorized: run 'flux auth login'` | Run `flux auth login [--env test]` |
| `not found` | Verify the ID exists for the current env/project |
| `product_not_configured` | The project has no scenes or use cases; pick another `--project` |
| `not supported by this server` | The server is older than the CLI; wait for the server release |
| Non-zero exit | Check stderr for the error message |

Exit codes: 0 = success, non-zero = error (check stderr).
