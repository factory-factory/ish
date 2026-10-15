# Write ish file parser

[brendan@berg.industries 2023-05-28 11:55:23 -0400]

- state: closed
- assignees: brendan@berg.industries
- estimate: 8h

---

[brendan@berg.industries 2023-05-28 11:55:23 -0400]

Write a parser that scans the issues directory and extracts relevant metadata
from **ish** files.

See this issue #1PJRQ-WQMB1

## Header

- The first line should start with `#` and the issue id in square brackets and
  be followed by the issue title
- The second line should be empty
- The third line should contain the email address of the user who created the
  issue and the timestamp in ISO-8601 format:
  `[<brendan@berg.industries> 2023-05-28T11:55:23-0400]`
- The fourth line should be empty
- Following the fourth line is an ordered list of issue metadata

## Conversation

- The list of metadata is optionally followed by a series of comments
- Each comment starts with a blank line followed by a line containing three
  hyphens to indicate a horizontal rule
- Following the horizontal rule, the comment may have an optional newline
  followed by the email address of the commenting user and the ISO-8601 timestamp
- After the user info, a blank line
- All following text until the next horizontal rule is interpreted as markdown
