# Define new filters for the filter drop-down

[<brendan@berg.industries> 2026-10-15T23:44:19-07:00]

- state: open
- assignees: <brendan@berg.industries>
- relation.depends-on: #3CZWN-EBVEH

---

[<brendan@berg.industries> 2026-10-15T23:44:19-07:00]

Possible filters:

- **Assigned to me** `{assignees = get(user.email)}`
- **Created by me** `{@author = get(user.email)}`
- **Mentioned** `{@comment = '@{get(user.email)}'}`
- **Recent activity** `{@updated > date(@today - 1w)}`
