# Project Overview

ish is a distributed, file-based issue tracker. The project is written in go and
relies on sqlite to maintain the issue database. Design goals for the project
include simplicity, resiliency, and extensibility.

The ish command-line utility first and foremost provides a terminal-based interactive
UI for viewing and editing issues. Additional subcommands may be used to make smaller,
targeted edits to issues such as adding or modifying attributes, appending comments, or
editing existing comments.

## Development Rules

- All code must be written in go.
- Code must follow the [Google Go Style Guide](https://google.github.io/styleguide/go/)
- Focus on the task described in one file at a time. When starting a task, if it is
  currently in the `open` state, change the state to `in progress`.
- Append comments in the issue markdown files to document changes and request clarification
  when instructions are not clear.
- When a task is complete and all questions have been answered, change the issue's state
  to `in review`.

## Dependencies — APPROVED LIST

Only use dependencies from this list. Ask before adding anything new.

- `charm.land/bubbletea/v2` v2.1.0
- `charm.land/glamour/v2` v2.0.1
- `charm.land/lipgloss/v2` v2.0.6
- `charm.land/bubbles/v2` v2.2.1
- `github.com/muesli/termenv` v0.16.0
- `github.com/spf13/cobra` v1.10.2
- `github.com/yuin/goldmark` v1.8.6
- `github.com/yuin/goldmark-emoji` v1.0.6

## Git Behavior

- After completing each task, stage all changed files
- Commit message format: "[issue number]: [what was done]" and reference the issue number
  of the assigned task.
- Do NOT push automatically. Leave push for the user.

## Architecture Rules

- Parse markdown files and create records in sqlite
- Use command line utility to:
  - make individual, atomic updates to issues
  - view and edit an issue
- Export issues from sqlite to mardown

## Testing Rules

- Example markdown files are in the `issues/` directory. DO NOT MODIFY THE FILES IN
  THE `issues/` DIRECTORY.
- In keeping with go convention, tests should be in files colocated with the code they
  are testing, with a `_test.go` suffix.
- Fix any test or type errors until the whole suite is green.
- Add or update tests for the code you change, even if nobody asked.
