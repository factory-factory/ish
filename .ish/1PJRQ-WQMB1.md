# What is ish?

[brendan@berg.industries 2023-05-26T23:35:36-0400]

- state: in progress
- assignees: brendanberg, jhc

---

[brendan@berg.industries 2023-05-27T13:41:50-0400]

**ish** is a file format convention and a set of utilities for distributed, file-based issue tracking.

**ish** consists of a library and a command line utility.
The library provides a directory scanner, parser, and database API,
while the command line app provides a terminal UI to interact with issues.

---

[brendan@berg.industries 2023-05-27T23:37:18-0400]

The **ish** command line tools rely on go modules from [charmbracelet](https://github.com/charmbracelet):

- [Bubbletea](https://github.com/charmbracelet/bubbletea), a terminal UI framework
- [Glamour](https://github.com/charmbracelet/glamour), a markdown formatting tool
- [Lipgloss](https://github.com/charmbracelet/lipgloss), a terminal style library

## TODO:

- [x] Learn the basics of the charmbracelet tools
- [x] Write parser
- [ ] Build viewing and editing UI
  - [x] Rendering issues
  - [x] Navigating issue compontents
  - [x] Editing components
  - [x] Updating to-do items
  - [ ] Appending comments
  - [ ] Adding new metadata attributes
- [ ] Build ish DB
- [ ] Create ish sub-commands
  - [ ] ish init
  - [ ] ish new [TITLE]
  - [ ] ish list [QUERY]
  - [ ] ish edit [HASH]

---

[brendan@berg.industries 2023-06-04T16:18:23-0400]

Once upon a time, there was a variable named `foo`.

- The variable was sad, and lonely

  Until the programmer declared a variable named `bar`

- A separate variable, named `quux` became enraged with jealousy.

  ```
  for dip in glorp:
    frob.hiss
  ```

1. Jig
1. Karp
1. Lob
1. Meem
1. Nin
