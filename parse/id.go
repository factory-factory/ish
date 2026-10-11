package parse

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// idAlphabet is Crockford's base32 alphabet, which leaves out I, L, O, and U
// to avoid confusion with 1, 0, and V.
const idAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// An issue ID is a timestamp and random characters, joined by idSeparator.
// The timestamp is the number of minutes since idEpoch in base32, most
// significant digit first and padded with zeros, so IDs sort in the order
// they were created.
const (
	idTimeLength   = 5
	idSeparator    = "-"
	idRandomLength = 5
	idLength       = idTimeLength + len(idSeparator) + idRandomLength
)

// idEpoch is the time ID timestamps count minutes from. Five base32 digits
// count about 64 years of minutes from it, until 2083.
var idEpoch = time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)

// maxIDAttempts limits how many IDs are tried before giving up on finding one
// that isn't in use.
const maxIDAttempts = 100

// NewIssueID returns an issue ID for an issue created now that no issue file
// in dir uses.
func NewIssueID(dir string) (string, error) {
	return newIssueID(dir, time.Now(), rand.Reader)
}

func newIssueID(dir string, created time.Time, random io.Reader) (string, error) {
	timestamp, err := idTimestamp(created)

	if err != nil {
		return "", err
	}

	b := make([]byte, idRandomLength)

	for range maxIDAttempts {
		if _, err := io.ReadFull(random, b); err != nil {
			return "", err
		}

		// 256 is a multiple of 32, so each byte maps uniformly to a character.
		for i := range b {
			b[i] = idAlphabet[int(b[i])%len(idAlphabet)]
		}

		id := timestamp + idSeparator + string(b)
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

// idTimestamp returns the timestamp part of the ID of an issue created at t:
// the number of whole minutes since idEpoch in base32.
func idTimestamp(t time.Time) (string, error) {
	minutes := int64(t.Sub(idEpoch) / time.Minute)
	limit := int64(1) << (5 * idTimeLength)

	if t.Before(idEpoch) || minutes >= limit {
		return "", fmt.Errorf("can't create an issue ID for %s: the time must be from %s until %s",
			t.Format(time.RFC3339), idEpoch.Format(time.RFC3339),
			idEpoch.Add(time.Duration(limit)*time.Minute).Format(time.RFC3339))
	}

	b := make([]byte, idTimeLength)

	for i := idTimeLength - 1; i >= 0; i-- {
		b[i] = idAlphabet[minutes%32]
		minutes /= 32
	}

	return string(b), nil
}
