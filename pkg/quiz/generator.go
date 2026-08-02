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

const model = "openai/gpt-5.6-luna"

const maxTokens = 1500

const systemPrompt = "You are a One Piece trivia expert. Generate exactly 3 multiple-choice quiz questions in the requested JSON format.\n\n" +
	"Rules:\n" +
	"- IMPORTANT: Base every question AND every answer option ONLY on information from the episode summaries provided. Do not reference characters, places, events, or outcomes that appear later in the series \u2014 the person answering may not have watched beyond these episodes.\n" +
	"- Focus on memorable, significant moments: major character decisions, important plot turning points, key reveals, and defining actions \u2014 not obscure details or exact dialogue.\n" +
	"- Questions should be answerable by someone who watched the episode a while ago and remembers the highlights, not someone reading a transcript.\n" +
	"- Each question must be grammatically unambiguous: the correct answer must directly and sensibly answer the question as written.\n" +
	"- Do not embed the answer inside the question text.\n" +
	"- Keep every answer option brief \u2014 a few words or a short phrase, not full sentences.\n" +
	"- Wrong options should be plausible alternatives drawn from the provided summaries (e.g. other characters, locations, or outcomes that appear in those episodes) \u2014 not from later in the series.\n" +
	"- All four answer options must be the same type of thing: if the correct answer is a character name, all options must be character names; if it is a location, all must be locations; if it is an action or outcome, all must be actions or outcomes. A player should not be able to eliminate options by noticing they are a different category.\n" +
	"- When referring to characters, use their anime name rather than their manga name (e.g. prefer the Funimation/Crunchyroll anime romanisation). If the anime name is unknown, the manga name is acceptable.\n" +
	"- Before finalising a question, verify: does each wrong option clearly not answer the question, and does the correct option clearly and uniquely answer it?"

const userPromptHeader = "Generate 3 multiple-choice trivia questions about the following One Piece episodes.\n" +
	"Each question must have exactly 1 correct answer and 3 wrong answers.\n" +
	"IMPORTANT: Only use information from the summaries below \u2014 do not reference anything from later in the series to avoid spoilers.\n" +
	"Focus on the most memorable and significant moments from the summaries \u2014 things that would stick in a viewer's memory.\n" +
	"Avoid obscure details, minor background events, or exact dialogue.\n\n"

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

// NewGeneratorWithModel returns a Generator that uses the given OpenRouter model ID.
func NewGeneratorWithModel(apiKey, model string) *Generator {
	return &Generator{
		client: openrouter.New(
			openrouter.WithSecurity(apiKey),
		),
		model: model,
	}
}

// NewGenerator returns a Generator that authenticates with the given API key.
func NewGenerator(apiKey string) *Generator {
	return NewGeneratorWithModel(apiKey, model)
}

const maxAttempts = 3

// GenerateQuestions calls the LLM and returns exactly 3 validated questions.
// usedQuestions contains the text of previously answered questions to avoid
// repeating them. It retries up to maxAttempts times when the LLM returns
// invalid output.
func (g *Generator) GenerateQuestions(ctx context.Context, episodes []EpisodeSource, usedQuestions []string) ([]RawQuestion, error) {
	prompt := buildPrompt(episodes, usedQuestions)

	effort := components.ChatRequestEffortMedium
	trueVal := true
	maxT := int64(maxTokens)

	req := components.ChatRequest{
		Model: new(g.model),
		Messages: []components.ChatMessages{
			components.CreateChatMessagesSystem(components.ChatSystemMessage{
				Role:    components.ChatSystemMessageRoleSystem,
				Content: components.CreateChatSystemMessageContentStr(systemPrompt),
			}),
			components.CreateChatMessagesUser(components.ChatUserMessage{
				Role:    components.ChatUserMessageRoleUser,
				Content: components.CreateChatUserMessageContentStr(prompt),
			}),
		},
		MaxTokens: optionalnullable.From(&maxT),
		Reasoning: &components.ChatRequestReasoning{
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

	var lastErr error
	for range maxAttempts {
		questions, err := g.sendRequest(ctx, req)
		if err == nil {
			return questions, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (g *Generator) sendRequest(ctx context.Context, req components.ChatRequest) ([]RawQuestion, error) {
	resp, err := g.client.Chat.Send(ctx, req, nil)
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
	if !ok || content == nil {
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
	lines.WriteString(userPromptHeader)

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
