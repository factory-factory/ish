# Detect changed files and re-index them on start-up

[brendan@berg.industries 2026-10-11T23:40:06-07:00]

- state: closed
- assignees: brendan@berg.industries

---

[brendan@berg.industries 2026-10-11T23:40:06-07:00]

From issue #3CT6E-ZZPJQ:

> After a `git pull` or an edit in another editor, the database will be stale until `ish init` is run again. The `sha256` column makes it possible to detect this at startup and re-index only the files that changed, were added or were removed. I recommend a separate issue for that, to keep this one focused. Until then, `ish init` brings the database up to date.

## Detecting changes

The `sha256` column is present to detect when issue markdown files diverge from the content that has been read into the metadata database. On `ish` startup, compute the SHA-256 hash of the file at the specified path for each issue. If the hash differs from the value stored in the `sha256` column, remove the stale row and insert a new row by re-indexing the file.

---

[Claude 2026-10-11T23:51:43-07:00]

Done, and moved to `in review`.

## What happens at startup

When `ish` starts and the database already exists, a new `store.Sync` brings it up to date with the issue files in one transaction:

- **Changed:** for each issue in the database, it hashes the file at the stored `path`. If the SHA-256 differs from the `sha256` column, it deletes the stale rows (attributes and comments go with them) and re-indexes the file, as this issue describes.
- **Removed:** if there's no file at the stored path, it deletes the issue's rows.
- **Added:** an `.md` file with no issue in the database is indexed.

Your description covers changed files. I also handled added and removed files, because the comment you quoted from 3CT6E-ZZPJQ includes them, and a `git pull` can do all three. Without them, new issues would stay missing from the list and deleted ones would stay in it.

## Details

- Unchanged files are only hashed, not parsed, and their rows are left alone.
- Saving from the TUI already records the new checksum, so ish's own edits aren't picked up again as changes.
- A new or changed file that can't be indexed (it doesn't parse, or has a duplicate key or a key with characters that aren't allowed) is reported, and the stale rows of a changed file are still removed. Such a file has no row, so it's tried and reported again on every start until it's fixed, and then it's indexed as a new issue. That seemed better than letting it disappear from the list without a word.
- `ish` reports only when something changed, for example `Re-indexed changed issue: 1PJRQ-WQMB1`, `Indexed new issue: …` and `Removed deleted issue: …`. A normal start prints nothing.
- If the database can't be updated at all, `ish` says so, suggests running `ish init` to rebuild it, and exits.
- If there's no database yet, `runInit` builds it as before, so there's nothing to sync.

## Testing

- `store` tests cover:
  - no changes, where unchanged rows are left alone (checked by their UUIDs)
  - a changed file, where only its rows are replaced and a second sync finds nothing
  - added and removed files, including the cascading delete
  - files that can't be indexed, including being reported again on the next sync
  - a file saved through ish not being seen as changed
  - a missing database
- I checked that the "left alone" tests fail if every file is re-indexed regardless of its checksum.
- `main.go` tests cover the startup report: a file that still can't be indexed is reported again, it's indexed once fixed, and nothing is printed when nothing changed.
- On a copy of `.ish` with a database, I edited one file, added one and deleted one outside ish. The first start reported all three and the database matched the files (14 issues, 14 files). The second start printed nothing.
