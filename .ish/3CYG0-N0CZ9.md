# App hangs if terminals don't respond to termenv.HasDarkBackground()

[brendan@berg.industries 2026-10-14T23:56:02-07:00]

- state: open
- type: bug
- assignees: brendan@berg.industries

---

[brendan@berg.industries 2026-10-14T23:56:02-07:00]

During work on #3CTTK-4AQMZ, Claude discovered that cards call termenv.HasDarkBackground() every time they're drawn, and that call can ask the terminal for its background colour. In Claude's test terminal, which never answers, opening any issue stalled. The previous commit stalls the same way, so Claude's changes didn't cause it, and Claude tested with `TERM=screen-256color` to avoid it. Terminals that answer seem fine. It might be worth its own issue, since the issue view already receives the background colour from Bubble Tea.
