# Implement encoding / decoding filter expression functions

["Brendan Berg" <brendan@berg.industries> 2026-10-22T21:32:14-07:00]

- state: in review
- assignees: <brendan@berg.industries>
- relation.parent: #3CZWN-EBVEH

---

["Brendan Berg" <brendan@berg.industries> 2026-10-22T21:32:14-07:00]

Implement functions for Base64 and JSON encoding and decoding.

`:base64.encode` Takes a UTF-8 encoded string and returns a Base64-encoded string as a response.

`:base64.decode` Takes a Base64-encoded string and returns a decoded UTF-8 string.

`:json.encode` Takes a string, number, boolean, or array and encodes it as a JSON string.

`:json.decode` Takes a JSON string and returns the string, number, boolean, or array it represents. If the string cannot be parsed, the result is `NULL`.

`:hash.sha256`, `:hash.sha512`, `:hash.md5` Takes a UTF-8 string and returns the hash digest as calculated by the selected algorithm.

---

[Claude <noreply@anthropic.com> 2026-10-22T21:59:30-07:00]

I've reviewed this and moved it to `in progress`. Below is my plan, then some questions, each with what I'd suggest. I'll start once they're answered.

## Plan

- **The functions:** these are expression functions, like those from #3D97C-F62P7, evaluated in SQL by functions written in Go. They take no arguments, so they chain as in `(.user.email :lower :hash.sha256)`.
- **Names with dots:** function names can currently only have letters and digits. I'd allow dots between the parts of a name, as in `:base64.encode`.
- **Types:** each function takes a string, except `:json.encode`. A value of another type is a type error, and `null` gives `null`, as with the other functions.

## Questions

1. **Which Base64:** I'd suggest the standard alphabet with `=` padding (RFC 4648, section 4), for both encoding and decoding. Should decoding also accept the URL-safe alphabet, or unpadded text?
2. **Invalid input to `:base64.decode`:** what if the text isn't valid Base64, or decodes to bytes that aren't valid UTF-8? I'd suggest `null` for both, as for `:json.decode`.
3. **JSON objects:** filter expressions have no table type, so what should `:json.decode` give for a JSON object, such as `{"a": 1}`? I'd suggest `null`, as for text that can't be parsed. An array containing an object would be `null` too.
4. **How `:json.encode` writes:** I'd suggest compact JSON, without spaces.
   - Numbers would be written as elsewhere, such as `3` and `2.5`.
   - Characters such as `<`, `>` and `&` would be left as they are, rather than escaped for HTML.
   - `null` would be encoded as `"null"`, rather than giving `null`.

   Is that all right?
5. **Digests:** should they be lowercase hexadecimal, as `sha256sum` writes them? I'd suggest yes. Base64 is the other common form.

---

["Brendan Berg" <brendan@berg.industries> 2026-10-22T22:01:00-07:00]

1. Base64 encode should use RFC 4648 with `=` padding, but decode should accept RFC 4648 as well as the URL-safe alphabet.
2. Invalid input to `:base64.decode` should return `NULL`, similar to `:json.decode`.
3. Yes, returning `NULL` when decoding a JSON string with objects makes sense.
4. Your suggestion makes sense.
5. Hash function digests should be lowercase hexadecimal.

---

[Claude <noreply@anthropic.com> 2026-10-22T22:06:15-07:00]

Done, following your answers, and moved to `in review`.

## The functions

- **`:base64.encode`:** this encodes a string's UTF-8 bytes in Base64, with the standard alphabet and `=` padding (RFC 4648, section 4).
- **`:base64.decode`:** this decodes Base64 in the standard or the URL-safe alphabet, with padding. Text that isn't Base64, or that decodes to bytes that aren't valid UTF-8, gives `null`.
- **`:json.encode`:** this encodes a string, number, boolean or list as compact JSON text.
  - Numbers are written as elsewhere, such as `3` and `2.5`.
  - `<`, `>` and `&` are left as they are.
  - `null` encodes as the text `null`. It's the only function that's given `null` rather than giving `null` for it.
- **`:json.decode`:** this decodes JSON text to the string, number, boolean, list or `null` it represents. Text that isn't JSON gives `null`, including text after the value, as in `[1] x`. So does JSON with an object anywhere in it.
- **The hashes:** `:hash.sha256`, `:hash.sha512` and `:hash.md5` give the lowercase hexadecimal digest of a string's UTF-8 bytes.
- **Types:** a value of the wrong type is a type error. For example, `:hash.sha256 needs a value that's a string, not a number`. Apart from `:json.encode`, they give `null` for `null`.

## Names with dots

- **In filters:** function names can now have dots between their parts, as in `:base64.encode`. An unknown name, such as `:base64.nope`, is reported as before.
- **In SQL:** names can't contain dots, so the SQL functions use underscores, as in `ish_base64_encode`.

## Testing

- **Base64:** encoding, decoding either alphabet, round trips, empty text, invalid Base64 and invalid UTF-8.
- **JSON:**
  - encoding each type, including lists and `null`, without escaping for HTML
  - decoding each type and round trips
  - text that isn't JSON, objects at the top and inside lists, and trailing values or text
- **Hashes:** SHA-256, SHA-512 and MD5 digests of `abc`, compared with known values.
- **Type errors and `null`:** for each kind of function.
- **Names with dots:** printing, unknown names, arguments and trailing dots.
- **Against a database:** decoded Base64 and JSON compared with attributes, and invalid JSON as UNKNOWN in an `or`. This runs the SQL functions with underscores in SQLite.
