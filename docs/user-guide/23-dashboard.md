# Dashboard and HUD

The Reflect dashboard provides system status and monitoring information.

---

## Opening the Dashboard

In the TUI:

```
/ecosystem         # Ecosystem status (Relay and token pipeline)
/path              # Developer path readiness
/preflight         # Quick health check
```

---

## Ecosystem Status

Shows the status of all Reflect components:

| Component | Status |
|-----------|--------|
| Relay (providers) | Ready / Error |
| token pipeline | Available |
| memory | Ready |

---

## Developer Path

Check readiness to chat:

```bash
reflect path
```

This verifies:
- Configuration setup
- Credential status
- Security readiness

---

## Usage Display

View credit usage and cost:

```
/usage
```

Shows:
- Token consumption
- Estimated costs
- Model breakdown

---

## Where to Go Next

| Document | What You Will Learn |
|----------|-------------------|
| [Monitoring Usage](24-monitoring-usage.md) | Telemetry details |

---

© 2026 GrayCode AI. All rights reserved.