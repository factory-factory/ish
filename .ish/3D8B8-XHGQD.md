# Update string literals to support expression interpolation

["Brendan Berg" <brendan@berg.industries> 2026-10-22T00:04:52-07:00]

- state: in review
- assignees: <brendan@berg.industries>
- relation.parent: #3CZWN-EBVEH
- relation.depends-on: #3D97C-F62P7

---

["Brendan Berg" <brendan@berg.industries> 2026-10-22T00:04:52-07:00]

Update string literal syntax in filter expressions to allow string interpolation. Expressions enclosed in parentheses can be inserted into strings if they are preceeded by a backslash character.

The content between the opening `\(` of an expression to its closing `)` should be parsed identically to parenthesized expressions as implemented in #.

The value resulting from the evaluation of the expression should be converted to a string and inserted in place of the parenthesized expression.

Expression syntax was specified and implemented in #3D97C-F62P7.

---

[Claude <noreply@anthropic.com> 2026-10-22T17:12:06-07:00]

I've reviewed this and moved it to `in progress`. Below is my plan, then some questions. I'll start once they're answered.

## Plan

- **Syntax:** in a single- or double-quoted string, `\(` starts an expression, which runs to its matching `)`, as in `'#\(.issue.id)'` or `"\(.user.name :lower)@example.com"`.
  - The expression is parsed exactly as a parenthesized expression from #3D97C-F62P7 is, including nested parentheses and strings.
  - So `'a\(('b' :upper))c'` works, even with the same quotes inside.
  - `\(` isn't a valid escape today, so no existing filter changes meaning.
  - A literal `\` followed by `(` is still written `\\(`.
- **Where:** interpolation works in every string: a value compared with an attribute, a string inside an expression, and a function's argument. Regular expression literals (`/…/`) are taken literally, as now, so they don't interpolate.
- **Evaluation:**
  - a string with interpolations becomes an expression, which joins its text and the inserted values
  - it's evaluated as other expressions are: by an SQL function written in Go, and once in Go when the filter is applied, to report errors
  - lookups come from the same snapshot
- **Display:** the string is printed as it's written. The string's text is green, as now. `\(` and `)` are yellow, like parentheses, and the expression inside is coloured as expressions are.

## Questions

1. **Converting to strings:**
   - strings are inserted as they are
   - numbers as written in JSON, such as `3` or `2.5`
   - booleans as `true` and `false`

   Is that right?
2. **`null`:** in SQL, `'a' || NULL` is `NULL`. Following the earlier answer to use SQL's behaviour for `null`, I'd make the whole string `null` when an inserted value is, so a comparison with it is UNKNOWN. Or would you rather `null` were inserted as empty text, or as the word `null`?
3. **Lists:** should a list be an error, leaving it to `:join` to make a string? Or should it be inserted somehow, such as joined with `, `, or as JSON? I'd suggest an error.
4. **Regular expressions:** in a string used as a pattern, as in `{title ~ '^\(.prefix)'}`, should inserted text be inserted as it is? Then its `.` and `*` act as pattern characters. Or should it be escaped, so it matches literally? I'd suggest inserting it as it is, for consistency, with a `:quotemeta`-style function later if needed.

---

["Brendan Berg" <brendan@berg.industries> 2026-10-22T17:15:38-07:00]

1. Yes, that is correct.
2. If an expression evaluates to `NULL`, insert the empty string.
3. Lists should be errors.
4. When an interpolated string is used as a pattern to match against with `~`, interpolated values should be inserted as is.

---

[Claude <noreply@anthropic.com> 2026-10-22T17:31:34-07:00]

Done, following your answers, and moved to `in review`.

## Syntax

- **Inserting an expression:** in a single- or double-quoted string, `\(` starts an expression that runs to its matching `)`, as in `'issue \(.count)'` or `"\(.user.email :lower)"`.
  - The expression is parsed exactly as a parenthesized expression from #3D97C-F62P7 is.
  - Nested parentheses and strings work, even with the same quotes, as in `'a\(('b' :upper))c'`.
- **Literal text:** `\\(` is still a backslash followed by a parenthesis. `\(` wasn't a valid escape before, so no existing filter changes meaning. A string without `\(` is a literal, as before.
- **Where it works:** in every string: a value compared with an attribute, a string inside an expression, and a function's argument. Regular expression literals (`/…/`) are unchanged, and don't interpolate.
- **Errors:** inside `\(…)` they're reported as in other expressions. For example, `'x\()'` is "expected a value".
- **Display:** strings print as they're written. The text is green, `\(` and `)` are yellow, and the expression inside is coloured as expressions are.

## Values

- **Converting:** strings are inserted as they are, numbers as written in JSON (`3`, `2.5`), and booleans as `true` and `false`.
- **`null`:** this is inserted as empty text, including a lookup that finds nothing. So `'[\(.missing)]'` is `[]`.
- **Lists:** a list is an error: "can't insert a list in a string; use :join to make a list a string".
- **Patterns:** with `~`, inserted values are inserted as they are, so they act as pattern characters.

## Evaluation

- **As an expression:** an interpolated string is an expression, so it's handled as other expressions are.
  - It's evaluated in SQL by a new SQL function, `ish_interpolate`, which joins the text and the inserted values.
  - It's evaluated once in Go when the filter is applied, which reports errors such as a list.
  - Its lookups come from the same snapshot, and are passed as parameters.

## Testing

- **Printing:** strings print as they're written, including nested quotes, escapes, lists and patterns. Strings without `\(`, and with `\\(`, are literals.
- **Values:** numbers, booleans, functions, arithmetic, nested strings, escapes and `null` as nothing. An interpolated string can have functions applied to it. A list is an error, and a joined list isn't.
- **Syntax errors:** an empty expression, an unclosed expression and an unclosed string.
- **Rendering:** each part's colour.
- **Against a database:** compared with attributes and fields, as a pattern, with `null`, and inside an expression.
- **In a real terminal,** on a clone of this repository with an uppercase `user.email`, `{assignees = '\(.user.email :lower)'}` listed the issues assigned to `@Brendan Berg`, with `\(` drawn in yellow.
