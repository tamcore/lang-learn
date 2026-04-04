package generator

import "github.com/user/lang-learn/internal/models"

// Blueprints returns the built-in course generation blueprints.
func Blueprints() map[string]models.Blueprint {
	return map[string]models.Blueprint{
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
