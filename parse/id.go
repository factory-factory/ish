package parse

import (
	"crypto/rand"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// idAlphabet is Crockford's base32 alphabet, which leaves out I, L, O, and U
// to avoid confusion with 1, 0, and V.
const idAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// idLength is the number of characters in an issue ID.
const idLength = 10

// maxIDAttempts limits how many IDs are tried before giving up on finding one
// that isn't in use.
const maxIDAttempts = 100

// NewIssueID returns a random issue ID that no issue file in dir uses.
func NewIssueID(dir string) (string, error) {
	return newIssueID(dir, rand.Reader)
}

func newIssueID(dir string, random io.Reader) (string, error) {
	b := make([]byte, idLength)

	for range maxIDAttempts {
		if _, err := io.ReadFull(random, b); err != nil {
			return "", err
		}

		// 256 is a multiple of 32, so each byte maps uniformly to a character.
		for i := range b {
			b[i] = idAlphabet[int(b[i])%len(idAlphabet)]
		}

		id := string(b)
		_, err := os.Stat(filepath.Join(dir, id+".md"))

		if errors.Is(err, os.ErrNotExist) {
			return id, nil
		}

		if err != nil {
			return "", err
		}
	}

	return "", errors.New("couldn't find an unused issue ID")
}
