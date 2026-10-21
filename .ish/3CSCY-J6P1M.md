# Add assignee column to Issue list view

[<brendan@berg.industries> 2026-10-11T08:58:11-07:00]

- state: closed
- assignees: <brendan@berg.industries>

---

[<brendan@berg.industries> 2026-10-11T08:58:11-07:00]

Add a column to the list view to show the assignee of each issue. The column should be to the right of the title.

---

[Claude <noreply@anthropic.com> 2026-10-11T09:24:47-07:00]

Implemented in the working tree; nothing is committed yet.

- The issue list has an **Assignee** column between Title and State.
- Values come from the `assignees` tag, or from `assignee`: this issue uses the singular key, while the older issues use the plural. If an issue has both, all of their values are shown, separated by commas.
- The column is as wide as its longest value, up to 30 characters, so titles keep most of the width. Longer values are cut off with "…".
- Values are shown as written. 3CRTZ-EBD9H gives its assignee as `<brendan@berg.industries>`, so the brackets appear in the list. Editing that issue to drop them would make it consistent.
- Added `Issue.Assignees()` in `models`, with tests, and list tests for the column's position, alignment and truncation.

Unrelated fix: moving the issues to `.ish/` broke two `parse` tests that read sample issues from `../issues`. They now read from `IssuesDir`, so they follow the directory if it moves again. `agents.md` still says the sample issues are in `issues/`.

Questions:

- Should the `assignee` and `assignees` keys be standardised on one name? The column handles both either way.
- Should the state of this issue change now? I've only appended this comment, so it still says `open`.

3CSCZ-5858S (an "Opened" column) is a separate issue and isn't done.

---

[<brendan@berg.industries> 2026-10-11T09:23:48-07:00]

In response to the previous questions:

- Please update the issue to `in progress`.
- The key name should be standardized on `assignees`.
- If you think the task is complete, update the issue to `in review`.

---

[Claude <noreply@anthropic.com> 2026-10-11T09:33:40-07:00]

Done, and moved to `in review`; nothing is committed yet.

- The Assignee column now reads only the `assignees` key. `Issue.Assignees()` ignores `assignee`, and a test checks that.
- I renamed `assignee` to `assignees` in the two issues that used it: this one and 3CRTZ-EBD9H. Only that line changed in each. Every issue with assignees now uses `assignees`, and both still show their assignee in the list.
- 3CRTZ-EBD9H's value still has its angle brackets, `<brendan@berg.industries>`, since that wasn't part of this change.
