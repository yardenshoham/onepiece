package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/yardenshoham/onepiece/internal/web"
	"github.com/yardenshoham/onepiece/pkg/crunchyroll"
	"github.com/yardenshoham/onepiece/pkg/poller"
	"github.com/yardenshoham/onepiece/pkg/tracker"
)

func newWebCmd() *cobra.Command {
	var (
		email           string
		password        string
		addr            string
		pollInterval    time.Duration
		healthcheckUUID string
	)

	cmd := &cobra.Command{
		Use:   "web",
		Short: "Start the One Piece tracker web server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			logger := cmd.Context().Value(loggerKey{}).(*slog.Logger)

			// Resolve flags from env if not set
			if email == "" {
				email = os.Getenv("ONEPIECE_CR_EMAIL")
			}
			if password == "" {
				password = os.Getenv("ONEPIECE_CR_PASSWORD")
			}
			if email == "" || password == "" {
				return fmt.Errorf("email and password are required (use --email/--password flags or ONEPIECE_CR_EMAIL/ONEPIECE_CR_PASSWORD env vars)")
			}

			if !cmd.Flags().Changed("addr") {
				if envAddr := os.Getenv("ONEPIECE_ADDR"); envAddr != "" {
					addr = envAddr
				}
			}
			if !cmd.Flags().Changed("poll-interval") {
				if envInterval := os.Getenv("ONEPIECE_POLL_INTERVAL"); envInterval != "" {
					d, err := time.ParseDuration(envInterval)
					if err != nil {
						return fmt.Errorf("invalid ONEPIECE_POLL_INTERVAL: %w", err)
					}
					pollInterval = d
				}
			}

			if healthcheckUUID == "" {
				healthcheckUUID = os.Getenv("ONEPIECE_HEALTHCHECK_UUID")
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

			// Start poller in background (blocks on initial fetch)
			pollErrCh := make(chan error, 1)
			go func() {
				pollErrCh <- p.Start(ctx)
			}()

			// Wait briefly for initial fetch to complete or fail
			// The poller's Start blocks on the first fetch, so we need to wait
			// for either the dashboard to be set or an error
			for {
				select {
				case err := <-pollErrCh:
					if err != nil {
						return fmt.Errorf("poller failed: %w", err)
					}
					// poller returned nil = context cancelled during initial fetch
					return nil
				default:
				}

				if p.Dashboard() != nil {
					break
				}
				time.Sleep(50 * time.Millisecond)
			}

			// Create and start web server
			server := web.NewServer(logger, p)
			return server.ListenAndServe(ctx, addr)
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "Crunchyroll email ($ONEPIECE_CR_EMAIL)")
	cmd.Flags().StringVar(&password, "password", "", "Crunchyroll password ($ONEPIECE_CR_PASSWORD)")
	cmd.Flags().StringVar(&addr, "addr", ":8080", "Listen address ($ONEPIECE_ADDR)")
	cmd.Flags().DurationVar(&pollInterval, "poll-interval", time.Hour, "Poll interval ($ONEPIECE_POLL_INTERVAL)")
	cmd.Flags().StringVar(&healthcheckUUID, "healthcheck-uuid", "", "Healthchecks.io check UUID ($ONEPIECE_HEALTHCHECK_UUID)")

	return cmd
}
