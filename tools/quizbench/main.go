// Command quizbench compares OpenRouter models for quiz generation by running
// the production generator (pkg/quiz) against a fixed set of episodes.
//
// Usage:
//
//	ONEPIECE_OPENROUTER_API_KEY=... go run ./tools/quizbench [-runs 3] MODEL...
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/yardenshoham/onepiece/pkg/onepiecewiki"
	"github.com/yardenshoham/onepiece/pkg/quiz"
)

// episodes is the fixed benchmark input: the end of the Arlong Park arc.
var episodes = []quiz.EpisodeSource{
	{Number: 42, Title: "Explosion! Fishman Arlong's Fierce Assault from the Sea!"},
	{Number: 43, Title: "The End of the Fishman Empire! Nami's My Friend!"},
	{Number: 44, Title: "Setting Out with a Smile! Farewell, Hometown Cocoyashi Village!"},
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	runs := flag.Int("runs", 3, "runs per model")
	flag.Parse()
	models := flag.Args()
	if len(models) == 0 {
		return errors.New("usage: quizbench [-runs N] MODEL [MODEL...]")
	}
	apiKey := os.Getenv("ONEPIECE_OPENROUTER_API_KEY")
	if apiKey == "" {
		return errors.New("ONEPIECE_OPENROUTER_API_KEY not set")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	wiki := onepiecewiki.NewClient(slog.New(slog.DiscardHandler))
	for i := range episodes {
		desc, err := wiki.FetchLongDescription(ctx, episodes[i].Number)
		if err != nil {
			return fmt.Errorf("episode %d: %w", episodes[i].Number, err)
		}
		episodes[i].Description = desc
	}

	for _, m := range models {
		gen := quiz.NewGeneratorWithModel(apiKey, m)
		for i := 1; i <= *runs; i++ {
			start := time.Now()
			qs, err := gen.GenerateQuestions(ctx, episodes, nil)
			fmt.Fprintf(os.Stdout, "== %s run %d: %.1fs\n", m, i, time.Since(start).Seconds())
			if err != nil {
				fmt.Fprintf(os.Stdout, "   error: %v\n", err)
				continue
			}
			for _, q := range qs {
				fmt.Fprintf(os.Stdout, "   Q: %s\n      ✓ %s  ✗ %s\n", q.Question, q.CorrectOption, strings.Join(q.WrongOptions, " | "))
			}
		}
	}
	return nil
}
