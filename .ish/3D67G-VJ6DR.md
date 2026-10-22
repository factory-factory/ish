# Add object lookup syntax to filter expressions

["Brendan Berg" <brendan@berg.industries> 2026-10-20T11:56:53-07:00]

- state: in review
- assignees: <brendan@berg.industries>
- relation.parent: #3CZWN-EBVEH
- relation.depends-on: #3D6HA-7FCAK

---

[<brendan@berg.industries> 2026-10-20T11:56:53-07:00]

Currently, filter expressions allow attribute comparisons against literal strings, numbers, booleans, and null. Implement object lookup syntax that allows access to the configuration state of the application.

## Syntax

Object lookup uses dot notation, which can be chained to traverse nested objects. The application can choose to expose certain objects to the filter mechanism.

For example, if the application made a `user` object available with properties for things like `name` and `email`, a filter could reference those values in a property comparison like `{assignees = .user.email}`.

## Semantics

An object lookup can be used in any position where a literal value is used.

## Rendering

Object lookup terms should be rendered in aqua (6) text.

## Query Behavior

Object lookup should be done on a snapshot of the application state such that no modifications to the state are made between first and last lookup. A lookup must not modify application state.

The object lookup in the filter expression should be replaced with a placeholder when transforming into a SQL query. The value from the application state should be passed to the query as an argument.

---

[Claude <noreply@anthropic.com> 2026-10-21T20:32:21-07:00]

I've reviewed this and moved it to `in progress`. Below is my plan, then some questions about cases the ticket doesn't cover. I'll start once they're answered.

## Plan

- **Syntax:** a lookup is a `.` followed by a key, repeated, as in `.user.email`. It can be used wherever a literal value can, so it's always on the right of an operator: `{assignees = .user.email}`. A key made of digits indexes an array, as in `.remotes.0.url`, matching `config.Lookup` from #3D6HA-7FCAK.
- **The syntax tree:** a lookup is a new kind of `Value`. `String()` prints it as it's written, and `Render()` colours it aqua (6).
- **Resolving lookups:**
  - When a filter is applied, the app takes one snapshot of the application state.
  - A new `filter.Resolve(node, snapshot)` returns a copy of the expression, with each lookup replaced by its value from that snapshot. Every lookup is done on the same snapshot, and nothing is changed.
  - The SQL generator works as it does now. Each value becomes a `?` placeholder, and the looked-up value is passed as its argument.
  - The parsed expression keeps the lookup, so the filter field still shows `.user.email`, and the next query looks it up again.
- **What's exposed:** filters can look up the whole configuration from `config.toml`. Your project file's `[user]` table gives `.user.name` and `.user.email`.

## Questions

1. **A missing value:** what should `{assignees = .user.email}` do if `user.email` isn't configured?
   - (a) **An error:** like a syntax error, shown under the filter, such as "no setting .user.email". The list is left as it was. I'd suggest this.
   - (b) **Like `null`:** `= null` matches issues without the attribute, so it would quietly list every issue with no assignees.
   - (c) **Match nothing:** `!=` would then match everything, keeping it the opposite of `=`.
2. **Arrays and tables:** what if the value is an array, such as `.team.members`?
   - **An array:** I'd suggest it matches if the attribute equals any of the values, like SQL's `IN`. `!=` would match if it equals none of them.
   - **A table:** this can't be compared, so I'd make it an error.
3. **Type checks:** a literal's type is checked when the filter is parsed. For example, `~` needs a regular expression, and `null` only works with `=` and `!=`. A lookup's type is only known once it's looked up, so I'd apply the same checks then, as errors shown under the filter. Is that all right?
4. **Git defaults:** without a `config.toml`, there's no `user`. Should `.user.email` and `.user.name` fall back to git's `user.email` and `user.name`, as bylines do? A config file would override them. I'd suggest yes, so the lookups work out of the box, but that's a default for the configuration from #3D6HA-7FCAK, so it's your call.
5. **Key characters:** I'd allow letters, digits, `_` and `-` in keys, which covers TOML's bare keys. Quoted TOML keys with other characters couldn't be looked up. Is that enough?
6. **The preset filter:** should "Issues assigned to me" become `{assignees = .user.email} and {state != 'closed'}`, using the lookup? Or should it stay as it is for now?

---

["Brendan Berg" <brendan@berg.industries> 2026-10-21T21:24:55-07:00]

1. If a value is missing, the comparison cannot be made. Therefore the entire attribute expression should be considered UNKNOWN and return zero results (unless the attribute expression is joined to another expression with an `or` operator, in which case the other expression's matches are returned.
2. If the value is an array, the term will match if the operator applies to any of the values, with the exception of `!=` which will match if the operator applies to none of them. Tables cannot be compared, so it is a type error.
3. Yes, type checks can be performed after the lookup's value and type are known. To be clear, though, `~` can accept either a regular expression literal (`/.../`) or a string.
4. Do not make changes to configuration loading in this ticket.
5. Allowed characters in keys should be letters, digits, `_` and `-`.
6. Please don't increase the scope. Leave that change for a follow-up ticket.

---

[Claude <noreply@anthropic.com> 2026-10-21T21:46:45-07:00]

Done, following your answers, and moved to `in review`.

## Syntax

- **The form:** a lookup is a `.` followed by a key, repeated, as in `.user.email`. Keys are letters, digits, `_` and `-`. A key of digits indexes an array, as in `.remotes.0.url`.
- **Where it goes:** a lookup can be used wherever a literal value can, as in `{assignees = .user.email}` or `{@author != .user.email}`.
- **Errors:** an empty key, as in `.user.` or `.user..email`, is a syntax error, as is any other character in a key.
- **Display:** lookups are printed as they're written, and rendered in aqua (6).

## Resolving lookups

- **`filter.Resolve(node, lookup)`:**
  - It returns a copy of the expression, with each lookup replaced by its value.
  - `lookup` reads a single snapshot. The app passes `Snapshot().Lookup` from the application state, so every lookup in a query comes from the same snapshot, and nothing is changed.
  - The parsed expression keeps its lookups, so each query looks them up again.
- **SQL:** generation is unchanged. A looked-up value becomes a `?` placeholder, and the value is passed as its argument.
- **Types:**
  - strings and booleans are used as they are
  - integers and floats become numbers
  - an array becomes a list of values
- **Type checks:** these are applied to the values once they're looked up, just as they're applied to literals when they're parsed:
  - `~` takes a string, which must be a valid regular expression, or a `/regexp/` literal
  - `true` and `false` can only be compared with `=` and `!=`
- **Type errors:** a table, an array containing a table, or a value of another type, such as a TOML date, is a type error. For example, `.team.lead is a table, which can't be compared`. As with a syntax error, it's shown under the filter, and the filter isn't applied.

## Missing values and arrays

- **A missing value:** the expression is UNKNOWN, written as SQL's `NULL`. It matches nothing, whatever the operator, and `not` of it is still UNKNOWN. In an `or`, the other side's matches are returned, and in an `and`, nothing matches.
- **An array:** it matches if the comparison holds for any of its values. `!=` matches if `=` holds for none. An empty array matches nothing with `=`, and everything with `!=`.

## In the app

- **Applying a filter:** this resolves it, so type errors are caught then.
- **Reloading:** the list is reloaded, for example after an issue is closed. The filter is resolved again from the current state, ready for when settings can change while ish runs.
- **Unchanged:** configuration loading and the "Issues assigned to me" filter are left as they were, as you asked.

## Testing

- **Parsing:** lookups in each position, including fields, printing them back, and the syntax errors.
- **Rendering:** lookups in aqua.
- **Resolving:**
  - strings, integers, floats, booleans, regular expressions, arrays, empty arrays and missing values
  - the parsed expression unchanged, and each lookup made once from the function given
- **Type errors:** tables, tables in arrays, numbers with `~`, `true` with `<` and invalid regular expressions.
- **Against a database:**
  - scalars, fields, numbers and regular expressions
  - arrays with `=` and `!=`, and empty arrays
  - missing values with every operator, `not`, `and` and `or`
  - values passed as parameters
- **In the app:** filtering with a lookup, a type error shown without applying the filter, a missing value matching nothing except through `or`, and reloading after the state changes.
- **In a real terminal,** on a clone of this repository with `user.email` in `.ish/config.toml`, `{assignees = .user.email}` applied, with the lookup drawn in aqua.
