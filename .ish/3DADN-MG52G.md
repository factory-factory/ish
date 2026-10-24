# Support indexing modification dates from git and new @dateModified filter field

["Brendan Berg" <brendan@berg.industries> 2026-10-23T11:29:23-07:00]

- state: in review
- assignees: <brendan@berg.industries>

---

["Brendan Berg" <brendan@berg.industries> 2026-10-23T11:29:23-07:00]

This change has multiple phases.

## Update SQLite schema to include a history table

Add a `history` table with the following fields:

- `uuid`: primary key
- `issueId`: foreign key on `issue.id`
- `dateModified`: ISO datestamp when the edit was committed

Create a unique index on `issueId, dateModified`.

## Update indexing process on init and startup

Add a function run during initialization that requests commit dates and files changed for all issue markdown files in the issues directory. Insert a row for each issue with edits for a particular commit timestamp.

Additionally, update the startup check for file modificatins to include commit dates and each changed file. In addition to re-indexing the changed files that is already occuring, insert the commit dates for each file into the `history` table.

## Update the filter expression parser

Update filter expressions so the `@dateModified` field can be used to query issues based on the `dateModified` column in the `history` table.

---

[Claude <noreply@anthropic.com> 2026-10-23T14:50:42-07:00]

I've reviewed this and moved it to `in progress`. Below is my plan, then some questions. The first two decide what `@dateModified` means, so they matter most. I'll start once they're answered.

## Plan

- **The schema:**
  - a `history` table: `uuid`, `issueId` referencing `issue.id` (deleted with its issue), and `dateModified`, in UTC RFC 3339 like the other dates
  - a unique index on `(issueId, dateModified)`
  - the schema version goes up to 6, so databases are rebuilt
- **Reading git:** `git log --format='%H %cI' --name-only -- .ish` lists each commit's date and the issue files it changed, as below. Each file's name gives its issue's ID, and each commit gives one row for each issue it touched.

  ```
  7d49af8… 2026-10-23T14:57:05-07:00
  .ish/3DACD-RNJ3F.md
  ```
- **When:**
  - **`ish init`:** the whole history is indexed.
  - **At start-up:** along with the existing check for changed files, the commits since the newest date in `history` are added, starting from that date. Rows that are already there are skipped, thanks to the unique index. So start-up stays quick on a long history, and nothing is missed or counted twice.
- **No repository:** without git, or outside a repository, `history` stays empty, and nothing fails, as with users.
- **Files that can't be matched:** a commit touching a file whose issue isn't in the database, such as one since deleted or renamed, is skipped.

## Questions

1. **What `@dateModified` compares:** an issue has a row for every commit that changed it.
   - **(a) The latest modification only:** `{@dateModified < ('-1mo' :fromnow)}` then means "not changed in the last month". I'd suggest this, since it's how a modified date usually works.
   - **(b) Any modification**, like `@comments.*`: `{@dateModified > ('-1w' :fromnow)}` means "changed in the last week". But `{@dateModified < ('-1mo' :fromnow)}` would match nearly every issue older than a month, even ones changed yesterday.
2. **Commit date or author date:** the ticket says "when the edit was committed", so I've used the commit date (`%cI`). For users in #3D6MY-1X334 you chose author dates. They differ after a rebase or a cherry-pick. Which should this use?
3. **Edits that aren't committed:** an issue edited in ish but not yet committed has no commit for that edit. Should `@dateModified` ignore it until it's committed, as the ticket implies? Or should the file's modification time count as a modification?
4. **An issue with no commits:** a new issue that isn't committed yet has no history. Should `@dateModified` be `null` for it, so it matches only `!=`, and `= null` finds uncommitted issues? Or should it be its `@dateCreated`?
5. **Which branches:** should history come from the checked-out branch only, since the issue files are from that branch? Or from every branch (`--all`), as for users? I'd suggest the checked-out branch.
6. **Renamed files:** history before a rename belongs to the old file name. When IDs were changed earlier in the project, old names stopped matching any issue. Should renames be followed, so an issue keeps its history from before the rename? That needs git's rename detection on each file, which is slower on a long history. I'd suggest not following them for now.

---

["Brendan Berg" <brendan@berg.industries> 2026-10-23T15:13:34-07:00]

1. It's best to keep query/filter behavior consistent. `{@dateModified > ('-1w' :fromnow)}` would match issues with at least one modification in the past 7 days. To match issues that have *not* been changed in the past 7 days, the filter would be `not {@dateModified > ('-1w' :fromnow)}`.
2. I had forgotten about the distinction. Please use author dates.
3. `@dateModified` should only reflect committed changes for now. A future issue will address changes that have not been committed / automatically committing edits.
4. An issue with no commits should not have an entry in the `history` table. `{@dateModified = null}` should list uncommitted issues.
5. History should come from the checked-out branch only.
6. Since a rename changes the issue ID, it's fine to break history by not following renames.

---

[Claude <noreply@anthropic.com> 2026-10-23T17:13:43-07:00]

Done, following your answers, and moved to `in review`.

## The `history` table

- **Columns:** `uuid`; `issueId`, which references `issue.id`, so it's deleted with its issue; and `dateModified`, the commit's author date in UTC RFC 3339, like the other dates.
- **The index:** a unique index on `(issueId, dateModified)`.
- **The schema version:** it's now 6, so databases are rebuilt.

## Indexing

- **Where the history comes from:** `git log --format=%x00%aI --name-only -- .` in the issue directory. It lists the checked-out branch's commits with their author dates, and the files each changed. Each issue file a commit changed gives a row.
- **What's skipped:** other files, such as `config.toml`, and files whose issue isn't in the database, such as deleted or renamed ones. Renames aren't followed.
- **`ish init`:** this indexes the whole history, and reports it, as in "Indexed 141 changes from the git history".
- **At start-up:**
  - After the existing sync of changed files, ish adds the commits made since the newest modification it has. Rows it already has are skipped, by the unique index.
  - git can only ask for commits by when they were committed, not by their author dates. So it asks for commits committed since the newest author date. That's never later than the commits it already has.
- **No repository:** without git, or outside a repository, there's no history, and nothing fails.

## Keeping history when issues are re-indexed

- **The problem:** the start-up sync and saving from the issue view re-index an issue by deleting its rows and adding them again. That would have deleted its history, which doesn't come from its file.
- **The fix:** they now keep the issue's history and add it back.
- **Deleted issues:** a deleted issue's history is still deleted with it.

## `@dateModified`

- **Matching:** it compares the dates of the commits that changed an issue, as `@comments.*` compares its comments:
  - `{@dateModified > ('-1w' :fromnow)}` matches issues with a commit in the last week
  - `not {@dateModified > ('-1w' :fromnow)}` matches issues without one
- **`null`:** an issue with no commits has no history, so `{@dateModified = null}` lists uncommitted issues, and `!= null` committed ones.
- **Uncommitted edits:** as you said, these don't count until they're committed.

## Your "Mentions me" preset

`app/menu.go` has your uncommitted "Mentions me" preset, which I haven't included. With it, `TestMenuSeparatesCustomFilter` fails, since it expects "Created by me" to be the last saved filter before the rule. So I ran the whole suite with the committed `app/menu.go`, and it passed. That test will need updating with your preset.

## Testing

- **Parsing history:** dates in UTC, issue files only, and commits without dates.
- **`ish init`:** the whole history, without issues that aren't in the database, and without duplicates.
- **Start-up:** only commits since the newest modification are asked for, and those already counted are skipped. Without git, nothing fails.
- **Re-indexing:** the sync and `IndexIssue` keep history, and deleting an issue deletes its history.
- **The table:** its unique index, and its reference to issues.
- **With a real git repository:** author dates in UTC, for issue files only.
- **Filters:**
  - `@dateModified` with dates and `:fromnow`
  - `not` for issues without recent commits, and `!=`
  - `= null` and `!= null`
  - parsing and printing
- **On a clone of this repository:**
  - `ish init` indexed 141 changes for all 54 issues. #3DACD-RNJ3F's matched `git log`: `2026-10-23T14:35:07-07:00`, stored as `2026-10-23T21:35:07Z`.
  - After I committed a change to that issue, starting ish kept its earlier modification and added the new one.
