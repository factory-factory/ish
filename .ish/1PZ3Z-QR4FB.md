# Specify workflow configuration file format

[<brendan@berg.industries> 2023-06-04T18:23:24-04:00]

- state: planning
- assignees: <brendan@berg.industries>

---

[<brendan@berg.industries> 2023-06-04T18:23:24-04:00]

The **ish** library should provide the ability to enforce rules
when moving issues from one state to another.

A workflow definition declares state transitions that require
specific conditions to be met for certain edits to be committed.
For example, a workflow rule may require that an issue have a
time estimate, a single assignee, and at least one comment in
order to be moved into the "in progress" state.

The `<type>.workflow.<fmt>` file defines a workflow for issues
with type `<type>`.

The available predicates are:

- Presence of a given metadata key
- Value count for a given metadata key
- Specific value of a given metadata key
