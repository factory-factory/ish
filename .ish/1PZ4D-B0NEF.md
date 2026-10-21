# Implement 'ish init' to create the metadata database

[<brendan@berg.industries> 2023-06-04T18:37:55-04:00]

- state: closed
- assignees: <brendan@berg.industries>

---

[<brendan@berg.industries> 2023-06-04T18:37:55-04:00]

The issue database is a local metadata cache for all issues. It provides a mechanism for efficiently searching and filtering issues as well as for generating reports.

The database uses the SQLite engine, and the database file is saved in the `.ish` base directory as `.metadata.ish`.

The database schema is described below.

## Issue

- `id`: the issue ID as seen in the markdown filename
- `path`: the path to the issue relative to the ish root directory (`.ish`)
- `title`: the issue title
- `author`: the author string from the byline
- `dateCreated`: the ISO-8601 date string (converted to UTC) from the byline
- `tzOffset`: the timezone offset from the date string in the byline
- `sha256`: the SHA-256 checksum of the file

## Attribute

- `uuid`: a unique identifier for the metadata attribute
- `issueId`: [foreign key] the Issue ID the attribute belongs to
- `key`: the metadata attribute key (alphanumeric plus period, underscore, hyphen, colon, and forward slash)
- `value`: the metadata attribute value, as JSON

The attribute table should have a unique key constraint on the `issueID, key` pair.

## Comment

- `uuid`: a unique identifier for the comment
- `issueId`: [foreign key] the Issue ID the comment belongs to
- `sortIndex`: [uint] the position in the list of comments
- `author`: the author string from the comment byline
- `dateCreated`: the ISO-8601 date string (converted to UTC) from the comment byline
- `tzOffset`: the timezone offset from the date string in the byline
- `content`: the raw markdown text of the comment

The comment table should have a unique key constraint on the `issueID, sortIndex` pair.

---

[<brendan@berg.industries> 2026-10-11T21:42:24-07:00]

To start out, create a subcommand of the utility so that running `ish init` looks for the `.ish` base directory and metadata store. If the directory doesn't exist, it should create it, and if the metadata store doesn't exist, it should create the SQLite database and install the empty schema.

In the case that there is already an `.ish` directory with issue markdown files, `ish init` should parse each of the files and populate the metadata DB with the information from the file.

---

[Claude <noreply@anthropic.com> 2026-10-11T21:53:21-07:00]

I've reviewed this and drafted a plan, below, but I need answers before implementing. Questions 1 and 2 block the work. For the rest, I'll go with my recommendation unless you say otherwise.

## Questions

1. **SQLite driver (blocking).** No SQLite driver is on the approved dependency list. Which may I add?
   - `modernc.org/sqlite`: pure Go, so no C compiler is needed and ish cross-compiles easily. It makes the binary several MB larger and is somewhat slower. I recommend this one.
   - `github.com/mattn/go-sqlite3`: the most widely used, and faster, but it uses cgo, so building ish needs a C toolchain.
2. **Attributes with several values (blocking).** Tags can have several values, such as `assignees: brendanberg, jhc` in 1PJRQ-WQMB1. That doesn't fit a single `value` column with a unique (`issueId`, `key`) pair. Options:
   - (a) One row per value, unique on (`issueId`, `key`, `value`). This makes filtering simple, such as finding every issue assigned to jhc. I recommend this.
   - (b) One row per key with the values joined, `brendanberg, jhc`. This keeps the constraint as written, but filtering has to match inside the text.
   - (c) One row per key with the values stored as a JSON array.
3. **Duplicate keys.** The file format allows the same key twice in one issue (no current issue does this). With (a), I'd merge their values. With (b) or (c), should a duplicate be an error?
4. **Key characters.** This issue limits keys to letters, digits and `. _ - : /`, but the parser accepts any key without spaces, and the "add field" editor from 3CSGN-E1JWT only removes spaces. My recommendation: `ish init` reports a key outside the allowed characters as an error for that file, and enforcing the same rule in the editor becomes a separate issue.
5. **Existing database.** If `.metadata.ish` already exists, should `ish init` leave it untouched, rebuild it from the files, or update only the issues whose SHA-256 changed? Since it's a cache, I recommend rebuilding it for now, which is simple and always correct, and adding a faster sync later.
6. **Files that don't parse.** Should `ish init` stop without changing the database, or skip the file, index the rest, and report it? I recommend skipping and reporting each failed file, and exiting with a non-zero status so scripts notice.
7. **`dateCreated` format.** Bylines use three different date formats across the current issues. Should the database store the date exactly as written, or converted to RFC 3339 in UTC so dates sort and compare correctly in SQL? I recommend converting.
8. **Finding `.ish`.** Should `ish init` use `.ish` in the current directory, as the TUI does today, or search parent directories like git does? I recommend the current directory for now, to match the TUI.
9. **Version control.** `.ish/` is committed, but the database is a local cache, and SQLite can also create `.metadata.ish-wal` and `.metadata.ish-journal` files next to it. Should `ish init` create `.ish/.gitignore` ignoring `.metadata.ish*`? I recommend yes, so the cache is never committed by accident.

## Plan

1. **Dependency:** add the SQLite driver you approve (question 1) to `go.mod` and to the approved list in `agents.md`.
2. **New `store` package** for the database:
   - `Init(dir)` creates `dir` if needed (and `.gitignore`, per question 9), then creates `.metadata.ish` if it's missing, all in one transaction. It turns on SQLite foreign keys and sets `PRAGMA user_version = 1`, so later schema changes can be detected and migrated.
   - **Schema:**
     - `issue`: `id` is the primary key and `path` is unique, plus `title`, `author`, `dateCreated` and `sha256`.
     - `attribute`: `uuid`, then `issueId` referencing `issue(id)` with `ON DELETE CASCADE`, plus `key` (with a `CHECK` on its characters) and `value`. Its unique constraint depends on question 2, and there's an index on (`key`, `value`) for filtering.
     - `comment`: `uuid`, then `issueId` referencing `issue(id)` with `ON DELETE CASCADE`, plus `sortIndex` (`CHECK (sortIndex >= 0)`), `author`, `dateCreated` and `content`. It's unique on (`issueId`, `sortIndex`).
   - **UUIDs** are random version-4 UUIDs from `crypto/rand`, so no UUID library is needed.
3. **Indexing:** for each `*.md` file in `.ish/`, read it, parse it with the existing parser, compute the file's SHA-256, and insert the issue, its attributes and its comments (in order, with `sortIndex` from 0). Failures are handled per question 6.
4. **`ish init` subcommand:** registered with cobra in `main.go`. It reports what it did, for example `Created .ish/`, `Created .ish/.metadata.ish`, and `Indexed 10 issues`, plus any files it couldn't index.
5. **Tests**, all in temporary directories:
   - creating the directory and the database, and the schema's tables, constraints and `user_version`
   - running `init` twice (question 5)
   - indexing sample issues: attributes, comment order, `sortIndex`, `sha256` and dates
   - files that don't parse, and keys with characters that aren't allowed
   - the unique constraints and cascading deletes

Not in this issue: the TUI still reads the markdown files directly. Using the database for listing, searching or filtering would be later issues.

---

[<brendan@berg.industries> 2026-10-11T21:57:05-07:00]

To clarify and unblock:

1. Add `modernc.org/sqlite` as a dependency to both go.mod and agents.md.
2. Instead of one row per value for multi-value tags, let's make the value column use JSON values. This allows us to maintain the unique index as described as well as allow complex types in the value field. (Which I see you already thought of in option (c) in your comment.)
3. Duplicate keys should be an error, and we can address the UI for validating key uniqueness in a separate issue.
4. Your recommendation that `ish init` reports a key outside the allowed characters as an error for that file is spot-on. Enforcing the same rule in the editor can be handled in a separate issue.
5. Since the markdown files should always be treated as ground truth, `ish init` should rebuild the database if it finds the file already exists.
6. Skip and report files that fail to parse. Exiting with a non-zero status makes sense if there are files that don't parse.
7. The date fields should be stored consistently in the database. This means we should convert all timestamps to UTC and store the timezone offset as a separate column.
8. Let's keep things simple for now. Use `.ish` in the current directory.
9. I was going to suggest we handle gitignore as a separate issue, but your intuition is correct. Let's also have `ish init` create `.ish/.gitignore` with an entry to ignore `.metadata.ish*`.

---

[Claude <noreply@anthropic.com> 2026-10-11T22:28:53-07:00]

Done, following your answers, and moved to `in review`.

## `ish init`

- Run in the directory containing `.ish`. It creates `.ish/` if it doesn't exist, adds `.metadata.ish*` to `.ish/.gitignore` (creating the file, or appending to an existing one, if the entry isn't already there), and builds `.ish/.metadata.ish` from the issue files.
- An existing database is always rebuilt from the files. The new database is built in a temporary file next to it and then swapped in, so if the build fails the old database is left intact. The temporary file's name also matches the `.gitignore` entry.
- It reports what it did, for example `Created .ish/.metadata.ish` or `Rebuilt .ish/.metadata.ish`, then `Indexed 12 issues`. Files it couldn't index are listed as `Couldn't index <path>: <reason>`, and it then exits with status 1.
- A file is skipped, and nothing from it is added, if it doesn't parse, has a duplicate metadata key or a key with characters other than letters, digits and `. _ - : /`, or has a byline date that can't be read. Each file is added under its own savepoint, so one that fails part-way leaves no partial rows.

## Database

- `issue`: `id` (primary key), `path` (unique; the file name within `.ish`), `title`, `author`, `dateCreated`, `dateCreatedOffset` and `sha256` (of the whole file, as hex).
- `attribute`: `uuid` (primary key), `issueId`, `key` and `value`, unique on (`issueId`, `key`). The database checks the key's characters and that the value is valid JSON. There's an index on `key`.
- `comment`: `uuid` (primary key), `issueId`, `sortIndex` (0 for the first comment, and never negative), `author`, `dateCreated`, `dateCreatedOffset` and `content` (the raw markdown), unique on (`issueId`, `sortIndex`).
- `issueId` references `issue(id)` with `ON DELETE CASCADE`, and `store.Open` turns on foreign key enforcement, which SQLite needs for every connection.
- `PRAGMA user_version` is 1, so later schema changes can be detected.

Decisions within your answers:

- **Values** are always a JSON array of strings, even for one value, such as `["open"]`, so every row has the same shape. Filtering works with SQLite's JSON functions, for example:

  `SELECT attribute.issueId FROM attribute, json_each(attribute.value) WHERE attribute.key = 'assignees' AND json_each.value = 'jhc'`

- **Dates** are RFC 3339 in UTC, such as `2023-05-27T03:35:36Z`. `dateCreatedOffset` is the byline's offset in minutes east of UTC, such as `-240` for `-04:00`, which is exact and enough to rebuild the original time.
- **UUIDs** are random version-4 UUIDs from `crypto/rand`, so there's no UUID dependency. They're new each time the database is rebuilt, since it's a cache.

## Other changes

- Added `modernc.org/sqlite` v1.60.1 to `go.mod` and to the approved list in `agents.md`.
- New `store` package, with `Init` and `Open`. `ish init` is a cobra subcommand in `main.go`.
- Exported the byline date parser as `models.ParseDate` so `store` can use it.

## Testing

- `store` tests, all in temporary directories, cover:
  - creating the directory, `.gitignore` and schema
  - indexing: rows, UTC dates and offsets, SHA-256, JSON values, comment order and UUIDs
  - rebuilding after files change or are removed
  - keeping an existing `.gitignore`
  - each kind of file that's skipped, with no partial rows (I checked this test fails if the savepoint rollback is removed)
  - leaving the old database intact when a rebuild fails
  - every unique, check and foreign key constraint, and cascading deletes
- `ish init` tests in `main_test.go` cover its output and exit status.
- I ran `ish init` on copies of `.ish`, never on the real one. On the 12 current issues it built 12 issues, 26 attributes and 26 comments. The values and dates matched the files, and a JSON filter found 1PJRQ-WQMB1 as the only issue assigned to jhc. With an unparseable file added, it skipped that file, reported it and exited with status 1. The TUI still lists issues with the database present.

The TUI doesn't use the database yet; as planned, that's for later issues. Two follow-ups you mentioned: checking key characters, and duplicate keys, in the issue view's metadata editor.
