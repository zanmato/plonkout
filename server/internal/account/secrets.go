package account

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/zanmato/plonkout/server/internal/platform/api"
)

// recoveryCodeCount is how many one time codes a user holds.
const recoveryCodeCount = 10

// Crockford style alphabet without the letters people misread.
var recoveryEncoding = base32.NewEncoding("0123456789abcdefghjkmnpqrstvwxyz").WithPadding(base32.NoPadding)

// newToken returns a random token for a cookie and its hash for the database.
// Only the hash is stored, so a leaked table cannot be replayed.
func newToken() (token string, hash []byte, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	return token, hashToken(token), nil
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// newRecoveryCodes returns codes formatted for people, e.g. "k3m9x-2pq7r".
func newRecoveryCodes() ([]string, error) {
	codes := make([]string, recoveryCodeCount)
	for i := range codes {
		raw := make([]byte, 7)
		if _, err := rand.Read(raw); err != nil {
			return nil, err
		}
		encoded := recoveryEncoding.EncodeToString(raw)[:10]
		codes[i] = encoded[:5] + "-" + encoded[5:]
	}
	return codes, nil
}

// hashRecoveryCode normalizes what a person typed and hashes it.
func hashRecoveryCode(code string) []byte {
	normalized := strings.ToLower(strings.NewReplacer("-", "", " ", "").Replace(code))
	normalized = strings.NewReplacer("o", "0", "i", "1", "l", "1").Replace(normalized)
	return hashToken(normalized)
}

// deriveKey derives a purpose specific key from the configured secret, so one
// secret can sign several kinds of thing without one forging another.
func deriveKey(secret, purpose string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(purpose))
	return mac.Sum(nil)
}

func encodeCredentialID(id []byte) string {
	return base64.RawURLEncoding.EncodeToString(id)
}

func decodeCredentialID(id string) ([]byte, error) {
	raw, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil || len(raw) == 0 {
		return nil, fmt.Errorf("%w: no such passkey", api.ErrNotFound)
	}
	return raw, nil
}

// uniqueAs replaces a unique violation with a specific problem, and passes
// every other error through.
func uniqueAs(err error, problem *api.Problem) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return problem
	}
	return err
}
