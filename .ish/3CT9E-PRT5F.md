# Update markdown rendering style to indent wrapped lines in lists

[<brendan@berg.industries> 2026-10-12T00:10:24-07:00]

- state: closed
- assignees: <brendan@berg.industries>

---

[<brendan@berg.industries> 2026-10-12T00:10:24-07:00]

Currently, when markdown is rendered in comments, the text in bulletted and numbered lists isn't indented when lines wrap. Use a custom style to align the start of a wrapped line with the start of the line above it.

---

[Claude <noreply@anthropic.com> 2026-10-13T00:48:09-07:00]

I've looked into this, and it can't be done with a glamour style. I need you to choose an approach before I change any code.

## What happens now

I rendered a card with long list items through the card's own glamour settings. Each continuation line starts at the list's indentation, not under the item's text:

```
• A list item with enough text in it that it has to wrap onto a second line and
maybe a third line too
  • A nested item with enough text in it that it also has to wrap onto another
  line
1. A numbered item with enough text in it that it has to wrap onto a second line
as well
[ ] A todo item with enough text in it that it has to wrap onto a second line
here
```

## Why a style can't fix it

- Glamour v2 renders a whole list as one block. It writes each item's marker (`• `, `1. `, `[ ] `) followed by its text, then wraps the entire block in a single pass and indents it as a unit.
- The wrapping step doesn't know where each item's marker ends, so it can't indent continuation lines to match.
- None of glamour's list style options can change that. List styles only offer prefixes, colours, a uniform indent, a margin, and the indent per nesting level. There's no hanging-indent option.

## Why fixing it afterwards isn't reliable

The obvious workaround is to take glamour's output, rejoin each item's wrapped lines, and wrap them again with a hanging indent. But glamour's wrapping (`lipgloss.Wrap`) also breaks directly after `,`, `.`, `-` and similar characters, without a space. For example, `a,b,c` can become `a,` and `b,c`, and `https://example.com/a/path` can become `https://example.` and `com/a/path`. The output doesn't show whether a space was removed at each break, so rejoining the lines would sometimes add stray spaces inside URLs, comma lists or hyphenated words.

## Options

- (a) **Change glamour itself**, so each list item is wrapped on its own with a hanging indent the width of its marker. This is the only clean fix. It's a moderate change to glamour's internals, because each item would have to be rendered as its own block. It could go upstream as a pull request, which would take time to be accepted, released and picked up. Meanwhile ish could use a patched copy of glamour kept in this repository, through a `replace` directive in `go.mod`. That's a change to how a dependency is sourced, so it needs your approval under `agents.md`.
- (b) **Re-wrap list items after rendering** anyway, accepting the occasional stray space described above. I don't recommend this.
- (c) **Leave it as it is** until glamour supports hanging indents, and close or defer this issue.

I recommend (a): open a glamour pull request, and use a patched copy here until it's released. Which would you like, and if (a), may I add the patched copy?

---

[Claude <noreply@anthropic.com> 2026-10-13T00:57:58-07:00]

Done, and moved to `in review`. You chose option (a) in our conversation: patch glamour locally, without an upstream pull request, since others are already working on this in glamour.

## The patch

- glamour v2.0.1 is now vendored in `third_party/glamour`, with its MIT licence. `go.mod` points at it with `replace charm.land/glamour/v2 => ./third_party/glamour`.
- The patch is small, and every change is marked "Local patch for ish":
  - `ansi/listitem.go` and `ansi/task.go` write an invisible private-use character after each item's marker (`• `, `1. `, `[ ] `), marking where the item's text starts.
  - Lists now set a new `HangingIndent` flag (`ansi/elements.go`), and `ansi/blockelement.go` uses it when a list finishes.
  - When a list finishes, a new `wrapHanging` (`ansi/hanging.go`) wraps each line that has the mark to the width left after the marker. It indents continuation lines by the marker's width and removes the mark. Other lines, such as nested lists that were already wrapped, are wrapped exactly as before.
- This works without rejoining any text, because when a list finishes, each item's text is still on a single unwrapped line. That avoids the stray-space problem I described.

## Result

```
• A list item with enough text in it that it has to wrap onto a second line and
  maybe a third line too
  • A nested item with enough text in it that it also has to wrap onto another
    line
1. A numbered item with enough text in it that it has to wrap onto a second line
   as well
[ ] A todo item with enough text in it that it has to wrap onto a second line
    here
```

Each item aligns to its own marker, so `10. ` gets a 4-column indent and `1. ` a 3-column one.

## Testing

- glamour's own test suite passes on the patched copy, as it did before the patch.
- New tests in `third_party/glamour/ansi/hanging_test.go` cover:
  - the indent for different marker widths
  - lines without the mark
  - several items
  - styled markers, whose width is measured without escape codes
  - a width too narrow for the marker, which falls back to glamour's normal wrapping
- A new ish test (`TestCardViewIndentsWrappedListItems`) renders a card with bullets, nested bullets, numbered items and todos, and checks each continuation line's indent. I confirmed it fails with the patch turned off.

## Something I noticed along the way

Glamour wraps card text at a fixed 80 columns, but a card's border and padding add 6 more. So in an 80-column terminal, the end of a long line in a card is cut off by the edge of the screen before it wraps. This is how it already behaved, and it affects every kind of text, not just lists. It might be worth its own issue: the card's wrap width could follow the terminal's width.
