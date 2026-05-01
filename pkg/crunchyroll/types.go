package crunchyroll

import "time"

// OnePieceSeriesID is the Crunchyroll series ID for One Piece.
const OnePieceSeriesID = "GRMG8ZQZR"

// Profile represents the user's Crunchyroll profile.
type Profile struct {
	ProfileName string `json:"profile_name"`
}

// authResponse represents the response from the auth endpoint.
type authResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	AccountID    string `json:"account_id"`
}

// WatchHistoryResponse represents the paginated watch history response.
type WatchHistoryResponse struct {
	Total int                 `json:"total"`
	Data  []WatchHistoryEntry `json:"data"`
}

// WatchHistoryEntry represents a single entry in the watch history.
type WatchHistoryEntry struct {
	ID           string    `json:"id"`
	DatePlayed   time.Time `json:"date_played"`
	FullyWatched bool      `json:"fully_watched"`
	Panel        Panel     `json:"panel"`
}

// Panel holds the display metadata for a watch history entry.
type Panel struct {
	ID              string          `json:"id"`
	Title           string          `json:"title"`
	Description     string          `json:"description"`
	SlugTitle       string          `json:"slug_title"`
	Images          PanelImages     `json:"images"`
	EpisodeMetadata EpisodeMetadata `json:"episode_metadata"`
}

// PanelImages holds image collections for a panel.
type PanelImages struct {
	// Thumbnail is a slice of image-set slices; each inner slice is a set of
	// sizes for the same image. We take the first set.
	Thumbnail [][]ThumbnailImage `json:"thumbnail"`
}

// ThumbnailAt returns the URL of the thumbnail closest to the requested width,
// or empty string if no thumbnails exist.
func (p PanelImages) ThumbnailAt(targetWidth int) string {
	if len(p.Thumbnail) == 0 || len(p.Thumbnail[0]) == 0 {
		return ""
	}
	best := p.Thumbnail[0][0]
	for _, img := range p.Thumbnail[0] {
		if abs(img.Width-targetWidth) < abs(best.Width-targetWidth) {
			best = img
		}
	}
	return best.Source
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ThumbnailImage is a single resolution variant of a thumbnail.
type ThumbnailImage struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Source string `json:"source"`
	Type   string `json:"type"`
}

// EpisodeMetadata holds episode-level information.
type EpisodeMetadata struct {
	EpisodeNumber  int       `json:"episode_number"`
	SeasonNumber   int       `json:"season_number"`
	SeasonID       string    `json:"season_id"`
	SeasonTitle    string    `json:"season_title"`
	SeriesID       string    `json:"series_id"`
	SeriesTitle    string    `json:"series_title"`
	EpisodeAirDate time.Time `json:"episode_air_date"`
	DurationMS     int       `json:"duration_ms"`
}

// Season represents a season of a series.
type Season struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	SeasonNumber     int    `json:"season_number"`
	NumberOfEpisodes int    `json:"number_of_episodes"`
	SlugTitle        string `json:"slug_title"`
}

// seasonsResponse wraps the seasons API response.
type seasonsResponse struct {
	Data []Season `json:"data"`
}

// Series represents series metadata.
type Series struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// seriesResponse wraps the series API response.
type seriesResponse struct {
	Data []Series `json:"data"`
}
