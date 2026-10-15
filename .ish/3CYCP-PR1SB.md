# Filter expression follow-up: more intuitive date comparisons

[brendan@berg.industries 2026-10-14T22:10:13-07:00]

- state: open
- assignees: brendan@berg.industries
- relation.previous: #3CSJ4-PEZ1N

---

[brendan@berg.industries 2026-10-14T22:10:13-07:00]

Date comparisons in filter expressions currently do lexicographical comparisons of RFC 3339 strings in UTC. Ideally, date comparisons should do local time conversions and allow an equality comparison with '2026-10-14' for example to match every issue opened the same day in the local timezone.

There are a couple ways of handling the comparisons. One is to update the filter expression language to include type casts to convert an RFC 3339 string to a UTC timestamp for comparison.

`{@dateCreated >= date('2026-10-01')}` would match all issues created on or after October 1, 2026 in the local time zone.

`{@dateCreated = date('2026-10-01')}` would match issues created between 00:00:00 and 23:59:59 local time on October 1, 2026.

---

[brendan@berg.industries 2026-10-14T23:49:09-07:00]

See also #3CVJY-W5VVN
