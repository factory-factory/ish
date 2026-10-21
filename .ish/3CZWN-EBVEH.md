# Update filter expression language to add functions

[<brendan@berg.industries> 2026-10-15T23:45:09-07:00]

- state: planning
- assignees: <brendan@berg.industries>

---

[<brendan@berg.industries> 2026-10-15T23:45:09-07:00]

Expanding value types in expressions to include dynamic values instead of just literals opens more possibilities for useful queries. Dynamic values include properties of the environment like `user.email` as well as functions like `currentTime()`.

It would be nice to unify both types of lookup...

## Expressions

When used as an r-value in place of a literal, expressions must be enclosed in parentheses.

It would be nice to be able to query against things like `user.email`

## String interpolation

String literal syntax should be expanded to allow expressions to be evaluated inline. The syntax to insert an expression inside a string is similar to Swift's string interpolation: the expression starts with a `\(` and ends at the matching close parenthesis `)`. This allows unambiguous identification of expressions inside string literals while normal parentheses remain easy to type.

`{repository = 'git@github.com/\(.user.name)/\(.repository.name)'}`

## Object lookups

It is often necessary to look up application state in a filter--for example, the current user's email address is stored in applicatin state.

Object lookups use dot notation, and can be chained to traverse nested objects. So in our example discussed earlier, `{assignees = (.user.email)}` would filter issues to show only items with the user's email in the assignee metadata.

## Functions

Functions are designed to be chained in filter expressions, and always start with a colon. Functions follow the

- Relative time: `:time`, `:date`, `:datetime`
- String manipulation: `:lower`, `:upper`, `:join`, `:split`, `:replace`, `:trim`
- Collections: `:length`, `:merge`, `:concat`, `:unique`/`:distinct`, `:lookup`, `:keys`, `:values`
- Numeric: `:abs`, `:ceil`, `:floor`, `:max`, `:min`, `:pow`
- Encoding: `:base64.decode`/`.encode`, `:json.decode`/`.encode`, `:hash.sha256`/`.sha384`/`.sha512`/`.md5`
- Type conversion: `:string`/`:number`/`:boolean`

Allowed effects are `decides` and `computes`. (additionally, all functions should converge)

```
(-435 :abs)

(.email :lower :hash.sha256)
(.path :replace('/', '\'))
('Mixed Case String' :lower)

('2026-10-17' :date)
('12:46:03' :time)
('-3h' :fromnow)
```
