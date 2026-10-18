# Update Issue navigation to unify j/k/up/down/tab/shift-tab key behavior

[brendan@berg.industries 2026-10-14T22:49:37-07:00]

- state: closed
- assignees: brendan@berg.industries

---

[brendan@berg.industries 2026-10-14T22:49:37-07:00]

The different behaviors in the issue view for cycling through all fields with `tab` / `shift-tab` vs cycling through just comments with `j`/`k` and up and down arrows is confusing.

Update arrow key and j/k behavior to also cycle through all selectable UI elements in an issue. The end result should be that `j`, `tab`, and down arrow share the same behavior and `k`, `shift-tab`, and up arrow share the same behavior.

---

[Claude 2026-10-16T17:33:51-07:00]

I've reviewed this and moved it to `in progress`. The keys behave differently in three ways besides which fields they visit. Each needs a single behaviour, so I'd like your choice before I start.

## How the keys differ now

- **Which fields they visit:**
  - `tab`/`shift+tab` visit the title, each metadata field and each comment.
  - `↓`/`j` and `↑`/`k` visit only comments. From the title or a field, `↓` jumps to the first comment.
- **Tall comments:**
  - On a comment taller than the screen, `↓`/`↑` scroll through it a line at a time, then move to the next comment once its end is in view.
  - `tab` jumps straight to the next field.
- **The ends:**
  - `tab` wraps around. After the last comment it reaches a state with nothing selected, then the title.
  - The arrows stop at the first and last comments.

## Plan

`↓`, `j` and `tab` will all move to the next selectable element: the title, then each metadata field, then each comment. `↑`, `k` and `shift+tab` will move to the previous one. The help footer will list them together, for example `↑/↓/tab move`.

## Questions

1. **Tall comments:** should all three keys scroll through a comment taller than the screen before moving on, as the arrows do now? Or should they all jump straight to the next element, as `tab` does, leaving `pgup`/`pgdn` for reading a long comment? I'd suggest scrolling through it. Otherwise the arrows lose the only line-by-line way to read a long comment.
2. **The ends:** should all three keys wrap around, as `tab` does now, or stop at the title and the last comment, as the arrows do? I'd suggest stopping. Wrapping makes `j` jump from the last comment back to the title, which is easy to do by accident when holding the key down.
3. **Nothing selected:** an issue opens with nothing selected. If the keys stop at the ends, they'd never return to that state. From it, `↓` would select the title, and `↑` would do nothing. Is that all right? If they wrap, should the cycle keep passing through "nothing selected", or go straight from the last comment to the title?

---

[brendan@berg.industries 2026-10-16T17:39:21-07:00]

Yes, the behavior is even more confusing than I originally thought.

- **`tab`/`shift-tab`** should keep their existing behavior cycling through all selectable fields, with one change: selecting a comment should scroll the comment into view such that the comment's byline is at the top of the viewport. Scrolling should still be pinned so the bottom of the final comment stops at the bottom of the screen when fully scrolled into view.
- **`j`/`k`** should scroll the next or previous comment to the top of the vieport. (If the last comment is already fully in the viewport, `k` should just change the selection.)
- **`up`/`down`** should *only* scroll the viewport one line at a time, without changing what's selected.

---

[Claude 2026-10-16T22:36:31-07:00]

Thanks, that's much clearer. Here's how I'm reading it, with two small points to confirm.

- **`tab`/`shift+tab`:** they keep cycling through every field, wrapping as they do now. Selecting a comment scrolls so that its byline is at the top of the screen. Near the end, the scrolling stops when the last comment's bottom reaches the bottom of the screen, so the comment may sit lower.
- **`j`/`k`:** they select the next or previous comment, and scroll it so its byline is at the top, with the same limit at the end.
- **`↑`/`↓`:** they scroll the view one line at a time, and the selection doesn't change. It can scroll out of view.

## To confirm

1. **The last comment:** I think the note "if the last comment is already fully in the viewport, `k` should just change the selection" is about `j`. Near the end, `j` can't scroll the next comment to the top, so it only moves the selection. `k` can always scroll the previous comment to the top. Did you mean `j`?
2. **Starting from a field, and the ends:**
   - From the title or a metadata field, or with nothing selected, I'd have `j` select the first comment and `k` select the last.
   - At the last comment, `j` would do nothing, and at the first comment, `k` would do nothing. They wouldn't wrap or move back up into the fields.

   Does that sound right?

---

[brendan@berg.industries 2026-10-16T22:38:32-07:00]

1. To clarify, I'm talking about cases where multiple comments are in the viewport that includes the final comment. If the penultimate comment is selected, but it and the final comment are both short enough that the final comment is fully in view, `k` should advance the selection to the final comment but since the bottom of the final comment is pinned to the bottom of the viewport, it can't be scrolled such that the byline is at the top of the viewport.
2. Yes, that is correct.

---

[Claude 2026-10-16T22:46:59-07:00]

Done, following your answers, and moved to `in review`.

## The keys

- **`tab`/`shift+tab`:** they cycle through every field as before. Selecting a comment now scrolls its byline to the top of the view.
- **`j`/`k`:** they select the next or previous comment, and scroll its byline to the top.
  - From the title, a metadata field or nothing selected, `j` selects the first comment and `k` the last.
  - They stop at the first and last comments.
- **The end of the issue:** the view never scrolls past the bottom of the last comment, so a comment near the end stops lower down. When the last comment is already in view, `j` only moves the selection. Your example described moving forward to the last comment, so I've applied this to `j`.
- **`↑`/`↓`:** they scroll the view one line at a time, and never change the selection. They stop at the top and at the end of the last comment.
- **Selecting todo items and links** is unchanged. `↑`/`k` and `↓`/`j` still move between them.

## Help

- The short help lists `j next comment` and `k previous comment`, in place of the arrows.
- The full help adds `↑ scroll up` and `↓ scroll down`, with the paging keys.

## Code

- The arrows call the viewport's own one-line scrolling.
- The old `scrollDown` and `scrollUp` are replaced by `moveComment`, which handles `j` and `k`, and `scrollToCard`, which `tab` uses too.

## Testing

- The arrows scroll a line at a time without changing the selection, whether nothing, the title or a comment is selected, and they stop at either end.
- `j` selects each comment with its byline at the top. Near the end, it stops when the last comment's bottom reaches the bottom of the view, and it does nothing on the last comment.
- `j` only moves the selection when the last comment is already in view.
- `k` from the title selects the last comment, then each earlier one at the top, and does nothing on the first. `j` from a tag selects the first comment, and `j`/`k` do nothing without comments.
- `tab`/`shift+tab` scroll the selected comment to the top, and stop at the end.
- The help lists the new keys.
- Two app tests that used `↓` to select a comment now use `j`.
- In a real terminal, on a copy of `.ish`, an issue with several comments handled `j`, `↓` and `k`, then `tab`, and quit normally.
