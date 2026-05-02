// Package quiz generates trivia questions from One Piece episode summaries
// using the OpenRouter LLM API, and tracks quiz state in memory.
package quiz

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openrouter "github.com/OpenRouterTeam/go-sdk"
	"github.com/OpenRouterTeam/go-sdk/models/components"
	"github.com/OpenRouterTeam/go-sdk/optionalnullable"
)

const model = "google/gemini-2.5-flash"

const maxTokens = 1500

// EpisodeSource is the episode information passed to the generator.
type EpisodeSource struct {
	Number      int
	Title       string
	Description string // LongDescription when available, else Crunchyroll description
}

// RawQuestion is the per-question schema returned by the LLM.
type RawQuestion struct {
	Question      string   `json:"question"`
	CorrectOption string   `json:"correct_option"`
	WrongOptions  []string `json:"wrong_options"`
}

type rawResponse struct {
	Questions []RawQuestion `json:"questions"`
}

// Generator creates quiz questions via OpenRouter.
type Generator struct {
	client *openrouter.OpenRouter
	model  string
}

// NewGenerator returns a Generator that authenticates with the given API key.
func NewGenerator(apiKey string) *Generator {
	return &Generator{
		client: openrouter.New(
			openrouter.WithSecurity(apiKey),
		),
		model: model,
	}
}

// GenerateQuestions calls the LLM and returns exactly 3 validated questions.
// usedQuestions contains the text of previously answered questions to avoid
// repeating them.
func (g *Generator) GenerateQuestions(ctx context.Context, episodes []EpisodeSource, usedQuestions []string) ([]RawQuestion, error) {
	prompt := buildPrompt(episodes, usedQuestions)

	effort := components.EffortMedium
	trueVal := true
	maxT := int64(maxTokens)

	req := components.ChatRequest{
		Model: new(g.model),
		Messages: []components.ChatMessages{
			components.CreateChatMessagesSystem(components.ChatSystemMessage{
				Role:    components.ChatSystemMessageRoleSystem,
				Content: components.CreateChatSystemMessageContentStr("You are a One Piece trivia expert. Generate exactly 3 multiple-choice quiz questions in the requested JSON format. Keep every answer option brief — a few words or a short phrase, not full sentences. Use correct canonical spelling of all names and places (e.g. Alabasta, not Arabasta)."),
			}),
			components.CreateChatMessagesUser(components.ChatUserMessage{
				Role:    components.ChatUserMessageRoleUser,
				Content: components.CreateChatUserMessageContentStr(prompt),
			}),
		},
		MaxTokens: optionalnullable.From(&maxT),
		Reasoning: &components.Reasoning{
			Effort: optionalnullable.From(&effort),
		},
		Provider: optionalnullable.From(&components.ProviderPreferences{
			RequireParameters: optionalnullable.From(&trueVal),
		}),
		ResponseFormat: responseFormatPtr(components.ChatFormatJSONSchemaConfig{
			Type: components.ChatFormatJSONSchemaConfigTypeJSONSchema,
			JSONSchema: components.ChatJSONSchemaConfig{
				Name:   "quiz_questions",
				Schema: quizSchema(),
				Strict: optionalnullable.From(&trueVal),
			},
		}),
	}

	resp, err := g.client.Chat.Send(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("OpenRouter request: %w", err)
	}

	if resp == nil || resp.ChatResult == nil {
		return nil, fmt.Errorf("unexpected nil response from OpenRouter")
	}

	choices := resp.ChatResult.GetChoices()
	if len(choices) == 0 {
		return nil, fmt.Errorf("OpenRouter returned no choices")
	}

	content, ok := choices[0].Message.Content.Get()
	if !ok {
		return nil, fmt.Errorf("OpenRouter response has no content")
	}
	if content.Str == nil {
		return nil, fmt.Errorf("OpenRouter response content is not a string")
	}

	var raw rawResponse
	if err := json.Unmarshal([]byte(*content.Str), &raw); err != nil {
		return nil, fmt.Errorf("parsing LLM response JSON: %w", err)
	}

	if err := validateQuestions(raw.Questions); err != nil {
		return nil, fmt.Errorf("invalid questions from LLM: %w", err)
	}

	return raw.Questions, nil
}

// validateQuestions checks that exactly 3 questions were returned and each has
// a question text, a correct option, and exactly 3 wrong options.
func validateQuestions(qs []RawQuestion) error {
	if len(qs) != 3 {
		return fmt.Errorf("expected 3 questions, got %d", len(qs))
	}
	for i, q := range qs {
		if q.Question == "" {
			return fmt.Errorf("question %d has empty text", i)
		}
		if q.CorrectOption == "" {
			return fmt.Errorf("question %d has empty correct_option", i)
		}
		if len(q.WrongOptions) != 3 {
			return fmt.Errorf("question %d has %d wrong_options, expected 3", i, len(q.WrongOptions))
		}
	}
	return nil
}

// buildPrompt constructs the user prompt from episode sources.
func buildPrompt(episodes []EpisodeSource, usedQuestions []string) string {
	var lines strings.Builder
	lines.WriteString("Generate 3 multiple-choice trivia questions about the following One Piece episodes. " +
		"Each question must have exactly 1 correct answer and 3 wrong answers. " +
		"Focus on specific plot events, character actions, and details from the summaries.\n\n")

	for _, ep := range episodes {
		fmt.Fprintf(&lines, "### Episode %d: %s\n%s\n\n", ep.Number, ep.Title, ep.Description)
	}

	if len(usedQuestions) > 0 {
		lines.WriteString("Avoid asking questions similar to these already-asked questions:\n")
		for _, q := range usedQuestions {
			fmt.Fprintf(&lines, "- %s\n", q)
		}
		lines.WriteString("\n")
	}

	return lines.String()
}

// quizSchema returns the JSON Schema map for structured output.
func quizSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"questions": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"question":       map[string]any{"type": "string"},
						"correct_option": map[string]any{"type": "string"},
						"wrong_options": map[string]any{
							"type":     "array",
							"items":    map[string]any{"type": "string"},
							"minItems": 3,
							"maxItems": 3,
						},
					},
					"required":             []string{"question", "correct_option", "wrong_options"},
					"additionalProperties": false,
				},
				"minItems": 3,
				"maxItems": 3,
			},
		},
		"required":             []string{"questions"},
		"additionalProperties": false,
	}
}

func responseFormatPtr(cfg components.ChatFormatJSONSchemaConfig) *components.ResponseFormat {
	v := components.CreateResponseFormatJSONSchema(cfg)
	return &v
}
