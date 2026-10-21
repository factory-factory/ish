# Allow adding and deleting metadata fields to an issue

[<brendan@berg.industries> 2026-10-11T10:57:08-07:00]

- state: closed
- assignees: <brendan@berg.industries>
- type: feature

---

[<brendan@berg.industries> 2026-10-11T10:57:08-07:00]

In the Issue view, there is currently no mechanism to add or delete metadata fields. A new keybinding should be added so that `m` creates a new Tag in the metadata array.

## Adding a metadata field

A new, completely blank Tag should be appended to the Metadata list. Since a blank Tag renders unintuitively, the Key field should be editable immediately. After the user presses enter to set the Key, the Value field should become editable. Once both the Key and Value fields are set, the Tag should be taken out of editing mode but remain selected.

If the Tag is taken out of editing mode having an empty Key field, the Tag should be deleted from the Metadata list.

## Removing a metadata field

When a Tag is selected, pressing delete should highlight the entire Tag with red text on a grey background. Pressing delete again should remove the tag. Any other key should put the Tag back into the selected state.

---

[Claude <noreply@anthropic.com> 2026-10-11T11:19:25-07:00]

Done, and moved to `in review`.

Adding a metadata field:

- `m` appends a blank tag to the metadata, selects it, and starts editing its key. Enter sets the key and moves on to editing the values, which are comma-separated like other tags. Enter with values set stops editing, and the tag stays selected.
- A tag that leaves edit mode with an empty key is removed, as the issue says, and the tag before it (or the title, if there are none) is selected again.
- Two cases the issue didn't cover:
  - Enter with no values keeps editing the values, since the tag should only stop editing once both are set.
  - Pressing `esc` while editing the values also removes the new tag. A key without a value can't be written to the issue file (the format needs `key: value`), so keeping it would make saving fail when leaving the issue. Tell me if you'd rather keep it and handle that another way.
- Keys can't contain spaces in the file format, so typed spaces are ignored and pasted whitespace is removed.

Removing a metadata field:

- With a tag selected, pressing delete highlights the whole tag in bold bright red on dark grey (ANSI colors 9 and 8). Pressing delete again removes it, and selects the tag that took its place, the tag before it, or the title.
- On Mac keyboards the key labelled "delete" sends backspace, so both backspace and forward delete work.
- Any other key cancels and the tag goes back to being selected. The key isn't also acted on, so pressing `j` to cancel doesn't move the selection. Moving the selection away also cancels.
- Backspace on the title or a comment does nothing, and while a field is being edited it deletes a character as usual.

Other details:

- Added and deleted fields mark the issue as changed, and they're written to the file when the issue is closed or the program quits, like other edits. An unfinished new tag is discarded when quitting.
- The help footer lists `m add field` and, with a tag selected, `⌫ delete field`. While confirming a delete it shows `⌫ delete field • any other key cancel`.
- Added tests for adding a tag (key, then values), spaces in keys, empty values, each way of abandoning a new tag, quitting while adding one, saving it, deleting with both keys, deleting the last tags, cancelling with another key or by moving away, and the delete key elsewhere. There's also an end-to-end test that adds and deletes fields in an issue file through the list.

---

[<brendan@berg.industries> 2026-10-11T11:22:32-07:00]

I've tested the new behavior in a freshly built binary. It largely looks good. I'm curious about something I've noticed though: when a new tag is created, it looks like there's a `k` under the cursor when edting the key attribute. It appears to be a rendering issue, as the key reflects what the user types when saved.

---

[Claude <noreply@anthropic.com> 2026-10-11T11:31:04-07:00]

Good catch: it was a rendering issue, and it's fixed. Moved back to `in review`.

- Cause: I'd given the new tag's key input a grey placeholder, `key`, as a hint. The input draws its cursor over the first character of the placeholder, and the rest of the placeholder wasn't drawn, so all you saw was a `k` under the cursor. Typing replaces the placeholder, which is why the saved key was always right.
- Fix: the placeholder is removed, so the cursor sits on an empty field. The bullet and the help footer already show that the field is being edited.
- Added a test that the new tag's line shows only the bullet before anything is typed, and the typed key afterwards. I checked that it fails with the placeholder put back (it shows ` • k`, matching what you saw).
- Checked in the real app: after the bullet, the terminal now draws the cursor on an empty space, with no `k`.
