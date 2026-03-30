package auth_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/lang-learn/internal/auth"
)

const testSecret = "super-secret-key-for-testing-only"

func TestIssueToken_ValidClaims(t *testing.T) {
	token, err := auth.IssueToken(testSecret, "user-123", false, 15*time.Minute)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := auth.Verify(testSecret, token)
	require.NoError(t, err)
	assert.Equal(t, "user-123", claims.UserID)
	assert.False(t, claims.IsAdmin)
}

func TestIssueToken_AdminClaim(t *testing.T) {
	token, err := auth.IssueToken(testSecret, "admin-1", true, 15*time.Minute)
	require.NoError(t, err)

	claims, err := auth.Verify(testSecret, token)
	require.NoError(t, err)
	assert.Equal(t, "admin-1", claims.UserID)
	assert.True(t, claims.IsAdmin)
}

func TestVerify_ExpiredToken(t *testing.T) {
	// Issue a token that expires in the past.
	token, err := auth.IssueToken(testSecret, "user-123", false, -1*time.Second)
	require.NoError(t, err)

	_, err = auth.Verify(testSecret, token)
	assert.ErrorIs(t, err, auth.ErrTokenExpired)
}

func TestVerify_TamperedSignature(t *testing.T) {
	token, err := auth.IssueToken(testSecret, "user-123", false, 15*time.Minute)
	require.NoError(t, err)

	_, err = auth.Verify(testSecret, token+"x")
	assert.Error(t, err)
	assert.NotErrorIs(t, err, auth.ErrTokenExpired)
}

func TestVerify_WrongSecret(t *testing.T) {
	token, err := auth.IssueToken(testSecret, "user-123", false, 15*time.Minute)
	require.NoError(t, err)

	_, err = auth.Verify("wrong-secret", token)
	assert.Error(t, err)
}

func TestVerify_EmptyToken(t *testing.T) {
	_, err := auth.Verify(testSecret, "")
	assert.Error(t, err)
}

func TestVerify_MalformedToken(t *testing.T) {
	_, err := auth.Verify(testSecret, "not.a.jwt.token")
	assert.Error(t, err)
}

func TestVerify_NonHMACAlgorithm(t *testing.T) {
	// Generate an RSA key and sign a valid RS256 token.
	// Our Verify function rejects non-HMAC algorithms.
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": "user-1",
		"exp": time.Now().Add(15 * time.Minute).Unix(),
	})
	signed, err := token.SignedString(privateKey)
	require.NoError(t, err)

	_, err = auth.Verify(testSecret, signed)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, auth.ErrTokenExpired)
}

func TestIssueToken_DifferentTTLs(t *testing.T) {
	tests := []struct {
		name string
		ttl  time.Duration
	}{
		{"short", 1 * time.Minute},
		{"access", 15 * time.Minute},
		{"refresh", 7 * 24 * time.Hour},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token, err := auth.IssueToken(testSecret, "user-1", false, tc.ttl)
			require.NoError(t, err)

			claims, err := auth.Verify(testSecret, token)
			require.NoError(t, err)
			assert.Equal(t, "user-1", claims.UserID)
		})
	}
}
