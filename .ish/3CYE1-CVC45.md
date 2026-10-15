# Update metadata UI to accept the key field and advance to the value field on tab and colon

[brendan@berg.industries 2026-10-14T22:53:31-07:00]

- state: closed
- assignees: brendan@berg.industries

---

[brendan@berg.industries 2026-10-14T22:53:31-07:00]

Since colons are not allowed in attribute keys, typing colon should save the contents of the key field and advance the cursor to the value field. Additionally, the tab key should advance to the key field.

While editing the value field, it should also be possible to shift-tab to switch back to editing the key field.

---

[Claude 2026-10-15T00:57:38-07:00]

I've reviewed this and moved it to `in progress`. I have a plan, but a few points need your answers before I start.

## How it works now

- Pressing `m` adds a tag and edits its key first. Enter saves the key and moves on to its values. Enter on an empty key removes the new tag.
- Only new tags have an editable key. Enter on an existing tag edits just its values.

## Plan

- **While editing a new tag's key:** `tab` or `:` saves the key and moves to the value field, as enter does now. The colon isn't typed into the key.
- **While editing that tag's values:** `shift+tab` returns to the key, keeping the values typed so far. `tab` or `:` then moves forward to them again.
- **The help footer** lists `tab`/`:` and `shift+tab` while a tag is being edited.

## Questions

1. **"The tab key should advance to the key field":** I've read this as advancing to the *value* field. Is that right?
2. **`shift+tab` on existing tags:** should it also work while editing the values of an existing tag? That would let you rename keys, which isn't possible today. Or should it only work on new tags, whose key hasn't been saved yet? I'd suggest new tags only, unless you want renaming. Renaming would also need a rule for a key that's already used by another tag.
3. **An empty key:** enter on an empty key removes the new tag. Should `tab` and `:` on an empty key do the same, or do nothing and leave you in the key field? I'd suggest doing nothing, since `esc` already cancels.
4. **Pasting `key: value`:** pasted text, such as `relation.parent: #3CSJ4-PEZ1N`, arrives in one piece. Should a colon in pasted text split it between the key and the value? Or should the paste be typed into the key, with the colon dropped, as other characters that can't be in a key are? I'd suggest splitting it, but it's an extra, so I'm happy to leave it out.

---

[Claude 2026-10-15T00:57:53-07:00]

A correction to question 4: the key field drops only spaces. It doesn't drop other characters that can't be in a key; those are typed, and the key is rejected when the issue is saved. So the second option would be "type the paste into the key, with the colon dropped".

---

[brendan@berg.industries 2026-10-15T01:09:46-07:00]

1. Yes, that's my bad. I meant the *value* field. I've updated the original comment.
2. `shift+tab` should only work on new tags.
3. `tab` and `:` on an empty key should do nothing.
4. Let's split pasted text on `:`. Since a space is likely to follow a colon, whitespace should be stripped from the value portion of the pasted text.

---

[Claude 2026-10-15T01:16:16-07:00]

Done, following your answers, and moved to `in review`.

## Keys while adding a tag

- **Key field:** `tab` or `:` saves the key and moves to the value field, as enter does. The colon isn't typed. On an empty key, they do nothing.
- **Value field:** `shift+tab` returns to the key. The values typed so far are kept, and come back when you move forward again with `tab`, `:` or enter.
  - This only applies to new tags. On an existing tag, `shift+tab` does nothing.
  - In the value field, `:` and `tab` are typed as usual, so values such as URLs can contain colons.
- **Help footer:** it shows `tab/: values` while the key is edited, and `shift+tab key` while a new tag's values are.

## Pasting

- Text pasted into the key is split at its first colon. The part before it goes into the key, and the part after it goes into the value field, without surrounding whitespace.
- For example, pasting `relation.parent: #3CSJ4-PEZ1N` fills in both.
- Pasted values replace any values kept from before a `shift+tab`. An empty part after the colon keeps them.
- With an empty key, the colon does nothing, as when it's typed, so the part after it is dropped.
- Text without a colon is pasted into the key as before.

## Testing

- `tab` and `:` move to the values from a key, do nothing on an empty key, and update the help.
- `shift+tab` returns to the key and keeps the values, and the tag is then saved with the edited key.
- `shift+tab` does nothing on an existing tag.
- Pasting with and without a colon, with surrounding space, with a URL value, with an empty value, and with an empty key, and replacing values kept from before a `shift+tab`.
- In a real terminal, on a copy of `.ish`, typing `prio`, `:`, `high`, `shift+tab`, `rity`, `tab` and enter saved `priority: high`. Pasting `relation.related: #3CSJ4-PEZ1N` saved that tag.

---

[brendan@berg.industries 2026-10-15T09:17:40-07:00]

While playing with the UI, I noticed that the value field disappears when `shift+tab` is pressed to return to the key field. I'd like to dim the existing value text when going back to edit the key.

---

[Claude 2026-10-15T09:22:13-07:00]

Done, and left in `in review`.

## The change

- After `shift+tab`, the values typed so far stay on the tag's line, dimmed, after the key: `• priority: high, low`, with `: high, low` in grey (8), the colour of the filter's placeholder text.
- The values follow directly after the key and the cursor, and move along as the key gets longer or shorter. Moving forward with `tab`, `:` or enter puts them back in their input, undimmed.

## How

- The key input pads itself with spaces to its full width, which would have pushed the values to the far edge.
- While values are shown after it, the input is now only as wide as the key. It's widened while each key press or paste is handled, so it doesn't scroll as the key grows.

## Testing

- After `shift+tab`, the values are dimmed, directly after the key and the cursor, including after typing more of the key.
- Moving forward shows them in their input again, undimmed.
- In a real terminal, on a copy of `.ish`, the line showed `prio` and the cursor, then `: high` in grey. After typing `r`, it showed `prior`, with `: high` still directly after the cursor.

---

[brendan@berg.industries 2026-10-15T10:01:34-07:00]

This is a minor tweak, but is it possible to get rid of the space between the key and the colon when navigating back with `shift+tab`? The cursor should be on the colon if possible.

---

[Claude 2026-10-15T10:04:53-07:00]

Done, and left in `in review`.

- After `shift+tab`, the colon now follows the key directly, as in `• priority: high`. With the cursor at the end of the key, the cursor is drawn on the colon.
- The gap was the cursor's own cell, which the text input always draws as a blank after the key. The colon is now shown in that cell, and only the values after it are dimmed.
- With the cursor moved back into the key, the colon still follows the key directly. With the key deleted, the line reads `• : high`.

## Testing

- The line reads `• prio: high, low` after `shift+tab`, and keeps the colon directly after the key while typing, with the cursor within the key, and with the key deleted.
- In a real terminal, on a copy of `.ish`, `prio` was followed by the colon in reverse video (the cursor), then ` high` in grey. After typing `r`, the cursor stayed on the colon after `prior`.

---

[brendan@berg.industries 2026-10-15T10:06:09-07:00]

Please make the dimmed color for values the same as the default color used for keys when not edititng them.

---

[Claude 2026-10-15T10:07:20-07:00]

Done, and left in `in review`. After `shift+tab`, the values after the key are now drawn in bright yellow (11), the colour of keys that aren't being edited, instead of grey. The colour is taken from the key style, so they'll stay the same if it changes. A test checks that they match.
