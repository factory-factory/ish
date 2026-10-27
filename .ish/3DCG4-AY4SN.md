# Usability improvements

["Brendan Berg" <brendan@berg.industries> 2026-10-24T22:56:39-07:00]

- state: closed
- assignees: <noreply@anthropic.com>

---

["Brendan Berg" <brendan@berg.industries> 2026-10-24T22:56:39-07:00]

## List of improvements

- [x] Change "Assignee" column to "Assignees"
- [x] Typing `/` in list view should jump to the filter area but not change the drop-down to the custom text field
- [x] Render ID column in blue, Assignees in aqua

---

[Claude <noreply@anthropic.com> 2026-10-26T23:09:20-07:00]

All three improvements are done. I haven't ticked the checkboxes above, because the project rules don't let me edit existing comments.

## Change "Assignee" column to "Assignees"

The issue list's header now reads **Assignees**. The column's width is still capped by `maxAssigneeWidth`.

## `/` jumps to the filter area without switching to the custom field

`/` now just selects the filter row, the same as moving up from the top of the list, and keeps the selected issue. If the filter menu is showing, the menu is selected but not opened or changed, so you can press enter to choose a preset. If the custom field has already replaced the menu, the field is selected but you're not editing it yet; press enter to edit. To start a custom filter, choose **Custom...** from the menu. I took `/` out of the help footer shown while the menu is selected, since pressing it there does nothing now.

## Render the ID column in blue and Assignees in aqua

`styles.IssueID` (color 4) and `styles.Assignees` (color 6) color the text of those two cells. The selected row is the one exception: it's drawn plain. The table library draws the selected row by wrapping the whole row in its blue highlight, and a colored cell's reset code cuts that highlight off partway across the row. A blue ID would also disappear on the blue highlight. The rows are re-rendered whenever the cursor moves, so the colors follow the selection.

## Tests

- `TestRowColors` checks that the colors follow the cursor.
- `TestSlashSelectsFilterMenu` replaces `TestSlashEditsCustomFilter`.
- `TestSlashKeepsSelectedIssue` now checks that `/` selects the filter row.
- Tests that used to press `/` to edit a custom filter now go through a new `editCustom` helper, which chooses **Custom...** from the menu.

Moving this to `in review`.
