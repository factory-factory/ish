# Improvements to custom filter UI

[<brendan@berg.industries> 2026-10-15T17:29:07-07:00]

- state: closed
- assignees: <brendan@berg.industries>

---

[<brendan@berg.industries> 2026-10-15T17:29:07-07:00]

The custom filter view that replaces the filter drop-down when the `Custom...` menu item is selected should be styled with a grey border when unfocused and blue when focused.

To the right of the text field, add two buttons: "Reset" which removes the custom filter view and switches back to the drop-down menu, and "Save" which will be implemented in a follow-up ticket.

The width of the text area should be the same as the width of the drop-down.

```
╭     ╮             ╭──────────────────────────────────╮
  ish      Filter:  | {assignees = null}               │  | Reset |  | Save |
╰     ╯             ╰──────────────────────────────────╯
```

---

[Claude <noreply@anthropic.com> 2026-10-15T18:38:09-07:00]

I've reviewed this and moved it to `in progress`. Some of it is already done, and the buttons need a few decisions before I start.

## Already done in #3CYHY-9EVBE

- The custom field's border is 11 when unfocused and blue (4) when focused, the same as the menu's trigger.
- The field is already the same width as the trigger.

## Plan for the buttons

- **Appearance:** "Reset" and "Save" are drawn to the right of the field as boxes, 3 lines tall like the field. They'd use the trigger's colours: border 11 and white (7) text, or a blue border and bright white (15) text while selected.
- **Keys:** with the custom field selected but not being edited, `→`/`l` and `←`/`h` move between the field, Reset and Save. Enter edits the field, as it does now, or presses the selected button. `↓` still moves to the issues.
- **Reset:** it does what `esc` on the field does now. It returns to the menu and applies the filter that was chosen from the menu before.
- **Save:** it's shown, but does nothing until the follow-up ticket.

## Questions

1. **Width:** with the buttons beside it, a field as wide as the trigger doesn't fit. In a 100-column terminal, the trigger is 78 wide, which leaves no room for the buttons, about 20 columns. Should the trigger and the field both shrink to make room for the buttons, so they stay the same width as each other? Or should only the field be narrower, by the width of the buttons? Your mock-up's field is narrower than the trigger in #3CYHY-9EVBE.
2. **Button borders:** should the buttons have full borders, like the field? Or should they be a single line, as in the mock-up (`| Reset |`), shown on the middle line with the field's text?
3. **Save until the follow-up:** should it look like the other button, or be dimmed to show it doesn't work yet? And should it be skipped when moving between the buttons?
4. **The keys above:** do they suit you? Moving out of the text box while editing would still need enter or `esc` first.
5. **`esc`:** on the field, while it's selected but not being edited, `esc` currently does what Reset will do. Should it keep doing that, or should Reset be the only way?

---

[<brendan@berg.industries> 2026-10-15T18:47:54-07:00]

## Answers, hopefully

1. It's ok to shrink the field so that the buttons can fit right-aligned with the terminal edge. The drop-down trigger shouldn't be narrowed to less than 60 characters even if the custom filter field has to shrink below 60 characters wide to fit the buttons.
2. The buttons are meant to be single-line and should not have borders. The pipe characters were to indicate the extents of the button background color. (There should be two spaces before and after the button text.)
3. The behavior of the save button will be handled in a follow-up.
4. Yes, the proposed key navigation looks good.
5. No, the `esc` behavior should be removed and replaced with the reset button.

---

[Claude <noreply@anthropic.com> 2026-10-15T21:43:39-07:00]

Done, following your answers, and moved to `in review`.

## The buttons

- **Layout:** "Reset" and "Save" are on the line with the field's text, at the heading's right edge. They have no border, and their background spans two spaces either side of each label. There are two spaces between them.
- **Colours:** your answers didn't name any, so I chose these. Each is a quick change in `styles.Button` and `styles.ButtonSelected`.
  - At rest, white (7) on grey (8), matching the trigger's white text.
  - Selected, bright white (15) on blue (4), like the selected todo item and the selected border.
- **Width:**
  - The field is as wide as the trigger when there's room. Otherwise it shrinks to leave room for the buttons and a two-space gap, even below 60 columns.
  - The trigger keeps its own width, 60 to 90 columns.
  - In a 100-column terminal, the trigger is 78 wide and the field 57.

## Keys

- **Moving:** with the custom filter row selected, but not being edited, `←`/`h` and `→`/`l` move between the field, Reset and Save. They stop at either end. Only the selected part is highlighted, so the field's border turns back to 11 while a button is selected.
- **Enter:** it edits the field, or presses the selected button. The help footer names what enter will do: "edit filter", "reset to the filter menu" or "save".
- **Reset:** it returns to the filter menu and applies the filter chosen from it before, as `esc` used to.
- **Save:** it does nothing until the follow-up ticket.
- **`esc`:** it no longer leaves the custom filter row.
- **`↓`:** it still moves to the issues. Coming back up selects the field again.

## Testing

- **Layout:** the buttons sit at the right edge, on the field's text line, in 80, 100 and 200 columns. The field shrinks to fit them, while the trigger keeps at least 60.
- **Button styles:** their colours, with no border.
- **Moving:** between the field and the buttons in both directions, stopping at the ends, with the highlight and the help changing.
- **Enter:** Save does nothing, enter on the field edits it, and the arrows move within the text while editing.
- **Leaving the row:** `↓` and back selects the field again.
- **Reset:** the menu test that used `esc` to return to the menu now checks that `esc` doesn't, then uses Reset.
- **In a real terminal,** on a copy of `.ish`, the buttons were white on grey beside the field. `→` highlighted Reset in bright white on blue and turned the field's border back to 11. Enter returned to the filter menu.

---

[<brendan@berg.industries> 2026-10-15T22:16:48-07:00]

I've changed some button styling and added a test to style the reset button differently than the save button when selected.
