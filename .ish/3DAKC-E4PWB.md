# Support multiple conditions matching the same comment or attribute in filter expressions

["Brendan Berg" <brendan@berg.industries> 2026-10-23T14:32:53-07:00]

- state: in review
- assignees: <brendan@berg.industries>

---

["Brendan Berg" <brendan@berg.industries> 2026-10-23T14:32:53-07:00]

Currently, `{@comments.author = 'alice@example.com'} and {@comments.text ~ 'LGTM'}` matches an issue where alice wrote any comment and any comment says LGTM, even if they aren't the same comment.

New syntax would be required to restrict matches to the same issue.

## Design Exploration

1. **Add logical operators within attribute expressions**

This change would allow `and` and `or` operators to join multiple clauses *within* the curly braces.

```
{@comments.author = 'alice@example.com' and @comments.text ~ /LGTM/}
```

- **Makes it easy to accidentally mix tables:** there would have to be an additional test to ensure that both field specifiers start with `@comments` so the user doesn't enter a meaningless filter like `{@comments.author = 'a@example.com' and state != 'closed}`.

- **And can change meaning depending on whether it's inside a brace:** The existing construct `{...} and {...}` matches if each condition individually matches some value; the new `{... and ...}` would mean the same value must match *both* conditions.

- **Or inside braces doesn't add expressivity:** `EXISTS(a OR b)` is the same as `EXISTS(a) OR EXISTS(b)`.

Ultimately, adding this construction makes it harder to reason about filter expressions and should not be used.

2. **Pull the collection name out of the attribute expressions**

Enclose attribute expressions in another type of expression that indicates enclosed expressions should all be true of the *same* row.

```
{@comments {@author = 'alice@example.com'} and {@text ~ /LGTM/}}
```

- **Makes it clear we're querying in a namespace:** Because it breaks up the `@comments` part from `@author` and `@text`, it's clear that we're querying the *comment's* author and content. This shadowing may seem intuitive to some, but also may not be what all users expect.

- **Doesn't handle the attribute case:** One downside of this approach is that it doesn't easily apply to attribute collections. Making sure an estimate is between 3 and 7 cannot be written as `{estimate {> 3} and {< 7}}`. Additional syntax would need to be developed if we want to support this behavior.

- **Tests against null values change behavior:** The `EXISTS` keyword treats a `NULL` condition as false instead of UNKNOWN. So a filter attempting to match a null or missing value would have unexpected results.

- **Can't use this syntax for attribute comparisons:** Without a way to refer to the outside value, inner comparisons can't test against attributes.

---

[Claude <noreply@anthropic.com> 2026-10-23T23:22:33-07:00]

## Recommended syntax: a subject, then row conditions

Everything inside one pair of braces is tested against the *same* row. After `{`, write the subject once, then one or more comparisons about it, joined by `and` and `not`. How a comparison starts depends on what kind of row the subject has.

**A single value per row** (an attribute's values, or `@dateModified`): the comparison starts with an operator, and applies to the value.

```
{estimate >= 3 and <= 7}
{assignees ~ /^a/ and ~ /@example\.com$/}
{@dateModified > ('-1w' :fromnow) and < ('-1d' :fromnow)}
```

**Several fields per row** (`@comments`): the comparison starts with a bare field name: `author`, `text` or `dateCreated`.

```
{@comments author = 'alice@example.com' and text ~ /LGTM/}
{@comments author = ('Bob' :user) and dateCreated > ('-1w' :fromnow)}
{@comments text ~ /LGTM/ and not author = 'alice@example.com'}
```

**Existing filters keep their meaning.**
- `{state != 'closed'}` is a scope with one comparison, and means exactly what it does today.
- `{@comments.author = 'x'}` is shorthand for `{@comments author = 'x'}`.

### Why it's intuitive

- **It reads like a sentence:** "estimate at least 3 and at most 7", or "a comment whose author is alice and whose text says LGTM".
- **The braces show what has to match together:**
  - `{a} and {b}` can be satisfied by two different rows, as now
  - `{a and b}` has to be satisfied by one
  - the subject is written once, so it's clear which rows are meant
- **It handles both cases:** comments, and attributes with several values, such as a range on one estimate, which neither design above covers.
- **No ambiguity for the parser:** after the subject, an operator means the subject is a value, and a name means it's a collection. One token of lookahead is enough.

### How it becomes SQL

Each scope becomes a single `EXISTS` over the subject's rows, with all of its conditions inside it:

```sql
-- {estimate >= 3 and <= 7}
EXISTS (SELECT 1 FROM attribute, json_each(attribute.value) AS v
        WHERE attribute.issueId = issue.id AND attribute.key = ?
          AND (ish_number(v.value) >= ? AND ish_number(v.value) <= ?))

-- {@comments author = 'alice@example.com' and text ~ /LGTM/}
EXISTS (SELECT 1 FROM comment
        WHERE comment.issueId = issue.id
          AND (comment.author = ? AND ish_regexp(?, comment.content)))

-- {@dateModified > … and < …}
EXISTS (SELECT 1 FROM history
        WHERE history.issueId = issue.id
          AND (history.dateModified > ? AND history.dateModified < ?))
```

- **Why it's a small change:** the SQL generator already builds each comparison's condition separately from the `EXISTS` around it. A scope joins those conditions with `AND` and `NOT`, then wraps them once.
- **Values:** lookups, expressions, `:user` and `:fromnow` work unchanged. Their values are resolved before the query and passed as parameters.

### Rules that prevent unintended consequences

1. **In a scope with more than one comparison, `!=` and `null` are errors.**
   - **`!=`:** at the top level, `{assignees != 'bob'}` means "no assignee is bob". Inside a row scope, it could only mean "this value isn't bob". Allowing it would make adding a second comparison change the meaning of the first.
   - **`not` instead:** `not` inside a scope applies to the row, as in `{@comments text ~ /LGTM/ and not author = 'a'}`. `not` before the braces, as in `not {…}`, means "no row matches".
   - **`null`:** this would ask for a row with no value, which the columns never have. `{x = null}` on its own keeps its meaning of "has none".
2. **A missing lookup makes the whole scope UNKNOWN.** `EXISTS` turns a `NULL` condition into false. Then `not {@comments author = .missing}` would match every issue, whereas a missing lookup otherwise matches nothing. `Resolve` already knows when a lookup is missing, so it can give `NULL` for the whole scope.
3. **Only the collection's own fields are allowed inside it.** In `{@comments …}`, only `author`, `text` and `dateCreated` are allowed. Issue fields such as `@title`, attributes, and nested scopes are errors. Otherwise `{@comments author = 'a' and state = 'open'}` would compile, and silently test the issue's state. This is the table-mixing problem described for design 1.

### Remaining pitfall

**`{assignees = 'alice' and = 'bob'}` never matches,** since a single value can't be both. "Assigned to both" is still written `{assignees = 'alice'} and {assignees = 'bob'}`. That follows from the one rule, so it's mostly a matter of documentation. The parser could also report an `and` of two `=` comparisons with different literals as an error.

### Compared with the designs above

- All three support comments and leave existing filter behavior unchanged.

| | Attributes | Meaning of `!=` |
|---|---|---|
| (1) `and` inside braces | partly; the name is repeated | changes inside braces |
| (2) `{@comments {…}}` | no; there's no name for the value | changes inside the scope |
| (3) this recommendation | yes | is an error inside a scope |

---

["Brendan Berg" <brendan@berg.industries> 2026-10-24T08:45:04-07:00]

The third option is the best overall. Let's go with that one. 

Some implementation notes:

- Only allow `and` and `not` as logical operators on conditions inside an expression.
- Update the expression parser and the SQL generation. (I don't think there's any new syntax highlighting that needs to be implemented...)

---

[Claude <noreply@anthropic.com> 2026-10-24T09:22:34-07:00]

Implemented the subject-then-conditions syntax.

- **Parser** (`filter/parse.go`): after the subject, braces hold conditions joined by `and`, each a comparison, `not` and a condition, or a condition in parentheses. A comparison starts with an operator, or, after `@comments`, with `author`, `text` or `dateCreated`. A single comparison still parses as an `AttributeExpression`, so existing filters are unchanged. The grammar comment is updated.
- **AST and SQL** (`filter/scope.go`): `ScopeExpression` becomes one `EXISTS` over the subject's rows (attribute values, `comment` or `history`) with every condition inside it. An issue field such as `@title` has one row, so its conditions are just joined. A single comparison, such as `{@comments author = 'x'}`, means exactly what `{@comments.author = 'x'}` does.
- **Resolving** (`filter/resolve.go`): lookups, expressions, `:user` and `:fromnow` resolve inside scopes. A missing lookup makes the whole scope UNKNOWN, so neither it nor `not` of it matches.
- **Highlighting:** none was needed. Scopes render with the existing styles.

Decisions:

- `or` inside braces is an error that suggests `{…} or {…}`.
- `!=` and `null` are errors in braces with more than one comparison; `not` is the way to negate a condition on a row.
- Parentheses are allowed inside braces, as in `{@comments not (author = 'a' and text ~ 'b')}`.
- Any attribute or field can be a subject, including `@title`, `@dateModified` and `@comments.text`.
- Only the collection's own fields are allowed inside `{@comments …}`; anything else, such as `@title`, is an error that lists the fields.

Tests are in `filter/scope_test.go`: printing and rendering, errors, and matching against databases for attribute ranges, values with several conditions, `not`, lists, expressions, lookups, `:user`, `:fromnow`, the same-comment versus any-comment distinction, and `@dateModified` ranges. The full suite passes. I also checked in the terminal that the same-comment filter matches the expected issues and that the `or` error appears.
