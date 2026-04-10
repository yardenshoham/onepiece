package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

type loggerKey struct{}

func newRootCmd() *cobra.Command {
	var debug bool

	var rootCmd = &cobra.Command{
		Use:          "onepiece",
		Short:        "",
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			level := slog.LevelInfo
			if debug {
				level = slog.LevelDebug
			}
			logger := slog.New(slog.NewTextHandler(cmd.OutOrStdout(), &slog.HandlerOptions{
				Level: level,
			}))
			cmd.SetContext(context.WithValue(cmd.Context(), loggerKey{}, logger))
		},
	}

	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newWebCmd())
	return rootCmd
}

func Execute() {
	rootCmd := newRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
