package quiz

import (
	"os"
	"testing"
)

func TestGenerateQuestions(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping OpenRouter integration test in short mode")
	}

	apiKey := os.Getenv("ONEPIECE_OPENROUTER_API_KEY")
	if apiKey == "" {
		t.Skip("ONEPIECE_OPENROUTER_API_KEY not set")
	}

	// Uses the production model: free OpenRouter models proved too unreliable
	// with structured output and reasoning, and a run costs well under a cent.
	g := NewGenerator(apiKey)

	episodes := []EpisodeSource{
		{
			Number: 1,
			Title:  "I'm Luffy! The Man Who Will Become the Pirate King!",
			Description: "Monkey D. Luffy, a young boy who ate the Gum-Gum Devil Fruit and gained the properties " +
				"of rubber, sets off to sea to find the legendary One Piece treasure and become the Pirate King. " +
				"He is rescued by the pirate Shanks who loses his arm saving Luffy, inspiring Luffy's dream.",
		},
		{
			Number: 2,
			Title:  "Enter the Great Swordsman! Pirate Hunter Roronoa Zoro!",
			Description: "Luffy arrives at a Marine base and learns of the legendary pirate hunter Roronoa Zoro, " +
				"who is imprisoned in the base's courtyard. Luffy frees Zoro, who agrees to join his crew after " +
				"retrieving his three swords from the corrupt Marine Captain Morgan's son Helmeppo.",
		},
		{
			Number: 3,
			Title:  "Morgan versus Luffy! Who's the Strange Fellow?",
			Description: "Luffy and Zoro battle Marine Captain Axe-Hand Morgan and his men. Luffy defeats Morgan " +
				"while Zoro takes down Helmeppo. Coby decides to join the Marines, and Luffy's crew sets sail.",
		},
	}

	questions, err := g.GenerateQuestions(t.Context(), episodes, nil)
	if err != nil {
		t.Fatalf("GenerateQuestions: %v", err)
	}

	if err := validateQuestions(questions); err != nil {
		t.Fatalf("invalid questions from LLM: %v", err)
	}
}
