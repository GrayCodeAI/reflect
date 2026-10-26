---
description: Extending reflect — how to write AGENTS.md files, custom specialists, skills, hooks, MCP servers, and plugins.
globs: "*.go, *.js, *.md, *.json, *.toml, *.yaml, *.yml"
alwaysApply: false
---

# Extending reflect

reflect is an open-source code intelligence platform. It lives in the `graycode-eco`
workspace alongside the ecosystem repos that power it (`relay`,
`shrike`, `harrier`, `swift`, `kestrel`, `merlin`). This document describes how to extend
reflect with custom tools, skills, hooks, and integrations.

## Development workflow

When starting any new work (feature, fix, refactor, chore), always create a feature branch from `main` first. Never commit directly to `main`. Use branch naming conventions like `feat/<description>`, `fix/<description>`, or `chore/<description>`. Open a PR, ensure CI is green, then merge.

## 1. Drop a project `AGENTS.md`

When reflect starts in a directory, it looks for project-level instructions and injects them into the system prompt. The lookup walks from your current working directory **up to the nearest git root** and reads the first matching file at each level — general rules at the repo root, more specific rules in sub-trees. Files are labeled with their directory in the prompt (e.g. `## Project guidelines (services/api/AGENTS.md)`).

Accepted file names, in priority order at each level:

| Path | Notes |
| --- | --- |
| `./AGENTS.md` | The classic spot — committed to your repo, shared with the team. |
| `./ZERO.md` | Brand-specific alias. Same format, lower priority. |
| `./.zero/AGENTS.md` | Project-local, hidden, gitignored. Personal notes that stay out of git. |

Matching is **case-insensitive** on the basename, so `AGENTS.md`, `Agents.md`, and `agents.md` resolve to the same file on Windows and macOS. The git-tracked filename in this repo is `AGENTS.md` — keep that on case-sensitive filesystems (Linux, the WSL filesystem, or a CI runner) to match what the loader looks for.

Both files use the same format. YAML frontmatter is optional; the markdown body is loaded as instructions for the agent. reflect reads the file once at session start, so changes take effect on the next launch — not mid-session.

```markdown
# Project conventions for <your project>

- Build with `make`, not `go build` directly.
- Tests live next to the source file (`foo_test.go` next to `foo.go`).
- Run `make lint` before opening a PR.
- Never edit files under `third_party/` — those are vendored.
```

Tips:

- Keep each file under ~8 KiB. reflect caps the **total** across all matched files at 32 KiB; everything past the cap is dropped.
- Re-state rules in the imperative voice: "Run `make lint`", not "you should consider running the linter".
- Don't put secrets, model IDs, or environment-specific paths in `AGENTS.md`. Use config files for those.
- In a monorepo, drop a narrower `AGENTS.md` in each sub-tree (e.g. `services/api/AGENTS.md`). reflect picks those up automatically when you launch from inside the sub-tree.
- A YAML frontmatter block (`---\n...\n---`) at the top is preserved verbatim in the injected prompt but is not parsed for `globs:` or `alwaysApply:` scoping today — keep the body self-contained.

### Personal guidelines, across every project

For preferences that follow *you*, not a specific repo (tone, tooling habits, workflow), drop a `ZERO.md` in your user config directory: `~/.reflect/ZERO.md` on Linux/macOS, `%AppData%\reflect\ZERO.md` on Windows — the same directory as config files and your personal specialists. Same format and 8 KiB cap as the project files above, and the same case-insensitive basename match.

This file is injected as its own `## User guidelines` section, before the project's `AGENTS.md`/`ZERO.md`, and is labeled as personal preference in the prompt: project guidelines are the later, more specific instruction and take precedence over it when the two conflict.

## 2. Custom specialists

Specialists are reflect's sub-agents. Three scopes, in priority order:

| Scope | Path | Shared? |
| --- | --- | --- |
| Built-in | compiled into reflect | yes |
| User | `~/.reflect/specialists/*.md` | no — your machine only |
| Project | `./.zero/specialists/*.md` | yes — the repo team |

Project overrides user overrides built-in when names collide.

A specialist is a markdown manifest with frontmatter and a system prompt:

```markdown
---
description: Reviews API changes for breaking-change risk and missing tests.
tools: read-only,plan
---

You review API changes. For every changed hunk in `internal/api/` or any file
that ends in `_api.go`:

1. Confirm the public signature is backward-compatible, or note the breaking
   change explicitly with the migration path.
2. Confirm a corresponding test exists in `internal/api/*_test.go` and that
   the new behaviour is exercised.
3. Flag any new exported symbol without a doc comment.

Reply with one JSON object per finding: `{"file", "line", "severity", "message", "fix"}`.
```

CLI management:

```bash
reflect specialist list
reflect specialist show api-reviewer
reflect specialist create api-reviewer \
    --project \
    --description "Reviews API changes" \
    --tools read-only,plan \
    --prompt "$(cat api-reviewer.md)"
reflect specialist edit api-reviewer --project
reflect specialist delete api-reviewer --project
reflect specialist path                       # prints the resolved specialists directory
```

## 3. Skills

reflect ships **no bundled skills** by default. Skills are markdown instruction
files that extend agent capabilities, sourced from the separate
`GrayCodeAI/graycode-skills` repo and installed on demand:

```bash
reflect skills search <query>          # find skills in graycode-skills
reflect skills install <owner/repo> [skill-name]   # install after user approval
reflect skills list                    # list installed skills
reflect skills remove <name>
```

Installed skills live in user or project scope:
- User-scoped: `~/.reflect/skills/`
- Project-scoped: `./.zero/skills/` or `./skills/`

A skill manifest:

```markdown
---
description: How to review Go code for security issues
globs: "*.go"
alwaysApply: true
---

When reviewing Go code for security:

1. Check for SQL injection patterns
2. Verify error handling doesn't expose sensitive data
3. Confirm secrets are not hardcoded
4. Validate input sanitization
```

## 4. Hooks

Hooks allow custom commands to run at specific lifecycle points:
- `beforeReview` — runs before code review starts
- `afterReview` — runs after code review completes
- `sessionStart` — runs at session initialization
- `sessionEnd` — runs at session teardown

```bash
reflect hook add beforeReview --command "lint-check"
reflect hook remove beforeReview
reflect hook list
```

## 5. MCP integration

MCP (Model Context Protocol) servers can expose tools to reflect:

```bash
reflect mcp add --name server --url http://localhost:8080
reflect mcp remove server
reflect mcp list
```

## 6. Plugins

Plugins extend reflect with custom tools and capabilities:

```bash
reflect plugin add --name my-plugin --path ./my-plugin
reflect plugin remove my-plugin
reflect plugin list
```

## 7. Verification

reflect includes a self-verification system to validate local changes before contributing:

```bash
reflect verify
reflect verify --fix
```

## Development

```bash
make lint
reflect verify
```

### Architecture note: cross-repo contracts

Cross-repo severity and finding contracts now live in reflect's `internal/contracts` (vendored from the removed `github.com/GrayCodeAI/eagle` module) — extensions and support repos must vendor the needed DTOs instead of Reflect internals until a published contracts module exists.

### Architecture note: provider ownership

Implement provider protocols, adapters, catalog metadata, credential mappings, and
provider contract tests in `../relay` (the relay engine's repo) first. Reflect consumes providers only
through Relay's stable engine facade; Reflect changes should be limited to host UX
and facade integration. Concentrate AI is a pay-as-you-go gateway implemented
with its native Responses API (`/v1/responses`) under the
`concentrate-payg` deployment.

### Workspace workflow (sibling repos)

reflect depends on ecosystem repos (`relay`, etc.) as independent sibling repos in the `graycode-eco` workspace. Reflect's `go.work` lists them as `../<repo>`, so local changes in any sibling are automatically picked up by reflect. Each sibling is its own git repo, versioned and released independently.

1. Edit + test in `../<repo>` — run its tests, run `make test` in reflect
2. Push from the sibling: `git push origin <branch>`
3. Open a PR in the sibling repo → merge to `main`
4. Ensure reflect's `go.mod` pins a version that resolves to (or is an ancestor of) the sibling's `main` — run `make sync` to verify parity
5. No pointer commits: reflect resolves the sibling via `go.work` for local dev and via the pinned `go.mod` version for standalone/module-mode builds (Docker, released consumers)

## Cloudflare

This repo has no Cloudflare deployment. If one is added it must follow the Vaultly-only policy (vault `cf-graycodeai`, account f4b522a3ebdf147fa6d51fa475331b1c) used by graycode-platform.
