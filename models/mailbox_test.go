package models

import "testing"

func TestSplitMailbox(t *testing.T) {
	for _, tt := range []struct {
		text    string
		box     Mailbox
		written string
		rest    string
	}{
		// A bare addr-spec.
		{"alice@example.com 2026-10-01", Mailbox{Email: "alice@example.com"}, "alice@example.com", "2026-10-01"},
		{"a.b+c@mail.example.com", Mailbox{Email: "a.b+c@mail.example.com"}, "a.b+c@mail.example.com", ""},

		// An addr-spec in angle brackets, with or without a name.
		{"<alice@example.com> 2026-10-01", Mailbox{Email: "alice@example.com"}, "<alice@example.com>", "2026-10-01"},
		{"Alice Liddell <alice@example.com> 2026", Mailbox{"Alice Liddell", "alice@example.com"}, "Alice Liddell <alice@example.com>", "2026"},
		{`"Liddell, Alice" <alice@example.com> 2026`, Mailbox{"Liddell, Alice", "alice@example.com"}, `"Liddell, Alice" <alice@example.com>`, "2026"},
		{`"Say \"hi\"" <a@example.com> x`, Mailbox{`Say "hi"`, "a@example.com"}, `"Say \"hi\"" <a@example.com>`, "x"},
		{`Dr "Alice" Liddell <a@example.com> x`, Mailbox{"Dr Alice Liddell", "a@example.com"}, `Dr "Alice" Liddell <a@example.com>`, "x"},
		{`"Dr. Alice Liddell" <a@example.com> x`, Mailbox{"Dr. Alice Liddell", "a@example.com"}, `"Dr. Alice Liddell" <a@example.com>`, "x"},
		{"José Núñez <jose@example.com> x", Mailbox{"José Núñez", "jose@example.com"}, "José Núñez <jose@example.com>", "x"},

		// The rest can have spaces, as some timestamps do.
		{"<a@example.com> 2023-05-28 11:55:23 -0400", Mailbox{Email: "a@example.com"}, "<a@example.com>", "2023-05-28 11:55:23 -0400"},
	} {
		box, written, rest, err := SplitMailbox(tt.text)

		if err != nil || box != tt.box || written != tt.written || rest != tt.rest {
			t.Errorf("SplitMailbox(%q) = %+v, %q, %q, %v, want %+v, %q, %q", tt.text, box, written, rest, err, tt.box, tt.written, tt.rest)
		}
	}

	// Not mailboxes.
	for _, text := range []string{
		"Claude 2026-10-01",
		"Alice alice@example.com 2026",
		"Alice <alice> 2026",
		"Alice <alice@example.com 2026",
		`"Alice <alice@example.com> 2026`,
		"<alice@example.com>2026",
		"alice@ 2026",
		"@example.com 2026",
		"alice..b@example.com",
		// A dot in an unquoted name is an obsolete form, so it's not accepted.
		"Dr. Alice <alice@example.com> 2026",
		"",
	} {
		if box, _, _, err := SplitMailbox(text); err == nil {
			t.Errorf("SplitMailbox(%q) = %+v, want an error", text, box)
		}
	}
}

func TestParseMailbox(t *testing.T) {
	if box, err := ParseMailbox("Alice <alice@example.com>"); err != nil || box != (Mailbox{"Alice", "alice@example.com"}) {
		t.Errorf("ParseMailbox() = %+v, %v, want Alice", box, err)
	}
	if _, err := ParseMailbox("<alice@example.com> more"); err == nil {
		t.Error("ParseMailbox() with text after the mailbox error = nil, want an error")
	}
}

func TestMailboxString(t *testing.T) {
	for box, want := range map[Mailbox]string{
		{"Brendan Berg", "brendan@berg.industries"}: `"Brendan Berg" <brendan@berg.industries>`,
		{`Say "hi" \o/`, "a@example.com"}:           `"Say \"hi\" \\o/" <a@example.com>`,
		{"", "a@example.com"}:                       "a@example.com",
	} {
		if got := box.String(); got != want {
			t.Errorf("%+v.String() = %q, want %q", box, got, want)
		}

		// It reads back as the same mailbox.
		if back, err := ParseMailbox(box.String()); err != nil || back != box {
			t.Errorf("ParseMailbox(%q) = %+v, %v, want %+v", box.String(), back, err, box)
		}
	}
}

func TestBylineAuthorNames(t *testing.T) {
	for author, want := range map[string]string{
		// The directory's name for the email, which is the newest.
		`"Bobby" <bob@example.com>`: "@Bob",
		// Otherwise, the byline's own name.
		`"Carol" <carol@example.com>`: "@Carol",
		"<carol@example.com>":         "carol@example.com",
	} {
		if got := people.author(author); got != want {
			t.Errorf("author(%q) = %q, want %q", author, got, want)
		}
	}
}
