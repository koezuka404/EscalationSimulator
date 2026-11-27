package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AccessTokens struct {
	secret   []byte
	issuer   string
	audience string
	ttl      time.Duration
}

func NewAccessTokens(secret []byte, issuer, audience string, ttl time.Duration) *AccessTokens {
	return &AccessTokens{secret: secret, issuer: issuer, audience: audience, ttl: ttl}
}

type accessClaims struct {
	Ver int `json:"ver"`
	jwt.RegisteredClaims
}

//ログイン用トークンを発行する
func (t *AccessTokens) Issue(userID string, authVersion int, now time.Time) (string, time.Time, error) {
	expiresAt := now.Add(t.ttl)
	claims := accessClaims{
		Ver: authVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    t.issuer,
			Audience:  jwt.ClaimStrings{t.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

//ログイン用トークンから利用者IDと認証の版を取る
func (t *AccessTokens) Parse(raw string) (string, int, error) {
	claims := &accessClaims{}
	parsed, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("署名の方式が違います")
		}
		return t.secret, nil
	}, jwt.WithIssuer(t.issuer), jwt.WithAudience(t.audience), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || parsed == nil || !parsed.Valid || claims.Subject == "" {
		return "", 0, err
	}
	return claims.Subject, claims.Ver, nil
}

type Secrets struct{}

//再ログイン用の印と、そのハッシュを作る
func (Secrets) New() (raw, hash string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	return raw, Secrets{}.Hash(raw), nil
}

//印のハッシュを返す
func (Secrets) Hash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
