# Memory

Memory lets Reflect recall facts, decisions, and patterns from earlier sessions. Reflect indexes saved information and searches it automatically, so new sessions can reuse relevant context.

---

## What Is Memory?

Without memory, each Reflect session starts fresh. When memory is enabled, Reflect can:

- Recall project conventions you explained before
- Reuse debugging steps that worked
- Carry architectural decisions forward across sessions
- Avoid re-asking questions it already has answers to

Memory is a local subsystem built into Reflect. It stores memories as files under
Reflect's state directory; no external service is required.

---

## Enabling Memory

Memory is on by default. To inspect what has been stored:

```
/learn    # lessons learned across sessions
```

### Settings

```json
// ~/.reflect/settings.json
{
  "memory": {
    "enabled": true
  }
}
```

---

## How Memory Is Stored

Memory is stored under Reflect's state directory (`~/.reflect/` by default).

| Location | Scope | Description |
|----------|-------|-------------|
| `~/.reflect/memories/` | Global | Core memories and preferences |
| `~/.reflect/` state files | Workspace | Project-specific memory and lessons |

Reflect combines core memories, automatic captures, evolving guidelines, and
retrieval metrics to rank what is relevant to the current turn.

---

## Working with Memory

### Remember

Ask Reflect to remember something:

```
/remember always open PR links after pushing
```

Reflect records entries as durable statements organized by topic.

### Forget

Ask what Reflect should forget:

```
/forget the snake_case convention
```

Forget is best-effort.

### Recall

Ask what Reflect remembers:

```
/what do you remember about auth?
```

Reflect searches across all memory sources and summarizes.

---

## Browsing Memory

Lessons learned are surfaced through:

```
/learn
```

---

## First-Turn Injection

On the first turn of each session, Reflect automatically searches memory for content relevant to the current project and injects it as context.

Configure injection:

```json
{
  "memory": {
    "first_turn_injection": true
  }
}
```

---

## Where to Go Next

| Document | What You Will Learn |
|----------|-------------------|
| [Headless Mode](14-headless-mode.md) | Non-interactive usage |
| [Agent Mode](15-agent-mode.md) | Agent configuration |

---

© 2026 GrayCode AI. All rights reserved.