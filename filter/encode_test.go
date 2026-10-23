package filter

import (
	"strings"
	"testing"
)

func TestEncodingFunctions(t *testing.T) {
	for _, tt := range []struct {
		text, want string
	}{
		// Base64, encoded with the standard alphabet and padding.
		{"('héllo <wörld>?' :base64.encode)", `"aMOpbGxvIDx3w7ZybGQ+Pw=="`},
		{"('' :base64.encode)", `""`},
		// Decoded from the standard or URL-safe alphabet.
		{"('aMOpbGxvIDx3w7ZybGQ+Pw==' :base64.decode)", `"héllo <wörld>?"`},
		{"('aMOpbGxvIDx3w7ZybGQ-Pw==' :base64.decode)", `"héllo <wörld>?"`},
		{"('x' :base64.encode :base64.decode)", `"x"`},
		// Text that isn't Base64, or isn't UTF-8 when decoded, is null.
		{"('not base64!' :base64.decode)", "null"},
		{"('/w==' :base64.decode)", "null"},

		// JSON, encoded compactly, without escaping for HTML.
		{"('a<b>' :json.encode)", `"\"a<b>\""`},
		{"(2.5 :json.encode)", `"2.5"`},
		{"(3 :json.encode)", `"3"`},
		{"(true :json.encode)", `"true"`},
		{"([1, 'x', [false]] :json.encode)", `"[1,\"x\",[false]]"`},
		{"(null :json.encode)", `"null"`},
		{"(.missing :json.encode)", `"null"`},
		// Decoded to the value it represents.
		{`('"a"' :json.decode)`, `"a"`},
		{"('2.5' :json.decode)", "2.5"},
		{"(' [1, true, null, \"x\"] ' :json.decode)", `[1,true,null,"x"]`},
		{"('null' :json.decode)", "null"},
		{"([1, 'x'] :json.encode :json.decode)", `[1,"x"]`},
		// Text that isn't JSON, or has an object in it, is null.
		{"('{' :json.decode)", "null"},
		{`('{"a": 1}' :json.decode)`, "null"},
		{`('[1, {"a": 1}]' :json.decode)`, "null"},
		{"('1 2' :json.decode)", "null"},
		{"('[1] x' :json.decode)", "null"},

		// Digests in lowercase hexadecimal.
		{"('abc' :hash.sha256)", `"ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"`},
		{"('abc' :hash.sha512)", `"ddaf35a193617abacc417349ae20413112e6fa4e89a97ea20a9eeee64b55d39a2192992a274fc1a836ba3c23a3feebbd454d4423643ce80e2a9ac94fa54ca49f"`},
		{"('abc' :hash.md5)", `"900150983cd24fb0d6963f7d28e17f72"`},
		{"('ABC' :lower :hash.md5)", `"900150983cd24fb0d6963f7d28e17f72"`},

		// Other functions give null for null.
		{"(null :base64.encode)", "null"},
		{"(.missing :hash.sha256)", "null"},
	} {
		if got, err := evalText(t, tt.text); err != nil || got != tt.want {
			t.Errorf("%s = %s, %v, want %s", tt.text, got, err, tt.want)
		}
	}
}

func TestEncodingTypeErrors(t *testing.T) {
	for _, tt := range []struct {
		text, message string
	}{
		{"(5 :base64.encode)", ":base64.encode needs a value that's a string, not a number"},
		{"([1] :base64.decode)", ":base64.decode needs a value that's a string, not a list"},
		{"(true :json.decode)", ":json.decode needs a value that's a string, not a boolean"},
		{"(1 :hash.sha256)", ":hash.sha256 needs a value that's a string, not a number"},
	} {
		if _, err := evalText(t, tt.text); err == nil || !strings.Contains(err.Error(), tt.message) {
			t.Errorf("%s error = %v, want one containing %q", tt.text, err, tt.message)
		}
	}
}

func TestFunctionNamesWithDots(t *testing.T) {
	if got := mustParse(t, "{a = ('x' :base64.encode :hash.md5)}").String(); got != "{a = ('x' :base64.encode :hash.md5)}" {
		t.Errorf("String() = %q", got)
	}

	for _, tt := range []struct {
		text, message string
	}{
		{"{a = ('x' :base64.nope)}", "unknown function :base64.nope"},
		{"{a = ('x' :base64)}", "unknown function :base64"},
		{"{a = ('x' :hash.md5('y'))}", ":hash.md5 takes no arguments"},
	} {
		if _, err := Parse(tt.text); err == nil || !strings.Contains(err.Error(), tt.message) {
			t.Errorf("Parse(%q) error = %v, want one containing %q", tt.text, err, tt.message)
		}
	}

	// A dot after a name, not followed by a letter, isn't part of it.
	if _, err := Parse("{a = ('x' :lower.)}"); err == nil {
		t.Error("Parse() with a trailing dot error = nil, want an error")
	}

	if got := sqlFunctionName("base64.encode"); got != "ish_base64_encode" {
		t.Errorf("sqlFunctionName() = %q, want ish_base64_encode", got)
	}
}
