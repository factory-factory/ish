package filter

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func mustParse(t *testing.T, text string) Node {
	t.Helper()

	node, err := Parse(text)

	if err != nil {
		t.Fatalf("Parse(%q) error = %v", text, err)
	}

	return node
}

func attr(name string, op Operator, raw string, data any) *AttributeExpression {
	return &AttributeExpression{Attribute: name, Operator: op, Value: Value{Raw: raw, Data: data}}
}

func TestParseTrees(t *testing.T) {
	a := func() *AttributeExpression { return attr("a", Equal, "1", json.Number("1")) }
	b := func() *AttributeExpression { return attr("b", Equal, "2", json.Number("2")) }
	c := func() *AttributeExpression { return attr("c", Equal, "3", json.Number("3")) }

	tests := []struct {
		text string
		want Node
	}{
		{"{a = 1}", a()},
		{"{a=1}", a()},
		{"  {  a  =  1  }  ", a()},

		// and binds more tightly than or, and not more tightly than both.
		{"{a = 1} or {b = 2} and {c = 3}", &OrExpression{LeftExpression: a(), RightExpression: &AndExpression{LeftExpression: b(), RightExpression: c()}}},
		{"{a = 1} and {b = 2} or {c = 3}", &OrExpression{LeftExpression: &AndExpression{LeftExpression: a(), RightExpression: b()}, RightExpression: c()}},
		{"not {a = 1} and {b = 2}", &AndExpression{LeftExpression: &NotExpression{Expression: a()}, RightExpression: b()}},
		{"not not {a = 1}", &NotExpression{Expression: &NotExpression{Expression: a()}}},

		// Binary operators group from the left.
		{"{a = 1} or {b = 2} or {c = 3}", &OrExpression{LeftExpression: &OrExpression{LeftExpression: a(), RightExpression: b()}, RightExpression: c()}},

		// Parentheses group, and are counted.
		{"({a = 1} or {b = 2}) and {c = 3}", &AndExpression{
			LeftExpression:  &OrExpression{LeftExpression: a(), RightExpression: b(), Parentheses: 1},
			RightExpression: c(),
		}},
		{"(({a = 1}))", &AttributeExpression{Attribute: "a", Operator: Equal, Value: Value{Raw: "1", Data: json.Number("1")}, Parentheses: 2}},
		{"not ({a = 1})", &NotExpression{Expression: &AttributeExpression{Attribute: "a", Operator: Equal, Value: Value{Raw: "1", Data: json.Number("1")}, Parentheses: 1}}},

		// Keywords need space only where a name would otherwise continue.
		{"({a = 1})and({b = 2})", &AndExpression{
			LeftExpression:  &AttributeExpression{Attribute: "a", Operator: Equal, Value: Value{Raw: "1", Data: json.Number("1")}, Parentheses: 1},
			RightExpression: &AttributeExpression{Attribute: "b", Operator: Equal, Value: Value{Raw: "2", Data: json.Number("2")}, Parentheses: 1},
		}},

		// Attribute names use letters, digits, and - _ . /
		{"{relation.parent = 'x'}", attr("relation.parent", Equal, "'x'", "x")},
		{"{a-b_c/d.e2 != 'x'}", attr("a-b_c/d.e2", NotEqual, "'x'", "x")},

		// Fields start with @.
		{"{@title ~ /^Filter/}", attr("@title", Match, "/^Filter/", Regexp("^Filter"))},
		{"{@author = 'alice'}", attr("@author", Equal, "'alice'", "alice")},
		{"{@dateCreated >= '2026-10-01'}", attr("@dateCreated", GreaterOrEqual, "'2026-10-01'", "2026-10-01")},
		{"{ @title!='x' }", attr("@title", NotEqual, "'x'", "x")},

		// Every operator.
		{"{a != 1}", attr("a", NotEqual, "1", json.Number("1"))},
		{"{a < 1}", attr("a", Less, "1", json.Number("1"))},
		{"{a > 1}", attr("a", Greater, "1", json.Number("1"))},
		{"{a <= 1}", attr("a", LessOrEqual, "1", json.Number("1"))},
		{"{a >= 1}", attr("a", GreaterOrEqual, "1", json.Number("1"))},
		{"{a ~ '^x'}", attr("a", Match, "'^x'", "^x")},

		// Every kind of value, keeping how it was written.
		{"{a = null}", attr("a", Equal, "null", nil)},
		{"{a = true}", attr("a", Equal, "true", true)},
		{"{a != false}", attr("a", NotEqual, "false", false)},
		{"{a = -1.50e+3}", attr("a", Equal, "-1.50e+3", json.Number("-1.50e+3"))},
		{"{a = 0}", attr("a", Equal, "0", json.Number("0"))},
		{`{a = "double"}`, attr("a", Equal, `"double"`, "double")},
		{`{a = 'it\'s'}`, attr("a", Equal, `'it\'s'`, "it's")},
		{`{a = "say \"hi\""}`, attr("a", Equal, `"say \"hi\""`, `say "hi"`)},
		{`{a = 'tab\there\né\\\/'}`, attr("a", Equal, `'tab\there\né\\\/'`, "tab\there\né\\/")},
		{`{a = '"'}`, attr("a", Equal, `'"'`, `"`)},
		{`{a = 'café'}`, attr("a", Equal, `'café'`, "café")},
	}

	for _, tt := range tests {
		if got := mustParse(t, tt.text); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Parse(%q) = %#v, want %#v", tt.text, got, tt.want)
		}
	}
}

func TestStringRoundTrip(t *testing.T) {
	tests := []struct {
		text string
		want string
	}{
		{"{state != 'closed'}", "{state != 'closed'}"},
		{"{ state!='closed' }", "{state != 'closed'}"},
		{`{a = "x"}   and   not   {b>=1.50}`, `{a = "x"} and not {b >= 1.50}`},
		{"(({a = 1}))", "(({a = 1}))"},
		{"({a = 1} or {b = 2}) and not ({c = 3} and {d = null})", "({a = 1} or {b = 2}) and not ({c = 3} and {d = null})"},
		// Strings follow JSON escaping, so a regular expression's backslashes
		// are doubled.
		{`{a ~ '(?i)^in\\s'}`, `{a ~ '(?i)^in\\s'}`},
		{`{a = 'it\'s'}`, `{a = 'it\'s'}`},
		{"{ @author='a' } or {@dateCreated<'2026'}", "{@author = 'a'} or {@dateCreated < '2026'}"},
	}

	for _, tt := range tests {
		node := mustParse(t, tt.text)

		if got := node.String(); got != tt.want {
			t.Errorf("Parse(%q).String() = %q, want %q", tt.text, got, tt.want)
		}

		// Printing and parsing again gives the same tree.
		if again := mustParse(t, node.String()); !reflect.DeepEqual(again, node) {
			t.Errorf("reparsing %q gave %#v, want %#v", node.String(), again, node)
		}
	}
}

func TestStringOfBuiltTrees(t *testing.T) {
	a := attr("a", Equal, "1", json.Number("1"))
	b := attr("b", Equal, "2", json.Number("2"))
	c := attr("c", Equal, "3", json.Number("3"))

	// Trees built in code, without recorded parentheses, still print text
	// that parses to the same meaning.
	tests := []struct {
		node Node
		want string
	}{
		{&AndExpression{LeftExpression: &OrExpression{LeftExpression: a, RightExpression: b}, RightExpression: c}, "({a = 1} or {b = 2}) and {c = 3}"},
		{&NotExpression{Expression: &AndExpression{LeftExpression: a, RightExpression: b}}, "not ({a = 1} and {b = 2})"},
		{&OrExpression{LeftExpression: a, RightExpression: &AndExpression{LeftExpression: b, RightExpression: c}}, "{a = 1} or {b = 2} and {c = 3}"},
		// A value without its original text is printed as JSON.
		{&AttributeExpression{Attribute: "a", Operator: Equal, Value: Value{Data: "x"}}, `{a = "x"}`},
	}

	for _, tt := range tests {
		if got := tt.node.String(); got != tt.want {
			t.Errorf("String() = %q, want %q", got, tt.want)
		}
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		text       string
		start, end int
		message    string
	}{
		{"", 0, 0, "expected a filter expression"},
		{"   ", 3, 3, "expected a filter expression"},
		{"state = 'open'", 0, 5, "expected an attribute expression"},
		{"android", 0, 7, "expected an attribute expression"},
		{"{state}", 6, 7, "expected an operator"},
		{"{state = }", 9, 10, "expected a value"},
		{"{state == 'open'}", 8, 9, "expected a value"},
		{"{state = Open}", 9, 14, "expected a value"},
		{"{state = True}", 9, 14, "expected a value"},
		{"{state = 'open'", 0, 1, "this brace is never closed"},
		{"{state = 'open' 'x'}", 16, 20, "expected a closing brace"},
		{"{state = 'open}", 9, 10, "this string is never closed"},
		{"{rel:parent = 1}", 4, 5, "can't contain colons"},
		{"{:x = 1}", 1, 2, "can't contain colons"},
		{"{st ate = 1}", 4, 7, "expected an operator"},
		{"{a$ = 1}", 2, 3, "may only contain letters"},
		{"{ = 1}", 2, 3, "expected an attribute name"},
		{"{@ = 1}", 1, 2, "expected a field name: @title, @author, @dateCreated, @comments.text, @comments.author, and @comments.dateCreated"},
		{"{@@title = 1}", 1, 2, "expected a field name"},
		{"{@state = 'open'}", 1, 7, "unknown field @state; the fields are @title, @author, @dateCreated, @comments.text, @comments.author, and @comments.dateCreated"},
		{"{@Title = 'x'}", 1, 7, "unknown field @Title"},
		{"{@title@ = 'x'}", 7, 8, "may only contain letters"},
		{"{a@b = 1}", 2, 3, "may only contain letters"},
		{"({a = 1}", 0, 1, "this parenthesis is never closed"},
		{"({a = 1} {b = 2})", 9, 11, "expected and, or, or a closing parenthesis"},
		{"{a = 1} {b = 2}", 8, 10, "expected and, or, or the end"},
		{"{a = 1})", 7, 8, "expected and, or, or the end"},
		{"{a = 1} andx {b = 2}", 8, 12, "expected and, or, or the end"},
		{"{a = 1} and", 11, 11, "expected an attribute expression"},
		{"not", 3, 3, "expected an attribute expression"},
		{"{a = 12ab}", 5, 10, "expected a number"},
		{"{a = 01}", 5, 8, "expected a number"},
		{"{a = -}", 5, 7, "expected a number"},
		{`{a = 'x\q'}`, 7, 9, `unknown escape \q`},
		{`{a = 'x\u12'}`, 7, 13, `expected four hex digits`},
		{"{a = 'x\ny'}", 7, 8, "control characters"},
		{"{a ~ 1}", 5, 6, "~ needs a regular expression"},
		{"{a = /x/}", 5, 8, "a regular expression literal can only follow ~"},
		{"{a != /x/}", 6, 9, "a regular expression literal can only follow ~"},
		{"{a ~ /x}", 5, 6, "this regular expression is never closed"},
		{`{a ~ /x\/}`, 5, 6, "this regular expression is never closed"},
		{"{a ~ /(/}", 5, 8, "invalid regular expression"},
		{"{a ~ /x/i}", 8, 10, "expected a closing brace"},
		{"{a ~ /x\ty/}", 7, 8, "control characters"},
		{"{a ~ '('}", 5, 8, "invalid regular expression"},
		{"{a < null}", 5, 9, "null can only be compared with = or !="},
		{"{a >= true}", 6, 10, "true can only be compared with = or !="},
		{"{a ~ false}", 5, 10, "false can only be compared with = or !="},
		// Positions count runes, not bytes.
		{"{a = 'é'} xx", 10, 12, "expected and, or, or the end"},
	}

	for _, tt := range tests {
		_, err := Parse(tt.text)

		var syntaxErr *SyntaxError
		if err == nil {
			t.Errorf("Parse(%q) error = nil, want %q", tt.text, tt.message)
			continue
		}

		syntaxErr, ok := err.(*SyntaxError)
		if !ok {
			t.Errorf("Parse(%q) error = %T, want *SyntaxError", tt.text, err)
			continue
		}

		if syntaxErr.Start != tt.start || syntaxErr.End != tt.end || !strings.Contains(syntaxErr.Message, tt.message) {
			t.Errorf("Parse(%q) error = [%d, %d) %q, want [%d, %d) containing %q",
				tt.text, syntaxErr.Start, syntaxErr.End, syntaxErr.Message, tt.start, tt.end, tt.message)
		}
	}
}

func TestRegexpLiterals(t *testing.T) {
	tests := []struct {
		text string
		want Regexp
	}{
		// The text between the slashes is taken literally, so backslashes
		// reach the regular expression unchanged.
		{`{a ~ /^in\s/}`, `^in\s`},
		{`{a ~ /\bfoo\b/}`, `\bfoo\b`},
		{`{a ~ /(?i)^OPEN$/}`, `(?i)^OPEN$`},
		{`{a ~ /it's "quoted"/}`, `it's "quoted"`},
		// \/ stands for a slash, as in JavaScript.
		{`{a ~ /a\/b/}`, `a/b`},
		// \\ is kept for the regular expression, so a slash after it closes
		// the literal.
		{`{a ~ /a\\/}`, `a\\`},
		{`{a ~ //}`, ``},
	}

	for _, tt := range tests {
		node := mustParse(t, tt.text).(*AttributeExpression)

		if node.Value.Data != tt.want {
			t.Errorf("Parse(%q) pattern = %q, want %q", tt.text, node.Value.Data, tt.want)
		}

		// The literal prints back as it was written.
		if node.String() != tt.text {
			t.Errorf("Parse(%q).String() = %q", tt.text, node.String())
		}
	}

	// A literal created without its text prints with its slashes escaped.
	built := &AttributeExpression{Attribute: "a", Operator: Match, Value: Value{Data: Regexp("a/b")}}

	if got := built.String(); got != `{a ~ /a\/b/}` {
		t.Errorf("String() = %q, want %q", got, `{a ~ /a\/b/}`)
	}
	if again := mustParse(t, built.String()).(*AttributeExpression); again.Value.Data != Regexp("a/b") {
		t.Errorf("reparsed pattern = %q, want %q", again.Value.Data, "a/b")
	}
}

func TestStringsAfterMatch(t *testing.T) {
	// Quoted strings still work with ~, with JSON escaping, so a backslash
	// for the regular expression is doubled.
	if got := mustParse(t, `{a ~ '^in\\s'}`).(*AttributeExpression).Value.Data; got != `^in\s` {
		t.Errorf("regular expression = %q, want %q", got, `^in\s`)
	}
}

func TestSyntaxErrorMessage(t *testing.T) {
	_, err := Parse("{state = }")

	if got := err.Error(); !strings.HasPrefix(got, "column 10: expected a value") {
		t.Errorf("Error() = %q, want the 1-based column and the message", got)
	}
}

func TestParseCommentFields(t *testing.T) {
	for _, text := range []string{
		"{@comments.text ~ /TODO/}",
		"{@comments.author = ('Bob' :user)}",
		"{@comments.dateCreated > ('-1w' :fromnow)}",
		"{@comments.text = null}",
	} {
		if got := mustParse(t, text).String(); got != text {
			t.Errorf("String() = %q, want %q", got, text)
		}
	}

	if _, err := Parse("{@comments.body ~ 'x'}"); err == nil || !strings.Contains(err.Error(), "unknown field @comments.body") {
		t.Errorf("Parse() of an unknown comment field error = %v, want one naming it", err)
	}
}
