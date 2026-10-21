# Implement custom syntax highlighting for rendering filter expressions

[<brendan@berg.industries> 2026-10-12T21:59:07-07:00]

- state: closed
- assignees: <brendan@berg.industries>
- relation.parent: #3CSJ4-PEZ1N

---

[<brendan@berg.industries> 2026-10-12T21:59:07-07:00]

Implement custom syntax highlighting for filter expressions in the `ish` TUI.

- Curly braces should be blue
- Attributes should be magenta
- Operators should be bold and bright white
- String and numeric values should be green
- Boolean values should be green and bold
- `null` should be grey
- Parentheses should be yellow
- Logical operators should be white

The syntax highlighting should be part of the bubbletea `Render` method on the AST nodes.

---

[Claude <noreply@anthropic.com> 2026-10-12T23:48:39-07:00]

Done, and moved to `in review`.

- Every node now has a `Render()` method alongside `String()` and `Where()`. It returns the same text as `String()`, with each kind of token coloured as this issue lists.
- Bubble Tea itself has no `Render` method; that's Lip Gloss's name for styling text. So I added `Render() string` to the `filter.Node` interface, and it uses Lip Gloss styles.
- The issue view's filter field (#3CVMN-Q41P7) now shows the applied filter through `Render()`. While editing, it still shows the raw text without colours.

## Colours

These are the standard 16 ANSI colours, like the rest of the TUI, and are defined in `styles/style.go`:

- braces: blue (4)
- attributes: magenta (5)
- operators: bold bright white (15)
- strings, numbers and regular expression literals: green (2)
- `true` and `false`: bold green
- `null`: grey (8)
- parentheses: yellow (3)
- `and`, `or` and `not`: white (7)

The issue's list predates regular expression literals (`/^in /`) from #3CTX4-6BP6E. They're values, so I coloured them green like strings.

## How it's built

- `String()` and `Render()` share one formatter, which takes a set of styling functions. `String()` passes ones that leave the text as it is, and `Render()` passes ones that colour it.
- This means the coloured text can't drift from the plain text. Parentheses added to keep the meaning of trees built in code are coloured like any others.
- Spaces between tokens aren't styled.

## Testing

- With its colour codes removed, `Render()` matches `String()` for parsed expressions and for a tree built in code.
- Each kind of token (braces, attributes, every operator, strings, numbers, regular expressions, booleans, `null`, parentheses, `and`, `or` and `not`) is rendered in its own style.
- Each style has the colour and boldness this issue gives.
- The filter field shows the highlighted filter, and the raw text while editing.
- In the real app on a copy of `.ish`, the terminal drew the default filter with braces blue, `state` magenta, `!=` bold bright white and `'closed'` green.

---

[Claude <noreply@anthropic.com> 2026-10-12T23:48:46-07:00]

A correction to my previous comment: the filter field from #3CVMN-Q41P7 is in the issue _list_, not the issue view. That's where the highlighted filter appears.
