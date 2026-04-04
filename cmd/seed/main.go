// Command seed creates the initial users and seed courses.
package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/user/lang-learn/internal/models"
	"github.com/user/lang-learn/internal/store"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "/data"
	}

	ctx := context.Background()

	users, err := store.NewFileUserStore(filepath.Join(dataDir, "users"))
	if err != nil {
		slog.Error("user store", "err", err)
		os.Exit(1)
	}

	courses, err := store.NewFileCourseStore(filepath.Join(dataDir, "courses"))
	if err != nil {
		slog.Error("course store", "err", err)
		os.Exit(1)
	}

	audit, err := store.NewFileAuditStore(filepath.Join(dataDir, "audit"))
	if err != nil {
		slog.Error("audit store", "err", err)
		os.Exit(1)
	}

	now := time.Now().UTC()

	// Seed users
	philippHash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), 12)
	jankaHash, _ := bcrypt.GenerateFromPassword([]byte("user123"), 12)

	seedUsers := []models.User{
		{ID: "philipp", Username: "philipp", Email: "philipp@lang-learn.local", PasswordHash: string(philippHash), IsAdmin: true, CreatedAt: now, UpdatedAt: now},
		{ID: "janka", Username: "janka", Email: "janka@lang-learn.local", PasswordHash: string(jankaHash), IsAdmin: false, CreatedAt: now, UpdatedAt: now},
	}

	for _, u := range seedUsers {
		if err := users.Create(ctx, u); err != nil {
			slog.Warn("user exists", "user", u.Username, "err", err)
		} else {
			slog.Info("created user", "username", u.Username, "admin", u.IsAdmin)
		}
	}

	// Seed course pairs
	seedCourses := buildSeedCourses(now)
	for _, c := range seedCourses {
		if err := courses.Create(ctx, c); err != nil {
			slog.Warn("course exists", "course", c.ID, "err", err)
		} else {
			slog.Info("created course", "id", c.ID, "title", c.Title)
		}
	}

	// Audit entries for seeding
	for _, c := range seedCourses {
		_ = audit.Append(ctx, models.AuditEntry{
			ID: "seed-" + c.ID, Timestamp: now, Action: models.ActionCourseGenerated,
			ActorID: "philipp", TargetID: c.ID, TargetType: "course",
			Meta: map[string]any{"source": c.SourceLang, "target": c.TargetLang, "seeded": true},
		})
	}

	slog.Info("seed complete")
}

func buildSeedCourses(now time.Time) []models.Course {
	return []models.Course{
		buildCourse("sk-en-forward", "Slovak → English (Travel Basics)", "Learn essential English phrases from Slovak", "sk", "en", models.DirectionForward, now,
			[]lessonDef{
				{"Greetings & Introductions", []turnDef{
					{S, "Dobrý deň! Vitajte na prvej lekcii.", "Hello! Welcome to the first lesson.", false},
					{S, "Dnes sa naučíte základné pozdravy v angličtine.", "Today you will learn basic greetings in English.", false},
					{S, "Hello.", "Ahoj.", false},
					{U, "Hello.", "Ahoj.", true},
					{S, "Good morning.", "Dobré ráno.", false},
					{U, "Good morning.", "Dobré ráno.", true},
					{S, "My name is...", "Volám sa...", false},
					{U, "My name is...", "Volám sa...", true},
					{S, "How are you?", "Ako sa máte?", false},
					{U, "How are you?", "Ako sa máte?", true},
					{S, "I'm fine, thank you.", "Mám sa dobre, ďakujem.", false},
					{U, "I'm fine, thank you.", "Mám sa dobre, ďakujem.", true},
					{S, "Nice to meet you.", "Teší ma.", false},
					{U, "Nice to meet you.", "Teší ma.", true},
					{S, "Goodbye.", "Dovidenia.", false},
					{U, "Goodbye.", "Dovidenia.", true},
				}},
				{"Asking for Help", []turnDef{
					{S, "Dobrý deň! Poďme si zopakovať z minulej lekcie.", "Hello! Let's review from the last lesson.", false},
					{U, "Hello. How are you?", "Ahoj. Ako sa máte?", true},
					{S, "Výborne! Dnes sa naučíme ako požiadať o pomoc.", "Excellent! Today we'll learn how to ask for help.", false},
					{S, "Excuse me.", "Prepáčte.", false},
					{U, "Excuse me.", "Prepáčte.", true},
					{S, "Can you help me?", "Môžete mi pomôcť?", false},
					{U, "Can you help me?", "Môžete mi pomôcť?", true},
					{S, "I don't understand.", "Nerozumiem.", false},
					{U, "I don't understand.", "Nerozumiem.", true},
					{S, "Could you repeat that?", "Mohli by ste to zopakovať?", false},
					{U, "Could you repeat that?", "Mohli by ste to zopakovať?", true},
					{S, "Please speak slowly.", "Prosím, hovorte pomaly.", false},
					{U, "Please speak slowly.", "Prosím, hovorte pomaly.", true},
					{S, "Where is...?", "Kde je...?", false},
					{U, "Where is...?", "Kde je...?", true},
				}},
				{"Numbers & Basics", []turnDef{
					{S, "Poďme si zopakovať – Excuse me, can you help me?", "Let's review – Excuse me, can you help me?", false},
					{U, "Excuse me, can you help me?", "Prepáčte, môžete mi pomôcť?", true},
					{S, "Dnes sa naučíme čísla a základné frázy.", "Today we'll learn numbers and basic phrases.", false},
					{S, "One, two, three.", "Jeden, dva, tri.", false},
					{U, "One, two, three.", "Jeden, dva, tri.", true},
					{S, "Four, five.", "Štyri, päť.", false},
					{U, "Four, five.", "Štyri, päť.", true},
					{S, "How much is this?", "Koľko to stojí?", false},
					{U, "How much is this?", "Koľko to stojí?", true},
					{S, "Yes.", "Áno.", false},
					{U, "Yes.", "Áno.", true},
					{S, "No.", "Nie.", false},
					{U, "No.", "Nie.", true},
					{S, "Please.", "Prosím.", false},
					{U, "Please.", "Prosím.", true},
					{S, "Thank you.", "Ďakujem.", false},
					{U, "Thank you.", "Ďakujem.", true},
				}},
			},
		),
		buildCourse("en-sk-reverse", "English → Slovak (Travel Basics)", "Learn essential Slovak phrases from English", "en", "sk", models.DirectionReverse, now,
			[]lessonDef{
				{"Pozdravy a Predstavenie", []turnDef{
					{S, "Hello! Welcome to your first Slovak lesson.", "Ahoj! Vitajte na prvej lekcii slovenčiny.", false},
					{S, "Today you'll learn basic Slovak greetings.", "Dnes sa naučíte základné slovenské pozdravy.", false},
					{S, "Ahoj.", "Hello.", false},
					{U, "Ahoj.", "Hello.", true},
					{S, "Dobrý deň.", "Good day.", false},
					{U, "Dobrý deň.", "Good day.", true},
					{S, "Ako sa máte?", "How are you?", false},
					{U, "Ako sa máte?", "How are you?", true},
					{S, "Ďakujem, dobre.", "Thank you, well.", false},
					{U, "Ďakujem, dobre.", "Thank you, well.", true},
					{S, "Volám sa...", "My name is...", false},
					{U, "Volám sa...", "My name is...", true},
					{S, "Teší ma.", "Nice to meet you.", false},
					{U, "Teší ma.", "Nice to meet you.", true},
					{S, "Dovidenia.", "Goodbye.", false},
					{U, "Dovidenia.", "Goodbye.", true},
				}},
				{"Prosím o Pomoc", []turnDef{
					{S, "Let's review: Ahoj, ako sa máte?", "Poďme si zopakovať: Hello, how are you?", false},
					{U, "Ahoj, ako sa máte?", "Hello, how are you?", true},
					{S, "Great! Now let's learn to ask for help in Slovak.", "Výborne! Teraz sa naučíme požiadať o pomoc po slovensky.", false},
					{S, "Prepáčte.", "Excuse me.", false},
					{U, "Prepáčte.", "Excuse me.", true},
					{S, "Môžete mi pomôcť?", "Can you help me?", false},
					{U, "Môžete mi pomôcť?", "Can you help me?", true},
					{S, "Nerozumiem.", "I don't understand.", false},
					{U, "Nerozumiem.", "I don't understand.", true},
					{S, "Hovorte pomaly, prosím.", "Speak slowly, please.", false},
					{U, "Hovorte pomaly, prosím.", "Speak slowly, please.", true},
					{S, "Kde je...?", "Where is...?", false},
					{U, "Kde je...?", "Where is...?", true},
				}},
				{"Čísla a Základy", []turnDef{
					{S, "Review: Prepáčte, môžete mi pomôcť?", "Review: Excuse me, can you help me?", false},
					{U, "Prepáčte, môžete mi pomôcť?", "Excuse me, can you help me?", true},
					{S, "Now let's learn Slovak numbers.", "Teraz sa naučíme slovenské čísla.", false},
					{S, "Jeden, dva, tri.", "One, two, three.", false},
					{U, "Jeden, dva, tri.", "One, two, three.", true},
					{S, "Štyri, päť.", "Four, five.", false},
					{U, "Štyri, päť.", "Four, five.", true},
					{S, "Koľko to stojí?", "How much is this?", false},
					{U, "Koľko to stojí?", "How much is this?", true},
					{S, "Áno.", "Yes.", false},
					{U, "Áno.", "Yes.", true},
					{S, "Nie.", "No.", false},
					{U, "Nie.", "No.", true},
					{S, "Prosím.", "Please.", false},
					{U, "Prosím.", "Please.", true},
					{S, "Ďakujem.", "Thank you.", false},
					{U, "Ďakujem.", "Thank you.", true},
				}},
			},
		),
		buildCourse("en-de-forward", "English → German (Travel Basics)", "Learn essential German phrases from English", "en", "de", models.DirectionForward, now,
			[]lessonDef{
				{"Greetings & Introductions", []turnDef{
					{S, "Welcome to your first German lesson!", "Willkommen zu Ihrer ersten Deutschstunde!", false},
					{S, "Today we'll learn basic German greetings.", "Heute lernen wir grundlegende deutsche Begrüßungen.", false},
					{S, "Hallo.", "Hello.", false},
					{U, "Hallo.", "Hello.", true},
					{S, "Guten Morgen.", "Good morning.", false},
					{U, "Guten Morgen.", "Good morning.", true},
					{S, "Wie heißen Sie?", "What is your name?", false},
					{U, "Wie heißen Sie?", "What is your name?", true},
					{S, "Ich heiße...", "My name is...", false},
					{U, "Ich heiße...", "My name is...", true},
					{S, "Wie geht es Ihnen?", "How are you?", false},
					{U, "Wie geht es Ihnen?", "How are you?", true},
					{S, "Gut, danke.", "Good, thank you.", false},
					{U, "Gut, danke.", "Good, thank you.", true},
					{S, "Freut mich.", "Nice to meet you.", false},
					{U, "Freut mich.", "Nice to meet you.", true},
					{S, "Auf Wiedersehen.", "Goodbye.", false},
					{U, "Auf Wiedersehen.", "Goodbye.", true},
				}},
				{"Asking for Help", []turnDef{
					{S, "Let's review: Hallo, wie geht es Ihnen?", "Review: Hello, how are you?", false},
					{U, "Hallo, wie geht es Ihnen?", "Hello, how are you?", true},
					{S, "Today we'll learn to ask for help in German.", "Heute lernen wir, auf Deutsch um Hilfe zu bitten.", false},
					{S, "Entschuldigung.", "Excuse me.", false},
					{U, "Entschuldigung.", "Excuse me.", true},
					{S, "Können Sie mir helfen?", "Can you help me?", false},
					{U, "Können Sie mir helfen?", "Can you help me?", true},
					{S, "Ich verstehe nicht.", "I don't understand.", false},
					{U, "Ich verstehe nicht.", "I don't understand.", true},
					{S, "Können Sie das wiederholen?", "Could you repeat that?", false},
					{U, "Können Sie das wiederholen?", "Could you repeat that?", true},
					{S, "Sprechen Sie langsam, bitte.", "Speak slowly, please.", false},
					{U, "Sprechen Sie langsam, bitte.", "Speak slowly, please.", true},
					{S, "Wo ist...?", "Where is...?", false},
					{U, "Wo ist...?", "Where is...?", true},
				}},
				{"Numbers & Basics", []turnDef{
					{S, "Review: Entschuldigung, können Sie mir helfen?", "Review: Excuse me, can you help me?", false},
					{U, "Entschuldigung, können Sie mir helfen?", "Excuse me, can you help me?", true},
					{S, "Today: German numbers and basic phrases.", "Heute: Deutsche Zahlen und grundlegende Phrasen.", false},
					{S, "Eins, zwei, drei.", "One, two, three.", false},
					{U, "Eins, zwei, drei.", "One, two, three.", true},
					{S, "Vier, fünf.", "Four, five.", false},
					{U, "Vier, fünf.", "Four, five.", true},
					{S, "Was kostet das?", "How much is this?", false},
					{U, "Was kostet das?", "How much is this?", true},
					{S, "Ja.", "Yes.", false},
					{U, "Ja.", "Yes.", true},
					{S, "Nein.", "No.", false},
					{U, "Nein.", "No.", true},
					{S, "Bitte.", "Please.", false},
					{U, "Bitte.", "Please.", true},
					{S, "Danke.", "Thank you.", false},
					{U, "Danke.", "Thank you.", true},
				}},
			},
		),
		buildCourse("de-en-reverse", "German → English (Travel Basics)", "Learn essential English phrases from German", "de", "en", models.DirectionReverse, now,
			[]lessonDef{
				{"Begrüßungen & Vorstellungen", []turnDef{
					{S, "Willkommen! Heute lernen Sie englische Begrüßungen.", "Welcome! Today you'll learn English greetings.", false},
					{S, "Hello.", "Hallo.", false},
					{U, "Hello.", "Hallo.", true},
					{S, "Good morning.", "Guten Morgen.", false},
					{U, "Good morning.", "Guten Morgen.", true},
					{S, "What is your name?", "Wie heißen Sie?", false},
					{U, "What is your name?", "Wie heißen Sie?", true},
					{S, "My name is...", "Ich heiße...", false},
					{U, "My name is...", "Ich heiße...", true},
					{S, "How are you?", "Wie geht es Ihnen?", false},
					{U, "How are you?", "Wie geht es Ihnen?", true},
					{S, "Good, thank you.", "Gut, danke.", false},
					{U, "Good, thank you.", "Gut, danke.", true},
					{S, "Nice to meet you.", "Freut mich.", false},
					{U, "Nice to meet you.", "Freut mich.", true},
					{S, "Goodbye.", "Auf Wiedersehen.", false},
					{U, "Goodbye.", "Auf Wiedersehen.", true},
				}},
				{"Um Hilfe bitten", []turnDef{
					{S, "Wiederholung: Hello, how are you?", "Review: Hello, how are you?", false},
					{U, "Hello, how are you?", "Hallo, wie geht es Ihnen?", true},
					{S, "Heute lernen Sie, auf Englisch um Hilfe zu bitten.", "Today you'll learn to ask for help in English.", false},
					{S, "Excuse me.", "Entschuldigung.", false},
					{U, "Excuse me.", "Entschuldigung.", true},
					{S, "Can you help me?", "Können Sie mir helfen?", false},
					{U, "Can you help me?", "Können Sie mir helfen?", true},
					{S, "I don't understand.", "Ich verstehe nicht.", false},
					{U, "I don't understand.", "Ich verstehe nicht.", true},
					{S, "Could you repeat that?", "Können Sie das wiederholen?", false},
					{U, "Could you repeat that?", "Können Sie das wiederholen?", true},
					{S, "Please speak slowly.", "Sprechen Sie langsam, bitte.", false},
					{U, "Please speak slowly.", "Sprechen Sie langsam, bitte.", true},
					{S, "Where is...?", "Wo ist...?", false},
					{U, "Where is...?", "Wo ist...?", true},
				}},
				{"Zahlen & Grundlagen", []turnDef{
					{S, "Wiederholung: Excuse me, can you help me?", "Review: Entschuldigung, können Sie mir helfen?", false},
					{U, "Excuse me, can you help me?", "Entschuldigung, können Sie mir helfen?", true},
					{S, "Heute: Englische Zahlen und grundlegende Phrasen.", "Today: English numbers and basic phrases.", false},
					{S, "One, two, three.", "Eins, zwei, drei.", false},
					{U, "One, two, three.", "Eins, zwei, drei.", true},
					{S, "Four, five.", "Vier, fünf.", false},
					{U, "Four, five.", "Vier, fünf.", true},
					{S, "How much is this?", "Was kostet das?", false},
					{U, "How much is this?", "Was kostet das?", true},
					{S, "Yes.", "Ja.", false},
					{U, "Yes.", "Ja.", true},
					{S, "No.", "Nein.", false},
					{U, "No.", "Nein.", true},
					{S, "Please.", "Bitte.", false},
					{U, "Please.", "Bitte.", true},
					{S, "Thank you.", "Danke.", false},
					{U, "Thank you.", "Danke.", true},
				}},
			},
		),
	}
}

const (
	S = models.SpeakerSystem
	U = models.SpeakerUser
)

type turnDef struct {
	speaker     models.TurnSpeaker
	text        string
	translation string
	blurred     bool
}

type lessonDef struct {
	title string
	turns []turnDef
}

func buildCourse(id, title, desc, src, tgt string, dir models.CourseDirection, now time.Time, lessons []lessonDef) models.Course {
	c := models.Course{
		ID:          id,
		Title:       title,
		Description: desc,
		SourceLang:  src,
		TargetLang:  tgt,
		Direction:   dir,
		Perspective: models.PerspectiveMale,
		BlueprintID: "travel-basics-v1",
		LessonCount: len(lessons),
		CreatedAt:   now,
		GeneratedAt: now,
		GeneratedBy: "philipp",
	}

	for i, ld := range lessons {
		lessonID := id + "-L" + string(rune('1'+i))
		lesson := models.Lesson{
			ID:        lessonID,
			CourseID:  id,
			Sequence:  i + 1,
			Title:     ld.title,
			CreatedAt: now,
		}
		for j, td := range ld.turns {
			lesson.Turns = append(lesson.Turns, models.Turn{
				ID:           lessonID + "-T" + itoa(j+1),
				Sequence:     j + 1,
				Speaker:      td.speaker,
				Text:         td.text,
				Translation:  td.translation,
				AudioFile:    "",
				IsBlurred:    td.speaker == models.SpeakerUser,
				SpacedRepeat: td.speaker == models.SpeakerUser,
				DelayAfterMs: 2000,
			})
		}
		c.Lessons = append(c.Lessons, lesson)
	}
	return c
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}
