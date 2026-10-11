# Decoder Examples

Decoder is an AI coding agent that understands your codebase.

## Basic Usage

### Start a chat session

```bash
decoder chat
> Explain how the authentication system works
```

### Generate code

```bash
decoder "Add input validation to the user registration endpoint"
```

### Review changes

```bash
decoder review
```

## Advanced Examples

### Use with skills

```bash
decoder --skill code-review "Review my latest changes"
```

### Analyze codebase

```bash
decoder analyze --depth full
```

### Fix issues

```bash
decoder fix --auto
```

### Headless agent in CI

Copy [decoder-ci-exec.yml](github/decoder-ci-exec.yml) to run `decoder exec --ephemeral --json`
on pull requests (summarize diff, risk list, or your own prompt). Pin your
decoder install step and provider secrets before enabling the job.

### Report CI delivery context

Copy [decoder-delivery-context.yml](github/decoder-delivery-context.yml) to your
repository to report GitHub Actions runs to Decoder Cloud. Create a dedicated,
revocable device token for CI and keep its endpoint, device ID, project ID, and
token in GitHub Actions secrets.

## MCP Integration

Decoder can use MCP servers for extended capabilities:

```bash
# With harrier for persistent memory
harrier setup
decoder chat

# With swift for session capture
swift start
decoder "refactor the API layer"
swift stop
```

## Configuration

Create `.decoder/config.json`:

```json
{
  "model": "claude-sonnet-4-6",
  "maxTokens": 8192,
  "temperature": 0.7
}
```

See the [main README](../README.md) for full documentation.
