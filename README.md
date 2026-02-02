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

### List personas

```bash
flux personas list
```

### Environments

```bash
# Production (default) - Eyra dev projects
flux issues list

# Test environment - Flux dogfooding
flux issues list --env test
```

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
