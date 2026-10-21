# Allow users to add comments to issues while editing

[<brendan@berg.industries> 2026-10-11T10:20:04-07:00]

- state: closed
- assignees: <brendan@berg.industries>

---

[<brendan@berg.industries> 2026-10-11T10:20:04-07:00]

It's not currently possible to append comments to an Issue in the Issue view. Create a new keybinding so that pressing `c` will append a new comment card at the end of the issue, update the selection index so the new card is selected, and put the card in editing mode.

---

[Claude <noreply@anthropic.com> 2026-10-11T10:31:38-07:00]

Done, and moved to `in review`; nothing is committed yet.

- In the issue view, `c` appends an empty comment card at the end of the issue, selects it, scrolls it into view, and starts editing it. As with other comments, `ctrl+s` saves and `esc` cancels. The help footer lists `c comment`.
- The new comment's byline is the current user, from the same lookup new issues use (git's `user.email`, falling back to the OS user name), and the current time in RFC 3339 format.
- If the new comment is cancelled, or saved empty or blank, it's removed and the comment before it is selected again, so an empty comment is never written to the file. A comment that's saved with text marks the issue as changed, and it's written when the issue is closed or the program quits, like other edits.
- `c` works from anywhere in the issue view, including while selecting a comment's todo items. While a field is being edited, `c` is typed into the editor as usual.
- `models` can't run git, so the issue gets its comment author through a new `Issue.SetAuthor`, like `SetSave`. The issue list and `ish <path>` both set it. Without an author, `c` is disabled. The lookup is now exported as `app.CurrentUser` so `main` can use it.
- Added tests for adding, saving and abandoning a new comment (cancelled, empty and blank), an issue with no comments, `c` without an author, `c` while editing and while selecting todos, scrolling the new comment into view, and an end-to-end test that writes a new comment to an issue file through the list.

---

[<brendan@berg.industries> 2026-10-11T10:32:54-07:00]

Testing the new behavior in a fresh build. So far it looks good.
