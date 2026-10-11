# Specify ish DB schema

[brendan@berg.industries 2023-06-04T18:37:55-04:00]

- state: open
- assignees: brendan@berg.industries

---

[brendan@berg.industries 2023-06-04T18:37:55-04:00]

The issue database is a local metadata cache for all issues.
It is a SQLite database that is either git-ignored or stored
outside of the local working directory.

## Issue

`uuid`, `path`, `title`, `author`, `dateCreated`, `commitHash`

## Attribute

`uuid`, `issueId`, `key`, `value`

## Comment

`uuid`, `issueId`, `author`, `dateCreated`, `content`
