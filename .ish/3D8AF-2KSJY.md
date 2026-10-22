# Define lookup order for user name and email

["Brendan Berg" <brendan@berg.industries> 2026-10-21T23:39:45-07:00]

- state: closed
- assignees: <brendan@berg.industries>

---

["Brendan Berg" <brendan@berg.industries> 2026-10-21T23:39:45-07:00]

## Only get the git `user.name` and `user.email` if it hasn't been set in config

If the user or project config file specifies a user name and email, don't look for it from the version control system. If the config file does not specify the name and email, set the values by querying `git config`. The *only* time `git config` should be queried for those values is during startup when loading the config.

## Pull user name and email for bylines from the `config` object

When creating a new Issue or Comment byline, populate the name and email fields by fetching the `user.name` and `user.email` from a snapshot of the `config` object that manages application state.

---

[Claude <noreply@anthropic.com> 2026-10-21T23:53:29-07:00]

I've reviewed this and moved it to `in progress`. Below is my plan, and one question.

## Plan

- **Loading:** at start-up, `config.Load` reads the user's and the project's files as it does now. Then, only for whichever of `user.name` and `user.email` neither file set, it asks `git config`. git's values are the lowest layer, so either file overrides them. This is the only place git is asked for them.
- **Bylines:** new issues and comments get their byline author from a snapshot of the application state, as `"Name" <email>` from `.user.name` and `.user.email`. With no name, the byline is just the email.
- **Removing the other git calls:** `app.CurrentUser` and `CurrentAuthor` currently ask git directly, each time an issue is opened or created. Both would go. The app's `user` field, used by the old "Issues assigned to me" filter, would go too, since your preset change uses `.user.email` instead.

## Question

Without an email in either file or in git, there's no email for a byline. Until now, ish fell back to the operating system's user name, but a byline without an email no longer parses, so the issue couldn't be saved. What should happen?

- (a) **Keep the fallback:** as now, saving fails with an error until an email is configured.
- (b) **Disable new issues and comments:** without an email, the help footer explains why. I'd suggest this, since it stops someone writing a comment and then losing it.
- (c) **Something else.**

---

["Brendan Berg" <brendan@berg.industries> 2026-10-21T23:54:42-07:00]

If no email or name is provided, the application should exit with an error status and print that the required values are not configured.

(In the future, if a read-only mode is implemented, the app would start in read-only mode.)

---

[Claude <noreply@anthropic.com> 2026-10-22T00:01:07-07:00]

Done, following your answer, and moved to `in review`.

## Loading the user

- **Where they come from:** `config.Load` reads `~/.ish/config.toml` and `.ish/config.toml` as before. Then it asks `git config` for `user.name` and `user.email`, but only for whichever of them neither file sets.
- **Precedence:** git's values are the lowest layer, so a file's setting always wins.
- **The only place git is asked:** this is the only place ish asks git for them, once, at start-up. `app.CurrentUser`, `CurrentAuthor` and their git calls are gone, as is the app's `user` field.
- **A missing user:**
  - `ish` exits with status 1 if `user.name` or `user.email` is still missing, or blank.
  - It does this before touching the database, and prints, for example, `Couldn't start ish: user.name and user.email aren't configured. Set them in ~/.ish/config.toml or .ish/config.toml, as in […] or with git config`.
  - `ish init` doesn't need a user, so it isn't affected.

## Bylines

- **Where the author comes from:** new issues and comments take their byline author from a snapshot of the application state, as `"Name" <email>` from `.user.name` and `.user.email`.
- **A fix for single-issue mode:** `ish path/to/issue.md` opens the issue before the state is set. So setting the state now also gives an already-open issue its author. Without that, new comments were disabled in that mode, which the real-terminal check below caught.

## Your preset filters

- **Your change:** your uncommitted edits to `app/menu.go` and `app/menu_test.go` didn't build, since the tests still used `presetMine`. They also overlapped this change, which removes the app's `user` field. So I've finished them as part of this ticket, keeping your design.
- **The menu:** it now offers "All unclosed issues", "Assigned to me" (`{assignees = .user.email} and …`), "Created by me" (`{@author = .user.email} and …`) and "Custom...". The rule sits between "Created by me" and "Custom...".
- **Tests:** the menu tests use the new presets and labels. A new test checks "Created by me", including a byline with a name and one without.
- **`quote`:** this was only used by the old preset, so I removed it and its test.

## Testing

- **Loading:**
  - git fills in both settings when neither file sets them, and only the missing one when a file sets the other
  - git isn't asked when the files set both
  - without git's settings, there's no user, and a `user` that isn't a table is left alone
  - the `config` tests stub git, so your machine's settings don't affect them
- **Start-up:** a missing name, a missing email, a blank name and both missing, with the message for each.
- **Bylines:** the author from the state with a name, with quotes escaped and without a name. A new issue's byline has the name and adds its user.
- **Single-issue mode:** opening an issue, then setting the state, gives new comments the state's author.
- **On a clone of this repository:**
  - with no config file and no git user, `ish` printed the message and exited with status 1
  - with only your git settings, it started
  - with the project's `config.toml` setting a different user, a new comment's byline was `["Config Name" <config@example.com> …]`, rather than your git user
