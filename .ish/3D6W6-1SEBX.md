# Create filter expression function to convert display names to email addresses

["Brendan Berg" <brendan@berg.industries> 2026-10-20T22:58:40-07:00]

- state: in review
- assignees: <brendan@berg.industries>
- relation.parent: #3CZWN-EBVEH
- relation.depends-on: #3D6BW-FYMNH

---

["Brendan Berg" <brendan@berg.industries> 2026-10-20T22:58:40-07:00]

In filter expressions, matching against assignees requires an email address enclosed in angle brackets. An expression like `{assignees = 'Brendan Berg'}` won't work because the display name isn't stored in the metadata database. A user knowing that email addresses are used may expect an expression like `{assignees = 'hello@example.org'}` to work, but this too does not match any issues because email addresses are stored with enclosing angle brackets.

Issue #3D6WQ-KR22E will address the second case by stripping angle brackets from email addresses when storing them in the metadata database.

To address the first case, we need a function that converts display names to email addresses.

## Desired behavior

- When creating filter expressions, a user should be able to specify a display name that gets converted to an email address

This would allow an expression like `{assignees = ('Brendan Berg' :user)}` to match based on the email address stored in the user directory for Brendan Berg.

## Required changes

- Add a `:user` function that looks up a display name in the user directory and returns the user's email address.

---

[Claude <noreply@anthropic.com> 2026-10-22T17:58:53-07:00]

I've reviewed this. It's marked `planning`, so I've left the state as it is and haven't started. Below is how I'd approach it, then some questions.

## Plan

- **The function:** `:user` is an expression function, like those from #3D97C-F62P7. `('Brendan Berg' :user)` gives the email of the user with that display name, from the `user` table, such as `brendan@berg.industries`. Display names are unique since #3D6MY-1X334, so a name finds at most one user.
- **In SQL:** it's evaluated against the `user` table in the same query, so it sees the same users as the issues it filters. Emails are stored without angle brackets since #3D6WQ-KR22E, so the result compares directly with stored assignees.
- **Checking when the filter is applied:** as with other expressions, the app evaluates `:user` once in Go to report errors, and to find a `null` result. For that, `filter.Resolve` would also be given a way to look up users, alongside the state snapshot. The app already reads the users for display names, so it would pass those.

## Questions

1. **No such user:** what if no user has the name?
   - (a) `null`, so the comparison is UNKNOWN and matches nothing, like a missing lookup. I'd suggest this.
   - (b) An error under the filter, such as "no user named Brendan".
2. **Matching names:** should a name match exactly, or regardless of case, so `('brendan berg' :user)` works? Names are unique as stored, but two could differ only in case, such as `Sam` and `sam`. Then a case-insensitive match would be ambiguous.
3. **A leading `@`:** names are shown as `@Brendan Berg`. Should `('@Brendan Berg' :user)` work too, ignoring the `@`?
4. **An email:** should `('brendan@berg.industries' :user)` give the email back, so the function also accepts an email? Or should it only look up names, giving `null` for an email?
5. **A list:** should `(['Brendan Berg', 'Alice'] :user)` give a list of emails, so a filter can match any of several people? Or should `:user` take only a string, with a list as a type error?
6. **The state:** this is in `planning`. Should I move it to `in progress` once these are answered?

---

["Brendan Berg" <brendan@berg.industries> 2026-10-22T18:00:34-07:00]

1. If no user matches, the result is `null` and behaves like a missing lookup.
2. Names should match case insensitive.
3. A leading `@` should be ignored.
4. The function should only look up names.
5. Expanding scope to accept lists is a good idea and worth a try.
6. I've moved the issue into `in progress`.

---

[Claude <noreply@anthropic.com> 2026-10-22T18:06:11-07:00]

Done, following your answers, and moved to `in review`.

## `:user`

- **The function:** `('Brendan Berg' :user)` gives the email of the user with that display name, from the `user` table, such as `brendan@berg.industries`. That compares directly with stored assignees and authors, as in `{assignees = ('Brendan Berg' :user)}`.
- **Matching:** a leading `@` is ignored, so `('@Brendan Berg' :user)` works too. Names match regardless of case.
- **Names that differ only in case:** your answers didn't cover this, so I chose:
  - an exact match wins, so `Sam` finds `Sam` even if there's also a `sam`
  - otherwise, a name matching one user regardless of case finds them
  - one matching several, such as `SAM`, finds no one
- **No match:** a name with no user gives `null`, so the comparison is UNKNOWN, like a missing lookup. It matches nothing on its own, and an `or` returns its other side.
- **Emails:** only names are looked up, so an email finds no one.
- **Lists:** a list of names gives a list of emails, with `null` for names that find no one. So `{@author = (['Bob', 'Carol'] :user)}` matches either of them.
- **Errors:** a value that isn't a name or a list of names is a type error, as is an argument to `:user`.
- **Display:** the filter prints as it's written, with `:user` in aqua like other functions.

## How it's evaluated

- **Before the query:** unlike the other functions, `:user` isn't evaluated in SQL. An SQL function written in Go can't query the database it's called from, and a subquery handling `@`, case and lists would be unwieldy.
- **What happens instead:** when a filter is applied, `filter.ResolveEnv` replaces each `:user` call with the emails it finds. They're passed to the query as parameters, as looked-up values are.
- **Where the users come from:** the app reads them from the `user` table each time the filter is applied or the list reloads, so they're the same users the issues are indexed with.
- **The API:**
  - `filter.Resolve(node, lookup)` still works as before
  - `ResolveEnv` also takes the users, in a `filter.Env`
  - outside a resolved filter, such as in an unresolved expression, `:user` finds no one

## Testing

- **Looking up names:**
  - exact names, names regardless of case, and a leading `@`
  - an exact match winning, and several matches regardless of case finding no one
  - unknown names, and emails
- **Resolving:** single names, lists, chained functions, interpolation, unknown names and missing lookups as UNKNOWN, and type errors.
- **Parsing:** an argument to `:user` is a syntax error. Filters print as written, and resolving doesn't change the parsed filter.
- **Against a database:**
  - authors by name, with `@`, regardless of case, and from a list
  - `!=`, and no user, alone and with `or`
- **In the app:** a filter with `('@alice liddell' :user)` listed alice's issues, using the users in the database.
- **In a real terminal,** on a clone of this repository, `{assignees = ('@brendan berg' :user)}` listed the issues assigned to `@Brendan Berg`.
