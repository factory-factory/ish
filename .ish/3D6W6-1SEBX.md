# Create filter expression function to convert display names to email addresses

["Brendan Berg" <brendan@berg.industries> 2026-10-20T22:58:40-07:00]

- state: planning
- assignees: <brendan@berg.industries>
- relation.parent: #3D6BW-FYMNH

---

["Brendan Berg" <brendan@berg.industries> 2026-10-20T22:58:40-07:00]

In filter expressions, matching against assignees requires an email address enclosed in angle brackets. An expression like `{assignees = 'Brendan Berg'}` won't work because the display name isn't stored in the metadata database. A user knowing that email addresses are used may expect an expression like `{assignees = 'hello@example.org'}` to work, but this too does not match any issues because email addresses are stored with enclosing angle brackets.

Issue #3D6WQ-KR22E will address the second case by stripping angle brackets from email addresses when storing them in the metadata database.

To address the first case, we need a function that converts display names to email addresses.

## Desired behavior

- When creating filter expressions, a user should be able to specify a display name that gets converted to an email address

This would allow an expression like `{assignees = ('Brendan Berg' :user)}` to match based on the email address stored in the user directory for Brendan Berg.

## Required changes

- Add a `:user` function that looks up a display name in the user directory and returns the user's email address.
