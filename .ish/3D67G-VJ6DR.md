# Add object lookup syntax to filter expressions

["Brendan Berg" <brendan@berg.industries> 2026-10-20T11:56:53-07:00]

- state: planning
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
