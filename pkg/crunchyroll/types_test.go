package crunchyroll

import (
	"encoding/json"
	"testing"
)

func TestWatchHistoryEntryDateParsing(t *testing.T) {
	t.Parallel()

	raw := `{
		"id": "GR19Q7PK6",
		"date_played": "2026-04-10T09:58:34Z",
		"fully_watched": true,
		"panel": {
			"title": "Test Episode",
			"episode_metadata": {
				"episode_number": 1,
				"season_number": 1,
				"season_title": "East Blue (1-61)",
				"series_id": "GRMG8ZQZR",
				"series_title": "One Piece",
				"episode_air_date": "2000-08-16T00:00:00Z",
				"duration_ms": 1477912
			}
		}
	}`

	var entry WatchHistoryEntry
	if err := json.Unmarshal([]byte(raw), &entry); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if entry.DatePlayed.Year() != 2026 {
		t.Errorf("got year %d, want 2026", entry.DatePlayed.Year())
	}
	if entry.DatePlayed.Month() != 4 {
		t.Errorf("got month %d, want 4", entry.DatePlayed.Month())
	}
	if entry.DatePlayed.Day() != 10 {
		t.Errorf("got day %d, want 10", entry.DatePlayed.Day())
	}
}

func TestSeasonSlugTitle(t *testing.T) {
	t.Parallel()

	raw := `{
		"id": "GY3VWX3MR",
		"title": "East Blue (1-61)",
		"season_number": 1,
		"number_of_episodes": 61,
		"slug_title": "east-blue-1-61"
	}`

	var season Season
	if err := json.Unmarshal([]byte(raw), &season); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if season.SlugTitle != "east-blue-1-61" {
		t.Errorf("got slug_title %q, want %q", season.SlugTitle, "east-blue-1-61")
	}
	if season.NumberOfEpisodes != 61 {
		t.Errorf("got number_of_episodes %d, want 61", season.NumberOfEpisodes)
	}
}
