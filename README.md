# onepiece

[![Go Report Card](https://goreportcard.com/badge/github.com/yardenshoham/onepiece)](https://goreportcard.com/report/github.com/yardenshoham/onepiece)

A web dashboard that tracks your One Piece viewing progress on Crunchyroll.

## Features

- Connects to the Crunchyroll API to fetch your One Piece watch history
- Displays episodes watched, progress percentage, and current position
- Calculates watch rate (episodes/day) and estimates when you'll catch up
- Tracks viewing streaks (current and longest consecutive days)
- Auto-refreshes data every hour in the background

## Quick Start

### Prerequisites

- A [Crunchyroll](https://www.crunchyroll.com/) account with One Piece in your watch history

### Run with Go

```bash
export ONEPIECE_CR_EMAIL="your-email@example.com"
export ONEPIECE_CR_PASSWORD="your-password"
go run . web
```

Open http://localhost:8080 in your browser.

### Run with Docker

```bash
docker build -t onepiece .
docker run -p 8080:8080 \
  -e ONEPIECE_CR_EMAIL="your-email@example.com" \
  -e ONEPIECE_CR_PASSWORD="your-password" \
  onepiece
```

## Configuration

| Environment Variable        | Flag                 | Default    | Description                               |
| --------------------------- | -------------------- | ---------- | ----------------------------------------- |
| `ONEPIECE_CR_EMAIL`         | `--email`            | (required) | Crunchyroll account email                 |
| `ONEPIECE_CR_PASSWORD`      | `--password`         | (required) | Crunchyroll account password              |
| `ONEPIECE_ADDR`             | `--addr`             | `:8080`    | HTTP listen address                       |
| `ONEPIECE_POLL_INTERVAL`    | `--poll-interval`    | `1h`       | Data refresh interval                     |
| `ONEPIECE_HEALTHCHECK_UUID` | `--healthcheck-uuid` |            | Healthchecks.io check UUID for monitoring |
| `ONEPIECE_POSTHOG_KEY`      |                      |            | PostHog project API key for analytics     |
| `ONEPIECE_POSTHOG_HOST`     |                      |            | PostHog API host                          |

## Analytics

Optional PostHog analytics can be enabled by setting `ONEPIECE_POSTHOG_KEY`.

- `ONEPIECE_POSTHOG_KEY` enables the PostHog client when set
- `ONEPIECE_POSTHOG_HOST` overrides the PostHog API host
- If `ONEPIECE_POSTHOG_HOST` is unset, the app defaults to `https://eu.i.posthog.com`

## Monitoring

The dashboard supports optional [healthchecks.io](https://healthchecks.io/) monitoring. When configured, each poll cycle sends:

- A **start** signal before fetching data (enables execution time tracking)
- A **success** signal with diagnostics (profile name, episodes watched, duration) on completion
- A **failure** signal with error details if the poll fails

Signals use run IDs to correlate start/completion for accurate duration measurement, and include automatic retries with exponential backoff.

## Commands

- `onepiece web` — Start the web dashboard
- `onepiece version` — Print version information
