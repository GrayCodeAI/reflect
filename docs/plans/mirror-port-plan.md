# Plan: Port Mirror (reflection-oss/mirror) capabilities into Reflect

Source repo (cloned locally): `../mirror-ref` (reflection-oss/mirror, Apache-2.0).
Target: `reflect` (Go, monolithic TUI + relay engine).

Mirror is a Rust agent + Python CLI supervisor with AgentRPC workers. Reflect is a
monolithic Go TUI. We port **features and UX**, not the architecture (worker
supervision, AgentRPC) — Reflect's in-process agent loop is a strength, not a gap.

## Phase 0 — Inventory (done)

Mirror's CLI surface (`mirror-cli/src/mirror/cli/`) and features:

- Core: input/rendering, conversation, resume, rewind, branching
- Workspaces: roster, creation, switching, persistence (`workspace_controller`, `workspaces`, `workspaces_application`)
- Worktrees: parallel independent checkouts (`worktrees`)
- Sandbox: isolated file/shell tool execution (`sandbox/`, `sandbox_shell.py`, `sandboxes.py`)
- Extensions: Python extensions, commands, host tools, MCP declarations (`extension.py`, `extensions/`, `host_tools.py`, `mcp_configuration.py`, `user_extensions.py`)
- Settings: menu-driven settings with templates (`cli/extensions/settings/`)
- Themes: theme system (`themes.py`, `docs/themes.md`)
- Transcript: rendering/history (`transcript.py`, `transcript_view.py`)
- Approvals: bash/tool approvals (`bash_approvals.py`, `tool_approvals.py`)
- Onboarding, API keys, models (`onboarding.py`, `api_keys.py`, `models.py`)
- Noninteractive mode (`noninteractive.py`)
- Shell integration, terminal (`shell.py`, `terminal.py`, `extended_keys.py`)
- Safety, statsig telemetry, logging
- Memory (opt-in, cross-session)
- Project instructions, skills

## Mapping to Reflect (already present)

| Mirror feature | Reflect equivalent | Status |
|---|---|---|
| Worktrees / parallel tasks | `internal/multiagent`, mission mode, EnterWorktree/ExitWorktree | ✅ present |
| Resume conversations | session resume (`-r`/`-c`) | ✅ |
| Providers/models | relay engine (28+) | ✅ |
| Memory | conversation arc + memory | ✅ |
| Project instructions | AGENTS.md / ZERO.md | ✅ |
| Skills | skills system | ✅ |
| MCP | MCP client | ✅ |
| Approvals | permissions/autonomy + spec gate | ✅ (different UX) |
| Themes | (check Reflect theme) | partial |
| Transcript | session JSONL + TUI | partial |

## Gaps worth porting (prioritized)

### P1 — Sandbox workspaces (HIGH value, MEDIUM effort)
Mirror runs file/shell tools in an isolated sandbox (microsandbox/container) instead
of on the host. Reflect executes on the host with permission gates. Porting an
opt-in sandbox mode would be a major safety upgrade.

Scope:
- Add `internal/sandbox/` package wrapping a container runtime (Docker/Podman) or
  a lightweight namespace/chroot fallback.
- Wire into `tool.Bash`, `tool.Write`, `tool.Edit` behind a setting
  (`settings.sandbox = off|docker|microsandbox`).
- Add `/sandbox` command + status in `/path` and doctor.
- Config: `REFLECT_SANDBOX`, `REFLECT_SANDBOX_IMAGE`.
- Tests: sandbox round-trip, fallback to host when runtime missing.

### P2 — Conversation rewind / branching (MEDIUM value, MEDIUM effort)
Mirror's `/rewind` forks conversation history (not filesystem). Reflect has
session rewind? Check `internal/session`. If missing, add:
- `/rewind [n]` command to fork a session at turn n into a new session file.
- TUI picker over turns.
- Reuse existing session checkpoint/WAL.

### P3 — Themes (LOW-MEDIUM value, LOW effort)
Mirror has a theme system (`themes.py`, `docs/themes.md`). Reflect has
`internal/theme`. Audit parity: color tokens, custom theme files, `/theme` command.

### P4 — Host tools / Python extensions (MEDIUM value, HIGH effort)
Mirror allows Python extensions contributing commands, host tools, MCP decls.
Reflect has Go plugins + MCP. Porting Python extensions would broaden ecosystem
but is high effort. Defer unless demanded.

### P5 — Workspace tabs / roster UI (MEDIUM value, HIGH effort)
Mirror's multi-workspace tab UI. Reflect uses mission mode (background). Could
surface mission workers as tabs. Defer.

### P6 — Noninteractive / scripting mode parity
Mirror has `noninteractive.py`. Reflect has `decoder exec`/`-p`. Compare flags
and output formats; align.

### P7 — Onboarding / API key UX
Mirror's masked sign-in prompt. Reflect uses `/config`. Compare and improve.

## Phase plan

- Phase 1 (P1 sandbox) — 1-2 weeks
- Phase 2 (P2 rewind) — 3-5 days
- Phase 3 (P3 themes) — 2-3 days
- Phase 4 (P6, P7) — 1 week
- Phase 5 (P4, P5) — only if demanded

## Out of scope (not porting)

- Rust agent / AgentRPC worker supervision (Reflect is monolithic by design)
- Python CLI runtime (Reflect is Go)
- statsig telemetry (Reflect has its own observability)
- microsandbox specifically (use Docker/Podman abstraction)

## Open questions

1. Sandbox runtime preference (Docker vs Podman vs namespace)?
2. Should sandbox be default-on or opt-in?
3. Rewind: fork to new session or in-place truncate?

## Next action

Start Phase 1 (sandbox) unless user picks a different priority.
