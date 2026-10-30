# Reflect vs Top 20 Coding Agents — Permission, Sandbox, Memory & Tools

> Researched 2026-10-30. Sources: Vellum, MightyBot, Totalum, Medium/chaos.architect, F3Fundit, Northflank, Kilo leaderboard, GitHub (Kilo #9138), VS Code approvals docs, Anthropic auto-mode writeup.

## Top 20 coding agents (2026)

| # | Agent | Surface | Permission model | Sandbox | Memory | Notable |
|---|---|---|---|---|---|---|
| 1 | **Claude Code** | Terminal + IDE + web | 3 modes (default / acceptEdits / bypass) + new **auto mode** (classifier auto-approves safe; blocks overeager). 93% prompts rubber-stamped → collapsed to tiered auto. | OS sandbox (file/network) | Session + CLAUDE.md + memory tool | Auto mode is the 2026 reference design. |
| 2 | **OpenAI Codex CLI** | Terminal | `--approval-mode` suggest/auto-edit/full-auto | Read-only FS by default; network blocked | AGENTS.md | Sandbox-first. |
| 3 | **Cursor** | IDE | Allow-all / ask / sandboxed terminal | Sandboxed terminal | .cursorrules | Review-trap UX (diff approval). |
| 4 | **Windsurf** | IDE | Local Cascade + cloud Devin | Sandboxed | Workspace memory | 1M users. |
| 5 | **Aider** | Terminal | `--yes` / `--auto-test` / per-edit yes-no | None (host) | Git-based history | Most honest solo tool; tight git. |
| 6 | **Cline** | IDE (VS Code) | Plan/Act modes; per-tool approval toggles | Optional | .clinerules | BYO model, self-host. |
| 7 | **Roo Code** | IDE | Modes (Code/Architect/Ask/Debug) + per-tool auto-approve | Optional | .roo/ | Fork of Cline. |
| 8 | **OpenCode** | Terminal | Rules: shell/edit/execute/external_directory → ask/allow/deny; `--auto` | None by default | AGENTS.md | Provider-agnostic; distinct build/plan agents. |
| 9 | **Gemini CLI** | Terminal | `--approval-mode` default/auto_edit/yolo | Sandbox | GEMINI.md | Free tier (60 rpm / 1k rpd). |
| 10 | **GitHub Copilot CLI** | Terminal + IDE | Enterprise policy; tool approval | Sandboxed | Repo instructions | Enterprise breadth. |
| 11 | **Amp (Sourcegraph)** | Terminal/IDE | Auto-approve reads; ask writes/exec | Sandboxed | Code-intel index | Best mono-repo code intelligence. |
| 12 | **Antigravity (Google)** | Agent IDE | Preview; Gemini 3 | Preview | Workspace | Preview-only. |
| 13 | **Devin** | Cloud | Fully autonomous; human review at PR | Cloud sandbox | Persistent | Autonomous tickets. |
| 14 | **OpenHands** | Self-hosted | Configurable approval; sandboxed runtime | Docker sandbox | Pluggable memory | Open-source platform. |
| 15 | **Replit Agent** | Cloud IDE | Auto within sandbox | Cloud sandbox | Project memory | Shipping-focused. |
| 16 | **Zed (agent host)** | Editor | Hosts other agents; per-agent approval | Sandboxed | Workspace | Cleanest host-other-agents play. |
| 17 | **Kilo Code** | IDE | Manual or allow-everything; moving to **OpenGuardrails** classifier (auto-approve safe bash). | Optional | .kilo/ | Adopting Claude auto-mode design. |
| 18 | **Kimi Code** | Terminal | Approval modes + sandbox | Sandbox | KIMI.md | Moonshot. |
| 19 | **Qwen Code** | Terminal | Configurable approval modes + sandbox options | Sandbox | QWEN.md | Multi-protocol. |
| 20 | **DeepSeek (via agents)** | API | n/a (model, not agent) | n/a | n/a | Cheapest capable model. |

## Top 20 open-source coding agents (GitHub stars, 2026-10)

From [awesome-cli-coding-agents](https://github.com/bradAGI/awesome-cli-coding-agents) (last updated 2026-10-26) and the pinggy blog.

| # | Agent | Stars | Lang | Permission model (from docs) | Sandbox | Memory |
|---|---|---|---|---|---|---|
| 1 | **Hermes Agent** | 257k | Python | approval prompts; persistent memory | Unix socket RPC sandbox | Persistent memory + skills |
| 2 | **OpenCode** | 222k | TS | rules: shell/edit/execute/external_dir → ask/allow/deny; `--auto` | none by default | AGENTS.md |
| 3 | **Claw Code** | 195k | Rust/Py | (Claude Code clone) | — | — |
| 4 | **Codex CLI** | 131k | Rust | suggest/auto-edit/full-auto | read-only FS; network blocked | AGENTS.md |
| 5 | **Pi** | 118k | TS | (harness) | — | skills |
| 6 | **Gemini CLI** | 108k | TS | default/auto_edit/yolo | sandbox | GEMINI.md |
| 7 | **OpenHands** | 91.9k | Python | configurable + sandboxed runtime | Docker sandbox | pluggable |
| 8 | **Cline CLI** | 70.9k | TS | plan/act; per-tool approval toggles | optional | .clinerules |
| 9 | **Oh My OpenAgent** | 70.3k | TS | (multi-harness) | — | — |
| 10 | **Open Interpreter** | 68.7k | Python | per-exec approval | host | — |
| 11 | **Goose** | 55.8k | Rust | ask per action | host + MCP | — |
| 12 | **Aider** | 49.8k | Python | `--yes` / `--auto-test` / per-edit | none (host) | git history |
| 13 | **Codewhale** | 41.1k | Rust | 3-level permission posture | workspace snapshots | — |
| 14 | **OH-MY-PI** | 36.7k | TS/Rust | — | — | — |
| 15 | **Continue CLI** | 36.3k | TS | — | — | privacy-first |
| 16 | **Reasonix** | 36k | Go | approval gates | workspace sandbox | checkpoints |
| 17 | **Deep Agents (LangChain)** | 30.6k | Python | HITL approvals | — | — |
| 18 | **Crush** | 28.8k | Go | — | — | LSP |
| 19 | **Qwen Code** | 28.7k | TS | configurable approval modes | sandbox | — |
| 20 | **Kilo Code CLI** | 27.8k | TS | manual / allow-everything → OpenGuardrails classifier | optional | checkpoints |

### Permission/sandbox highlights from OSS leaders
- **Codex CLI**: read-only FS by default + network blocked — strongest default sandbox.
- **OpenHands**: Docker sandboxed runtime; configurable approval.
- **OpenCode**: granular rules (shell/edit/execute/external_directory) with ask/allow/deny.
- **Kilo**: moving from blunt allow-everything to an **OpenGuardrails classifier** (Claude auto-mode style) — explicit direction change in #9138.
- **Maki**: tree-sitter-parsed permission gating for bash commands (novel).
- **Orca**: OS-level sandboxing (Seatbelt, bwrap, Landlock+seccomp, fail-closed).
- **agentty**: sandboxed by default (Bubblewrap/sandbox-exec).
- **ipsupport-code**: opt-in OS-level sandboxing (Seatbelt/Landlock) for shell tool.
- **DvalinCode**: enforced network egress + OS-sandboxed subprocesses + tamper-evident audit trail.
- **Octomind**: guardrails-as-code.
- **Hazzel**: approval-first; `/undo` reverts only agent hunks.

## Reflect's current permission model (12 mechanisms)

1. 5 autonomy tiers (Supervised/Basic/Semi/Full/YOLO) → product tiers scout/builder/operator/autonomous
2. Dry-run kill switch
3. Spec workflow gate (Specify→Plan→Tasks)
4. Bash classifier (safe/unsafe/unknown)
5. Persisted exact rules
6. UnifiedGrants + memory grants
7. Never-allow ceiling
8. Bypass kill-switch (scoped, time-bound)
9. PreToolUse hooks
10. Governance ceiling (org policy)
11. External/untrusted tool gating
12. Tool categorization (read/write/network/shell)

## Industry convergence (what the leaders do)

- **3 tiers, not 5**: ask / auto-safe / auto (Claude Code, Codex, Gemini CLI, OpenCode all converge here).
- **Classifier auto-approves safe commands** (Claude auto-mode, Kilo/OpenGuardrails) — eliminates 93% of rubber-stamp prompts.
- **Sandbox by default** for filesystem + network (Codex, OpenHands, Devin, Replit).
- **Human review at the PR/commit boundary**, not per-tool-call (Devin, Cursor cloud).
- **Memory via instruction files** (AGENTS.md / CLAUDE.md / GEMINI.md) + session persistence — not a separate grant store.
- **Plan/Act or spec modes** are UX modes, not permission layers (Cline, Roo).

## Recommended simplification for Reflect

### Collapse to 3 layers (drop 9 of 12 mechanisms)

| Keep | Merge/Drop |
|---|---|
| 1. Tier: `ask` / `auto-safe` / `auto` (was 5 levels) | Drop Basic/Semi/Full distinctions |
| 2. Classifier: safe/unsafe/unknown Bash | Keep as-is (already aligned with auto-mode) |
| 3. Hard deny: destructive + never-allow + governance | Merge governance into never-allow; drop separate governance ceiling unless org policy needed |
| — | Merge Spec gate → a `plan` mode (not a permission layer) |
| — | Merge Bypass kill-switch → `auto` tier with optional time-box |
| — | Merge ExactRules + UnifiedGrants + Memory + AutoAllow → one "remembered rules" store |
| — | Drop PreToolUse hooks from hot path → optional plugin |
| — | Drop dry-run as separate → flag on `ask` tier |

### Defaults & UX
- Default new users to `auto-safe` (not Supervised) — matches Claude Code's post-auto-mode default.
- Ship a **sandbox** (read-only FS + no network by default) for Bash, like Codex/OpenHands.
- Keep human review at commit/PR boundary (`reflect pr`, `SmartCommit` already exist).

### What Reflect already does well
- Classifier-based safe Bash auto-allow (just added for Builder tier).
- Spec workflow gate (unique; keep as a mode).
- Never-allow ceiling (good safety).
- Remembered rules with identity (good).

### Gaps vs leaders
- No OS-level sandbox for Bash (Codex/OpenHands/Devin have it).
- 5 tiers where 3 suffice (cognitive load).
- 4 separate rule stores (ExactRules/UnifiedGrants/Memory/AutoAllow) — should be one.
- No auto-mode classifier for *edits* (only Bash) — Claude auto-mode also gates file writes.
- Default tier too conservative (Supervised) vs leaders (auto-safe).

## Action plan (prioritized)

1. **Collapse tiers 5→3** + update `/autonomy` picker and settings. (Low risk, big UX win.)
2. **Merge rule stores** into one remembered-rules table with allow/deny + identity + expiry.
3. **Move spec gate** out of permission engine into a `plan` mode flag.
4. **Add Bash sandbox** (read-only FS by default; opt-in network).
5. **Default to `auto-safe`** for new installs.
6. **Extend classifier** to file writes (safe = within project, no secrets paths).

Pick a number (or "all") and I'll implement.