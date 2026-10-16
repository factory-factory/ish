# Dim the selected row when the issue list is not in focus

[brendan@berg.industries 2026-10-15T22:50:57-07:00]

- state: closed
- assignees: brendan@berg.industries

---

[brendan@berg.industries 2026-10-15T22:50:57-07:00]

When navigating up to the filter menu by pressing the up arrow from the first issue, the selected issue currently remains highlighted with a blue background.

- Add an unfocused state to the issue table that renders the selected issue in normal (non-bold) text with a grey (8) background.
- If the table loses focus because the filter menu is selected by pressing `/`, the currently selected issue should not change.

---

[Claude 2026-10-15T22:57:15-07:00]

Done, and moved to `in review`. I had no questions.

- **While the filter has the focus,** the selected issue is in normal (non-bold) text on a grey (8) background. That covers the filter menu, the custom field, its buttons, and editing a filter. The text keeps the terminal's default colour.
- **When the table has the focus again,** the selected issue goes back to bold bright white on blue.
- **How:** the bubbles table draws its selected row the same way whether or not it has the focus. So ish now switches the table's styles whenever the focus moves between the table and the filter. The styles come from a new `tableStyles` function in `app/app.go`.
- **The selection doesn't change** when `/` or `↑` moves the focus to the filter. Applying the same filter and returning to the table keeps it too. This already worked, and it's now tested.

## Testing

- The focused and unfocused styles: bold on blue, and normal text on grey.
- `↑` from the first issue dims its row and keeps it selected. `↓` highlights it again.
- `/` from the third issue keeps it selected and dims its row. Applying the filter and returning to the table keeps it selected and highlighted.
- In a real terminal, on a copy of `.ish`, the first issue was bold bright white on blue. After `↑` it was normal text on grey, and after `↓` it was highlighted again.
