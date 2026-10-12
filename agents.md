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
- You may edit only the issue file for the task you've been assigned, and in it
  you may only:
  - change the `state` line, and
  - append comments at the end of the file.
    Never edit the title, other metadata, or existing comments, including your own.
- Sign appended comments with the byline `[Claude <RFC 3339 timestamp>]`.
- Use comments to document changes and to ask for clarification when instructions
  are unclear.
- When the task is complete and all questions have been answered, change the
  issue's state to `in review`.

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
- `modernc.org/sqlite` v1.60.1

## Git Behavior

- After completing each task, commit all changed files including changes to the
  issue markdown.
- Commit message format: "[[issue number]] [what was done]" and reference the
  issue number of the assigned task.
- Do NOT push automatically. Leave push for the user.

## Architecture Rules

- Parse markdown files and create records in sqlite
- Use command line utility to:
  - make individual, atomic updates to issues
  - view and edit an issue
- Export issues from sqlite to mardown

## Testing Rules

- Issue files are in the `.ish/` directory, and tests may read them as samples.
  Tests must never write to `.ish/`; use a temporary directory instead.
- Never create, rename, delete, or edit issue files, except as described under
  Development Rules for the issue you've been assigned.
- In keeping with go convention, tests should be in files colocated with the code they
  are testing, with a `_test.go` suffix.
- Fix any test or type errors until the whole suite is green.
- Add or update tests for the code you change, even if nobody asked.
