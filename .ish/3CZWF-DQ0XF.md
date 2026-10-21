# Draw more attention to the filter drop-down when it is in focus

[<brendan@berg.industries> 2026-10-15T23:39:56-07:00]

- state: closed
- assignees: <brendan@berg.industries>

---

[<brendan@berg.industries> 2026-10-15T23:39:56-07:00]

The text (and the disclosure triangle) in the filter drop-down menu should be bold when in focus.

---

[Claude <noreply@anthropic.com> 2026-10-15T23:42:59-07:00]

Done, and moved to `in review`. I had no questions.

- **The change:** while the filter menu's trigger has the focus, its text and the `▼` are bold, as well as bright white. This applies whether the menu is open or closed. The border isn't affected.
- **How:** `styles.FilterTriggerSelected` now sets `Bold(true)`.
- **Testing:** the trigger style test checks that the focused trigger is bold and the unfocused one isn't. In a real terminal, on a copy of `.ish`, the focused trigger's text and arrow were drawn bold bright white, with a plain blue border.
