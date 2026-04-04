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

Scenes (in progressive order — earlier scenes must be covered in earlier lessons):
%s

Pimsleur method rules:
- Each lesson builds on ALL previous lessons
- Start with simple sounds and cognates, progress to full sentences
- 3-5 new vocabulary items per lesson
- Every lesson after the first must begin with recall of key phrases from previous lessons
- Use spaced repetition: revisit vocabulary from 2-3 lessons ago
- The progression is: single words → short phrases → questions → full sentences → dialogues
- Lesson titles should reflect the primary new content introduced

Respond ONLY with a JSON array of strings (lesson titles). No extra text.
Example: ["Lesson 1: First Sounds", "Lesson 2: Core Words"]`, lessonCount, sourceLang, targetLang, blueprint.Name, strings.Join(scenes, "\n"))
}

// BuildLessonTurnsPrompt generates the prompt for creating turns within a lesson.
func BuildLessonTurnsPrompt(title, sourceLang, targetLang string, perspective models.Perspective, lessonNum int) string {
	recallInstruction := ""
	if lessonNum > 1 {
		recallInstruction = fmt.Sprintf(`
- This is lesson %d. Begin with 3-5 recall turns testing vocabulary from previous lessons.
- Recall turns should ask the learner to translate phrases they learned earlier.`, lessonNum)
	}

	return fmt.Sprintf(`You are a language course designer using the Pimsleur method.

Generate the dialogue turns for: "%s" (Lesson %d)
The learner speaks %s and is learning %s.
Learner perspective: %s

Pimsleur method rules:
- 40-60 turns total
- Each turn is either "system" (instructor speaks/demonstrates) or "user" (learner practices)
- The instructor speaks in BOTH languages: gives instructions/translations in %s, demonstrates target phrases in %s
- System turns: instructor says a word/phrase in the target language, then explains in source language
- User turns: instructor asks learner to say something — text field contains what the learner should say in the TARGET language, translation is the source-language meaning
- is_blurred=true means the learner should try to recall the phrase before seeing it
- For new vocabulary: introduce the target word, give its meaning, have the learner repeat it 3+ times
- Spaced repetition: reintroduce each new word after 1, 3, 5, and 10 turns with increasing gaps
- delay_after_ms: 2000-3000 for system turns, 4000-6000 for user practice turns (time to think/speak)
- Mix instruction, demonstration, repetition, and free practice naturally%s

IMPORTANT: All "text" fields for system turns should be in %s (the target language being taught).
All "text" fields for user turns should be in %s (what the learner should say in the target language).
All "translation" fields should be in %s (the learner's native language).

Respond ONLY with a JSON array of objects:
[{"speaker":"system","text":"...","translation":"...","is_blurred":false,"spaced_repeat":false,"delay_after_ms":2000},
 {"speaker":"user","text":"...","translation":"...","is_blurred":true,"spaced_repeat":true,"delay_after_ms":5000}]

No extra text outside the JSON array.`, title, lessonNum, sourceLang, targetLang, perspective,
		sourceLang, targetLang,
		recallInstruction,
		targetLang, targetLang, sourceLang)
}
