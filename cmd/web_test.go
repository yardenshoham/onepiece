package cmd

import (
	"strings"
	"testing"
	"time"
)

func TestWebFlagEnvNamesExist(t *testing.T) {
	t.Parallel()
	flags := newWebCmd().Flags()
	for _, fe := range webFlagEnv {
		if flags.Lookup(fe.flag) == nil {
			t.Errorf("flag %q (from %s) is not registered", fe.flag, fe.env)
		}
	}
}

func TestResolveFlagsFromEnv(t *testing.T) {
	t.Run("fills unset flags", func(t *testing.T) {
		t.Setenv("ONEPIECE_CR_EMAIL", "luffy@example.com")
		t.Setenv("ONEPIECE_POLL_INTERVAL", "15m")

		cmd := newWebCmd()
		if err := resolveFlagsFromEnv(cmd.Flags()); err != nil {
			t.Fatalf("resolveFlagsFromEnv: %v", err)
		}
		if got := cmd.Flags().Lookup("email").Value.String(); got != "luffy@example.com" {
			t.Errorf("email = %q, want luffy@example.com", got)
		}
		if got := cmd.Flags().Lookup("poll-interval").Value.String(); got != (15 * time.Minute).String() {
			t.Errorf("poll-interval = %q, want 15m0s", got)
		}
	})

	t.Run("keeps flags passed on the command line", func(t *testing.T) {
		t.Setenv("ONEPIECE_ADDR", ":9999")

		cmd := newWebCmd()
		if err := cmd.Flags().Set("addr", ":1234"); err != nil {
			t.Fatalf("setting addr: %v", err)
		}
		if err := resolveFlagsFromEnv(cmd.Flags()); err != nil {
			t.Fatalf("resolveFlagsFromEnv: %v", err)
		}
		if got := cmd.Flags().Lookup("addr").Value.String(); got != ":1234" {
			t.Errorf("addr = %q, want :1234", got)
		}
	})

	t.Run("leaves defaults when env is empty", func(t *testing.T) {
		t.Setenv("ONEPIECE_ADDR", "")

		cmd := newWebCmd()
		if err := resolveFlagsFromEnv(cmd.Flags()); err != nil {
			t.Fatalf("resolveFlagsFromEnv: %v", err)
		}
		if got := cmd.Flags().Lookup("addr").Value.String(); got != ":8080" {
			t.Errorf("addr = %q, want :8080", got)
		}
	})

	t.Run("reports an invalid value", func(t *testing.T) {
		t.Setenv("ONEPIECE_POLL_INTERVAL", "not-a-duration")

		err := resolveFlagsFromEnv(newWebCmd().Flags())
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		if want := "invalid ONEPIECE_POLL_INTERVAL"; !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	})
}
