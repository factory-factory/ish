# Update SQLite routines to only store author email addresses

["Brendan Berg" <brendan@berg.industries> 2026-10-20T23:15:32-07:00]

- state: closed
- assignees: <brendan@berg.industries>
- tested-by: <brendan@berg.industries>

---

["Brendan Berg" <brendan@berg.industries> 2026-10-20T23:15:32-07:00]

There are two processes that extract email addresses from issues and insert them in the metadata database. The first is from parsing bylines (which was updated in #3D6MY-1X334), and the second is from metadata attributes. Both behaviors need to be updated.

## Save only the email part of bylines in the `author` column of SQLite tables

Update the parsing behavior so that only the email address is saved to the metadata database. It should be stripped of surrounding angle brackets, if they are present.

## Strip angle brackets from emails when populating SQLite

Email addresses stored in SQLite should not include surrounding angle brackets. This should _not_ affect saving Issue markdown files, since those are written from the structure in memory. Metadata `Tag` objects in memory should still store emails as they are read from the markdown.

---

[Claude <noreply@anthropic.com> 2026-10-21T11:15:01-07:00]

Done, and moved to `in review`. I had no questions.

## What's stored

- **Authors:** the `author` columns of the `issue` and `comment` tables hold only the byline's email, without a display name or angle brackets. For example, `brendan@berg.industries` for `["Brendan Berg" <brendan@berg.industries> …]`.
- **Attribute values:** a value that's an email in angle brackets is stored without the brackets. Other values are stored as they are.
- **Unchanged:**
  - tags in memory keep their values as they're read, brackets and all
  - issue files are written from memory, so they're unchanged
  - filters, `@author` included, compare against the stored emails
- **The schema version:** it's now 5, so existing databases are rebuilt with the new values.

## Knock-on changes

- **"Issues assigned to me":** this matches `{assignees = "you@example.com"}` again. The extra match for `<you@example.com>` from #3D6BW-FYMNH is no longer needed, since the database never holds the brackets.
- **The Assignee column:** this reads the stored values, so it now shows a user's name for a bare email too.
- **Value suggestions:** values used with a key now come from the database without brackets. Typing the start of an email suggests it as stored, without brackets. Typing `@` still suggests users' names, which are saved as `<email>`.

## Something to check

`.ish/3D19G-TTPZE.md` has `- assignees: @Brendan Berg`, saved literally while accepting an `@` suggestion still stored the name. It's stored as that text, so it doesn't match "Issues assigned to me". Retyping it, or accepting the suggestion again, would save `<brendan@berg.industries>`. I haven't edited it, since it isn't this task's file.

## Testing

- **The store:** issue and comment authors are stored as emails, with a name and without brackets alike. Bracketed emails in values are stored without brackets, other values as they are, and the file is unchanged.
- **Showing values:** bare emails from the database show as users' names.
- **Suggestions:** typing an email suggests it as stored, and `@` suggests names.
- **"Issues assigned to me":** this compares with the plain email, and still finds an issue whose file has `<alice@example.com>`.
- **The schema test** expects version 5.
- **On a clone of this repository, with your current issue files:** `ish init` indexed 48 issues. Authors were stored as `brendan@berg.industries` and `noreply@anthropic.com`, no stored email has brackets, and the issue list shows `@Brendan Berg`.
