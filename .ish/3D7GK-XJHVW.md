# Implement tab / shift-tab navigation between issue list and filter drop-down

["Brendan Berg" <brendan@berg.industries> 2026-10-21T09:51:53-07:00]

- state: closed
- assignees: <brendan@berg.industries>

---

["Brendan Berg" <brendan@berg.industries> 2026-10-21T09:51:53-07:00]

Currently, the only ways to navigate to the filter menu are either to use the up arrow to scroll past the top of the issue list, or to type `/` to bring up the custom filter expression text area.

Implement tab navigation so that `tab`/`shift+tab` cycle focus between the filter drop-down and the issue list. Additionally, if the custom filter expression text area is visible, the "Reset" and "Save" buttons should also be part of the navigation.

(Additionally, when using tab navigation while the custom filter text area is in edit mode, the edits should be saved before focus moves.)

---

[Claude <noreply@anthropic.com> 2026-10-21T11:46:40-07:00]

Done, and moved to `in review`. I had no questions.

## The focus order

- **With the filter menu showing:** `tab` and `shift+tab` move the focus between the menu and the issue list.
- **With the custom filter showing:** `tab` moves through the field, Reset, Save and the issue list. `shift+tab` goes the other way.
- **At the ends:** both keys wrap around. So `shift+tab` from the issue list focuses Save, or the menu.
- **What focus does:** a focused menu opens with enter. A focused button is highlighted and presses with enter.
- **What's unchanged:** `↑` from the top of the list, `↓` back to it, `←`/`→` between the field and the buttons, and `/` all work as before.
- **An open menu:** while the menu is open, `tab` doesn't move the focus, since keys go to the open menu.

## While editing a filter

- **Applying first:** `tab` and `shift+tab` apply the filter, as enter does, then move the focus. So `tab` moves to Reset, and `shift+tab` to the issue list.
- **An invalid filter:** if the filter doesn't parse, its error is shown and the focus stays in the field, since there's nothing valid to save.

## Help

The help footer lists `tab/shift+tab focus` in the issue list, on the menu and on the custom filter row, and `tab/shift+tab apply and move` while editing.

## Testing

- **The menu:** `tab` and `shift+tab` between the menu and the list, enter opening the focused menu, and `tab` doing nothing while it's open.
- **The custom filter:** both directions through the field, Reset, Save and the list, wrapping, with the focused button highlighted.
- **Editing:** `tab` and `shift+tab` apply the filter and move to Reset or the list. An invalid filter keeps the focus, with its error.
- **The help footer** in each state.
- **In a real terminal,** on a clone of this repository, `tab` turned the menu trigger's border blue and dimmed the selected issue, and a second `tab` returned to the list.
