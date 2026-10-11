# Memory

Memory lets Decoder recall facts, decisions, and patterns from earlier sessions. Decoder indexes saved information and searches it automatically, so new sessions can reuse relevant context.

---

## What Is Memory?

Without memory, each Decoder session starts fresh. When memory is enabled, Decoder can:

- Recall project conventions you explained before
- Reuse debugging steps that worked
- Carry architectural decisions forward across sessions
- Avoid re-asking questions it already has answers to

Memory is a local subsystem built into Decoder. It stores memories as files under
Decoder's state directory; no external service is required.

---

## Enabling Memory

Memory is on by default. To inspect what has been stored:

```
/learn    # lessons learned across sessions
```

### Settings

```json
// ~/.decoder/settings.json
{
  "memory": {
    "enabled": true
  }
}
```

---

## How Memory Is Stored

Memory is stored under Decoder's state directory (`~/.decoder/` by default).

| Location | Scope | Description |
|----------|-------|-------------|
| `~/.decoder/memories/` | Global | Core memories and preferences |
| `~/.decoder/` state files | Workspace | Project-specific memory and lessons |

Decoder combines core memories, automatic captures, evolving guidelines, and
retrieval metrics to rank what is relevant to the current turn.

---

## Working with Memory

### Remember

Ask Decoder to remember something:

```
/remember always open PR links after pushing
```

Decoder records entries as durable statements organized by topic.

### Forget

Ask what Decoder should forget:

```
/forget the snake_case convention
```

Forget is best-effort.

### Recall

Ask what Decoder remembers:

```
/what do you remember about auth?
```

Decoder searches across all memory sources and summarizes.

---

## Browsing Memory

Lessons learned are surfaced through:

```
/learn
```

---

## First-Turn Injection

On the first turn of each session, Decoder automatically searches memory for content relevant to the current project and injects it as context.

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