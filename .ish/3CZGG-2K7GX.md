# Add an ish badge in the upper left of the list view

[brendan@berg.industries 2026-10-15T17:16:50-07:00]

- state: closed
- assignees: brendan@berg.industries

---

[brendan@berg.industries 2026-10-15T17:16:50-07:00]

I'd like a small ish logo badge in the upper left corner of the list view.

```
╭     ╮
  ish  
╰     ╯
```

The corner glyphs should be styled aqua (`lipgloss.Color("6")`) and ish should be styled magenta (`lipgloss.Color("5")`).

There should be 5 spaces between the right edge of the glyph and the filter UI.

The code for the badge should go in a `Badge` model that gets added to the list view.

---

[Claude 2026-10-15T17:41:11-07:00]

Done, and moved to `in review`. I had no questions.

## The badge

- A new `Badge` model in `app/badge.go` draws the badge, with the corner glyphs in aqua (6) and `ish` in magenta (5). The styles are `styles.BadgeCorner` and `styles.BadgeName`.
- The list view holds a `Badge` and draws it in the upper left corner, inside the heading's usual padding.

## Layout

- **Spacing:** there are 5 spaces between the badge's right edge and the text `Filter:`. The label's style has one column of padding on its left, so that padding counts as one of the 5.
- **The filter menu:** it moved down a line, so it sits beside `ish`, on the badge's middle line. The heading is now three lines tall instead of one. The menu's panel still opens directly under the trigger.
- **The custom filter field:** its text lines up with `ish` too, as its label already lined up with its first line of text. It's narrower by the width of the badge and the space after it.
- **Errors:** an error in a custom filter now starts under the filter, rather than under the badge.

## Testing

- The badge's three lines and their colours.
- The badge in the corner, with `Filter:` 5 spaces after it on the line with `ish`.
- The menu's panel opening under its trigger.
- The custom field lining up with `ish` and fitting the terminal.
- A filter error starting under the filter.
- In a real terminal, on a copy of `.ish`, the list showed the corners in aqua and `ish` in magenta, with `Filter: [ All unclosed issues ▼ ]` 5 spaces after the badge, on its middle line.
