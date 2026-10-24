# Define new filters for the filter drop-down

[<brendan@berg.industries> 2026-10-15T23:44:19-07:00]

- state: closed
- assignees: <brendan@berg.industries>
- relation.depends-on: #3CZWN-EBVEH, #3DACD-RNJ3F, #3DADN-MG52G

---

[<brendan@berg.industries> 2026-10-15T23:44:19-07:00]

Possible filters:

- **Assigned to me** `{assignees = .user.email}`
- **Created by me** `{@author = .user.email}`
- **Mentioned** `{@comments ~ .user.email}` or `{@comments ~ (.user.name :user)}`
- **Recent activity** `{@dateModified > ('-1w' :fromnow)}`
