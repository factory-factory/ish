# Implement parser for filter expression language

[brendan@berg.industries 2026-10-12T10:40:12-07:00]

- state: closed
- assignees: brendan@berg.industries
- relation.parent: #3CSJ4-PEZ1N

---

[brendan@berg.industries 2026-10-12T10:40:12-07:00]

Create a parser that converts freeform text into filter expressions. The parser output should be AST nodes representing the expression.

**NOTE:** The specification here differs slightly from the one discussed in #3CSJ4-PEZ1N. Discussion and specification in this ticket should supercede any previous spec.

## Language Specification

Filter expressions are comprised of attribute expressions joined by logical operators. Attribute expressions are always enclosed in curly braces and in the form `{` `attribute` `operator` `value` `}`.

Attributes are alphanumeric strings that match the same class of characters as issue metadata tags. (The special characters `-`, `_`, `.`, and `/` are allowed. Attributes must not contain whitespace or colons.)

The operator is one of the following symbols:

- `=`: equal to
- `!=`: not equal to
- `<`: less than
- `>`: greater than
- `<=`: less than or equal to
- `>=`: greater than or equal to
- `~`: regular expression match

Values are any JSON scalar value: `null`, `true`, `false`, numbers, or strings. (Strings deviate from the JSON spec slightly because they can be enclosed in either single or double quotes.)

The logical operators are `and`, `or`, and `not`. The unary `not` operator binds the most tightly. The binary operator `and` binds more tightly than `or`. Parentheses may be used for grouping.

## AST Nodes

The parser should create a non-lossy representation of the input string as AST Nodes. The AST Node types all adhere to the `Node` interface, which provide methods for both printing the AST as a string and for generating a SQL WHERE clause.

All nodes have a `parenthesized` attribute to indicate whether the expression was enclosed in parentheses when parsed. (This is so the AST can be converted back into a string with the same exact tokens as were parsed.)

**`AttributeExpression`**

- `Attribute`: string
- `Operator`: string-based enum
- `Value`: json scalar

**`AndExpression` and `OrExpression`**

- `LeftExpression`: Node
- `RightExpression`: Node

**`NotExpression`**

- `Expression`: Node

## Parser Errors

If an error is encountered, the parser should indicate the position in the string where an unexpected token was found so the UI can highlight the error.

---

[Claude 2026-10-12T22:34:11-07:00]

I've implemented the parser to this specification, along with the SQL generation it needs. Three points need your decision, so this stays `in progress` until they're settled. The SQL semantics in question 1 are implemented as I recommended on #3CSJ4-PEZ1N, so they're easy to change if you decide differently. Nothing is committed yet.

## Questions

1. **SQL semantics (from #3CSJ4-PEZ1N, questions 2 to 4, not yet answered).** `Where()` needs these rules. Currently:
   - An attribute with several values matches `=`, `<`, `>`, `<=`, `>=` or `~` if *any* of its values does. `!=` is the exact opposite of `=`, so `{assignees != 'bob'}` excludes an issue assigned to bob and alice.
   - An issue without the attribute matches only `!=`, so the default `{state != 'closed'}` keeps issues with no state. `{key = null}` matches issues without the attribute, and `{key != null}` those with it.
   - A number compares numerically, and only with values that are numbers. `{estimate > 3}` matches `5` but not `8h`, and `{estimate = 5.0}` matches `5`. A quoted `'5'` compares as text.
   - `true` and `false` match the text `true` and `false`. Strings compare as case-sensitive text. Regular expressions are Go's syntax, so `(?i)` ignores case.

   Do these match what you intend?
2. **Backslashes in regular expressions.** The spec says values are JSON scalars, so strings use JSON escapes. A regular expression must then double its backslashes: `{state ~ '^in\\s'}` for `^in\s`. Worse, `\b` is a JSON escape for backspace, so `{title ~ '\bfoo'}` silently searches for a backspace rather than a word boundary. Options:
   - (a) Keep strict JSON escaping, as now.
   - (b) For `~` only, treat the string literally, except that `\'` or `\"` escapes the enclosing quote. `'\bfoo\s'` then reaches the regular expression unchanged.

   I recommend (b), since regular expressions are where backslashes come up.
3. **Colons in names.** This issue disallows colons in attribute names, but the metadata key rule from #1PZ4D-B0NEF (in `store`, and the database's `CHECK`) allows them. A key with a colon could be stored but never filtered on. No current issue uses one. Should I remove colons from the key rule, so keys and filter names match exactly? I recommend yes. `ish init` would then reject keys with colons, like other disallowed characters.

## Decisions I made

- **Parentheses** are counted in a `Parentheses` field on every node. A true/false "parenthesized" flag can't tell `(x)` from `((x))`, and the spec asks for a lossless tree.
- **Printing:** `String()` reproduces the parsed tokens exactly, including each value as written (`'closed'` stays single-quoted, and `1.50` stays `1.50`). Only the space between tokens is normalized, to single spaces. Parsing the printed text gives back the same tree. Trees built in code without parentheses get the ones their meaning needs.
- **Errors** are a `*SyntaxError` with the half-open range `[Start, End)` of the unexpected text, counted in characters (runes, not bytes) so the interface can highlight it. At the end of the input, `Start` equals `End`. The message is plain, for example `column 10: expected a value: a quoted string, a number, true, false, or null`.
- **Checks while parsing:** `null`, `true` and `false` can only be compared with `=` and `!=`. `~` needs a quoted, valid regular expression, and an invalid one is reported at the value.
- **Keywords** (`and`, `or`, `not`, `null`, `true`, `false`) are lowercase only, as in JSON.
- **Spacing:** tokens may be separated by any amount of space, or none where it's unambiguous (`({a=1})and({b=2})`).
- **An empty or all-space expression** is an error. The interface can treat an empty field as "no filter" itself.

## What's built

- A new `filter` package, with `Parse` and the node types `AttributeExpression`, `AndExpression`, `OrExpression` and `NotExpression`. All of them implement `Node`, which provides `String()` and `Where()`.
- `Where()` returns an SQL condition and its arguments. Values and attribute names are always passed as parameters, never written into the SQL. Comparisons look inside each attribute's JSON array of values with `json_each`, using two Go functions registered with SQLite: `ish_number` (number comparisons) and `ish_regexp` (`~`).
- Nothing uses filters yet. Filtering the list is the next step.
- Tests cover:
  - parsing every operator and kind of value, precedence and grouping, round trips through `String()`, and every syntax error with its exact position
  - every rule in question 1, run as real queries on a database built by `store.Init` in a temporary directory, including a value containing SQL

---

[brendan@berg.industries 2026-10-12T22:38:06-07:00]

All good catches. Hopefully this clarifies things:

1. **SQL Semantics**
    - The comparison operator behavior of having operators other than `!=` match *any* values and `!=` match *all* values makes sense.
    - The behavior you described for matching expressions when the attribute it refers to does not exist on the issue makes sense. So does the behavior for testing whether the attribute is set by using `{key = null}` or `{key != null}`.
    - Numeric comparisons as described make sense. There's no expectation that `8h` would compare numerically.
    - Matching `true` and `false` against strings seems to be the right approach, since we have no way of putting non-string values in the database.

2. **Regular Expressions** Let's add another type that can go in the value part of the expression that behaves similar to JavaScripts regular expression literals. For now, a string enclosed in `/` forward slashes should be treated literally for regular expression search. Syntactically, a regexp literal can only follow the `~` operator.

3. **Colons in Attribute Names** It was an oversight to allow colons in metadata keys in the first place. Please update the database code to only allow the characters `. _ - /`. Also update the initialization code to reject keys with colons. I will open a separate follow-up ticket to handle changes to the editing UI.

---

[Claude 2026-10-12T23:07:54-07:00]

Done, following your answers, and moved to `in review`.

## 1. SQL semantics

These are unchanged, since they're the rules you confirmed. Tests run each one as a query against a real database.

## 2. Regular expression literals

- A value can now be a regular expression literal between slashes, such as `{state ~ /^in\s/}` or `{title ~ /\bfoo\b/}`. Its text is taken literally, so backslashes reach the regular expression unchanged and `\b` is a word boundary.
- As in JavaScript, `\/` stands for a slash inside the literal. Every other backslash is kept as written, including `\\`, so `/a\\/` is the pattern `a\\`.
- A literal anywhere but after `~` is an error (`a regular expression literal can only follow ~`). So are an unclosed literal, an invalid pattern, and a control character inside one.
- In the tree, it's a value whose data has the new type `filter.Regexp`. `String()` prints it back as written. One built in code is printed with its slashes escaped.
- Since you said "for now", flags after the closing slash, such as `/x/i`, aren't supported and are reported as an error. `(?i)` ignores case.
- Quoted strings still work after `~`, with JSON escaping as before. Only the literal is new.

## 3. Colons in keys

- Metadata keys may now only contain letters, digits, and `. _ - /`. That applies to the database's `CHECK` constraint and to the check `ish init`, the startup sync and saving all use. A file with a colon in a key is skipped and reported, for example `metadata key "ns:key" may only contain letters, digits, and . _ - /`. Filter attribute names now allow exactly the same characters.
- Because the `CHECK` is part of the schema, I raised the schema version (`PRAGMA user_version`) from 1 to 2. When `ish` starts with a database at an older version, it rebuilds it with `runInit` instead of syncing it. Your current database is at version 1, so the first start after this change will print `Rebuilt .ish/.metadata.ish`. I checked this on a copy of your `.ish`: 20 issues, and the next start printed nothing.
- No current issue uses a colon in a key, so nothing is skipped.

## Testing

- Parser tests for regular expression literals: literal text, `\/`, `\\`, an empty literal, round trips, a literal built in code, and each new error with its position.
- Query tests with literals, including `/\bbob\b/` matching only the issue assigned to bob.
- `store` tests for rejecting a colon in a key, both in the `CHECK` and when `ish init` indexes a file, and for the new `store.Current` (whether a database has the current schema).
- A `main.go` test that an old-schema database is rebuilt at startup.
