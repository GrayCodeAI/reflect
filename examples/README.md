# Reflect Examples

Reflect is an AI coding agent that understands your codebase.

## Basic Usage

### Start a chat session

```bash
reflect chat
> Explain how the authentication system works
```

### Generate code

```bash
reflect "Add input validation to the user registration endpoint"
```

### Review changes

```bash
reflect review
```

## Advanced Examples

### Use with skills

```bash
reflect --skill code-review "Review my latest changes"
```

### Analyze codebase

```bash
reflect analyze --depth full
```

### Fix issues

```bash
reflect fix --auto
```

### Headless agent in CI

Copy [reflect-ci-exec.yml](github/reflect-ci-exec.yml) to run `reflect exec --ephemeral --json`
on pull requests (summarize diff, risk list, or your own prompt). Pin your
reflect install step and provider secrets before enabling the job.

### Report CI delivery context

Copy [reflect-delivery-context.yml](github/reflect-delivery-context.yml) to your
repository to report GitHub Actions runs to Reflect Cloud. Create a dedicated,
revocable device token for CI and keep its endpoint, device ID, project ID, and
token in GitHub Actions secrets.

## MCP Integration

Reflect can use MCP servers for extended capabilities:

```bash
# With harrier for persistent memory
harrier setup
reflect chat

# With swift for session capture
swift start
reflect "refactor the API layer"
swift stop
```

## Configuration

Create `.reflect/config.json`:

```json
{
  "model": "claude-sonnet-4-6",
  "maxTokens": 8192,
  "temperature": 0.7
}
```

See the [main README](../README.md) for full documentation.
