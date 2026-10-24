# Support field lookup on comment text, author, and date created

["Brendan Berg" <brendan@berg.industries> 2026-10-23T10:49:47-07:00]

- state: in review
- assignees: <brendan@berg.industries>

---

["Brendan Berg" <brendan@berg.industries> 2026-10-23T10:49:47-07:00]

Allow queries against comment contents and authorship. This involves allowing three new fields in query expressions: `@comments.text`, `@comments.author`, and `@comments.dateCreated`. (Note that the `.text` qualifier refers to the `content` column of the `comment` table.)

---

[Claude <noreply@anthropic.com> 2026-10-23T14:01:56-07:00]

I've reviewed this and moved it to `in progress`. Below is my plan, then some questions. I'll start once they're answered.

## Plan

- **The new fields:** `@comments.text`, `@comments.author` and `@comments.dateCreated` compare against the `content`, `author` and `dateCreated` columns of the `comment` table.
- **An issue has many comments:** so these behave as a metadata attribute with several values does.
  - A comparison matches if any of the issue's comments matches. For example, `{@comments.text ~ 'TODO'}` matches issues with a comment mentioning `TODO`.
  - `!=` matches if `=` matches none of them, so it's always the opposite of `=`.
- **The rest is the same:** literals, lookups, expressions, `:user` and `:fromnow` all work with these fields as with others. For example, `{@comments.author = ('Brendan Berg' :user)}` or `{@comments.dateCreated > ('-1w' :fromnow)}`. Authors and dates are stored as emails and UTC RFC 3339 text, as `@author` and `@dateCreated` are.

## Questions

1. **The issue's description:** the first card of an issue, its description, is stored in the `comment` table as comment 0, with the issue's own byline. Should `@comments.*` include it? Or only the comments after it?
   - **Including it:** `{@comments.text ~ 'x'}` searches the whole issue, description and all.
   - **Leaving it out:** `{@comments.author = …}` finds people who commented, rather than the issue's author too, whom `@author` already covers.

   I'd suggest including it, as the simplest rule. But it depends how you'd use these fields.
2. **Issues with no comments:** if the description is left out, an issue may have no comments. As with a missing attribute, I'd make it match only `!=`, and `= null` would mean "has no comments" while `!= null` would mean "has comments". Is that right?
3. **Several conditions on one comment:** `{@comments.author = 'a@example.com'} and {@comments.text ~ 'LGTM'}` matches an issue where alice wrote any comment and any comment says LGTM. They needn't be the same comment. Matching within a single comment would need new syntax, so I'd leave that for later. Is that all right?

---

["Brendan Berg" <brendan@berg.industries> 2026-10-23T14:20:46-07:00]

1. Include the description card in the query. We don't have a separate way of querying a description, so `@comments.text` should search the entire issue's contents.
2. Issues with no comments are valid (bulk creation for future fleshing-out, for example). It will be helpful to be able to query for issues with no comments / no description. The `= null` and `!= null` filters create the desired behavior.
3. Yes, several conditions mean each could match any comment. I'll create a new issue to work on syntax so multiple conditions match the same comment.

---

[Claude <noreply@anthropic.com> 2026-10-23T14:35:07-07:00]

Done, following your answers, and moved to `in review`.

## The fields

- **What they compare:** `@comments.text`, `@comments.author` and `@comments.dateCreated` compare the `content`, `author` and `dateCreated` of an issue's comments.
- **The description:** this is included, so `@comments.text` searches the whole issue.
- **Authors and dates:** authors are emails, and dates are UTC RFC 3339 text, as for `@author` and `@dateCreated`.
- **Matching:** like a metadata attribute with several values:
  - a comparison matches if any of the issue's comments does, as in `{@comments.text ~ 'TODO'}`
  - `!=` matches if none does, so it's always the opposite of `=`
  - an issue without comments or a description matches only `!=`
- **`null`:** `= null` matches issues with no comments, and `!= null` those with some.
- **What else works with them:** literals, lookups, lists, expressions, `:user` and `:fromnow`, as in `{@comments.author = ('Bob' :user)}` or `{@comments.dateCreated > ('-1w' :fromnow)}`.
- **Several conditions:** as agreed, each can match a different comment.
- **Errors:** an unknown field, such as `@comments.body`, is an error listing all the fields.

## Testing

- **Parsing:** each field, printed back as written, and an unknown comment field.
- **Against a database**, with an issue with a description and two comments, one with a description only, and one with neither:
  - text, regular expressions and numbers, with the description searched
  - authors, including `:user` and a list
  - `!=`, including the issue without comments
  - dates, as text and with `:fromnow`
  - `= null` and `!= null`
  - two conditions matched by different comments
- **The field list** in the existing error messages.
- **In a real terminal,** on a clone of this repository, `{@comments.text ~ 'dropdown'}` listed exactly the 3 issues whose comments mention it.
