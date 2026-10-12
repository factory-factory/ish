# Allow users to create filter expressions to filter issues in the list view

[brendan@berg.industries 2026-10-11T11:44:11-07:00]

- state: open
- assignees: brendan@berg.industries

---

[brendan@berg.industries 2026-10-11T11:44:11-07:00]

**This feature makes assumptions about the project architecture that are addressed in other issues [which ones?]. We will be assuming that the sqlite work has been completed before planning and implementing the work for this issue.**

## The filter selector

At the top of the list view, there should be a drop-down menu to select one of many saved filters. The default filter's display name is "All unclosed issues", and the expression is `{state} != 'closed'`. The bottom item in the drop down is "Custom filter...", which expands to show the custom filter field when selected.

## The custom filter field

The filter selector should be replaced with an enclosed box to allow editing when "Custom filter..." is selected. 

The text entered in the filter field should be parsed as filter expressions and styled for ease of comprehension. Filter expression syntax and style will be discussed below.

The custom filter field should have buttons to apply the expression to the issue list, and to save the custom filter as a named entry in the filter drop-down.

## Parsing filter expressions

Filter expressions are comprised of attribute expressions joined by logical operators. Attribute expressions are always in the form `attribute` `operator` `value`. Attributes are alphanumeric strings in curly braces that identify metadata keys to query against. Operators are `=`, `<`, `>`, `<=`, `>=`, `!=`, `~` (regexp match), and `?` (collection contains element). Values are any JSON scalar value: `null`, `true`, `false`, numbers, or strings.

## Rendering filter expressions

Filter expressions should use ANSI terminal colors for readability. The attribute text (including braces) should be magenta, the operator should be blue, and the value should be green.
