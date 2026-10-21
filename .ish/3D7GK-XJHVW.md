# Implement tab / shift-tab navigation between issue list and filter drop-down

["Brendan Berg" <brendan@berg.industries> 2026-10-21T09:51:53-07:00]

- state: open
- assignees: <brendan@berg.industries>

---

["Brendan Berg" <brendan@berg.industries> 2026-10-21T09:51:53-07:00]

Currently, the only ways to navigate to the filter menu are either to use the up arrow to scroll past the top of the issue list, or to type `/` to bring up the custom filter expression text area.

Implement tab navigation so that `tab`/`shift+tab` cycle focus between the filter drop-down and the issue list. Additionally, if the custom filter expression text area is visible, the "Reset" and "Save" buttons should also be part of the navigation.

(Additionally, when using tab navigation while the custom filter text area is in edit mode, the edits should be saved before focus moves.)
