# Add horizontal separator between saved filters an the "Custom..." menu item

[brendan@berg.industries 2026-10-15T22:47:45-07:00]

- state: in review
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
