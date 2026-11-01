# Reflect and Relay workflow

Reflect owns the coding-agent workflow and terminal UI. Relay owns model
providers, credentials, model routing, and streaming transport. Both remain
independent repositories.

Each project has its own build, configuration, version, and release lifecycle.
Relay is reusable by any application; Reflect is usable in any codebase and by
other applications through its integration interfaces.

| Integration | Interface |
| --- | --- |
| Interactive coding in any repository | `reflect --cwd /path/to/project` |
| Scripted tasks | `reflect exec --cwd /path/to/project --json "Review this code"` |
| Streaming automation | `reflect exec --output-format stream-json "Run the tests"` |
| Editor integration | `reflect acp` |
| MCP clients | `reflect mcp serve` |
| Model access in another Go application | Relay's public `engine` facade |

## Work in Reflect

Run the built executable from the project you want to work on:

```sh
./reflect
```

Reflect uses its original full-width chat layout, with the input fixed at the
bottom. Model, permissions, workspace, and usage are shown in the footer.
`/status`, `/cost`, and `/keybindings` provide additional details.

| Control | Action |
| --- | --- |
| Shift+Tab | Cycle Plan → Build → Review |
| F2 or `/mode` | Choose Plan, Build, or Review |
| F3 | Choose the permission tier |
| `/plan [task]` | Enter Plan mode and optionally research a task |
| `/plan show` | Reopen this conversation’s saved plan |
| `/spec` | Open the optional advanced spec workflow |
| Ctrl+K | Open the command palette |
| Tab | Move between prompt and transcript |
| Shift+Enter | Add a newline |
| Esc | Cancel the active operation or dismiss a menu |

For everyday work, use Plan to research, Build to implement, and Review to inspect
the result. Plan saves the final response as one Markdown document in Reflect’s
project- and conversation-scoped user state, outside your source tree. Later Plan
turns replace that document; `/plan show` reopens it. A cancelled or failed turn
does not replace the saved plan. Switching modes does not start a model turn.

Use `/spec` for work that needs proposal, specification, design, plan, tasks, and
explicit implementation approval. The full spec workflow starts only when you
opt in. `/spec reset` leaves it. It does not start with ordinary planning.

Work mode, permission tier, and workspace trust are independent. Choosing Plan
or Review restricts mutating tools. A trusted workspace still follows its
permission policy. Starting a spec workflow adds its own implementation gates.

Loading an installed local skill reads instructions. Any commands, file edits,
or network tools used afterward follow their own permission checks. Plugins and
MCP servers continue to use the normal installation and trust boundaries.

## Work in Relay

Implement provider protocols, adapters, credentials, and model metadata in the
Relay repository. Reflect consumes Relay through its engine facade. Validate
provider changes in Relay before checking their behavior through Reflect.

For local development, an explicit local module replacement may point Reflect
at the sibling Relay checkout. Released builds use the version pinned in
Reflect's `go.mod`.

## Verify a change

Run focused tests for the changed behavior and build the Reflect executable.
For terminal UI changes, cover resizing, input routing, pending approvals,
concurrent tools, and session continuity. Provider changes also need Relay's
facade and streaming lifecycle checks.
