# Filter expression follow-up: allow filtering on non-attribute fields

[brendan@berg.industries 2026-10-12T22:18:19-07:00]

- state: closed
- assignees: brendan@berg.industries
- relation.parent: #3CSJ4-PEZ1N

---

[brendan@berg.industries 2026-10-12T22:18:19-07:00]

Currently attributes in filter expressions only match against the metadata keys in an issue. Update the expression parser and WHERE clause generator to also accept fields that are direct attributes of the issue like title and dateCreated.

Non-metadata fields should start with an `@` character. Accepted fields are `@title`, `@author`, and `@dateCreated`.

This should be a straightforward addition to the WHERE clause, since the field names match the SQLite column names to query on.

---

[Claude 2026-10-14T21:41:33-07:00]

Done, and moved to `in review`.

## What's new

- Filters can compare `@title`, `@author` and `@dateCreated`, as in `{@author = 'alice'}` or `{@title ~ /^Filter/}`. Each compares the `issue` table's column of the same name.
- Field names are case-sensitive. Any other name after `@` is an error that lists the three fields, such as "unknown field @state; the fields are @title, @author, and @dateCreated". An `@` on its own is an error too.
- `@` can only start a name, so it never clashes with metadata keys, which can't contain `@`.
- Fields are highlighted magenta, like attribute names.

## How comparisons work

Fields follow the same rules as attributes, with these differences:

- **`null`:** every issue has every field, so `{@author = null}` matches nothing and `{@author != null}` matches every issue.
- **`!=`:** it's still the opposite of `=`. For example, `{@title != 42}` matches every title that isn't the number 42, including titles that aren't numbers.
- **Numbers:** a number compares numerically with titles that are numbers, as attribute values do.

## Dates

`@dateCreated` compares as text, against the stored value: RFC 3339 in UTC, such as `2026-10-13T04:32:20Z`. So:

- `{@dateCreated >= '2026-10-13'}` and `{@dateCreated < '2026-10-01'}` work as you'd expect, because the format sorts by date.
- `{@dateCreated = '2026-10-13'}` matches nothing, because it would have to match the whole value. `{@dateCreated ~ /^2026-10-13/}` matches that day.
- Days are UTC days, not those of the byline's time zone. An issue opened at 8:30 pm Pacific on 13 October has a `dateCreated` of the 14th.

If you'd like dates to compare in local time, or to understand values such as `'2026-10-13'` as dates, that could be a follow-up issue.

## Testing

- Parsing each field, including spacing, and printing it back unchanged.
- Errors for unknown fields (including the wrong case), an `@` on its own, and `@` inside a name.
- Fields rendered in the attribute style.
- Matching against a database in a temporary directory: text, regular expressions, numbers, `true`, `null`, `!=`, dates around a UTC day boundary, and fields combined with attributes.
- Field conditions use only their column, and their value is passed as a parameter.
- `store.ListIssues` with a filter that combines a field and an attribute, alongside the list's own joins.
