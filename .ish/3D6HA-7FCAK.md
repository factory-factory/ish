# Create object to hold application state

[<brendan@berg.industries> 2026-10-20T17:10:12-07:00]

- state: in review
- assignees: <brendan@berg.industries>
- relation.parent: #3CZWN-EBVEH

---

[<brendan@berg.industries> 2026-10-20T17:10:12-07:00]

Create an object that can be made visible to the filter mechanism that exposes application configuration state.

- State includes application config, loaded at startup via `config.Load`.
- Configuration precedence is defaults, then config file, then env vars, then flags.
- Implement consistent reads using `sync`/`atomic.Pointer`. When executing a query, all lookups should be done on a snapshot so changes can't modify state between reads.

## Loading configuration from disk

The default application configuration should be read from `.ish/config.toml`, a TOML-formatted file that describes a nested object structure for the config object.

---

[Claude <noreply@anthropic.com> 2026-10-21T20:06:47-07:00]

I've reviewed this. I left the state at `planning` and haven't started, since a few things need settling first. Below is how I'd approach it, then my questions.

## Plan

- **A new `config` package:**
  - `config.Load` builds the configuration from defaults, then `.ish/config.toml`, then environment variables, then command-line flags, each overriding the one before.
  - The configuration is a nested object, so that a filter can look up a value by a dotted path, such as `.user.email` in #3CZWN-EBVEH.
- **An application state object:**
  - It holds the loaded configuration in an `atomic.Pointer`, and hands out immutable snapshots.
  - A filter takes one snapshot when it's run, and does every lookup on it. So a change to the state can't change a value partway through a query.
- **Wiring:**
  - `main` loads the configuration at start-up, and passes the state to the app.
  - The filter parser and SQL generator would be given a snapshot when the functions and lookups in #3CZWN-EBVEH are added. This issue would only make the state available.

## Questions

1. **The cut-off sentence:** the last sentence ends at "…a nested object structure for the config object". Is there more to the spec, such as what the file contains?
2. **A TOML library:** Go's standard library doesn't read TOML, and no library on the approved list in `agents.md` does either. I'd suggest adding `github.com/BurntSushi/toml` (v1.4.0), which is small, stable and has no dependencies. `github.com/pelletier/go-toml/v2` is the other common choice. Do I have your approval for one of them? Otherwise I'd need to write a TOML reader, which is a sizeable job for a full grammar.
3. **Which settings exist:**
   - **A fixed schema:** a struct with named fields, such as `user.name`, `user.email`, and `repository.name`. Unknown keys would be errors.
   - **Free-form:** any keys the file has, available to lookups.

   Which do you have in mind? If it's a fixed schema, which settings should it start with, and what are their defaults? For example, should `user.email` and `user.name` default to git's settings, as the current user does now?
4. **Environment variables:** how should they be named? I'd suggest `ISH_` followed by the path, upper-cased, with dots as underscores. So `user.email` would be `ISH_USER_EMAIL`. That only works for keys without underscores, though, so it suits a fixed schema best.
5. **Flags:** which settings should have command-line flags? A flag for every setting, such as `--user.email`, or only some? Should there also be a flag to choose a different config file?
6. **Where the file lives:** `.ish/config.toml` is in the repository, so it's shared by everyone who uses it. Per-person settings such as `user.email` would usually live elsewhere, such as `~/.config/ish/config.toml`, read before the repository's file. Should I support a per-user file too, or only `.ish/config.toml`?
7. **When the state changes:** an `atomic.Pointer` suggests the state can change while ish is running. What would change it? For example, re-reading `config.toml` when the file changes, or a setting changed from inside the TUI? If nothing changes it yet, I'd still build the snapshots, but leave reloading for a later issue.
8. **Scope:** should this issue stop at making the state available? Or should it also add `.path` lookups to the filter language, from #3CZWN-EBVEH?

---

["Brendan Berg" <brendan@berg.industries> 2026-10-21T20:08:44-07:00]

## Answers

1. That sentence was not cut off, it just didn't have a period at the end.
2. I agree with that recommendation. I've added `github.com/BurntSushi/toml` to `agents.md`.
3. Let's make the configuration free-form for now. A future task will be to define a schema for configuration files.
4. Let's put off environment variable config values until the configuration schema has been designed.
5. Same with flags.
6. Yes, I do like the addition of a per-user config file. The precedence should be adjusted such that per-user gets applied first, then the project config is applied on top. Let's look for the per-user config in `~/.ish/config.toml`.
7. In the future, settings could be edited from with in the `ish` application.
8. This issue should also add methods to perform path lookups, but no changes should be made to the filter language implementation.

---

[Claude <noreply@anthropic.com> 2026-10-21T20:19:21-07:00]

Done, following your answers, and moved to `in review`. I moved the state from `planning` to `in progress` when I started.

## Configuration: the `config` package

- **`config.Load(home, dir)`:**
  - It starts from the defaults, which are empty for now, since there's no schema yet.
  - It applies `~/.ish/config.toml`, then the project's `.ish/config.toml`.
  - A missing file is skipped. A file that can't be read is an error naming the file.
- **Merging:** tables are merged key by key, and any other value replaces the one before. So an array in the project's file replaces the user's, rather than adding to it.
- **The free-form `Config`:** a TOML table is a `map[string]any`, and an array is a `[]any`, including arrays of tables. Other values are strings, `int64`s, `float64`s, `bool`s or times, as `github.com/BurntSushi/toml` decodes them.
- **`Config.Lookup(path)`:**
  - It follows a dotted path, such as `.user.email`. The leading dot is optional.
  - A numeric key indexes an array, as in `.remotes.0.url`.
  - It returns a copy, so changing the result can't change the configuration.

## Application state: the `state` package

- **`state.State`:** it holds the current `Snapshot` in an `atomic.Pointer`. `Snapshot()` returns it, for a reader to do all of its lookups on.
- **`SetConfig`:** it stores a new snapshot with a copy of the configuration, for when settings can be edited from within ish. Earlier snapshots keep the values they were taken with.
- **`Snapshot.Lookup` and `LookupString`:** these perform the path lookups.

## Wiring

- **At start-up:** `main` loads the configuration and gives the state to the app, with `Model.SetState`.
- **A bad file:** it stops ish with, for example, `Couldn't read the configuration: .ish/config.toml: toml: line 1: …`.
- **Not used yet:** nothing reads the state. As agreed, the filter language is unchanged, and environment variables and flags are left for the schema.

## Dependency

`github.com/BurntSushi/toml` v1.6.0 is now in `go.mod`, as you approved. Your edit to `agents.md` that lists it is still uncommitted. I've left it for you.

## Testing

- **Loading:**
  - the project's file over the user's, with tables merged, arrays replaced, and arrays of tables as slices of maps
  - no files, no home directory
  - a bad file reported with its path
- **Lookups:** nested keys, with and without the leading dot, array indexes, missing keys, bad indexes, paths through non-tables, and copies that can't change the configuration.
- **State:**
  - earlier snapshots keep their values after a change
  - changing the configuration passed in doesn't change the state
  - `LookupString`, and a missing or empty state
  - concurrent reads and writes, run with the race detector, where values read from one snapshot always match
- **Start-up:** `loadState` reads the per-user file, and reports a bad file. The app's state can be set and read.
- **On a clone of this repository:** a bad `.ish/config.toml` stopped ish with the message and exit status 1, and a valid one started it normally.
