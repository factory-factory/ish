# Allow users to create filter expressions to filter issues in the list view

[brendan@berg.industries 2026-10-11T11:44:11-07:00]

- state: closed
- assignees: brendan@berg.industries

---

[brendan@berg.industries 2026-10-11T11:44:11-07:00]

**This feature makes assumptions about the project architecture that are addressed in other issues [which ones?]. We will be assuming that the sqlite work has been completed before planning and implementing the work for this issue.**

## The filter selector

At the top of the list view, there should be a drop-down menu to select one of many saved filters. The default filter's display name is "All unclosed issues", and the expression is `{state} != 'closed'`. The bottom item in the drop down is "Custom filter...", which expands to show the custom filter field when selected.

## The custom filter field

The filter selector should be replaced with an enclosed box to allow editing when "Custom filter..." is selected.

The text entered in the filter field should be parsed as filter expressions and styled for ease of comprehension. Filter expression syntax and style will be discussed below.

The custom filter field should have buttons to apply the expression to the issue list, and to save the custom filter as a named entry in the filter drop-down.

## Parsing filter expressions

Filter expressions are comprised of attribute expressions joined by logical operators. Attribute expressions are always in the form `attribute` `operator` `value`. Attributes are alphanumeric strings in curly braces that identify metadata keys to query against. Operators are `=`, `<`, `>`, `<=`, `>=`, `!=`, `~` (regexp match), and `?` (collection contains element). Values are any JSON scalar value: `null`, `true`, `false`, numbers, or strings.

## Rendering filter expressions

Filter expressions should use ANSI terminal colors for readability. The attribute text (including braces) should be magenta, the operator should be blue, and the value should be green.

---

[Claude 2026-10-12T00:03:30-07:00]

I've reviewed this issue against the database work. The issues it depends on are done: 1PZ4D-B0NEF (`ish init` and the database), 3CT6E-ZZPJQ (the list reads from the database) and 3CT8G-7VMS1 (re-indexing on startup).

It's a large feature with several separable parts, so I suggest splitting it into smaller issues (below) and keeping this one to track them. I need answers to questions 1 to 6 before starting; 7 to 10 have recommended defaults.

## Questions about the expression language

1. **Which logical operators?** The issue says attribute expressions are "joined by logical operators", but doesn't list them. I recommend `and`, `or` and `not`, with parentheses for grouping, and `and` binding more tightly than `or`.
2. **Comparing against several values.** The database stores every attribute's values as a JSON array of strings, and some attributes have several, such as `assignees: brendanberg, jhc`. What does `{assignees} = 'jhc'` mean when there are two assignees? I recommend:
   - `=`, `<`, `>`, `<=` and `>=` are true if _any_ value matches. With one value, that's the obvious meaning.
   - `!=` is true if _no_ value equals the given one, so it's always the opposite of `=`.
   - `~` is true if any value matches the regular expression.
   - `?` then overlaps with `=`. Should it be dropped, or mean something different, such as "has exactly this one value"?
3. **Issues without the attribute.** Should `{state} != 'closed'` include an issue with no `state` tag? With SQL's usual `NULL` handling, it wouldn't, so the default filter would hide such issues. I recommend:
   - a missing attribute makes `!=` true and every other comparison false,
   - `{key} = null` matches issues without the attribute, and `{key} != null` matches issues with it.
4. **Numbers, booleans and strings.** Values in issue files are always text, for example `estimate: 8h`. How should `{estimate} > 5` or `{done} = true` compare? I recommend:
   - a number compares numerically against values that are numbers, such as `"5"` or `"2.5"`, and is false for values that aren't, such as `"8h"`,
   - `true` and `false` match the text `true` and `false`,
   - strings compare as text, case-sensitively. Regular expressions can use `(?i)` to ignore case.
5. **String quotes.** The issue's example uses single quotes (`'closed'`), but JSON strings use double quotes. I recommend accepting both.
6. **Fields that aren't metadata.** Should filters be able to use the title, author, ID or opening date, for example `{title} ~ 'parser'`? If so, they need names that can't clash with metadata keys, such as `{@title}`. I recommend leaving them for a follow-up issue. Also, the issue says attribute names are alphanumeric, but keys may also contain `. _ - : /`. I recommend allowing those characters in braces too.

## Questions about the interface

7. **Where saved filters are kept.** They can't go in the database, because `ish init` rebuilds it from the issue files and would delete them. Should they be shared with everyone who clones the repository, in a committed file in `.ish/`, or personal to each user, in a file in your home directory? This may tie in with the configuration file issues, 1PZ3T-2PSBF and 1PZ3Z-QR4FB. I recommend a committed `.ish/filters.json` for now, with the built-in default always listed first.
8. **Using the selector and field from the keyboard.** A terminal has no mouse-driven drop-downs or buttons, so I recommend:
   - `f` opens the filter selector, `↑`/`↓` choose a filter, and enter applies it,
   - in the custom filter field, enter applies the expression, `ctrl+s` saves it under a name you're then asked for, and `esc` cancels,
   - the box shows these as labelled buttons, such as `[enter: Apply]` and `[ctrl+s: Save]`, which the help footer repeats.
9. **Invalid expressions.** I recommend showing the problem under the field, for example `expected a value after "!=" at column 9`, highlighting the faulty part in red, and not applying the expression.
10. **The default filter.** Should the built-in "All unclosed issues" be editable or removable? And should ish remember the last filter used? I recommend always starting with the default, and not letting it be removed.

## Plan

**How filtering works.** Each expression is parsed into a syntax tree and turned into an SQL `WHERE` clause for the existing list query. Every value is passed as a query parameter, never pasted into the SQL, so a filter can't change the query in unintended ways. Each attribute comparison looks inside the JSON array of values with SQLite's `json_each`. `~` uses a regular expression function implemented in Go and registered with SQLite.

**Suggested issues, in order:**

1. **Filter expression language (no interface).** A lexer and parser for expressions, with errors that point to the column where the problem is, and the conversion to SQL. It would include thorough tests of every operator, of precedence, of missing attributes, of several values, of numbers, and of malformed input. This is the core, and everything else depends on it.
2. **Filtering the list.** `store.ListIssues` takes a filter, and the list applies the default `{state} != 'closed'`. Closed issues then disappear from the list by default, which is a visible change. This also covers the JSON escaping fix above.
3. **Highlighting expressions.** A function that colours an expression (attributes magenta, operators blue, values green, errors red). It can be tested on its own.
4. **The custom filter field.** A boxed, editable field with live highlighting, Apply and error display. The standard text input can't colour parts of its text, so this needs a small custom input. That makes it the largest piece of interface work.
5. **Saved filters and the selector.** Storing saved filters (question 7), the drop-down selector, and saving a custom filter under a name.

I can't create issue files under `agents.md`'s rules, so if you agree with the split, please create the issues you want and I'll take them in order.

---

[Claude 2026-10-12T00:03:41-07:00]

A correction to my previous comment: plan item 2 refers to "the JSON escaping fix above", but I left the explanation out. Here it is.

When `ish init` stores attribute values, Go's JSON encoder escapes `<`, `>` and `&`. So 3CRTZ-EBD9H's assignee `<brendan@berg.industries>` is stored as `["\u003cbrendan@berg.industries\u003e"]`. That's still valid JSON, and SQLite's JSON functions decode it correctly, so filters built on `json_each` would work. But anything that matches the raw text would miss it, so values should be stored without that escaping. It's a one-line change in `store` and fits naturally with item 2.
