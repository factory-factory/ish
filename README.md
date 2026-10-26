# ish

A distributed issue tracker in your terminal

![how to ish][demo_reel]

## Simple and Robust

Issues sync alongside your code, managed by the same version control you use for
everything else. Contributors can create new issues, edit metadata, participate
in conversations about development tasks, and move issues through project
management workflowss. Updates are pushed to the entire team through the same
version control process they're already used to.

Issues are stored in a directory within the repository as Markdown files. (The
command line utility maintains a SQLite database for fast queries and project
analytics, but the filesystem is always ground truth.) The Markdown conventions
**ish** uses are straightforward and intuitive, so you could even use `vim` to
keep your work up-to-date in a pinch.

## Familiar Concepts, New

The idea of an issue in a bug tracker or task planning tool is fairly
ubiquitous. In **ish**, issues are constructed around conversations that happen
in comments appended to the issue. Each comment has a byline that records the
user's identity and the comment's timestamp.

Conversations provide an additional benefit in being a natural approach for
dispatching AI coding agents. Agents can read and edit Markdown, and are able to
respond by appending comments to their assigned issues. In fact, **ish** has
been built this way from the start.

Workflows build on the core concepts and allow teams to design operational
processes that ensure issues are edited correctly as they move towards
completion. Event hooks provide a framework for validation rules, notifications,
and automation.

## Easy to Get Started

```bash
git clone git@github.com:factory-factory/ish.git
cd ish
go build .
./ish
```

## Why **ish**?

**Works offline.** Off-grid doesn't mean out-of-the-loop. You can still view and
edit issues when you're in airplane mode and changes sync when you're back
online. There's no excuse for losing time to a sluggish web app in 2026!

**Everything's in one place.** Issues travel with the code, so they branch,
merge, and appear in history alongside the code they describe.

**No vendor lock-in.** We think our tools are great, but if you don't agree you
can easily migrate, because your data was never in someone else's walled garden.

**Auditable and Diffable.** Plain text is readable without a specialized app
(but we think our terminal app provides delightful editing and collaboration
features). Building on standard version control tools lets you view an issue's
history and see who's responsible for which edits.

[demo_reel]: https://github.com/user-attachments/assets/04f83b9a-a344-46cd-95c8-8e9405691579
