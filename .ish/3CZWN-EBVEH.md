# Update filter expression language to add functions

[<brendan@berg.industries> 2026-10-15T23:45:09-07:00]

- state: closed
- assignees: <brendan@berg.industries>

---

[<brendan@berg.industries> 2026-10-15T23:45:09-07:00]

Expanding value types in expressions to include dynamic values instead of just literals opens more possibilities for useful queries. Dynamic values include properties of the environment like `user.email` as well as functions like `currentTime()`.

## Object Lookup

It is often necessesary to look up application state in a filter. Fetching the current user's display name or email, the name of the developer's team or current project, or the repository URL of the issue store are all examples of application state accessible to the filter mechanism.

Object lookups use dot notation, and can be chained to traverse nested objects. So a filter query for issues opened by the current user would be written as `{@author = .user.email}`.

Note that object lookups work in any place a value is expected, so they do not need to be parenthesized if they are used on their own.

## Expressions

Expressions are syntactic constructs that transform values. They are used for mathematical operations, string manipulation, date manipulation, and type conversions.

When used as an r-value in place of a literal, expressions must be enclosed in parentheses.

## Binary operations

The simplest form of expression is a binary operation. The filter expression language supports the standard mathematical operators `+`, `-`, `*`, `/` `%` (remainder), and `**` (exponentiation); and comparison operators `=`, `!=`, `<`, `<=`, `>`, and `>=`.

## Functions

Functions in filter expressions are designed to be chained, so they always take at least one argument and return a value. The first argument is always the value being passed through the chain. Additional arguments may follow the function name if enclosed in parentheses. Functions always start with a colon.

For example, an expression to convert a string to lower case would look something like `('YELLING' :lower)`. A function like `join`, which takes a list to join and a string to join with would be written as `([1, 2, 3] :join(','))`.

Functions are to be implemented as application-defined SQLite functions written in go. The functions to be implemented are:

- **Date & time:**
  - `:time` - Converts a string in `hh:mm:ss` form into a time.
  - `:date` - Converts a string in `YYYY-MM-DD` form into a date.
  - `:datetime` - Converts a string in RFC 3339 format into a datetime.
  - `:fromnow` - Takes a string describing a duration and adds to the current datetime, returning a time relative to the present moment.

- **String manipulation:**
  - `:lower`
  - `:upper`
  - `:join`
  - `:split`
  - `:replace`
  - `:trim`

- **Collections:**
  - `:length`
  - `:distinct`
  - `:includes()`
  - `:indexof()`

- **Numeric:**
  - `:abs`
  - `:ceil`
  - `:floor`
  - `:max`
  - `:min`

- **Encoding:**
  - `:base64.decode`/`.encode`
  - `:json.decode`/`.encode`
  - `:hash.sha256`/`.sha384`/`.sha512`/`.md5`

- **Type conversion:**
  - `:string`/`:number`/`:boolean`

## String interpolation

String literal syntax should be expanded to allow expressions to be evaluated inline. The syntax to insert an expression inside a string is similar to Swift's string interpolation: the expression starts with a `\(` and ends at the matching close parenthesis `)`. This allows unambiguous identification of expressions inside string literals while normal parentheses remain easy to type.

`{repository = 'git@github.com/\(.user.name)/\(.repository.name)'}`

---

["Brendan Berg" <brendan@berg.industries> 2026-10-21T18:26:57-07:00]

## Child issues:

- #3D6HA-7FCAK: Create object to hold application state
- #3D67G-VJ6DR: Add object lookup syntax to filter expressions
- #3D6W6-1SEBX: Create filter expression function to convert display names to email addresses
- #3D97C-F62P7: Implement parenthesized expressions in filter language
- #3D8B8-XHGQD: Update string literals to support expression interpolation
- #3D9D0-MNFGA: Implement relative date filter expression function
- #3D9KG-P1X8Q: Implement encoding / decoding filter expression functions
