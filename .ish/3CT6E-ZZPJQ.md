# Update the Issue list view to pull data from the metadata database

[<brendan@berg.industries> 2026-10-11T22:34:12-07:00]

- state: closed
- assignees: <brendan@berg.industries>

---

[<brendan@berg.industries> 2026-10-11T22:34:12-07:00]

Instead of reading issue files from disk when rendering the Issue list view, use the metadata database.

For now, select all issues when populating the table. Create a query that joins the required attributes for the table columns.

---

[Claude <noreply@anthropic.com> 2026-10-11T22:49:44-07:00]

I've reviewed this. The query itself is straightforward, but the list now depends on the database staying in step with the issue files. I need an answer to question 1 before implementing. For questions 2 and 3, I'll go with my recommendation unless you say otherwise.

## Questions

1. **Keeping the database current after edits in the TUI (blocking).** The list loads when ish starts and again whenever you close an issue, but the issue view saves edits only to the markdown file. Read from the database as it stands, the list would show stale titles and states after an edit. Worse, issues created with `n` wouldn't appear at all until `ish init` is run again. Options:
   - (a) When the issue view saves an issue file, also replace that issue's rows in the database, in one transaction. That needs a small `store` function to re-index a single issue, and the issue view would save through it. I recommend this, so the list never falls behind the TUI's own edits.
   - (b) Rebuild the whole database whenever you return to the list. This is simple, but it re-reads every file each time, which undoes much of the point of the database.
   - (c) Leave the database stale until `ish init` is run, and handle updating it in a later issue. This is the smallest change, but until then the TUI would regress, with new issues missing from the list.
2. **No database yet.** If `ish` starts and `.ish/.metadata.ish` doesn't exist (for example, in a fresh clone, since it's git-ignored), should it build the database itself, as `ish init` does, or exit with a message telling you to run `ish init`? I recommend building it, since it's a cache of the files and needs no input from you.
3. **Files changed outside ish.** After a `git pull` or an edit in another editor, the database will be stale until `ish init` is run again. The `sha256` column makes it possible to detect this at startup and re-index only the files that changed, were added or were removed. I recommend a separate issue for that, to keep this one focused. Until then, `ish init` brings the database up to date.

## Plan

1. **Query, in `store`:** a new `ListIssues` returns one summary per issue, ordered by ID, with what the list's columns need: ID, title, assignees, opening date and state. It joins the two attributes it needs to each issue:

   ```sql
   SELECT issue.id, issue.title, issue.dateCreated, issue.dateCreatedOffset,
          assignees.value, state.value
   FROM issue
   LEFT JOIN attribute AS assignees
     ON assignees.issueId = issue.id AND assignees.key = 'assignees'
   LEFT JOIN attribute AS state
     ON state.issueId = issue.id AND state.key = 'state'
   ORDER BY issue.id
   ```

   The unique (`issueId`, `key`) constraint means each join matches at most one row. The JSON values are decoded in Go, which keeps their order, and joined with commas as the list shows them now. The opening date is rebuilt in the byline's own time zone from the UTC date and the offset, so the Opened column keeps showing the same day as the issue view.

2. **List view:** the list uses these summaries instead of parsing every file, both at startup and when returning from an issue. The columns, their order and their formatting stay the same.
3. **What doesn't change:** opening an issue still reads its markdown file, which is the source of truth, and saving still writes the file. With 1(a), the save also updates the database.
4. **A side effect:** today one file that doesn't parse stops the list from opening at all. With the database, `ish init` skips and reports such files, and the list shows everything else.
5. **Tests**, in temporary directories with databases built by `store.Init`:
   - the query, covering issues with no assignees or state, several assignees, and the date and offset
   - the list built from the database
   - returning to the list after editing an issue or creating one
   - per question 2, starting without a database

---

[<brendan@berg.industries> 2026-10-11T22:58:42-07:00]

To answer your questions:

1. Let's go with option (a).
2. If `ish` is run and no database is found, the `runInit` function should be called to do the initialization.
3. The functionality described here is best handled in a separate issue. I'm currently working on that.

---

[Claude <noreply@anthropic.com> 2026-10-11T23:10:21-07:00]

Done, following your answers, and moved to `in review`.

## The list reads from the database

- `store.ListIssues` runs the query from my plan: every issue, ordered by ID, joined to its `assignees` and `state` attributes. It returns a summary per issue: ID, title, assignees, state, and the opening date in the byline's own time zone.
- The list uses it at startup and whenever you return from an issue. The columns and their formatting are unchanged.
- Opening an issue still reads its markdown file.

## Edits update the database (question 1, option a)

- Saving an issue from the TUI now goes through `store.SaveIssue`. It writes the file as before, then `store.IndexIssue` replaces that issue's rows in the database in one transaction. If that update fails, for example because the file can't be read back, the transaction is rolled back and the old rows stay.
- `store.SaveIssue` first checks the database's key rules (allowed characters, no duplicate keys). An edit that breaks them is refused with nothing written, and the issue view shows the error as it does for other failed saves. Otherwise the editor could write a file that `ish init` would then skip.
- Both the list and `ish <path>` save this way.

## A missing database (question 2)

- If `.ish/.metadata.ish` doesn't exist when `ish` starts, it calls `runInit` first, so you see the same report as `ish init`.
- If `runInit` can't create a database at all, `ish` exits with status 1. If it only skipped some files, those are reported and the TUI still starts with the rest.

## Changes outside ish (question 3)

- As agreed, these aren't picked up until `ish init` is run again. A test records this, and the separate issue you're writing can change it.
- Tests that used to edit issue files directly and expect the list to notice now rebuild the database first, standing in for `ish init`. They still check how the list keeps its selection when it reloads.

## Other changes

- The list's error line now wraps to the terminal's width, as the issue view's already does. Before, a long error (for example, one with a long path) was cut off.
- An issue with an unreadable byline date can no longer reach the list, because `ish init` skips it, so I removed that case from the Opened column tests. `store` tests already cover skipping it.

## Testing

- `store` tests for `ListIssues`:
  - ID order
  - several assignees, kept in order
  - issues without the attributes
  - the time zone
  - a missing database, which doesn't create an empty one
- `store` tests for `SaveIssue` and `IndexIssue`:
  - an issue's rows are replaced, not added to
  - its checksum is updated
  - new issues are added
  - invalid metadata is refused with nothing written
  - a missing database
  - a bad file leaves the old rows
- `main.go` tests for building a missing database at startup, including one with a skipped file and one that can't be created.
- List tests for a reload failing because the database was removed, and for an outside edit appearing only after a rebuild.
- I ran `ish` on a copy of `.ish` with no database. It built the database itself (13 issues), then I renamed an issue and created one with `n`. Each showed in the list on returning, and the database ended with 14 issues for 14 files.
