# Specify configuration file format

[<brendan@berg.industries> 2023-06-04T18:18:40-04:00]

- state: planning
- assignees: <brendan@berg.industries>

---

[<brendan@berg.industries> 2023-06-04T18:18:40-04:00]

The `.ishrc` file should be in a sane format (TOML? INI?)
and provide configuration directives for the following:

- issue directory / branch
- id scheme (?)
- id prefix (?)

Further thoughts:

- Since configuration files inevitably grow into full scripting languages, there needs to be a roadmap to include defining custom functions and types.
- An effect system, or some form of restriction on computation.
