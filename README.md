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

Use `--project flux` or `--project next` to scope the lookup. The default is
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

When the creator ID is unavailable, `author_id` is omitted. Names are not unique;
compare `author_id` with verified `basecamp_person_id` to identify the signed-in
person's comments, within the same Basecamp account. Do not derive identity from
the name or persona attribution.

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

Supported commands are `issues`, `epics`, and `milestones` **create**, **update**
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
# Production (default) - Eyra dev projects
flux issues list

# Test environment - Flux dogfooding
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
```
