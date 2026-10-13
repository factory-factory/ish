# Update Issue list view to have filter field

[brendan@berg.industries 2026-10-12T23:13:26-07:00]

- state: closed
- assignees: brendan@berg.industries
- relation.parent: #3CSJ4-PEZ1N

---

[brendan@berg.industries 2026-10-12T23:13:26-07:00]

Update the Issue list view to use filter expressions to select the issues to display in the table.

## Requirements

- Remove the title that says "Issues" at the top of the screen. In its place, put "Issue Filter" in white text above a single row, vertically expandable text area.
- The text area should have a gray border when not selected and a blue border when selected, like the comment cards in the Issue view.
- When editing, the text area should display the raw filter string
- Upon committing the edit, the filter string should be parsed and the styled AST nodes should be renderd in the text area
- Additionally on commit, the WHERE clause to be used in the issue query should be generated and the updated query should be executed, refreshing the table contents

---

[Claude 2026-10-12T23:33:19-07:00]

I've implemented the filter field. Two points need your decision, so this stays `in progress` for now. The behaviour below follows my recommendations, so either answer is a small change. Nothing is committed yet.

## Questions

1. **The styling depends on #3CVJB-3HXW2.** This issue asks for the applied filter to be shown as "styled AST nodes", but how they're styled (blue braces, magenta attributes and so on) is #3CVJB-3HXW2, which puts the styling in a `Render` method on the nodes. For now, the field shows the parsed expression as plain text from `String()`, through a single `renderFilter` function. #3CVJB-3HXW2 can then switch that function to the nodes' `Render` without changing the field. Is that split OK, or would you rather I did #3CVJB-3HXW2 first, or as part of this issue?
2. **The starting filter.** The field starts empty, which lists every issue, as the list did before. The parent issue (#3CSJ4-PEZ1N) describes a default "All unclosed issues" filter, `{state != 'closed'}`, but that's part of the saved-filters selector, which isn't an issue yet. Should the field start with `{state != 'closed'}` now instead? That would hide closed issues by default.

## How it works

- "Issues" at the top is replaced by **Issue Filter** in white, above the field.
- The field has a grey rounded border, and a blue thick border while it's selected, matching the selected comment cards.
- **Selecting and editing**, which the issue didn't specify, works like the issue view:
  - `↑` from the top row of the table selects the field, and `↓` (or `j`) goes back to the table.
  - Enter on the selected field edits it, and `/` from anywhere in the list jumps straight to editing.
  - While editing, the field shows the raw filter text as you typed it. It starts as one row and grows as the text wraps, up to 5 rows, then scrolls.
  - Every key goes into the field, except `ctrl+c`, which quits.
- **Applying (enter)** parses the text, generates the `WHERE` clause, and runs the list query again. The field then shows the parsed expression and stays selected. Editing again brings back the text exactly as you typed it.
- **Other cases:**
  - An empty filter lists every issue, and the field shows the grey hint "No filter: every issue is listed".
  - If the text doesn't parse, you stay in the field, and the error (for example `column 10: expected a value`) is shown under it in red. The list is unchanged.
  - `esc` cancels an edit and keeps the filter that was applied.
  - If no issues match, the list says "No issues match the filter".
- The filter stays applied when you return from an issue, so an issue you edit so that it no longer matches drops out of the list.
- The help footer shows the keys for whichever part has the focus.

## Code

- `store.ListIssues` takes an optional filter and adds its `WHERE` clause and parameters to the list query.
- The field is in the new `app/filter.go`, with its styles in `styles/style.go`.

## Testing

- Tests cover:
  - the initial field and its grey border
  - editing and applying, including the raw text coming back when editing again
  - an invalid filter, then fixing it
  - cancelling, and clearing the filter
  - selecting the field with `↑`, editing it with enter, and returning with `↓`
  - keys being typed into the field, while `ctrl+c` still quits
  - the filter staying applied after closing an issue in the issue view
  - no matches
  - the field growing with long text while the screen stays the terminal's height
- I ran ish on a copy of `.ish` and applied `{state != 'closed'}`. The label and the parsed filter were shown, and none of the 10 closed issues appeared afterwards.

---

[brendan@berg.industries 2026-10-12T23:35:28-07:00]

1. Good catch. Put off styling until we wrap this up and move on to #3CVJB-3HXW2.

2. Yes, the default value for the filter should be `{state != 'closed'}`.

---

[Claude 2026-10-12T23:40:13-07:00]

Done, following your answers, and moved to `in review`.

1. **Styling:** left for #3CVJB-3HXW2, as you said. The applied filter is still shown as plain text through `renderFilter` in `app/filter.go`, which is the one function #3CVJB-3HXW2 needs to change.
2. **Default filter:** the list now starts with `{state != 'closed'}` in the field and applied, so closed issues are hidden until you change it. On a copy of `.ish`, ish starts showing your 11 unclosed issues and none of the 10 closed ones.

Changes this needed:

- `app.New` now takes just the directory, applies the default filter (`app.DefaultFilter`), and loads the matching issues itself, so the list owns its filter. `main.go` no longer loads the issues.
- When nothing is listed, the list now tells an empty project from a filter that matches nothing. It counts all issues with a new `store.CountIssues`, then says "No issues in .ish" if there are none, or "No issues match the filter" if there are. With the default filter, a project whose issues are all closed would otherwise have said it had no issues.
- A few existing list tests used `closed` as an example state and expected the issue listed. They now use `in review`, which keeps what they test. The filter field's tests clear the default before typing a new filter.

Added tests for the default filter at startup, both empty-list messages, clearing the filter to show closed issues, and `store.CountIssues`.
