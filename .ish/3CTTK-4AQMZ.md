# Enable references to other issues

[brendan@berg.industries 2026-10-12T09:19:47-07:00]

- state: closed
- assignees: brendan@berg.industries

---

[brendan@berg.industries 2026-10-12T09:19:47-07:00]

It's helpful to be able to reference other issues when editing an issue. There are two ways to reference an issue: (1) as a metadata reference, and (2) as a hyperlink within comment text.

## Metadata references

References to other tickets may be stored in metadata attribute values as markdown links in the form `#3CTTK-4AQMZ` for references to issues in the same source control repository.

Therefore, to link to an issue that blocks the issue being edited, a user would add the following metadata line to the issue:

```
- relation.blocked-by: #34567-DEFGH
```

(Note that the `relation.` prefix is only a convention to group similar attributes under the same category.)

## Comment hyperlinks

Within comment markdown, any string starting with `#` followed by between 3 and 20 non-whitespace characters should be tested to determine whether the text matches the ID of an existing issue.

## Rendering issue hyperlinks

Links to other issues should be rendered in blue text with an underline. A selected hyperlink should be rendered in bold blue text with an underline.

## Interacting with issue hyperlinks

When a metadata attribute contains a hyperlink, the right arrow should select the hyperlink. Pressing enter when a link is selected should close the current issue and open the linked issue.

Navigating hyperlinks in comments should behave similar to selecting checkboxes for to-do items. (When a card is selected, the right arrow key descends into all selectable items in the comment, and up and down arrows select between them.) Pressing enter on a to-do item should still update that item, and pressing enter on a hyperlink should close the current issue and open the linked issue.

---

[Claude 2026-10-14T23:26:47-07:00]

I've reviewed this and moved it to `in progress`. Below is my plan, then some questions. I'll start once you've answered them.

## Plan

- **Finding references:**
  - A reference is `#` followed by 3 to 20 non-whitespace characters, and it counts only if that text is the ID of an issue in the metadata database.
  - Metadata values are checked one value at a time. Comment text is checked outside code spans and fenced code blocks.
- **Rendering:**
  - Links are blue (4) and underlined. The selected link is bold, blue and underlined.
  - Comment links are found after glamour renders them, using a private-use marker, as todo items are.
- **Selecting in a card:**
  - `→`/`l` starts selecting when the card has todo items or links. The selectable items are in reading order, and `↑`/`↓` move between them.
  - Enter toggles a todo item and opens a link. `←`/`h`/`esc` stop selecting, as they do now.
  - The help footer's "todos" label becomes something like "select".
- **Selecting in metadata:** with the cursor on a tag that has links, `→`/`l` selects its first link. The same keys then move, open and stop as in a card.
- **Opening a link:**
  - The current issue is saved, as it is when you go back to the list.
  - The issue view then sends a message asking to open the linked ID, and the list replaces the current issue with the linked one.

## Questions

1. **Trailing punctuation:** "Non-whitespace" includes punctuation, so in `see #3CSJ4-PEZ1N.` or `(#3CSJ4-PEZ1N)` the whole run isn't an ID. Should I strip trailing punctuation (`. , ; : ! ? ) ] } ' "`) when the whole run doesn't match? A leading `(` is already outside the match, since the reference starts at `#`.
2. **Code:** should `#ID` inside code spans and fenced code blocks stay plain text, as I've planned? This ticket's own comment writes `#3CTTK-4AQMZ` in a code span as an example.
3. **Case:** IDs are uppercase Crockford base32. Should `#3cttk-4aqmz` match, or only the exact ID?
4. **Unknown IDs in metadata:** in comments, unknown IDs are plain text. In metadata, should `#34567-DEFGH` (this ticket's example, which doesn't exist) also be plain text? Or should it look like a broken link, for example red, and not be selectable?
5. **Going back:** after following a link, should `esc`/`w` return to the list, as I'd suggest, or to the issue you came from? A history stack is more work, but it might be what you expect from links.
6. **Opening a single file:** `ish path/to/issue.md` opens the issue view without the list. There, should a link open the linked issue in place, with `esc` then quitting as it does now? I'd do this by starting the list model with the issue already open and keeping the current quit-on-back behaviour, so both modes share one code path.
7. **The issue title:** should references in the title be linked as well? I've left titles out, since the ticket doesn't mention them.
8. **Splitting the work:** this could be two issues: (a) finding and rendering references, and (b) selecting and opening them. I'm happy to do it as one, but if you'd prefer to review it in two parts, (a) is a natural place to stop.

---

[brendan@berg.industries 2026-10-14T23:30:06-07:00]

1. Yes, strip trailing punctuation
2. IDs in code spans or code blocks should NOT be treated as hyperlinks
3. For the initial implementation, IDs should be matched exactly
4. Unknown IDs should be plain text
5. A history stack is outside the scope of this feature. `esc`/`w` should return to the list view
6. When executing ish with a file path argument, links should open in place and `esc` should quit
7. IDs in the issue title should not be linked
8. Please tackle this as one issue

---

[Claude 2026-10-14T23:51:17-07:00]

Done, following your answers, and moved to `in review`.

## Finding references

- A reference is `#` followed by the ID of an issue in the metadata database, matched exactly.
- The text after `#` runs to the next whitespace. If that isn't an ID, it's tried again without trailing `. , ; : ! ? ) ] } ' "`, so `(#3CSJ4-PEZ1N).` works.
- IDs are 3 to 20 characters long. References to unknown issues stay plain text.
- One addition: a `#` directly after a letter or digit, as in `issue#3CSJ4-PEZ1N`, doesn't start a reference.
- **In metadata,** each value is checked. A value can be just the reference, as in `relation.parent: #3CSJ4-PEZ1N`, or contain one.
- **In comments,** text is found with the same Markdown parser glamour uses. Code spans, code blocks, HTML tags and link destinations are left out. Text in headings, lists and link labels is included.
- **Titles** aren't linked.

## Rendering

- Links are blue and underlined, and the selected link is bold as well.
- In comments, the text after a link keeps the style glamour gave it.
- A link inside a selected todo item keeps the item's highlight colours and is underlined, so it stays visible on the blue background.

## Keys

- **`→`/`l`** on a card starts selecting its todo items and links together, in reading order. On a tag with links, it selects the first link. The help footer labels it "select", in place of "todos".
- **`↑`/`↓`** move between items, and **`←`/`h`/`esc`** stop selecting, as before.
- **Enter** toggles a todo item, or opens a link.
- **Opening a link** saves the issue's changes first, as going back does. If they can't be saved, the issue stays open with the error, as it does when going back.
- **From the list,** the linked issue replaces the open one, and `esc`/`w` return to the list, with the last issue selected.
- **With a file path argument,** links open in place, and `esc`/`w` quit.
  - This replaces the old behaviour, where `esc` did nothing in this mode.
  - Both modes now share the list's code for showing issues. With a file path, ish starts the list model with the issue open (`app.Open`), so saving, comments and links work the same way.
  - The help footer still calls the key "back" in this mode.

## Other changes

- `store.IssueIDs` reads the IDs that links are checked against. They're read each time an issue is opened, so new issues can be linked from the next one you open.
- `models/tag.go` no longer uses `github.com/akrennmair/slice`, which isn't on the approved list, so it's gone from `go.mod`.

## Notes

- Glamour counts the marker characters used to find links as text. A line with links may therefore wrap a column or two early.
- **Unrelated, already present:** while testing, I found that cards call `termenv.HasDarkBackground()` every time they're drawn. Each call can query the terminal, so in a terminal that doesn't answer, an open issue stalls. I saw this with the previous commit as well, and worked around it in my test terminal by setting `TERM=screen-256color`. Answering terminals don't seem to be affected, but it might be worth its own issue: the background colour could be read once, from Bubble Tea's `BackgroundColorMsg`, which the issue view already receives.

## Testing

- **Finding references:** trailing punctuation, words after the ID, unknown IDs and case, length limits, and `#` inside a word.
- **Comments:** references skipped in code spans, code blocks, HTML and link destinations.
- **Rendering:** links in cards and tags, the selected link, the style restored after a link, a link in a selected todo item, and no links without a lookup.
- **Keys:**
  - `→` on cards with only links, on cards with both, and on tags with and without links
  - enter opening the right link, and still toggling todo items
  - `←` and `tab` stopping the selection
  - saving before opening a link, and staying open when saving fails
- **The app:**
  - following a link from the list, and returning to the list
  - a reference to an issue that doesn't exist
  - a linked file that's removed after the issue was opened, which shows the list with the error
  - opening a single issue, following its link in place, and quitting with `esc`/`w`
- **In a real terminal,** on a copy of `.ish`:
  - In 3CVJY-W5VVN, `relation.parent: #3CSJ4-PEZ1N` was drawn blue and underlined, then bold when selected. Enter opened 3CSJ4-PEZ1N in place, and `esc` quit.
  - In 3CWYG-DF969, the comment's link to #3CVMN-Q41P7 was selectable with `→` and opened that issue.
