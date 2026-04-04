package generator

import (
	"fmt"
	"strings"

	"github.com/user/lang-learn/internal/models"
)

// BuildLessonOutlinePrompt generates the LLM prompt that asks for lesson titles.
func BuildLessonOutlinePrompt(blueprint models.Blueprint, sourceLang, targetLang string, lessonCount int) string {
	scenes := make([]string, len(blueprint.Scenes))
	for i, s := range blueprint.Scenes {
		scenes[i] = fmt.Sprintf("- %s: %s (vocab: %s)", s.Title, s.Description, strings.Join(s.Vocabulary, ", "))
	}

	return fmt.Sprintf(`You are a language course designer using the Pimsleur method.

Create %d lesson titles for a %s → %s course using the "%s" blueprint.

Scenes:
%s

Rules:
- Each lesson builds on the previous one
- 3-5 new vocabulary items per lesson
- Begin each lesson with recall of previous key phrases

Respond ONLY with a JSON array of strings (lesson titles). No extra text.
Example: ["Lesson 1: Greetings", "Lesson 2: Introductions"]`, lessonCount, sourceLang, targetLang, blueprint.Name, strings.Join(scenes, "\n"))
}

// BuildLessonTurnsPrompt generates the prompt for creating turns within a lesson.
func BuildLessonTurnsPrompt(title, sourceLang, targetLang string, perspective models.Perspective, lessonNum int) string {
	return fmt.Sprintf(`You are a language course designer using the Pimsleur method.

Generate the turns for: "%s" (Lesson %d)
Language pair: %s → %s
Learner perspective: %s

Rules:
- 40-60 turns total
- Each turn is either "system" (instructor) or "user" (learner practice)
- System turns have audio; user turns are for practice (blurred text)
- Include translations in the source language (%s)
- Repeat new vocabulary ≥3 times with increasing gaps (spaced recall)
- If lesson > 1, begin with brief recall of previous lesson's key phrases
- Mix instruction, repetition, and practice naturally

Respond ONLY with a JSON array of objects:
[{"speaker":"system","text":"...","translation":"...","is_blurred":false,"spaced_repeat":false,"delay_after_ms":2000},
 {"speaker":"user","text":"...","translation":"...","is_blurred":true,"spaced_repeat":true,"delay_after_ms":5000}]

No extra text outside the JSON array.`, title, lessonNum, sourceLang, targetLang, perspective, sourceLang)
}
