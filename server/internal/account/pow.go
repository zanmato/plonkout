package account

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/zanmato/plonkout/server/internal/account/accountdb"
	"github.com/zanmato/plonkout/server/internal/platform/api"
)

// A proof of work in the style of ALTCHA. The server picks a secret number,
// publishes the hash of salt+number and an HMAC over it, and the browser finds
// the number by brute force. It costs a person a second of a background worker
// and a script the same second per attempt, which is enough to make bulk
// signups pointless without keeping any rate limit state.
const (
	powAlgorithm = "SHA-256"
	// powMaxNumber sets the work. A desktop browser tries about 160 000 hashes
	// a second, a phone a fraction of that. Solving starts when the signup page
	// opens, so it runs while the person types a username.
	powMaxNumber = 300_000
	powTTL       = 10 * time.Minute
)

// PowChallenge is what the browser solves.
type PowChallenge struct {
	Algorithm string `json:"algorithm" example:"SHA-256"`
	Challenge string `json:"challenge" doc:"Hex SHA-256 of salt followed by the secret number."`
	MaxNumber int    `json:"maxNumber" doc:"The secret number is between 0 and this, inclusive."`
	Salt      string `json:"salt"`
	Signature string `json:"signature" doc:"Proves the server issued the challenge."`
}

// PowSolution is a solved challenge.
type PowSolution struct {
	Challenge string `json:"challenge"`
	Number    int    `json:"number" minimum:"0"`
	Salt      string `json:"salt"`
	Signature string `json:"signature"`
}

type pow struct {
	key []byte
	q   *accountdb.Queries
	now func() time.Time
}

func (p *pow) create() (PowChallenge, error) {
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return PowChallenge{}, err
	}
	secret, err := rand.Int(rand.Reader, big.NewInt(powMaxNumber+1))
	if err != nil {
		return PowChallenge{}, err
	}

	salt := hex.EncodeToString(nonce) + "?expires=" + strconv.FormatInt(p.now().Add(powTTL).Unix(), 10)
	challenge := powHash(salt, int(secret.Int64()))
	return PowChallenge{
		Algorithm: powAlgorithm,
		Challenge: challenge,
		MaxNumber: powMaxNumber,
		Salt:      salt,
		Signature: p.sign(challenge),
	}, nil
}

// verify checks a solution and spends its salt, so it cannot be used twice.
func (p *pow) verify(ctx context.Context, q *accountdb.Queries, s PowSolution) error {
	refused := api.Errorf(http.StatusUnprocessableEntity, "signup_refused", "the signup could not be verified, reload and try again")

	_, query, found := strings.Cut(s.Salt, "?")
	if !found {
		return refused
	}
	params, err := url.ParseQuery(query)
	if err != nil {
		return refused
	}
	expiresUnix, err := strconv.ParseInt(params.Get("expires"), 10, 64)
	if err != nil {
		return refused
	}
	expires := time.Unix(expiresUnix, 0)
	if !p.now().Before(expires) {
		return refused
	}
	if !hmac.Equal([]byte(p.sign(s.Challenge)), []byte(s.Signature)) {
		return refused
	}
	if powHash(s.Salt, s.Number) != s.Challenge {
		return refused
	}

	saltHash := sha256.Sum256([]byte(s.Salt))
	spent, err := q.SpendPow(ctx, accountdb.SpendPowParams{SaltHash: saltHash[:], ExpiresAt: expires})
	if err != nil {
		return fmt.Errorf("spend proof of work: %w", err)
	}
	if spent == 0 {
		return refused
	}
	return nil
}

func (p *pow) sign(challenge string) string {
	mac := hmac.New(sha256.New, p.key)
	mac.Write([]byte(challenge))
	return hex.EncodeToString(mac.Sum(nil))
}

func powHash(salt string, number int) string {
	sum := sha256.Sum256([]byte(salt + strconv.Itoa(number)))
	return hex.EncodeToString(sum[:])
}
