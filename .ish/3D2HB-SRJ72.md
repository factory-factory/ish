# Autosuggest delete behavior is still not correct

[<brendan@berg.industries> 2026-10-17T20:55:16-07:00]

- state: closed
- assignees: <brendan@berg.industries>
- relation.relates-to: #3CZWS-DA7QP
- resolution: can't reproduce

---

[<brendan@berg.industries> 2026-10-17T20:55:16-07:00]

Currently, when an autocomplete suggestion is displayed and the user presses delete, the suggestion is cleared and the cursor remains at the last letter that was typed. The next time delete is pressed, the autocomplete suggestion re-appears.
