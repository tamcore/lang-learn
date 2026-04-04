package generator

import "github.com/user/lang-learn/internal/models"

// Blueprints returns the built-in course generation blueprints.
// The pimsleur-complete-v1 blueprint follows the real Pimsleur 10-unit
// progressive arc extracted from official course materials.
func Blueprints() map[string]models.Blueprint {
	return map[string]models.Blueprint{
		"pimsleur-complete-v1": {
			ID:          "pimsleur-complete-v1",
			Name:        "Pimsleur Complete",
			Description: "Full 10-unit Pimsleur-style course: sounds → words → phrases → sentences → conversation",
			Scenes: []models.Scene{
				{ID: "basic-sounds", Title: "Basic Sounds & Cognates", Description: "Simple syllables, cognates, basic vowels and consonants of the target language", Vocabulary: []string{"yes", "no", "hello", "map", "lamp", "mama", "papa"}},
				{ID: "core-vocabulary", Title: "Core Vocabulary", Description: "Common nouns, pronouns (I, you, he, she), to be, basic sentence parts", Vocabulary: []string{"I", "you", "he", "she", "am", "are", "this", "that", "here", "there"}},
				{ID: "first-sentences", Title: "First Sentences", Description: "What is this? I am not. Please. Simple questions and negation", Vocabulary: []string{"what", "is", "not", "please", "where", "who", "home"}},
				{ID: "greetings-survival", Title: "Greetings & Survival Phrases", Description: "Hello/good day, how are you, I understand, I don't understand, basic food words", Vocabulary: []string{"good day", "how are you", "understand", "don't understand", "slowly", "beer", "salad", "fish"}},
				{ID: "restaurant", Title: "At a Restaurant", Description: "Ordering food and drinks, with milk/sugar, give me the menu please", Vocabulary: []string{"menu", "give me", "with milk", "with sugar", "ice cream", "bread", "a little"}},
				{ID: "shopping", Title: "Shopping & Transactions", Description: "Do you have...? Give me one. How much does it cost? Write it down for me.", Vocabulary: []string{"do you have", "how much", "costs", "one", "write", "post office", "coffee"}},
				{ID: "identity", Title: "Identity & People", Description: "Nationalities, family members, how do you say...?, at the restaurant", Vocabulary: []string{"American", "English person", "husband", "wife", "how do you say", "key", "well/good"}},
				{ID: "places-culture", Title: "Places & Culture", Description: "Cities, rivers, everyday objects, compound words, reading complex sounds", Vocabulary: []string{"river", "city", "star", "lunch", "sentence", "entrance", "dog"}},
				{ID: "numbers", Title: "Numbers & Counting", Description: "Numbers 1-12, zero, basic math, city, country", Vocabulary: []string{"one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten", "eleven", "twelve", "zero"}},
				{ID: "review-conversation", Title: "Review & Conversation", Description: "Integrating all previous vocabulary into practical multi-turn dialogues", Vocabulary: []string{"excuse me", "can you help me", "I would like", "thank you", "goodbye"}},
			},
		},
		"travel-basics-v1": {
			ID:          "travel-basics-v1",
			Name:        "Travel Basics",
			Description: "Greetings, introductions, asking for help, basic numbers",
			Scenes: []models.Scene{
				{ID: "greetings", Title: "Greetings", Description: "Hello, goodbye, please, thank you", Vocabulary: []string{"hello", "goodbye", "please", "thank you", "yes", "no"}},
				{ID: "introductions", Title: "Introductions", Description: "My name is, where are you from, I speak", Vocabulary: []string{"name", "from", "speak", "country", "language"}},
				{ID: "help", Title: "Asking for Help", Description: "Excuse me, can you help me, I don't understand", Vocabulary: []string{"excuse me", "help", "understand", "repeat", "slowly"}},
			},
		},
		"restaurant-v1": {
			ID:          "restaurant-v1",
			Name:        "At the Restaurant",
			Description: "Ordering food and drinks, paying the bill",
			Scenes: []models.Scene{
				{ID: "ordering", Title: "Ordering", Description: "I would like, the menu, water, beer, wine", Vocabulary: []string{"menu", "water", "beer", "wine", "food", "order"}},
				{ID: "paying", Title: "Paying", Description: "The bill, how much, credit card, cash", Vocabulary: []string{"bill", "pay", "price", "card", "cash", "tip"}},
			},
		},
		"directions-v1": {
			ID:          "directions-v1",
			Name:        "Directions",
			Description: "Asking for and giving directions",
			Scenes: []models.Scene{
				{ID: "asking", Title: "Asking Directions", Description: "Where is, how do I get to, left, right, straight", Vocabulary: []string{"where", "left", "right", "straight", "turn", "stop"}},
			},
		},
	}
}
