package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain", `["a","b"]`, `["a","b"]`},
		{"with fences", "```json\n[\"a\"]\n```", `["a"]`},
		{"with backticks only", "```\n{\"x\":1}\n```", `{"x":1}`},
		{"whitespace", "  [1,2]  ", `[1,2]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, cleanJSON(tt.input))
		})
	}
}

func TestBuildLessonOutlinePrompt(t *testing.T) {
	t.Parallel()
	bp := Blueprints()["travel-basics-v1"]
	prompt := BuildLessonOutlinePrompt(bp, "sk", "en", 3)
	assert.Contains(t, prompt, "sk")
	assert.Contains(t, prompt, "en")
	assert.Contains(t, prompt, "3 lesson titles")
}

func TestBuildLessonTurnsPrompt(t *testing.T) {
	t.Parallel()
	prompt := BuildLessonTurnsPrompt("Greetings", "sk", "en", "male", 1)
	assert.Contains(t, prompt, "Greetings")
	assert.Contains(t, prompt, "sk")
	assert.Contains(t, prompt, "male")
}

func TestBlueprintsContainsExpectedKeys(t *testing.T) {
	t.Parallel()
	bp := Blueprints()
	assert.Contains(t, bp, "travel-basics-v1")
	assert.Contains(t, bp, "restaurant-v1")
	assert.Contains(t, bp, "directions-v1")
}
