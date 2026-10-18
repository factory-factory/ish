# Change autosuggest source when values start with # and @

[brendan@berg.industries 2026-10-16T23:40:56-07:00]

- state: in review
- assignees: brendan@berg.industries
- relation.depends-on: #3D26C-1320A

---

[brendan@berg.industries 2026-10-16T23:40:56-07:00]

Instead of searching existing values for items starting with `#`, the search should be across all issue IDs. Similarly, when typing a value starting with `@`, the autocompletions should come from user display names.

---

[Claude 2026-10-17T21:43:02-07:00]

Done, and moved to `in review`. With #3D26C-1320A in place, I had no questions. The ticket doesn't settle a few details, so I chose them, as described below. Each is a small change if you'd like it different.

## `#` values

- **Suggestions:** a value that starts with `#` suggests `#` followed by an issue ID. It searches the IDs of every issue in the metadata database, for any key, rather than the values already used with that key.
- **Left out:** the issue's own ID, and IDs already in the tag.
- **Not issues:** a value used with the key that isn't an issue's ID, such as a reference to a deleted issue, isn't suggested.

## `@` values

- **Suggestions:** a value that starts with `@` suggests `@` followed by a user's name from the user table. A name used with several emails is suggested once.
- **The `@` stays in the value,** as `#` does, so accepting `@Bre` gives `@Brendan Berg`.
- **Names with a comma** aren't suggested, since saving one would split it into two values.

## Like other suggestions

- **The threshold:** the 3-character threshold counts the `#` or `@`, so suggestions start at `#3C` or `@Br`.
- **Matching and order:** matches ignore case and are sorted alphabetically. Each comma-separated value is suggested separately.
- **Keys:** `tab` and enter accept a suggestion, as before.
- **Which tags:** this works in the values of new and existing tags.

## Where they come from

- **When they're read:** when an issue is opened, the app reads the issue IDs, which it already reads for links, and the users from `store.Users`.
- **How they reach the issue view:** through a new `Issue.SetReferences`.

## Testing

- **`#` suggestions:**
  - IDs are suggested regardless of case and for any key
  - the issue's own ID and IDs already in the tag are left out
  - the threshold counts the `#`
  - a value used with the key that isn't an issue isn't suggested
- **`@` suggestions:** names are suggested regardless of case and in order, after other values, and a name with a comma is left out.
- **Accepting and editing:** enter accepts a reference and saves the tag, and an existing tag's values get `@` suggestions.
- **In the app:** user names are de-duplicated, and an opened issue suggests another issue's ID.
- **In a real terminal,** on a clone of this repository after `ish init`:
  - `@Bre` and enter saved `watchers: @Brendan Berg`
  - `#3CT` and enter saved `see: #3CT6E-ZZPJQ`
