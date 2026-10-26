# ish

A distributed issue tracker in your terminal

![how to ish][demo_reel]

## Why **ish**?

**Works offline.** Off-grid doesn't mean out of the loop. You can still view and
edit issues when you're in airplane mode and sync your changes when you're back
online. There's no excuse for losing time to a sluggish web app in 2026!

**Everything's in one place.** Issues travel with the code, so they branch,
merge, and appear in history alongside the code they describe.

**No vendor lock-in.** We think our tools are great, but if you don't agree you
can easily migrate, because your data was never in someone else's walled garden.
Your issues can be grepped, scripted, and processed with any tool that reads
text.

**Auditable and Diffable.** Plain text is readable without a specialized app
(but we think our terminal app provides delightful editing and collaboration
features). Building on standard version control tools lets you view an issue's
history and see who's responsible for which edits with `git log` and `git blame`.

## Simple and Robust

Issues sync alongside your code, managed by the same version control you use for
everything else (currently `git`, but additional integrations are in the works).
Contributors can create new issues, edit metadata, participate in conversations
about development tasks, and move issues through project management workflows.
Updates are pushed to the entire team through the same version control process
they're already used to.

Issues are stored in a directory within the repository as Markdown files. (The
command line utility maintains a SQLite database for fast queries and project
analytics, but the filesystem is always ground truth.) The Markdown conventions
**ish** uses are straightforward and intuitive, so you could even use `vim` to
keep your work up to date in a pinch.

**ish** is still in the early stages of development, but it's being used to
develop **ish** itself.

## Familiar Concepts, New Possibilities

![ish flavored markdown and CLI rendering][ish_flavored_markdown]

In **ish**, issues are constructed around conversations that happen
in comments appended to the issue. Each comment has a byline that records the
user's identity and the comment's timestamp.

Conversations provide an additional benefit in being a natural approach for
dispatching AI coding agents. Agents can read and edit Markdown, and are able to
respond by appending comments to their assigned issues. In fact, **ish** has
been built this way from the start.

Workflows build on the core concepts and allow teams to design operational
processes that ensure issues are edited correctly as they move towards
completion. For example, a team could decide an issue can't be marked "closed"
without a resolution being recorded in the issue metadata.

Event hooks (coming soon!) provide a framework for validation rules,
notifications, and automation. Event hooks open up endless possibilities for
integration and customization.

## Get Started with **ish**

After making sure you have `go` version 1.27 or greater installed, clone the
repository and build from source:

```bash
git clone git@github.com:factory-factory/ish.git
cd ish
go build .
```

Then run **ish**. It will find the project's own issues so you can get a
hands-on demo in a real project!

```bash
./ish
```

[demo_reel]: https://github.com/user-attachments/assets/04f83b9a-a344-46cd-95c8-8e9405691579
[ish_flavored_markdown]: https://github.com/user-attachments/assets/1dae6c4d-2067-446d-893c-965a17ca1298
