# Allow dropdowns for value fields in metadata attributes

[<brendan@berg.industries> 2026-10-15T23:55:03-07:00]

- state: planning
- assignees: <brendan@berg.industries>
- relation.depends-on: #1PZ3T-2PSBF

---

[<brendan@berg.industries> 2026-10-15T23:55:03-07:00]

When a metadata key is defined to be an enum type in the project configuration, it should be rendered as a drop-down menu. The menu trigger should be a single line tall and use a background color to identify it as an interaction target. It should have a disclosure triangle on the right side of the target.

When the menu is open, it should show each of the defined values of the enum. The values can be found in the project configuration file.

Exiting from an issue should run validations that all attributes have valid types as defined in the project configuration.
