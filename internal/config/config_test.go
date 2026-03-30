package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/lang-learn/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-value")
	t.Setenv("DATA_DIR", "")
	t.Setenv("PORT", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("OPENROUTER_API_KEY", "")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "/data", cfg.DataDir)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, 15*time.Minute, cfg.AccessTokenTTL)
	assert.Equal(t, 7*24*time.Hour, cfg.RefreshTokenTTL)
	assert.Equal(t, 12, cfg.BcryptCost)
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("JWT_SECRET", "super-secret")
	t.Setenv("DATA_DIR", "/tmp/mydata")
	t.Setenv("PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("OPENROUTER_API_KEY", "sk-openrouter-key")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "super-secret", cfg.JWTSecret)
	assert.Equal(t, "/tmp/mydata", cfg.DataDir)
	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "sk-openrouter-key", cfg.OpenRouterAPIKey)
}

func TestLoad_MissingJWTSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}

func TestLoad_JWTSecretRequired(t *testing.T) {
	t.Setenv("JWT_SECRET", "   ")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}
