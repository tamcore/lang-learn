package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteJSON(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, map[string]string{"hello": "world"})

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Empty(t, env.Error)

	data, ok := env.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "world", data["hello"])
}

func TestWriteJSON_NilData(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusNoContent, nil)

	assert.Equal(t, http.StatusNoContent, rec.Code)

	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Nil(t, env.Data)
	assert.Empty(t, env.Error)
}

func TestWriteError(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	writeError(rec, http.StatusBadRequest, "invalid input")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Equal(t, "invalid input", env.Error)
	assert.Nil(t, env.Data)
}

func TestReadJSON(t *testing.T) {
	t.Parallel()

	type payload struct {
		Name string `json:"name"`
	}
	body := strings.NewReader(`{"name":"test"}`)
	r := httptest.NewRequest(http.MethodPost, "/", body)

	var p payload
	require.NoError(t, readJSON(r, &p))
	assert.Equal(t, "test", p.Name)
}

func TestReadJSON_UnknownField(t *testing.T) {
	t.Parallel()

	type payload struct {
		Name string `json:"name"`
	}
	body := strings.NewReader(`{"name":"test","extra":"bad"}`)
	r := httptest.NewRequest(http.MethodPost, "/", body)

	var p payload
	err := readJSON(r, &p)
	assert.Error(t, err)
}
