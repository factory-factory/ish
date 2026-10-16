# Add horizontal separator between saved filters an the "Custom..." menu item

[brendan@berg.industries 2026-10-15T22:47:45-07:00]

- state: closed
- assignees: brendan@berg.industries

---

[brendan@berg.industries 2026-10-15T22:47:45-07:00]

Add a non-selectable horizontal rule between the filter names and the "Custom..." menu item in the filter drop-down menu.

---

[Claude 2026-10-15T23:00:48-07:00]

Done, and moved to `in review`. I had no questions.

- **What you'll see:** the open filter menu has a horizontal rule between "Issues assigned to me" and "Custom...". It's in the panel's border colour and spans the panel's width.
- **Why the vendored copy changed:** `bubble-dropdown` has no way to add a separator or an item that can't be selected, so I patched our vendored copy. The changes are marked "Local patch for ish", like the overlay patch.
  - A new `WithSeparators(indices...)` option draws a rule above the items at those indices.
  - The rule isn't an item, so the arrow keys move straight past it, the mouse can't choose it, and item indices and `ItemChosenMsg` are unchanged.
  - The panel's height includes the rule. A rule above the first item is ignored, as is one at the top of a list that's scrolled.
- **In ish:** the filter menu uses `WithSeparators(presetCustom)`.
- I left the vendored `dropdown.go` as it was formatted upstream. A few of its struct fields aren't `gofmt`-aligned, which was already the case before this change.

## Testing

- **In the vendored package:** the rule is drawn between the right items, and the panel's height includes it. Down from the item above it moves to the item below it, and the mouse finds no item on its row. A rule at the top of a scrolled list isn't shown.
- **In ish:** the open menu shows the rule on its own line between the saved filters and "Custom...", and the item after it chooses the custom filter.
- **In a real terminal,** on a copy of `.ish`, the rule was drawn between "Issues assigned to me" and "Custom...".

---

[brendan@berg.industries 2026-10-15T23:02:44-07:00]

This looks good, but can the separator be the same border color used elsewhere in the UI? (Grey 11).

---

[Claude 2026-10-15T23:04:58-07:00]

Done, and left in `in review`. The rule is now grey (11), the border colour of unselected fields, rather than the panel's blue border colour.

- The vendored `bubble-dropdown` has a new `WithSeparatorColor` option, alongside `WithSeparators` and also marked "Local patch for ish". Without it, rules still take the border's colour.
- ish sets it to 11, with a `separatorColor` constant in `app/menu.go`.

## Testing

- **In the vendored package:** a rule is drawn in the border's colour by default, and in its own colour when one is set.
- **In ish:** the menu test checks that the rule is drawn in 11.
- **In a real terminal,** on a copy of `.ish`, the rule was drawn in 11, between the panel's blue borders.
