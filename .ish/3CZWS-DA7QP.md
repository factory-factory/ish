# Implement auto-complete for attribute keys and values

[<brendan@berg.industries> 2026-10-15T23:49:34-07:00]

- state: closed
- assignees: <brendan@berg.industries>

---

[<brendan@berg.industries> 2026-10-15T23:49:34-07:00]

It would be helpful to be able to autocomplete key names and values when adding metadata attributes.

## Autocomplete in the key field

Tab should no longer commit the key and move to the value field. After typing three letters, query the SQLite database for all unique keys starting with the letters typed. Display the first match such that the cursor stays in place and the rest of the suggestion appears as placeholder text after the cursor. `tab` and `shift+tab` should cycle between the set of matched keys.

Pressing `delete`/`backspace` should clear the autocomplete suggestion before returning to the typical behavior of deleting characters to the left of the cursor.

Pressing `return`/`enter` should accept the suggested completion and move the cursor to the value field.

## Autocomplete in the value field

Shift-tab should no longer move the cursor back to the key field.

The autocomplete query for the value field should search for values that have already been used with the current key.

The keybindings for delete and return, etc. should be the same as in the key field.

---

[Claude <noreply@anthropic.com> 2026-10-16T23:16:03-07:00]

I've reviewed this and moved it to `in progress`. Below is my plan, then some questions. Several of them come from the keys added in #3CYE1-CVC45. I'll start once you've answered them.

## Plan

- **Suggestions:** the bubbles text input has built-in suggestions. It shows the rest of the first match after the cursor in a dimmed style, and has keys for cycling through the matches. I'd bind those keys to `tab` and `shift+tab`, and add the `backspace` behaviour you describe.
- **The database:** ish would look up the keys, and the values used with a key, from the metadata database. As with links, the issue view would be handed the lookups, rather than querying the database itself.

## Questions

1. **`:` in the key field:** it currently saves the key and moves to the values. Should it keep doing that? If a suggestion is showing, should `:` accept it, as enter would, or save only what's been typed?
2. **Enter in the value field:** enter in the key field accepts the suggestion and moves to the values. In the value field, enter already saves the tag. With a suggestion showing, should enter accept it and save the tag in one press? Or should the first enter accept it, and a second save the tag?
3. **Going back to the key:** with `shift+tab` now cycling suggestions in the value field, there's no way back to the key while adding a tag. Should I remove the back-to-key feature from #3CYE1-CVC45 completely, including the dimmed values and the colon under the cursor? Or should it move to another key, such as `ctrl+b`?
4. **Several values:** values are comma-separated, as in `assignees: alice, bob`. I'd suggest completions for the value being typed after the last comma, and leave the earlier values alone. Should the 3-letter threshold apply to each value too? Short values such as `open` and `bob` would then only be suggested after 3 letters.
5. **Which suggestions, and in what order:**
   - I'd leave out keys this issue already has, since a key can only appear once.
   - I'd leave out values already in the tag being edited.
   - I'd match the letters exactly, including case, and sort the matches alphabetically.
   - Would you prefer the most often used first, or matching regardless of case?
6. **Existing tags:** the ticket is about adding attributes. Should editing the values of an existing tag also get suggestions? I'd suggest yes, since the value field works the same way.
7. **Suggestion colour:** should the rest of the suggestion use grey (8), like the filter field's placeholder text? Or 11, like the values kept after the key in #3CYE1-CVC45?

---

[<brendan@berg.industries> 2026-10-16T23:18:31-07:00]

Great call to use as much of the built-in bubbles text input behavior as possible.

## Answers to your questions

1. The behavior of `:` should remain the same.
2. In the value key, enter should first accept the suggestion, then a second press save the tag.
3. Remove the keybinding, but don't remove the implementation code.
4. We should treat comma-separated values as separate items. For values that contain a comma-separated list of items, the query should search for the value being typed at any position within the JSON list in the database column. So if I started typing `ali`, The autosuggest would match `alice` from a value like `["zaphine","alice"]`.
5. Your proposed exclusions sound good. For now, match regardless of case and sort suggestions alphabetically.
6. Yes, editing values in existing tags should also get autocomplete.
7. Let's try grey (8) for now. I suspect I'll ask to tweak it during acceptance testing.

---

[Claude <noreply@anthropic.com> 2026-10-16T23:37:19-07:00]

Done, following your answers, and moved to `in review`.

## Suggestions

- **Keys:** after 3 letters, ish suggests keys used in any issue that start with what's been typed, regardless of case. It leaves out keys this issue already has.
- **Values:** values used with the tag's key are suggested for the value after the last comma, found anywhere in the stored lists. So `ali` suggests `alice` from `["zaphine","alice"]`. Values already in the tag are left out, and the 3-letter threshold applies to each value.
- **Order:** matches are sorted alphabetically, regardless of case.
- **Display:** the cursor stays where it is, on the suggestion's next letter, and the rest of the suggestion follows in grey (8). The style is `styles.Suggestion`.
- **Which fields:** the key of a new tag, and the values of new and existing tags.
- **Built on bubbles:** this uses the text input's own suggestions. I bound its keys for cycling to `tab` and `shift+tab`, and turned off its own key for accepting a suggestion, so that enter can do it.

## Keys

- **`tab`/`shift+tab`:** they cycle through the matches. They no longer save the key, and `shift+tab` no longer returns from the values to the key. The help footer lists them while there's more than one match.
- **Enter:**
  - In the key field, it accepts the suggestion and moves on to the values.
  - In the value field, the first enter accepts it, and a second saves the tag.
  - Without a suggestion, it works as before.
- **`:`:** unchanged. It saves what's been typed and moves on to the values, without accepting a suggestion.
- **Backspace:** with a suggestion showing, it dismisses the suggestion. After that it deletes as usual, and typing brings suggestions back.

## Going back to the key

As you asked, the binding is gone, but the code isn't. `editKeyAgain`, the dimmed values after the key, and the colon under the cursor are all still there, and their tests now call `editKeyAgain` directly.

## Where suggestions come from

`store.MetadataValues` reads each key's distinct values from the metadata database when an issue is opened, as the issue IDs for links are. The issue view filters them as you type, rather than querying the database for each key press. So values saved after an issue is opened are suggested from the next issue you open.

## Testing

- **Keys:** the 3-letter threshold, the first match in order, the rest of the suggestion after the cursor in grey, `tab`/`shift+tab` cycling, the help listing them, and enter accepting and moving on.
- **Matching:** regardless of case, and leaving out keys the issue already has.
- **`:`:** it keeps the typed key, even with a suggestion showing.
- **Backspace:** it dismisses the suggestion, then deletes, and typing suggests again. Enter after dismissing saves only what was typed.
- **Values:** the threshold, a value after a comma suggested regardless of case, enter accepting and then saving, and values already in the tag left out.
- **Existing tags and no data:** an existing tag's values are suggested, and nothing is suggested without metadata values.
- **`store.MetadataValues`:** distinct values, sorted regardless of case.
- **Updated tests:** the tests from #3CYE1-CVC45 that used `tab` and `shift+tab` now use `:`, or call `editKeyAgain` directly.
- **In a real terminal,** on a copy of `.ish`:
  - adding a tag, `rel` suggested `relation.depends-on`, and enter accepted it
  - then `#3C` suggested `#3CZWN-EBVEH`, enter accepted it, and a second enter saved `relation.depends-on: #3CZWN-EBVEH`

---

[<brendan@berg.industries> 2026-10-16T23:49:06-07:00]

I think we're almost there.

## Key bindings

Let's change some behavior. Instead of cycling through matches with `tab`, pressing `tab` should accept the suggested match. If the cursor is in the key field pressing `tab` a second time should move to the value field. This also lets us revert the `shift+tab` behavior when editing the value field.

Pressing `enter` in the key field should accept the match _and_ move to the value field. Pressing `enter` in the value field should accept the match and save the attribute.

## Backspace behavior

The current implementation shows suggestions while deleting text. Once the suggestion is dismissed, no suggestions should be shown until characters are added to the field.

---

[Claude <noreply@anthropic.com> 2026-10-17T00:06:31-07:00]

Done, and left in `in review`.

## Keys

- **`tab`:** it accepts the suggestion shown after the cursor. In the key field, a second `tab` moves on to the values, as `:` does. In the value field, `tab` without a suggestion does nothing.
- **Enter:**
  - In the key field, it accepts the suggestion and moves on to the values.
  - In the value field, it accepts the suggestion and saves the tag in one press.
  - Without a suggestion, it works as before.
- **`shift+tab`:** it returns from a new tag's values to its key again, keeping the values typed so far and showing them dimmed after the key, as in #3CYE1-CVC45.
- **Cycling through matches:** there's no longer a key for it. Only the first match, in alphabetical order, is suggested, so typing more letters narrows it down. I turned off the text input's own keys for cycling, which default to `↑`/`↓` and `ctrl+n`/`ctrl+p`, so they don't change the suggestion without any sign of it. If you'd like cycling back on another key, it's a small change.
- **Help footer:** it shows `tab accept suggestion` while a suggestion is shown, `tab/: values` while a new tag's key is edited, and `shift+tab key` while its values are.

## Backspace

Once a suggestion is dismissed, nothing more is suggested while deleting, even with 3 or more characters left. Adding a character suggests again.

## Testing

- **`tab`:** it accepts a key suggestion, and a second `tab` moves on to the values. `tab` and `:` move on from a key without a suggestion. In the value field, `tab` accepts a suggestion and keeps editing.
- **Enter:** it accepts a key suggestion and moves on in one press, and accepts a value suggestion and saves the tag in one press.
- **`shift+tab`:** it returns from a new tag's values to its key, and does nothing on an existing tag.
- **Backspace:** after dismissing, deleting to 3 letters suggests nothing, and typing a letter suggests again.
- **The help footer** for each of these.
- **In a real terminal,** on a copy of `.ish`:
  - `rel` and `tab` accepted `relation.depends-on`, and a second `tab` moved to the values
  - `#3C` and enter accepted `#3CZWN-EBVEH` and saved the tag

---

[<brendan@berg.industries> 2026-10-17T20:53:27-07:00]

I've re-enabled cycling through suggestsions (with `↑`/`↓` and `ctrl+p`/`ctrl+n`). Otherwise, everything looks good.
