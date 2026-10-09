---
name: flux
description: |
  Manage Flux project issues, epics, milestones, scenes, use cases, and AppSignal incidents via the Flux CLI.
  Use for ANY question or action about Flux issues, epics, milestones, scenes, use cases, or incidents.
triggers:
  - flux issue
  - flux epic
  - flux milestone
  - flux issues
  - flux epics
  - flux milestones
  - flux scene
  - flux scenes
  - flux use case
  - flux usecases
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
  - list scenes
  - create scene
  - list use cases
  - create use case
  - link use case
  - flux incident
  - appsignal incident
invocable: true
argument-hint: "[action] [args...]"
---

# /flux - Flux Project Management

CLI for managing issues, epics, milestones, scenes, use cases, and AppSignal incidents in Flux.

## Agent Invariants

**MUST follow these rules:**

1. **Use the Flux CLI only** — never invoke Flux MCP tools or restore an MCP connection. Always pass `--env prod --project <key> --json` for operational commands. Only omit `--json` when presenting results directly to a human in prose.
2. **Production is the source of truth** — production contains the Flux, Next, and Feldspar backlogs. Use `--env test` only for explicitly requested verification against the test deployment, never to manage Flux's real backlog.
3. **Select the project explicitly** — use `flux` for Flux Platform, `next` for Next Platform, and `feldspar` for Feldspar. Production also has `devops`, `scriptdev`, and `website`. Resolve the key from the request or project configuration; do not rely on CLI defaults. Discover available projects with `flux projects list --env prod --project <key> --json` using a known key.
4. **Check auth first** — if a command fails with "unauthorized", run `flux auth login --env prod --project <key> --json` and retry. For explicit test verification, authenticate to `--env test` instead.
5. **Stage emojis belong in titles** — when advancing beyond Specification, the title must include the stage emoji at the end: ✏️ Design, 💻 Development, 🧪 Testing, ✅ Done. Use `issues update --title`; `issues advance` does not accept `--title`.
6. **IDs are Basecamp recording IDs** — long integers like `9958752901`. Always pass the exact ID. Scenes and use cases also take their code (`SC-Next-02`, `UC-NEXT-01`).
7. **Model attribution** — content-writing commands support `--ai-model <model>` for a caller-declared footer. Preserve it when supplied; do not invent a model identity. Flux has no personas: content is written as the signed-in caller, so don't pass `--persona` (it is ignored).

## Quick Reference

Replace `<key>` with the resolved production project key in every example. Scenes and use cases exist only in projects with a Product to-do set (today `next`).

| Task | Command |
|------|---------|
| Sign in | `flux auth login --env prod --project <key> --json` |
| Auth status | `flux auth status --env prod --project <key> --json` |
| List projects | `flux projects list --env prod --project <key> --json` |
| List people | `flux people list --env prod --project <key> --json` |
| List issues | `flux issues list --env prod --project <key> --json [--stage testing] [--context dev]` |
| Get issue | `flux issues get <id> --env prod --project <key> --json` |
| Create issue | `flux issues create --title "..." --app web [--stage specification] [--size M] [--context Dev] [--epic <id>] [--milestone <id>] [--usecase <id-or-code>] --env prod --project <key> --json` |
| Update issue | `flux issues update <id> [--title "..."] [--context Dev] [--epic <id>] [--milestone <id>] [--usecase <id-or-code>] --env prod --project <key> --json` |
| Advance issue | `flux issues advance <id> --stage testing --comment "..." --env prod --project <key> --json` |
| Assign issue | `flux issues assign <id> --assignees <person_id,...> --env prod --project <key> --json` |
| Link issue to epic | `flux issues link <id> --target-type epic --target-id <epic_id> --env prod --project <key> --json` |
| Link issue to use case | `flux issues link <id> --target-type usecase --target-id <use-case> --env prod --project next --json` |
| Add comment | `flux issues comment <id> --content "..." --env prod --project <key> --json` |
| Update comment | `flux comments update <comment_id> --content "..." --env prod --project <key> --json` |
| Delete comment | `flux comments delete <comment_id> --env prod --project <key> --json` |
| Delete issue | `flux issues delete <id> --env prod --project <key> --json` |
| List epics | `flux epics list --env prod --project <key> --json` |
| Get epic | `flux epics get <id> --env prod --project <key> --json` |
| Create epic | `flux epics create --title "..." --env prod --project <key> --json` |
| List epic issues | `flux epics issues <id> --env prod --project <key> --json` |
| Resync epic linked issues | `flux epics resync <id> --env prod --project <key> --json` |
| List milestones | `flux milestones list --env prod --project <key> --json` |
| Get milestone | `flux milestones get <id> --env prod --project <key> --json` |
| Resync milestone linked issues | `flux milestones resync <id> --env prod --project <key> --json` |
| List scenes | `flux scenes list [--completed] --env prod --project next --json` |
| Get scene | `flux scenes get <id-or-code> [--no-thread] --env prod --project next --json` |
| Create scene | `flux scenes create --title "..." --area Next [--status ...] --env prod --project next --json` |
| Update scene | `flux scenes update <id-or-code> [--title ...] [--code ...] [--status ...] --env prod --project next --json` |
| Next free scene code | `flux scenes next-code --area Next --env prod --project next --json` |
| Comment on scene | `flux scenes comment <id-or-code> --content "..." --env prod --project next --json` |
| Use cases of a scene | `flux scenes usecases <id-or-code> --env prod --project next --json` |
| Resync scene linked use cases | `flux scenes resync <id-or-code> --env prod --project next --json` |
| List use cases | `flux usecases list [--scene <id-or-code>] [--completed] --env prod --project next --json` |
| Get use case | `flux usecases get <id-or-code> [--no-thread] --env prod --project next --json` |
| Create use case | `flux usecases create --title "..." --area NEXT [--scene <id-or-code>] --env prod --project next --json` |
| Update use case | `flux usecases update <id-or-code> [--title ...] [--code ...] [--status ...] --env prod --project next --json` |
| Next free use case code | `flux usecases next-code --area NEXT --env prod --project next --json` |
| Comment on use case | `flux usecases comment <id-or-code> --content "..." --env prod --project next --json` |
| Link use case to scene | `flux usecases link <id-or-code> --target-type scene --target-id <scene> --env prod --project next --json` |
| Unlink use case | `flux usecases unlink <id-or-code> --target-type scene --target-id <scene> --env prod --project next --json` |
| Issues of a use case | `flux usecases issues <id-or-code> --env prod --project next --json` |
| Resync use case linked issues | `flux usecases resync <id-or-code> --env prod --project next --json` |
| Upload image | `flux images upload --file <path> [--caption "..."] --env prod --project <key> --json` |
| Render diagram | `flux diagrams render --file <path.mmd> --env prod --project <key> --json` |
| Render diagram (inline) | `flux diagrams render --mermaid "graph TD; A-->B" --env prod --project <key> --json` |
| AppSignal apps | `flux appsignal apps --env prod --project <key> --json` |
| AppSignal incidents | `flux appsignal incidents list --app <app> --env prod --project <key> --json` |

## Environment & Project Selection

```bash
# Flux Platform backlog
flux issues list --env prod --project flux --json

# Next Platform backlog
flux issues list --env prod --project next --json

# Feldspar backlog
flux issues list --env prod --project feldspar --json
```

For explicitly requested test-deployment verification only, substitute `--env test` and select the relevant test project explicitly. A product's deployment/testing stage does not change the tracker environment: real work remains in production.

## Scenes and Use Cases

Projects with a Product to-do set plan product work as **Scene → Use Case → Issue**.

- **Scene** (formerly "User Journey"): an actor-centred view of the system with one angle and one zoom level; the actor isn't necessarily human. Scenes together make up the whole system and may overlap. Written by the business developer.
- **Use Case**: part of a Scene worked out as a complete software design (main success scenario, alternative flows, exceptions). Each can ship to production on its own. Written by the software designer.
- **Issue**: delivery work that implements (part of) a Use Case.

**Codes** start the title: `SC-<Area>-NN` for scenes (`SCN-<Area>-NN` and legacy `UJ-<Area>-NN` are still accepted) and `UC-<AREA>-NN` for use cases. Matching ignores case and leading zeros; `xx` marks an unnumbered draft. Don't pick numbers yourself: `create --area <Area>` gives the title the next free code, `--code` sets one explicitly, and a title that already starts with a code keeps it. `next-code --area <Area>` prints the next free code without creating anything.

**Rules:**
- Each child has at most one parent: a use case belongs to one scene, an issue to one use case. Linking to another parent moves the child.
- `issues create` and `issues update` link through `--epic`, `--milestone` and `--usecase`; on update an empty value (`--usecase ""`) removes that link. With `--usecase`, the command reads the use case first, so an unknown use case, a project without Product or a server without Scenes fails before anything is written ("no issue was created: …"). If the issue is created but a link still fails, the command fails with "issue <id> was created, but could not link …": link it with `flux issues link` instead of creating it again.
- Epics are not linked to milestones. Link issues to an epic or milestone with `flux issues link`; `flux epics link` and `flux milestones epics` were removed.
- An issue's **context** is its bracketed title prefix (`[Dev]`, `[Web]`, or something else such as `[UC-NEXT-01]`), formerly called the program. `--context` on `issues create`/`update` sets it, replacing any existing prefix; `issues list --context` filters on it. `--program` is a deprecated alias; don't use it.
- A use case is optional on an issue. Link product work to its use case when one exists; bugs, chores and triage findings may have none. Epics and milestones still work as before, next to the use case.
- Scenes and use cases are completed by hand in Basecamp; there is no complete command.
- `--status` is the name of a to-do list group (for example `Refine`, `Ready to pick up`); `update --status none` moves the item out of its group. Status is reported as the group name, `done` once completed, or `null`.
- Links live in the "Managed by Flux" section of both to-dos. Never edit it by hand; use `link`/`unlink`.
- A parent holds a copy of each child's title. After renaming children, run `resync` on the parent.

A project without a Product to-do set fails with `product_not_configured`; an older server fails with "not supported by this server".

## Issue Stages

Stages in order: `triage` → `specification` → `design` → `development` → `testing` → (done)

Stage emojis (add to title when advancing beyond specification):
- Design: ✏️
- Development: 💻
- Testing: 🧪
- Done: ✅ (also mark complete in Basecamp)

```bash
# Advance to development (add emoji to title)
flux issues advance <id> --stage development --comment "Starting implementation" --env prod --project <key> --json
flux issues update <id> --title "[Dev] Fix the thing 💻" --env prod --project <key> --json
```

## Common Workflows

### Create and link an issue to an epic

```bash
# Create issue in Development stage, linked to the epic
flux issues create \
  --title "[Dev] Implement feature X 💻" \
  --app web \
  --stage development \
  --size M \
  --epic <epic_id> \
  --env prod --project <key> --json

# Or link an existing issue
flux issues link <issue_id> --target-type epic --target-id <epic_id> --env prod --project <key> --json
```

### Break a use case down into issues

```bash
# Read the use case and the issues it already has
flux usecases get UC-NEXT-01 --env prod --project next --json
flux usecases issues UC-NEXT-01 --env prod --project next --json

# Create an issue linked to the use case
flux issues create --title "[Web] Confirm account link during sign-in" --app web --stage specification --usecase UC-NEXT-01 --env prod --project next --json

# Or link an existing issue
flux issues link <issue_id> --target-type usecase --target-id UC-NEXT-01 --env prod --project next --json
```

### Add a use case to a scene

```bash
# Numbered with the next free UC-NEXT-NN code and linked to the scene in one step
flux usecases create --title "Link existing account during SURFconext sign-in" --area NEXT --scene SC-Next-02 --env prod --project next --json

# Or link an existing use case (moves it if it had another scene)
flux usecases link UC-NEXT-01 --target-type scene --target-id SC-Next-02 --env prod --project next --json
```

### Advance an issue through stages

```bash
# Specification → Design
flux issues advance <id> --stage design --env prod --project <key> --json
flux issues update <id> --title "[Web] Fix the thing ✏️" --env prod --project <key> --json

# Design → Development
flux issues advance <id> --stage development --comment "Design approved" --env prod --project <key> --json
flux issues update <id> --title "[Web] Fix the thing 💻" --env prod --project <key> --json

# Development → Testing
flux issues advance <id> --stage testing --comment "PR #42 merged" --env prod --project <key> --json
flux issues update <id> --title "[Web] Fix the thing 🧪" --env prod --project <key> --json
```

### Add a comment

```bash
flux issues comment <id> \
  --content "Investigated root cause: the session store is evicting tokens too early." \
  --env prod --project <key> --json
```

### Check AppSignal incidents

```bash
# List available apps
flux appsignal apps --env prod --project <key> --json

# List open incidents
flux appsignal incidents list --app <app_name> --state open --env prod --project <key> --json

# Get incident details
flux appsignal incidents get --app <app_name> --number <N> --env prod --project <key> --json
```

## Auth

```bash
flux auth login --env prod --project <key> --json   # Sign in to production
flux auth logout --env prod --project <key> --json  # Sign out of production
flux auth status --env prod --project <key> --json  # Check production status
```

Credentials stored in `~/.config/flux/credentials.json`, one entry per environment. API keys are no longer supported: `--api-key` fails and `FLUX_API_KEY` is ignored. Sign in with `flux auth login`.

`auth status` verifies the active credentials through the selected server's
`GET /api/delivery/identity` endpoint, using the saved credentials for the
selected environment.
Missing, invalid, or unverifiable authentication exits nonzero without success
JSON; a local credential file alone is not proof of authentication.

An explicit `--env` overrides `FLUX_ENV`, even `--env prod` when `FLUX_ENV=test`.
Otherwise `FLUX_ENV` applies, then the default `prod`.

## Comment Formatting

The server processes comment and description content submitted through the CLI:

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
metadata, not verified identity. Never infer it from
credentials or environment. It does not invoke/select a model or change
authentication, environment, or project selection.

Supported content writes:

- `flux issues create` / `update` with `--description`
- `flux epics create` / `update` with `--description`
- `flux milestones create` / `update` with `--description`
- `flux scenes create` / `update` and `flux usecases create` / `update` with `--description`
- `flux issues comment`, `flux epics comment`, `flux milestones comment`, `flux scenes comment`, `flux usecases comment`
- `flux comments update`
- `flux issues advance` for its optional `--comment` only, not its automatic stage comment

```bash
flux issues comment <id> --content "Investigated the failure." --ai-model "openai/gpt-5" --env prod --project <key> --json
flux comments update <comment_id> --content "Revised findings." --ai-model "anthropic/claude-sonnet-4" --env prod --project <key> --json
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
and `url`; use cases include `scene`). Scene and use case writes, links and
resyncs return the server's response. Other mutations return:

```json
{"ok": "true", "id": "<id>"}
```

Use `--json` output to chain commands: extract the `id` field from create responses to use in subsequent link or advance calls.

`flux auth status --json` returns only verified identity and environment fields:

```json
{"signed_in":true,"env":"prod","basecamp_account_id":"123","basecamp_person_id":"456","display_name":"Alex"}
```

Account and person IDs are strings identifying the signed-in person. No provider
credentials or tokens are included.

`flux issues get <id> --env prod --project <key> --json` scopes the issue lookup
to that project; issues outside it return an error. Each `thread` comment
preserves `author` and adds `author_id`, the Basecamp creator's string person ID.
Missing creator IDs are omitted or `null` as the server sends them, never
inferred from a name:

```json
{"id":"789","author":"Alex","author_id":"456","date":"2026-10-06","content":"Comment text"}
```

Names can be shared. Within the same Basecamp account, compare a comment's
`author_id` to the verified `basecamp_person_id` to determine whether it belongs
to the signed-in principal; do not compare display names.

## Error Handling

| Error | Fix |
|-------|-----|
| `unauthorized` | Run `flux auth login --env prod --project <key> --json` (use `--env test` only for explicit test verification) |
| `not found` | Verify the ID or code exists in the selected project |
| `ambiguous_code` | Several items share the code (e.g. an `xx` draft); the error lists the matching IDs, use one of them |
| `product_not_configured` | The project has no scenes or use cases; check `--project` |
| `not supported by this server` | The server is older than the CLI; wait for the server release |
| Non-zero exit | Check stderr for the error message |

Exit codes: 0 = success, non-zero = error (check stderr).
