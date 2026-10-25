# App hangs if terminals don't respond to termenv.HasDarkBackground()

[<brendan@berg.industries> 2026-10-14T23:56:02-07:00]

- state: closed
- type: bug
- assignees: <brendan@berg.industries>

---

[<brendan@berg.industries> 2026-10-14T23:56:02-07:00]

During work on #3CTTK-4AQMZ, <noreply@anthropic.com> discovered that cards call `termenv.HasDarkBackground()` every time they're drawn, and that call can ask the terminal for its background colour.

In Claude's test terminal, which never answers, opening any issue stalled. The previous commit stalls the same way, so Claude's changes didn't cause it, and Claude tested with `TERM=screen-256color` to avoid it.

Terminals that answer seem fine. It might be worth its own issue, since the issue view already receives the background colour from Bubble Tea.

---

["Brendan Berg" <brendan@berg.industries> 2026-10-24T23:06:43-07:00]

As discussed in this issue https://github.com/charmbracelet/lipgloss/issues/73:

> Is there any update on having this issue fixed within `lipgloss` itself?
>
> My understanding is that the issue arises because `Bubbletea` holds a lock on the `os.Stdout` file descriptor, which prevents `lipgloss` from performing terminal background color checks required for adaptive colors. 
>
> Calling `lipgloss.SetHasDarkBackground(termenv.HasDarkBackground())` does indeed fix the issue. A potential alternative solution might involve using a shared mutex to synchronize access to the file descriptor, but this would only be needed if the background color was changing (though I'm sure there's also some caching involved that would need to be updated too).

The discussion implies the issue has been resolved in Lip Gloss v2. However, I believe we're using v2 and are still encountering a similar issue.

Bubble Tea and Lip Gloss are competing for use of `stdout` which Lip Gloss needs to query to determine the background color.

It's pretty gross but you can work around the issue by simply performing the background query prior starting Bubble Tea:

```go
_ = lipgloss.DefaultRenderer().HasDarkBackground() // ugh

if _, err := p.NewProgram(model{}).Run(); err != nil {
    fmt.Fprintf(os.Stderr, "Oof: %v", err)
}
```

---

[Claude <noreply@anthropic.com> 2026-10-24T23:40:15-07:00]

## What was happening

Each comment card called `termenv.HasDarkBackground()` every time it rendered its markdown, and the issue view re-renders every card on each cursor move, scroll, and keystroke while editing. termenv doesn't cache the answer by default, so each call wrote a background-colour query and a cursor-position query to the terminal, then waited up to 5 seconds per byte for the reply. It skips the query when `TERM` starts with `screen`, `tmux` or `dumb`, when output isn't a terminal, or when ish isn't the foreground process.

- **Why you haven't seen it:** a real terminal answers within milliseconds, so each render only cost a few unnoticed round trips. Inside tmux or screen, the query never happens at all. My test terminal accepted the queries but never answered, so every card waited out the timeout on every render.
- **Why it was worth fixing anyway:** Bubble Tea reads the same input at the same time, so if it read the reply first, termenv waited out the timeout even in a terminal that answers. Over SSH, every render paid the network round trip several times over. And inside tmux or screen, a light background was never detected, so cards always used the dark style.
- **About the Lip Gloss workaround:** it fixes Lip Gloss v1, which made the same query implicitly through `lipgloss.DefaultRenderer()`. Lip Gloss v2 doesn't query the terminal itself, and that function no longer exists. The query here was ish's own termenv call, so querying once before Bubble Tea starts wouldn't have stopped the per-render queries.

## The change

Cards now use the background colour Bubble Tea already reports, which the app and the issue view request at start-up.

- Each card has a `light` setting, and renders its markdown and code blocks from it rather than querying the terminal. Until Bubble Tea answers, the background is assumed dark, as the rest of the issue view already assumes.
- The issue view passes Bubble Tea's answer to every card and redraws them. New comments start with the current setting, and so does an issue reloaded after an outside change.
- Nothing in ish uses termenv any more, so `go mod tidy` removed it, and its `go-osc52` dependency, from `go.mod`. The vendored `bubble-dropdown` still uses it in its own tests, under its own `go.mod`. termenv can stay on the approved list or come off it, as you prefer.

Tests are in `models/background_test.go`: cards render for a dark background by default and a light one once set, Bubble Tea's answer reaches every card and the viewport, a change back to dark is applied, and new comments and reloads keep the setting. The whole suite passes.

In a real terminal (tmux, with its default `TERM`), this issue opens without stalling. Its code block is painted dark in a dark pane and light in a pane with a white background. Before, it was always dark under tmux.
