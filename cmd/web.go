package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/yardenshoham/onepiece/internal/web"
	"github.com/yardenshoham/onepiece/pkg/crunchyroll"
	"github.com/yardenshoham/onepiece/pkg/onepiecewiki"
	"github.com/yardenshoham/onepiece/pkg/poller"
	"github.com/yardenshoham/onepiece/pkg/quiz"
	"github.com/yardenshoham/onepiece/pkg/tracker"
)

// webFlagEnv pairs each web flag with the environment variable that supplies
// its fallback value.
var webFlagEnv = []struct{ flag, env string }{
	{"email", "ONEPIECE_CR_EMAIL"},
	{"password", "ONEPIECE_CR_PASSWORD"},
	{"addr", "ONEPIECE_ADDR"},
	{"poll-interval", "ONEPIECE_POLL_INTERVAL"},
	{"healthcheck-uuid", "ONEPIECE_HEALTHCHECK_UUID"},
	{"posthog-key", "ONEPIECE_POSTHOG_KEY"},
	{"posthog-host", "ONEPIECE_POSTHOG_HOST"},
	{"openrouter-key", "ONEPIECE_OPENROUTER_API_KEY"},
}

// resolveFlagsFromEnv fills in every flag that was not passed on the command
// line from its environment variable, letting pflag parse the value.
func resolveFlagsFromEnv(flags *pflag.FlagSet) error {
	for _, fe := range webFlagEnv {
		if flags.Changed(fe.flag) {
			continue
		}
		value := os.Getenv(fe.env)
		if value == "" {
			continue
		}
		if err := flags.Set(fe.flag, value); err != nil {
			return fmt.Errorf("invalid %s: %w", fe.env, err)
		}
	}
	return nil
}

func newWebCmd() *cobra.Command {
	var (
		email           string
		password        string
		addr            string
		pollInterval    time.Duration
		healthcheckUUID string
		posthogKey      string
		posthogHost     string
		openrouterKey   string
	)

	cmd := &cobra.Command{
		Use:   "web",
		Short: "Start the One Piece tracker web server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			logger := cmd.Context().Value(loggerKey{}).(*slog.Logger)

			// Resolve flags from env if not set
			if err := resolveFlagsFromEnv(cmd.Flags()); err != nil {
				return fmt.Errorf("resolving flags from environment: %w", err)
			}
			if email == "" || password == "" {
				return fmt.Errorf("email and password are required (use --email/--password flags or ONEPIECE_CR_EMAIL/ONEPIECE_CR_PASSWORD env vars)")
			}

			// Setup signal-based context
			ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer cancel()

			// Create Crunchyroll client
			logger.Info("connecting to Crunchyroll")
			client, err := crunchyroll.NewClient(ctx, logger, email, password)
			if err != nil {
				return fmt.Errorf("creating Crunchyroll client: %w", err)
			}

			// Create tracker and poller
			tr := tracker.NewTracker(logger)
			p := poller.NewPoller(logger, client, tr, pollInterval, healthcheckUUID)

			// Enable wiki enrichment when quiz generation is configured.
			if openrouterKey != "" {
				wikiClient := onepiecewiki.NewClient(logger)
				p.SetWikiEnricher(wikiClient)
				logger.Info("wiki enrichment enabled")
			}

			// Perform initial data fetch synchronously
			if err := p.Fetch(ctx); err != nil {
				return fmt.Errorf("initial fetch failed: %w", err)
			}

			// Start background polling loop
			go func() {
				if err := p.Start(ctx); err != nil {
					logger.Error("poller stopped with error", "error", err)
				}
			}()

			// Create and start web server
			var quizGen *quiz.Generator
			if openrouterKey != "" {
				quizGen = quiz.NewGenerator(openrouterKey)
				logger.Info("quiz generation enabled")
			}
			server := web.NewServer(logger, p, web.Config{
				PostHogAPIKey: posthogKey,
				PostHogHost:   posthogHost,
				QuizGenerator: quizGen,
			})
			return server.ListenAndServe(ctx, addr)
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "Crunchyroll email ($ONEPIECE_CR_EMAIL)")
	cmd.Flags().StringVar(&password, "password", "", "Crunchyroll password ($ONEPIECE_CR_PASSWORD)")
	cmd.Flags().StringVar(&addr, "addr", ":8080", "Listen address ($ONEPIECE_ADDR)")
	cmd.Flags().DurationVar(&pollInterval, "poll-interval", time.Hour, "Poll interval ($ONEPIECE_POLL_INTERVAL)")
	cmd.Flags().StringVar(&healthcheckUUID, "healthcheck-uuid", "", "Healthchecks.io check UUID ($ONEPIECE_HEALTHCHECK_UUID)")
	cmd.Flags().StringVar(&posthogKey, "posthog-key", "", "PostHog project API key ($ONEPIECE_POSTHOG_KEY)")
	cmd.Flags().StringVar(&posthogHost, "posthog-host", "", "PostHog API host ($ONEPIECE_POSTHOG_HOST)")
	cmd.Flags().StringVar(&openrouterKey, "openrouter-key", "", "OpenRouter API key for quiz generation ($ONEPIECE_OPENROUTER_API_KEY)")

	return cmd
}
