# Flux CLI

Command-line interface for Flux project management.

## Installation

```bash
go install github.com/eyra/flux-cli@latest
```

## Usage

### List issues

```bash
# List all issues
flux issues list

# Filter by stage
flux issues list --stage development

# JSON output
flux issues list --json
```

### Get issue details

```bash
flux issues get 12345
```

Use `--project <key>` (for example `flux`, `next` or `feldspar`; `flux projects
list` shows all) to scope the lookup. The default is
`next` on production and `flux` on test. An issue outside the selected project
returns an error.

### Verified authentication status

```bash
flux auth status --env test --json
```

Status verifies the active credentials with the server, rather than trusting a
local credential file. On success it returns:

```json
{"signed_in":true,"env":"test","basecamp_account_id":"123","basecamp_person_id":"456","display_name":"Alex"}
```

The Basecamp account and person IDs are strings identifying the actual signed-in
principal (including the bot for API keys), not a persona. No tokens are returned.
Missing, invalid, or unverifiable authentication exits nonzero without success
JSON. Credential precedence remains `--api-key`, `FLUX_API_KEY`, then the selected
environment's saved personal credentials.

### Issue thread JSON

`flux issues get <id> --project flux --json` retains the `thread` array. Each comment
includes its display name as `author` and, when supplied by Basecamp, its stable
string person ID as `author_id`:

```json
{"id":"789","author":"Alex","author_id":"456","date":"2026-10-06","content":"Comment text"}
```

When the creator ID is unavailable, `author_id` is omitted or `null`, exactly as
the server sends it: `--json` output repeats the server's JSON and keeps every
field, including `ref`, `epic`, `milestone`, `use_case` and `url`. Names are not unique;
compare `author_id` with verified `basecamp_person_id` to identify the signed-in
person's comments, within the same Basecamp account. Do not derive identity from
the name or persona attribution.

### Scenes and use cases

Scenes and use cases live in a project's Product to-do set (Next Platform:
`--project next`). The model is Scene → Use Case → Issue:

- A **scene** (formerly "User Journey") is an actor-centred view of the system
  with one angle and one zoom level. Code: `SCN-<Area>-NN`.
- A **use case** works out part of a scene as a complete software design.
  Code: `UC-<AREA>-NN`.
- An **issue** implements (part of) a use case.

An issue belongs to at most one use case, and a use case to at most one scene;
linking to another parent moves the child. A use case is optional on an issue:
bugs, chores and triage findings may have none. IDs may be Basecamp IDs or
codes: `SCN-Next-02` (legacy `UJ-Next-02` works too) or `UC-NEXT-01`, in any
case and with or without leading zeros. Unnumbered drafts use `xx`; a code that
several items share returns `ambiguous_code`, so use the Basecamp ID.

```bash
# Scenes
flux scenes list [--completed]
flux scenes get SCN-Next-02 [--no-thread]
flux scenes create --title "Donate data" --area Next [--code SCN-Next-04] [--status Refine] [--description ...]
flux scenes update SCN-Next-02 [--title ...] [--code ...] [--status none] [--description ...]
flux scenes comment SCN-Next-02 --content "..."
flux scenes next-code --area Next     # print the next free code, e.g. SCN-Next-04
flux scenes usecases SCN-Next-02      # use cases linked to the scene
flux scenes resync SCN-Next-02        # refresh the linked use case titles

# Use cases
flux usecases list [--scene SCN-Next-02] [--completed]
flux usecases get UC-NEXT-01 [--no-thread]
flux usecases create --title "Upload data" --area NEXT [--scene SCN-Next-02]
flux usecases update UC-NEXT-01 [--title ...] [--code ...] [--status ...] [--description ...]
flux usecases comment UC-NEXT-01 --content "..."
flux usecases next-code --area NEXT   # print the next free code, e.g. UC-NEXT-03
flux usecases link UC-NEXT-01 --target-type scene --target-id SCN-Next-02
flux usecases unlink UC-NEXT-01 --target-type scene --target-id SCN-Next-02
flux usecases issues UC-NEXT-01       # issues linked to the use case
flux usecases resync UC-NEXT-01       # refresh the linked issue titles

# Issues
flux issues create --title "..." --app web --usecase UC-NEXT-01 [--epic ...] [--milestone ...]
flux issues update 12345 --usecase UC-NEXT-02   # moves it; --usecase "" removes the link
flux issues link 12345 --target-type usecase --target-id UC-NEXT-01 [--unlink]
```

`issues create` and `issues update` link the issue through `--epic`,
`--milestone` and `--usecase` after writing it, with the same link endpoint as
`issues link`. If a link fails after the issue was created, the command exits
with "issue <id> was created, but could not link …"; link it with `issues link`
rather than creating it again.

On create, the title gets `--code`, else the code it already starts with, else
the next free code in `--area`. `--status` is the name of a group in the list;
on update, `--status none` moves the item out of its group. Scenes and use cases
are completed by hand in Basecamp. A parent keeps a copy of each child's title:
run `resync` on it after renaming its children.

A project without a Product to-do set returns a `product_not_configured` error.
A server older than this CLI returns "not supported by this server".

### API namespaces

The CLI calls `/api/delivery` for issues, milestones and epics, and
`/api/product` for scenes and use cases. Against an older server without
`/api/delivery` it falls back to `/api/dev` automatically.

### List personas

```bash
flux personas list
```

### AI model attribution on content

Use optional `--ai-model` to declare which model generated text you supply:

```bash
flux issues comment 12345 --content "Investigated the failure." --ai-model "openai/gpt-5" --json
flux comments update 67890 --content "Updated findings." --ai-model "anthropic/claude-sonnet-4" --json
```

Supported commands are `issues`, `epics`, `milestones`, `scenes`, and `usecases`
**create**, **update**
(for supplied descriptions), and **comment**; **comments update**; and
**issues advance** (for its optional `--comment`, not the automatic stage comment).
The flag is not global and is unavailable on reads, deletes, links, assignments,
resyncs, and other operations.

The CLI forwards the exact string as JSON `ai_model`, including provider/model
IDs, whitespace, and HTML-looking text. When omitted, the JSON field is omitted;
no model is inferred from a persona, credentials, or environment. This is
caller-declared display metadata, not verified identity: it does not invoke or
select a model, change authentication, or change environment/project selection.

The server renders a single code-block footer, with only Flux and the model name bold and no italics:

- Without a model, or with an empty/whitespace-only value: `<pre>Assisted by <strong>Flux</strong></pre>`
- With `--ai-model "openai/gpt-5"`: `<pre>Assisted by <strong>Flux</strong> · Generated with <strong>openai/gpt-5</strong></pre>`

The server escapes model text as HTML and does not expand its `@mentions`.
Content edits replace the previous generated footer rather than accumulating
footers; editing content without a model removes its old model suffix.
Independent AI-disclosure text is retained. Metadata-only updates without a
description leave the historical content and model footer unchanged, even if
`--ai-model` is supplied. Advancing without `--comment` likewise attributes no
user text.

### Environments

```bash
# Production (default) - every project's real backlog, including Flux's own
flux issues list --project flux

# Test environment - only for verifying the eyra-flux-test deployment
flux issues list --env test
```

An explicit `--env` takes precedence over `FLUX_ENV`, including `--env prod` when
`FLUX_ENV=test`. Without either, the environment is `prod`.

## Configuration

The CLI connects to:
- **prod**: `https://eyra-flux.fly.dev` (default)
- **test**: `https://eyra-flux-test.fly.dev`

## Development

```bash
# Build
go build -o flux .

# Run
./flux issues list --env test

# Run against a local server
FLUX_BASE_URL=http://localhost:4040 ./flux scenes list --project next --api-key <key>
```
