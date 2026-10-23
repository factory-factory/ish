# Implement date & time filter expression functions

["Brendan Berg" <brendan@berg.industries> 2026-10-22T18:04:51-07:00]

- state: in review
- assignees: <brendan@berg.industries>
- relation.parent: #3CZWN-EBVEH

---

["Brendan Berg" <brendan@berg.industries> 2026-10-22T18:04:51-07:00]

Implement the `:fromnow` function.

- Takes a string describing a temporal offset from the current moment and returns the RFC3339 string representing the resulting time. Offsets consist of a number and a unit of time. Numbers may be positive or negative.
- Allowed units are `m` or `min` (minutes), `h` or `hr` (hours), `d` (days), `w` (weeks), `mo` (calendar months), and `y` or `yr` (calendar years). Note that adding calendar months clamp to the end of the month, so `2026-01-31 +1mo` would be `2026-02-28`. (Similarly, `2028-02-29 +1yr` would be `2029-02-28`.)

---

[Claude <noreply@anthropic.com> 2026-10-22T21:42:23-07:00]

I've reviewed this and moved it to `in progress`. Below is my plan, then some questions. I'll start once they're answered.

## Plan

- **The function:** `:fromnow` is an expression function. `('-3h' :fromnow)` gives the moment 3 hours ago, as an RFC 3339 string, such as `2026-10-22T22:04:51Z`.
- **Units:** `m`/`min`, `h`/`hr`, `d`, `w`, `mo` and `y`/`yr`, with an optional `+` or `-` sign. Months and years are calendar months and years, clamped to the end of the month, as the ticket describes.
- **Results in UTC:** `@dateCreated` is stored in UTC as RFC 3339 text since #3CVJY-W5VVN. So a result in UTC compares correctly with it as text, as in `{@dateCreated > ('-7d' :fromnow)}`.
- **One moment per query:** "now" is taken once, when the filter is applied or the list reloads. So every `:fromnow` in a query, and every issue it's compared with, uses the same moment.
- **How it's evaluated:** like `:user` in #3D6W6-1SEBX, it's resolved before the query, and its result is passed as a parameter. An SQL function would read the clock separately for every issue, and couldn't promise a single moment.
- **Errors:** an offset that can't be read, such as `('3 fortnights' :fromnow)`, is an error under the filter, naming the allowed units.

## Questions

1. **The form of offsets:**
   - Whole numbers only, or decimals too, as in `1.5h`?
   - Can there be a space between the number and the unit, as in `-3 h`?
   - Should several parts be allowed, as in `1d 12h`? The ticket describes one number and one unit, so I'd suggest only that, with whole numbers, and no space.
2. **Which time zone calendar units use:**
   - **(a) Local:** months, years and days are counted in your local time zone, then converted to UTC. So `-1mo` at 10pm on 31 January, Pacific time, is the end of December in Pacific time. I'd suggest this, since that's how you'd count them.
   - **(b) UTC:** simpler, but near midnight it can land on a different day, or clamp in a different month, than you'd expect locally.

   Days and weeks are exact multiples of 24 hours in UTC, but not in local time across a daylight saving change. With (a), I'd count days and weeks on the calendar, so `-1d` is the same time yesterday.
3. **Precision:** stored dates have whole seconds, so I'd give whole seconds too, without fractions. Is that all right?
4. **Dates without times:** metadata values such as `due: 2026-10-30` are only dates. As text, `2026-10-30` is less than any time on that day, such as `2026-10-30T00:00:00Z`. So `{due < ('+1d' :fromnow)}` would compare a date with a time. That's beyond this ticket, but should I note it for the `:date` function in #3CZWN-EBVEH?
5. **Scope:** the title says "date & time functions", but the ticket describes only `:fromnow`. I've assumed `:time`, `:date` and `:datetime` from #3CZWN-EBVEH are for later. Is that right?

---

["Brendan Berg" <brendan@berg.industries> 2026-10-22T21:46:37-07:00]

1. Offsets are whole numbers only, but may contain multiple units like `-1d 6h`. Spaces may separate different units, but there should be no space between the number and the unit.
2. The offset should be computed in the local time zone and then converted to UTC.
3. Whole seconds is the expected level of precision.
4. This will be addressed at a later date.
5. The other functions are on hold since it's not clear exactly how they'll be used.

---

[Claude <noreply@anthropic.com> 2026-10-22T21:54:44-07:00]

Done, following your answers, and moved to `in review`.

## `:fromnow`

- **The function:** `('-3h' :fromnow)` gives the moment 3 hours ago, as an RFC 3339 string in UTC with whole seconds, such as `2026-10-22T19:46:37Z`. It compares directly with `@dateCreated`, as in `{@dateCreated > ('-7d' :fromnow)}`.
- **Offsets:**
  - an offset is one or more parts, separated by spaces, as in `-1d 6h`
  - each part is a whole number and a unit, with no space between them
  - the units are `m`/`min`, `h`/`hr`, `d`, `w`, `mo` and `y`/`yr`
- **Signs:** your answers didn't say how signs apply to several parts, so I chose: a part's sign applies to the parts after it, until another sign. So `-1d 6h` is a day and six hours ago, and `-1d +6h` is eighteen hours ago.
- **Interpolation:** an offset can be built with interpolation, as in `('-\(.days)d' :fromnow)`.
- **Errors:** an offset that can't be read is an error under the filter, naming the part and the allowed units. For example: `:fromnow can't read "soon"; an offset is a whole number and a unit: m or min, h or hr, d, w, mo, or y or yr, as in -3h or +1mo 2d`. A value that isn't a string is a type error, and `null` gives `null`.

## Calendar arithmetic

- **Where it's done:** in the local time zone, and the result is converted to UTC.
- **Years and months:** applied first, clamped to the end of the month. So January 31 plus a month is February 28, and February 29, 2028 plus a year is February 28, 2029.
- **Days and weeks:** then calendar days, keeping the time of day, so `-1d` is the same time yesterday, even across a change to daylight saving time.
- **Hours and minutes:** last, as exact durations, so `-24h` is exactly 24 hours earlier.

## One moment per query

- **Reading the clock:** it's read once each time a filter is resolved, when it's applied or the list reloads. So every `:fromnow` in a query counts from the same moment.
- **How it's evaluated:** like `:user`, `:fromnow` is replaced by its result before the query, and the result is passed as a parameter.
- **Tests:** `filter.Env` has a `Now` field, which they use to fix the moment and time zone.

## Testing

- **Each unit:** minutes, hours, days, weeks, months and years, from a moment where the local date differs from UTC's.
- **Several parts:** with signs carried and changed, and extra spaces.
- **Clamping:** at the end of months and on leap days, forwards and backwards, and across years.
- **Daylight saving:** calendar days compared with exact hours across the change.
- **Errors:** an empty offset, a number or unit missing, decimals, a space before the unit, unknown units, capital letters, double signs and other separators.
- **Resolving:** interpolated offsets, `null` as UNKNOWN, type errors, printing as written, and one moment for several `:fromnow`s.
- **Against a database:** `@dateCreated` compared with hours, days, weeks and months ago.
- **In a real terminal,** on a clone of this repository, `{@dateCreated > ('-2d' :fromnow)}` listed exactly the 3 issues created in the last two days.
