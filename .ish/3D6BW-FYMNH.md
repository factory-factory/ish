# Render user email addresses as '@Display Name'

[<brendan@berg.industries> 2026-10-20T14:16:58-07:00]

- state: in review
- assignees: <brendan@berg.industries>

---

[<brendan@berg.industries> 2026-10-20T14:16:58-07:00]

When rendering issue and comment bylines, replace the email address with its associated `name` from the `user` table.

Additionally, transform values in metadata attribute values and in comment text so that when displayed, they show as the display name, but when written (to both SQLite and Markdown files) they use the email associated with the display name.

## Visual representation

In bylines, the display name should be rendered in bold grey text with a preceding `@` symbol.

In metadata attributes and comment body text, the display name should be rendered in aqua colored (6) text with a preceeding `@` symbol.

## Implementation

**For metadata fields:**

- Create a new type for `Tokens` that have two string fields: `display` and `canonical`.
- Update `Tag` values to accept either a string or `Token` object.
- Update `Tag.renderValue` to render the display string when rendering a Token.
- When writing metadata to either markdown or sqlite, use the `canonical` field.

**For comment text:**

- When reading markdown files, search for email addresses enclosed in `<` and `>` characters and replace with the display name associated with that email, enclosed by Unicode private use glyphs to indicate a string that should be replaced on write.
- When writing markdown files, search for display names enclosed by the Unicode private use start and stop characters and replace with the email address associated with the display name, enclosed in `<` and `>`.
- Replacements should not be done in code spans or code blocks.

---

[Claude 2026-10-20T17:29:20-07:00]

I've reviewed this and moved it to `in progress`. Below is my plan, then some questions. The first question concerns the approach for comment text, so I'd like your view on it before I start.

## Plan

- **Names for emails:** the issue view gets a lookup from email to display name, read from the `user` table when an issue is opened, the same way it gets issue IDs for links.
- **Bylines:** an author that's a known email is shown as `@Display Name` in bold grey, in both the issue's byline and each comment's. Other authors, such as `Claude`, are shown as they are now.
- **Metadata:**
  - A tag's values become `Token`s, each with a `display` and a `canonical` string. A value that's a known email has that user's name as its display. Other values have no display, and show their canonical text.
  - The tag shows the display in aqua (6) with an `@`.
  - Writing the Markdown file, indexing into SQLite, filters, `State()` and `Assignees()` all use the canonical value, so the files and the database still hold emails.
- **Comment text:**
  - An email in `<…>` is shown as `@Display Name` in aqua.
  - goldmark already parses `<email>` as an email autolink. So finding these from the parsed Markdown skips code spans and code blocks for free, as it does for issue references.
- **Editing:**
  - A tag's values are edited as `@Display Name`.
  - Saving turns `@Display Name` back into that user's email. An accepted `@` suggestion from #3D19G-TTPZE saves the suggested user's email, rather than `@Brendan Berg` as it does now.
  - A typed email is kept as it is, and shown as the name afterwards.

## Questions

1. **When comment text is changed:** the ticket replaces `<email>` with a marked display name when a file is read, and replaces it back when the file is written. I'd suggest keeping the email in memory, and replacing it only when the comment is drawn. Here's why:
   - **Two users can share a name.** The `user` table allows the same name with different emails. Once `<email>` has become a name, writing the file can't tell which email it was.
   - **The editor would show the markers.** Editing a comment would show the private-use characters in the text, and they could be deleted or moved.
   - **The parser would need the database.** Reading files would need the `user` table. Currently `parse` doesn't use the database at all, and the store's indexing reads the same files.

   With my approach, editing a comment shows `<email>`, as the file has it, and the comment shows the name once it's saved. Is that acceptable? If you'd rather see names while editing, I could keep the email inside the markers, so nothing is lost.

2. **The `Token` type:** Go has no tidy "either a string or a `Token`". I'd change `Tag.Values` to `[]Token`, where a plain value is a `Token` with no display. Is that all right?
3. **Which metadata values become names:** only values that are exactly a known user's email, as in `assignees: brendan@berg.industries`, for any key? Should `<email>` in a value count as well?
4. **One email, several names:** if someone changed their git name, the table has more than one name for the same email, and it doesn't record which is newer. Which should be shown? I'd suggest the first alphabetically, so it's at least consistent, unless you'd like the indexing to track which name is most recent.
5. **One name, several emails:** when someone types `@Brendan Berg` in a tag, and that name has two emails, which email should be saved? I'd suggest keeping the text as typed, unless it matches exactly one user. An accepted suggestion always knows its user, so this only affects typing the whole name.
6. **The issue list:** should the Assignee column show display names too? The "Issues assigned to me" filter would keep matching emails.
7. **Bare emails in comments:** GFM also turns a bare `alice@example.com` into a link. Should that be shown as a name, or only `<alice@example.com>`, as the ticket says?

---

[<brendan@berg.industries> 2026-10-20T17:32:05-07:00]

## Answers, hopefully:

1. The goal is to allow users to type display names in comments and have them get resolved to emails when saving. Use the disambiguation guidelines enumerated below to handle cases where two emails can have the same display name or two display names can share the same email.
2. Making all values into `Tokens` sounds good. A token with no display can use the canonical value.
3. All metadata values should be checked for email addresses enclosed in angle brackets. (Existing records don't use angle brackets, but I will go through and update them throughout the saved markdown files.)
4. For now, fetch the first result for the email, and in a future ticket, we will change the user table to have a unique index on email and replace old display names with new ones as they're encountered.
5. If a display name is ambiguous, keep it as typed.
6. Yes, the Assignee column should also show display names.
7. For now, only emails in angle brackets should be tested to see if a display name matches. If an email does not have an associated display name, it should be rendered as an ordinary email address.

---

[Claude 2026-10-20T18:22:15-07:00]

Done, following your answers, and moved to `in review`.

## Where names are shown

- **Bylines:** an author that's a user's email is shown as `@Display Name`, in the byline's bold style. That covers the issue's byline and each comment's. Other authors, such as `Claude`, are shown as they are.
- **Metadata:** a value that's a user's email in angle brackets, such as `<brendan@berg.industries>`, is shown as `@Brendan Berg` in aqua (6). An email in angle brackets with no user is shown as an ordinary email, without the brackets. Values without brackets are shown as they're written.
- **Comments:** a user's email in angle brackets, outside code spans and code blocks, is shown as `@Display Name` in aqua. The text after it keeps glamour's styling. An email with no user is shown by glamour as an ordinary email.
- **The issue list:** the Assignee column shows names in the same way.
- **Which name:** an email with several names in the `user` table is shown with the first, alphabetically.

## Typing names

- **In a tag's values:** names are edited in place of emails. Saving turns `@Display Name` into `<email>` if exactly one user has the name. An ambiguous or unknown name is kept as it's typed. An accepted `@` suggestion from #3D19G-TTPZE is now saved as the email.
- **In a comment:** starting to edit replaces each user's `<email>` with `@Display Name`. Saving turns typed names back into emails, outside code.
  - A name must start a word and end one. So the `@` in an email like `bob@Bob.example` isn't read as a name, and `@Bobby` isn't read as `@Bob`.
  - Where one name starts another, as with `@Alice` and `@Alice Lee`, the longer one is used.
  - An email whose name is ambiguous stays as `<email>` while editing, so saving can't change it.
- **Saving without changes:** saving a comment or tag unchanged doesn't mark the issue as changed.

## Comments: no markers in the content

- **What I did instead:** I didn't add private-use markers to the content when files are read. The content in memory, the files and the database always hold `<email>`.
- **Where the names come in:**
  - names are shown only when a comment is drawn
  - they're swapped in when an edit starts, and back when it's saved
  - drawing marks the names temporarily, as it does for links, to colour them
- **Why:** this gives what you described in your answer: typing names, saved as emails, with ambiguous names kept as typed. The parser doesn't need the database, and no marker characters show in the editor.

## Code

- **`models.Token`:** this has `Display` and `Canonical`. `Tag.Values` is now `[]Token`, and a plain value is a token with no display.
- **What uses the canonical value:** the Markdown writer, the SQLite indexing, filters, `State()` and `Assignees()`.
- **`models.Directory`:** this maps emails and names both ways, using the rules above. The app reads it from the `user` table and passes it to opened issues with `Issue.SetDirectory`.
- **"Issues assigned to me":** this now matches your email with or without angle brackets, since you're moving the files to the bracketed form.

## Testing

- **The directory:** the first name for an email, ambiguous and unknown names, a missing directory, and how values are shown.
- **Tags:**
  - names shown in aqua for bracketed emails, unknown emails without brackets, and plain values as they're written
  - the canonical values are what's written
  - names edited in place of emails, with single users' names saved as emails, and ambiguous or unknown names kept
  - an unchanged save leaves the issue unchanged
- **Bylines:** both by name, and an author who isn't a user as they are.
- **Comments:**
  - names drawn in aqua, except in code spans and code blocks, with no markers left behind
  - names swapped in for editing and back for saving, with the cases above
  - an unchanged save leaves the comment unchanged
- **New comments and tags** get the directory.
- **Files:** a tag shown by name is written with the email.
- **The app:**
  - the Assignee column shows names
  - an opened issue shows names, and a name typed in a tag is saved as the email
  - with no database, there are no users
  - "Issues assigned to me" matches bracketed emails, and quoting keeps `<` and `>`
- **In a real terminal,** on a clone of this repository with `Brendan Berg` in the `user` table:
  - the byline and a bracketed assignee showed `@Brendan Berg`
  - in a comment, `<brendan@berg.industries>` showed as `@Brendan Berg` in aqua, while the same email in a code span stayed as it was
  - quitting without edits left the file unchanged

---

[brendan@berg.industries 2026-10-20T18:32:13-07:00]

Please fix this issue:

- **Bylines:** It looks like you're expecting emails in bylines to not be enclosed in angle brackets. Let's make sure emails get parsed with or without angle brackets.

---

[Claude 2026-10-20T18:48:20-07:00]

Fixed, and left in `in review`. Bylines now recognise a user's email with or without angle brackets, so `[<brendan@berg.industries> …]` and `[brendan@berg.industries …]` both show `@Brendan Berg`. As in metadata, an email in angle brackets with no user is shown as an ordinary email, without the brackets. Any other author, such as `Claude`, is shown as it is.

Your sample files with bracketed bylines already parse, and they write back unchanged.

## Testing

- **Bylines:** both forms of a user's email show their name, in the issue's byline and in a comment's. An unknown email is shown without brackets, and `Claude` is shown as it is.
- **In a real terminal,** on the clone, a bracketed byline showed `@Brendan Berg opened this issue`.

## Two notes

- **New comments:** these still write the author's email without angle brackets, from git's `user.email`. Should new bylines use the bracketed form, to match the files you've updated?
- **The Go toolchain:** Homebrew upgraded Go to 1.27.3 today, but my shell's `GOROOT` still pointed to the old 1.27.0 toolchain, so builds failed with "version go1.27.0 does not match go tool version go1.27.3". I ran the tests with `GOROOT` unset rather than change your environment. You may want to remove the `GOROOT` setting from your shell profile.
