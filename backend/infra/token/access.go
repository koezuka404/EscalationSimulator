package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type accessClaims struct {
	Ver int `json:"ver"`
	jwt.RegisteredClaims
}

func IssueAccess(secret []byte, issuer, audience, userID string, authVersion int, ttl time.Duration, now time.Time) (string, time.Time, error) {
	expiresAt := now.Add(ttl)
	claims := accessClaims{
		Ver: authVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

func NewSecretToken() (raw, hash string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(raw))
	return raw, hex.EncodeToString(sum[:]), nil
}
