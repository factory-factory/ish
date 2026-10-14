# Implement the filter drop-down selector

[brendan@berg.industries 2026-10-13T21:32:20-07:00]

- state: closed
- assignees: brendan@berg.industries
- relation.parent: #3CSJ4-PEZ1N

---

[brendan@berg.industries 2026-10-13T21:32:20-07:00]

Update the Issue list view to have a drop-down menu for selecting active filters. The top row should be layed out with a label followed by the drop down menu.

```
Filter: [ All unclosed issues                                  ▼ ]
```

Selecting an item from the menu should apply the filter to the issue list.

The menu should have labels for the following filters:

- All unclosed issues
- Issues assigned to me
- Custom...

The custom menu item should replace the drop-down menu with the custom filter widget created in #3CVMN-Q41P7.

This feature should use the `bubble-dropdown` module (which has been added to the allowed modules list).

---

[Claude 2026-10-13T22:36:29-07:00]

The filter menu is implemented and its tests pass, but adding `bubble-dropdown` brought in a start-up problem that needs your decision before this can be finished. This stays `in progress`, and nothing is committed yet.

## The problem: Bubble Tea v1 runs at start-up

- `bubble-dropdown` is built on Bubble Tea v2, like ish. But to draw its panel it imports the root package of `github.com/madicen/bubble-overlay`, which imports **Bubble Tea v1** and **Lip Gloss v1**. Both are now compiled into ish alongside v2.
- Bubble Tea v1 has a package `init` function that runs before `main()`. It asks Lip Gloss v1 whether the terminal is dark, so `termenv` writes a background-colour query to the terminal and waits up to 5 seconds for the reply on standard input.
- In my test terminal, which never answers that query, ish stalled before drawing anything, and the keys I typed were read as part of the expected reply. A build of the last commit, without the dropdown, starts and quits normally in the same terminal. A goroutine dump shows the wait inside `github.com/charmbracelet/bubbletea.init.0`.
- Most terminals answer the query quickly, so this may not be visible in yours. But it means every `ish` command, `ish init` included, now does terminal I/O before Bubble Tea v2 has set up the terminal. It can swallow keys typed during start-up, and in terminals that don't answer the query (some tmux setups, for example) ish waits 5 seconds before starting.
- `bubble-overlay` has a `v2` subpackage, but it doesn't include the functions the dropdown uses (`OverlayView`, `ModalCellSize`, `CellInModal`, `Fixed`), so the dropdown can't simply be pointed at it.

## Options

- (a) **Accept it as it is.** It works in terminals that answer the query, with the risks above.
- (b) **Patch `bubble-overlay` locally**, like glamour: vendor it into `third_party` and use it through a `replace` directive. In the root package, only `focus.go` and `stack.go` import Bubble Tea v1. They hold the overlay's model-stacking helpers, which the dropdown doesn't use. Without them, Bubble Tea v1 and its start-up query should disappear, though Lip Gloss v1 would still be linked. I haven't built this yet, so I'd confirm it compiles and that the stall is gone.
- (c) **Report it to `bubble-dropdown`/`bubble-overlay`**, and wait for a release that doesn't depend on v1.

I recommend (b): it's small, it keeps the dropdown you chose, and it removes the problem completely. It's a change to how a dependency is sourced, so it needs your approval under `agents.md`.

## What's built

- **The filter row:** "Issue Filter" is replaced by a single row, `Filter: [ All unclosed issues ▼ ]`, using `bubble-dropdown` with the blue accent of the selected card borders.
- **The menu's items:** "All unclosed issues" (`{state != 'closed'}`, the default), "Issues assigned to me", and "Custom...". Choosing one applies its filter to the list.
- **Keys:**
  - `↑` from the top row of the table selects the menu, and enter or space opens it.
  - In the open menu, `↑`/`↓` move, enter chooses and `esc` cancels.
  - `↓` returns to the table.
  - While the menu is selected but closed, `↑`/`↓` don't change its selection. The dropdown would otherwise change it without applying it.
  - Keys typed while the menu is open go to the menu, except `ctrl+c`.
- **Custom...** replaces the menu with the custom filter field from #3CVMN-Q41P7, already editing, and starting from the filter that's applied so it can be adjusted. `/` from the table does the same.
- **Leaving the custom filter:** `esc` on the custom field, while it's selected but not being edited, returns to the menu and re-applies the filter chosen from it before.
- **Help:** the help footer shows the keys for the menu, the open menu and the custom field.
- **Width:** the mock-up shows the menu spanning the row. `bubble-dropdown` sizes its trigger to its longest item and has no option to make it wider, so it uses that natural width.

## Questions

1. **The dependency problem above:** (a), (b) or (c)?
2. **"Issues assigned to me"** uses the same identity as new issues and comments (your git `user.email`, through `app.CurrentUser`), as `{assignees = "you@example.com"}`. Taken literally, it includes closed issues. Should it leave them out (`... and {state != 'closed'}`)? Also, some older issues give assignees as names (`brendanberg`) or in angle brackets (`<brendan@berg.industries>`). Those won't match your email, so they won't be listed.

## Other changes

- `TestHighlightStyles` in `filter` was failing before I started. Your "style tweak" commit changed `null` from grey to bright yellow (11), but the test still expected grey. I've updated the test, and the comment in `filter/render.go`, to your colour.
- `go.mod` now lists `bubble-dropdown` as a direct dependency.

## Testing

- Tests cover:
  - the filter row's layout and the default filter
  - selecting the menu, and `↑` not changing its selection
  - opening the menu and seeing every item
  - choosing each item, and choosing a filter again
  - quoting user names (including quotes and backslashes) in the "assigned to me" filter
  - Custom... and `/` starting from the applied filter
  - returning from the custom field to the menu
  - cancelling the menu
  - keys while the menu is open, with `ctrl+c` still quitting
- I couldn't check the menu in a real terminal because of the start-up problem above.

---

[brendan@berg.industries 2026-10-13T22:39:08-07:00]

1. I think the best way to resolve the start-up issue is to vendor and patch bubble-dropdown.
2. The query for "Issues assigned to me" should include `... and {state != 'closed'}`.

---

[Claude 2026-10-14T08:52:33-07:00]

Done, following your answers, and moved to `in review`.

## 1. The start-up problem: patched `bubble-dropdown`

- `bubble-dropdown` v2.0.1 is vendored in `third_party/bubble-dropdown`, with its MIT licence, and `go.mod` points at it with a `replace` directive, like glamour.
- The patch removes the dropdown's import of `bubble-overlay`, the source of Bubble Tea v1. Its four uses are replaced in a new `third_party/bubble-dropdown/overlay.go`, marked "Local patch for ish":
  - `modalCellSize`, `cellInModal` and `clampedOrigin` are ported from `bubble-overlay` (also MIT, same author, credited in the file).
  - The panel is drawn over the list with Lip Gloss v2's `Compositor` and layers, instead of `bubble-overlay`'s Lip Gloss v1 renderer.
- **Bubble Tea v1, Lip Gloss v1 and `bubble-overlay` are now gone** from ish's `go.mod` and from the packages built into it.
- The vendored dropdown's own test suite passes with the patch.
- In the same test terminal that stalled before, ish now starts and quits on `q` at once. Opening the menu, choosing "Issues assigned to me" and quitting worked, and the menu's trigger showed the chosen filter afterwards.

## 2. "Issues assigned to me" leaves out closed issues

Its filter is now `{assignees = "you@example.com"} and {state != 'closed'}`, using your git `user.email`.

## Testing

- The menu tests from my previous comment pass, with "Issues assigned to me" now expected to leave out closed issues.
- The preset filter test checks that the filter requires both the assignee and `{state != 'closed'}`.

Everything else is as described in my previous comment.
