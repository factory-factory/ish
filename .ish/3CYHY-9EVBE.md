# Improvements to list view filter drop-down UI

[brendan@berg.industries 2026-10-15T00:58:04-07:00]

- state: in review
- assignees: brendan@berg.industries

---

[brendan@berg.industries 2026-10-15T00:58:04-07:00]

The `bubble-dropdown` element is not really performing the way I'd like it to. The focused state is not visually distinct enough from unfocused.

I'd like the trigger button to have a full border on all sides that changes from grey to blue when focused. Additionally, the text should change from grey to white when focused.

I'd like the trigger element to be a minimum of 60 characters wide and a maximum of 90.

The drop-down menu overlay can remain as currently implemented.

```
╭     ╮            ╭───────────────────────────────────────────────────────╮
  ish      Filter: │ All unclosed issues                                 ▼ │
╰     ╯            ╰───────────────────────────────────────────────────────╯
```

## Implementation Guidelines

- Since `bubble-dropdown` has already been vendored, it's OK to make changes to it.
- If a style can be applied with the public `bubble-dropdown` API surface, make the change in `ish` code.
- Make the smallest possible changes to update the style.

---

[Claude 2026-10-15T17:48:27-07:00]

I've reviewed this and moved it to `in progress`. Below is my plan, then some questions. I'll start once you've answered them.

## Plan: draw the trigger in ish, with no change to `bubble-dropdown`

- **Why not use the dropdown's own trigger:** `bubble-dropdown`'s `TriggerView` always draws `[ Label ▼ ]`, with the brackets built in, and it sizes itself to the longest item. A border set with `WithTriggerStyle` would wrap the brackets (`│ [ All unclosed issues ▼ ] │`), and there's no option to set its width. Getting your mock-up from it would mean patching the vendored code.
- **What I'd do instead:** draw the trigger in ish, as a bordered box around the chosen filter's label (from the dropdown's public `Selected()`) and the `▼`.
- **What stays with the dropdown:** opening, the keys and the menu overlay. ish already tells it the trigger's position and size through `SetBounds`, so it would pass the box's size there, and the panel opens just under the box.
- **What changes:** only ish code. `TriggerView` and `TriggerSize` would no longer be used.

## Questions

1. **Width:** does the 60–90 include the border? Between those limits, I'd make the box fill the space left after the badge and label, minus the heading's padding.
   - Should it shrink below 60 in a narrower terminal, so it still fits? Or keep 60 and run off the edge?
   - The mock-up's box is 57 wide, so I've assumed it isn't drawn to scale.
2. **Colours:** I'd use grey (8) and blue (4), the accent the selected cards and the filter menu already use.
   - For "white", I'd use bright white (15), the colour of the `Filter:` label. Or would you prefer plain white (7)?
   - Should the `▼` change colour with the text?
3. **Which state counts as focused:** I'd count the trigger as focused while it's selected, whether the menu is closed or open. Is that right?
4. **The border:** when focused, should the border stay rounded and only change colour? The custom filter field (`FilterBox`) switches from a rounded grey border to a thick blue one when it's selected.
5. **The custom filter field:** choosing "Custom..." replaces the trigger with the custom filter field. That field spans the full width that's left, with its own grey and blue borders. Should it take the same 60–90 width and colours, so the heading doesn't change size when you switch between them? Or should it stay as it is, since the ticket only mentions the trigger?

---

[brendan@berg.industries 2026-10-15T17:57:22-07:00]

## Responding to questions from the previous comment:

1. The width should include the border. UI layout behavior in terminals narrower than 80 characters is a task for a separate issue. The mock-up is drawn to fit in the width of a comment box in ish.
2. The colors you've selected sound good. Bright white is preferable. The triangle glyph should be styled the same as the text.
3. Correct. The trigger is focused whether the drop-down is open or not.
4. Let's start out with keeping the border thin and rounded for the drop-down trigger. The thick square-cornered border can help distinguish the drop-down trigger from the custom filter field.
5. The custom filter field should take the same width and height as the drop-down trigger, and the focused/unfocused border colors should be the same for both.

---

[Claude 2026-10-15T18:11:31-07:00]

Done, following your answers, and moved to `in review`.

## The trigger

- ish now draws the trigger itself, as described in my plan. It shows the chosen filter's name, with the `▼` at the box's right edge.
- `bubble-dropdown` is unchanged. It still handles opening, the keys and the menu overlay, and ish tells it the box's position and size, so the menu opens just under the box.
- **Colours:** the border is thin and rounded. It's grey (8), with grey text and arrow. While the trigger is selected, whether the menu is open or not, the border is blue (4) and the text and arrow are bright white (15). The styles are `styles.FilterTrigger` and `styles.FilterTriggerSelected`.
- **Width:** the box fills the space after the `Filter:` label, between 60 and 90 columns, border included.
- **Layout:** the box's top and bottom line up with the badge's. The label and the filter's name are on the line with `ish`.

## The custom filter field

- **Width:** the same as the trigger.
- **Height:** at rest it's one line of text, like the trigger. A filter too long for it is cut short with `…`. It used to wrap onto more lines.
- **Colours:** its border colours were already grey and blue, and still are. When selected, it keeps its thick border, which tells it apart from the trigger.
- **While editing:** I've kept the field growing to up to 5 lines, from #3CVMN-Q41P7, so a long filter can be seen while you change it. This is the one place it can be taller than the trigger. If you'd like it to stay one line tall while editing too, it's a small change.

## Note

In an 80-column terminal, the 60-column minimum doesn't quite fit. The heading needs 82 columns: the padding, the badge, the gap, the label and the box. So the box's right edge is cut off. I've left this for the separate issue about narrow terminals, which you mentioned.

## Testing

- **Width:** the box fills the space in 100 columns, stops at 90 in 200 columns, and stays at 60 in 80 columns.
- **Trigger:** its rounded box, with the name and the arrow at the right edge. Grey at rest, and in the selected style when selected and while the menu is open.
- **Styles:** the trigger's colours and rounded border. The custom field's border colours match the trigger's.
- **Custom field:** the same width and height as the trigger at rest, including with a long filter, which is cut short with `…`.
- **Menu panel:** it opens under the box. The tests that looked for the old `[ … ▼ ]` trigger now look for the box.
- **In a real terminal,** on a copy of `.ish`, the trigger was grey at rest. When selected, it had a blue border and bright white text, and the menu opened under it and showed its items.
