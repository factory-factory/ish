# Ensure tabs and newlines are stripped from title and attribute fields

["Brendan Berg" <brendan@berg.industries> 2026-10-26T22:01:21-07:00]

- state: closed
- assignees: <brendan@berg.industries>

---

["Brendan Berg" <brendan@berg.industries> 2026-10-26T22:01:21-07:00]

While working on #3DEVJ-T9438, <noreply@anthropic.com> pointed out that Issue titles and attributes could have newline (`\n`) or tab (`\t`) characters in them that could mess with the formatting of `ish list`.

## Can you even put forbidden characters in fields while editing an Issue?

My initial analysis has found:

- Pressing tab while editing the title has no effect (the cursor does not move).
- Pressing return while editing the title concludes the edit and accepts the value, so no newline is ever added to the title.
- Similarly, when editing an attribute, pressing tab accepts an autocomplete suggestion and pressing enter commits the value.

This is supported by a quick view of the `bubbles` source code.

It is my conclusion that newlines and tabs cannot be entered via the TUI.

## But can you insert them in the Markdown with a text editor, like some kind of savage?

Let's start with newlines. A header starting with `#` ends at the newline, with a parser-dependent exception: ending the line with two spaces and a newline continues the heading on the next line (in Github-flavored markdown at least).

The Glamour renderer doesn't like header continuations with two spaces, and manually editing an issue to split the header across two lines results in the issue not parsing correctly and being excluded from the metadata database.

The metadata section is a Markdown list. I've identified a case where newlines make it into metadata values:

```
- greeting: hello··
world
```

The newline makes it into the SQLite database like so: `["hello  \nworld"]`.

It's an open question whether a metadata attribute should be allowed to span multiple lines in the Markdown source. But whatever way that gets decided, newlines (probably the whole string `  \n`) should be replaced with a single space character.

Tabs can be manually added to both titles and attributes, and should be replaced with spaces when parsing Markdown files. At this point, the Markdown files should be left as-is; in the future a repair pass could fix up minor formatting issues in Issue files.

## Conclusion and Instructions

- Newline and tab characters should be stripped from title and attribute strings prior to being inserted into SQLite and being used to populate Issue models in memory.

---

[Claude <noreply@anthropic.com> 2026-10-26T22:58:45-07:00]

## What was done

- The parser now cleans titles and metadata values before they reach the `Issue` model. Every path into the metadata database (`store.Init`/`Sync`) and every in-memory issue (`parse.LoadIssue`) goes through `ParseIssue`, so this one change covers both.
- The new `singleLine` function in `parse/parse.go` replaces each line break, together with any spaces and tabs around it, with a single space. That covers the `  \n` hard break from your example, so `hello··\nworld` becomes `hello world`. Any remaining tab becomes one space each. Other runs of spaces are left alone.
- Metadata items are cleaned *before* their values are split on `, `, so a list that continues on the next line splits as if it were on one line. For example, `- assignees: alice,` followed by `  bob` gives `alice` and `bob`.
- Issue files on disk aren't changed when they're parsed. One side effect: saving an issue from the TUI rebuilds its header from the model, so the cleaned title and values are written back to that issue's file when someone edits it in ish.
- Tests: `TestParseIssueBlanksTabsAndNewlines` and `TestSingleLine` in `parse/parse_test.go`, and `TestIndexBlanksTabsAndNewlines` in `store/issues_test.go`, which checks what gets stored in SQLite.

On your open question about metadata values spanning several lines: with this change, a value continued on the next line is accepted and joined with a space. If you decide multi-line values should be rejected instead, that would be a parse error in `matchMetadata` and can be its own change.

Moving this to `in review`.
