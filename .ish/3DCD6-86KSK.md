# Use fsnotify to watch for git state updates

["Brendan Berg" <brendan@berg.industries> 2026-10-24T21:22:23-07:00]

- state: closed
- assignees: <noreply@anthropic.com>
- watchers: <brendan@berg.industries>

---

["Brendan Berg" <brendan@berg.industries> 2026-10-24T21:22:23-07:00]

## <noreply@anthropic.com> write in #3D0Y1-C8EY4:

> you're right, changes in `.ish/` can't be told apart from a `git pull`. Some history changes don't touch `.ish/` at all; committing an issue you'd already edited adds a modification date without changing any issue file.
>
> - **What I'd do instead:** watch `.git/logs/HEAD`, which git adds a line to every time HEAD moves. That covers commit, pull, merge, rebase, checkout and reset. When it changes, I'd update the commit history and the user table. The file is found with `git rev-parse --git-path logs/HEAD`, so worktrees work too.
> - **Limitation:** this relies on git's reflog, which is on by default but can be turned off. If it is, history goes back to updating only at start-up.
> - **A gap in the current code:** history indexing only asks git for commits newer than the newest one already stored. A pull that brings in older commits, as merges often do, would leave them out until the next full index. I'd change it to look for commits it doesn't have yet, which is still cheap.

In this ticket, implement watch notifications for git state to trigger re-indexing commit history and users. The reflog requirement is understandable. A future issue may address this by testing whether the reflog is on, but that is outside the scope of this ticket.

---

[Claude <noreply@anthropic.com> 2026-10-24T21:44:17-07:00]

Done.

- **Watching** (`watch` package): `watch.Reflog` finds HEAD's reflog with `git rev-parse --git-path logs/HEAD`, so a worktree's own reflog is used. `watch.NewFile` watches that one file, waiting 100 ms after the last of a burst of changes, as the issue watcher does. Outside a git repository, or in one without commits yet, nothing extra is watched and no warning is shown.
- **On each change** (`app`): the issue files are synced first, so issues a pull adds are in the database before their history is indexed. Then `IndexHistory` and `IndexUsers` run, and the list reloads. If an issue is open, its user names are refreshed instead.
- **The `IndexHistory` gap:** history is no longer indexed by date. The database records the commit HEAD was at when history was last indexed, in a new `historyHead` table. The schema goes to version 7, so existing databases are rebuilt once at start-up.
  - If HEAD has moved forward from that commit, only the commits in between are indexed, whatever their dates. A merge that brings in older commits now adds them.
  - If it hasn't, as after a checkout of another branch, a reset, or a rebase, history is rebuilt from HEAD. Commits that are no longer on the branch are removed.
- **Issues added after their commits:** `Sync` now adds the history of the files it adds, such as a file fixed after failing to parse, with one `git log` limited to those files.

Like the file watcher, this all runs in the UI's update loop, so it never writes to the database at the same time as ish's own saves. An ordinary pull only reads the new commits, so it's quick. A full rebuild after switching branches runs `git log` over the whole history, which in a very large repository could pause the UI briefly.

Tests:

- `store/history_test.go`: indexing the commits since the recorded one, rebuilding when history is rewritten, working without git, history for files `Sync` adds, and a test against a real repository covering a commit, a merge of an older commit, and a reset.
- `watch/watch_test.go`: watching a single file, finding the reflog in a repository and in a worktree, commits and checkouts being reported, and a repository without commits.
- `app/history_test.go`: commits, new committers' names, issues brought in by a pull, and the open issue being deleted by one.

The whole suite passes, and `-race` is clean on `watch` and `app`. In a real terminal, I filtered the list on `@dateModified`; a commit made while ish was running added the issue to it, and resetting the commit removed it again.

---

["Brendan Berg" <brendan@berg.industries> 2026-10-24T22:19:32-07:00]

I'm not sure about the table name `historyHead`. Something like `versionHeadMarker` is more descriptive. Please make that change and I think this will be good.

---

[Claude <noreply@anthropic.com> 2026-10-24T22:31:07-07:00]

Renamed the table to `versionHeadMarker`. I also bumped the schema to version 8, so a database already built with the old `historyHead` name is rebuilt at start-up instead of failing to find the table. The full suite passes.
