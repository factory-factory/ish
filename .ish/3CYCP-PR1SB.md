# Filter expression follow-up: more intuitive date comparisons

[<brendan@berg.industries> 2026-10-14T22:10:13-07:00]

- state: closed
- assignees: <brendan@berg.industries>
- relation.previous: #3CSJ4-PEZ1N
- relation.depends-on: #3CZWN-EBVEH

---

[<brendan@berg.industries> 2026-10-14T22:10:13-07:00]

Date comparisons in filter expressions currently do lexicographical comparisons of RFC 3339 strings in UTC. Ideally, date comparisons should do local time conversions and allow an equality comparison with '2026-10-14' for example to match every issue opened the same day in the local timezone.

There are a couple ways of handling the comparisons. One is to update the filter expression language to include type casts to convert an RFC 3339 string to a UTC timestamp for comparison.

`{@dateCreated >= date('2026-10-01')}` would match all issues created on or after October 1, 2026 in the local time zone.

`{@dateCreated = date('2026-10-01')}` would match issues created between 00:00:00 and 23:59:59 local time on October 1, 2026.

---

[<brendan@berg.industries> 2026-10-14T23:49:09-07:00]

See also #3CVJY-W5VVN

---

["Brendan Berg" <brendan@berg.industries> 2026-10-21T12:46:45-07:00]

Current thinking on match syntax / semantics:

- **Issues created on a specific day:** `{@dateCreated = ('2026-10-01' :date)}`
- **Issues created on or after a specific date:** `{@dateCreated >= ('2026-10-01' :date)}`
- **Issues created in the past week:** `{@dateCreated > ('-7d' :fromnow)}`

---

["Brendan Berg" <brendan@berg.industries> 2026-10-24T10:27:59-07:00]

After #3DAKC-E4PWB, there's now a more expressive way to filter by date range.

To match issues created on a specific day, the following filter could be used:

```
{@dateCreated >= '2026-10-20' and < '2026-10-21}
```
