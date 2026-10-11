# Allow adding and deleting metadata fields to an issue

[brendan@berg.industries 2026-10-11T10:57:08-07:00]

- state: open
- assignees: brendan@berg.industries

---

[brendan@berg.industries 2026-10-11T10:57:08-07:00]

In the Issue view, there is currently no mechanism to add or delete metadata fields. A new keybinding should be added so that `m` creates a new Tag in the metadata array.

## Adding a metadata field

A new, completely blank Tag should be appended to the Metadata list. Since a blank Tag renders unintuitively, the Key field should be editable immediately. After the user presses enter to set the Key, the Value field should become editable. Once both the Key and Value fields are set, the Tag should be taken out of editing mode but remain selected.

If the Tag is taken out of editing mode having an empty Key field, the Tag should be deleted from the Metadata list.

## Removing a metadata field

When a Tag is selected, pressing delete should highlight the entire Tag with red text on a grey background. Pressing delete again should remove the tag. Any other key should put the Tag back into the selected state.
