// Package auth provides JWT issuance and verification for the lang-learn API.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrTokenExpired is returned when the token has passed its expiry time.
var ErrTokenExpired = errors.New("token expired")

// ErrTokenInvalid is returned for all other token verification failures
// (wrong secret, malformed, unsupported algorithm, etc.).
var ErrTokenInvalid = errors.New("token invalid")

// Claims holds the user-facing fields extracted from a verified JWT.
type Claims struct {
	UserID  string
	IsAdmin bool
}

// jwtClaims is the internal JWT payload structure.
type jwtClaims struct {
	IsAdmin bool `json:"is_admin"`
	jwt.RegisteredClaims
}

// IssueToken signs an HS256 JWT containing userID and isAdmin, expiring after ttl.
func IssueToken(secret, userID string, isAdmin bool, ttl time.Duration) (string, error) {
	now := time.Now()
	c := jwtClaims{
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// Verify parses and validates a signed HS256 JWT, returning the embedded claims.
// Returns ErrTokenExpired when the token has expired; ErrTokenInvalid otherwise.
func Verify(secret, tokenStr string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %s", ErrTokenInvalid, err.Error())
	}

	c := parsed.Claims.(*jwtClaims)

	return &Claims{
		UserID:  c.Subject,
		IsAdmin: c.IsAdmin,
	}, nil
}
