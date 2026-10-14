# Update markdown rendering style to give code blocks a grey background

[brendan@berg.industries 2026-10-12T10:20:45-07:00]

- state: closed
- assignees: brendan@berg.industries

---

[brendan@berg.industries 2026-10-12T10:20:45-07:00]

When rendering markdown in comment cards, code blocks are difficult to distinguish from regular text. Update the glamour styles to give code blocks a grey background.

---

[Claude 2026-10-13T01:05:10-07:00]

I've tried this with glamour's styles alone. It works, but doesn't look good, so I'd like you to choose between two approaches before I change any code.

## Styles alone

Glamour highlights code blocks with chroma. Chroma's terminal output deliberately doesn't paint a theme's background colour, so a background only appears where each token's own style sets one. Setting a grey background on every chroma token style (plus the plain style used for code without a language) gives grey behind the characters, but:

- **The right edge is ragged:** each line is grey only as far as its text goes.
- **Blank lines in the code have no grey at all,** which splits the block into pieces.
- **Tabs stay unpainted,** because terminals don't paint the background under the cells a tab skips. Indented code then has gaps.
- **The left margin isn't painted.**

Here's a preview, where `«BG»` marks where the grey starts:

```
  «BG»func«BG» «BG»main«BG»()«BG» «BG»{«BG»
  «BG»
  «BG»\t«BG»fmt«BG».«BG»Println«BG»(«BG»"hi"«BG»)«BG»
  «BG»}«BG»
```

## Options

- (a) **Styles only, as the issue says.** Grey behind the text, with the gaps above. This needs no change to glamour.
- (b) **Styles plus a small patch to the vendored glamour** (`third_party/glamour`), marked "Local patch for ish" like the list indent patch from #3CT9E-PRT5F. Code block lines would be padded with grey to the block's full width, and tabs turned into spaces, so each code block becomes a solid grey rectangle. I'd add tests in glamour and in ish, as before.

I recommend (b), since its whole point is making code blocks stand out from the text around them. Which would you like?

---

[brendan@berg.industries 2026-10-13T14:13:03-07:00]

This is a test comment with a code block.

```
func stringPtr(s string) *string {
    return &s
}
```

---

[Claude 2026-10-13T14:46:10-07:00]

Done, and moved to `in review`. In our conversation you suggested a third approach, applying the background after glamour has rendered the text, which turned out to be better than either of my options. It needs no change to glamour.

## How it works

- The card's glamour style sets the code block's `BlockPrefix` and `BlockSuffix` to two private-use characters, which glamour writes just before and just after each code block. Syntax highlighting is otherwise unchanged.
- After rendering, `paintCodeBlocks` (in `models/codeblock.go`) finds the lines between those markers and paints them grey, then removes the markers. Glamour pads every rendered line to the full width, so each code block becomes a solid grey rectangle, including any blank lines in the code.
- Painting a line takes two details:
  - **Resets:** glamour's output turns all styling off dozens of times per line, which would also turn the grey off. The grey is switched back on after each reset.
  - **Tabs:** terminals don't paint the background under a tab, so tabs are turned into spaces, up to the next 4-column tab stop. The line is then re-padded to its original width.
- The grey is 236 on dark terminals and 254 on light ones, from the 256-colour palette. It's chosen the same way the card picks glamour's dark or light style.

## Other changes

- This replaces your uncommitted experiment in `models/card.go`. That one gave code blocks a whole new style with a grey `BackgroundColor`, which turned off chroma's syntax highlighting and the block's margin. Now only `BlockPrefix` and `BlockSuffix` are set, so code keeps its highlighting, and the unused `stringPtr` helper is gone.

## Testing

- Tests cover:
  - expanding tabs, including around escape sequences
  - painting a line: the grey is restored after every reset, and the line keeps its colours and width
  - finding and painting blocks, including an empty block, and keeping the width of lines that held a marker
  - a full card with a Go block (containing a blank line and a tab) and a block without a language, checking that:
    - every code line is painted
    - the text around it isn't painted
    - every line keeps the full width
    - syntax highlighting is kept
- I checked that the painting tests fail if the grey isn't restored after resets.
- In the real app, a code block's lines were drawn with the grey background, and the text before and after it without.
