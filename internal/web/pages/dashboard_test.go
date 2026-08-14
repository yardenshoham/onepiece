package pages

import (
	"strings"
	"testing"

	"github.com/yardenshoham/onepiece/pkg/tracker"
)

func renderCards(t *testing.T, episodes []tracker.EpisodeInfo) string {
	t.Helper()

	var sb strings.Builder
	if err := recentEpisodeCards(episodes).Render(&sb); err != nil {
		t.Fatalf("rendering cards: %v", err)
	}
	return sb.String()
}

func TestRecentEpisodeCardsStoryEpisode(t *testing.T) {
	t.Parallel()

	got := renderCards(t, []tracker.EpisodeInfo{
		{Number: 592, EpisodeLabel: "592", Title: "Legendary Assassins Descend!"},
	})

	if !strings.Contains(got, "wiki/Episode_592") {
		t.Errorf("story episode should link to its wiki page, got %q", got)
	}
	if !strings.Contains(got, "#592") {
		t.Errorf("story episode should be labeled #592, got %q", got)
	}
}

func TestRecentEpisodeCardsSpecialDoesNotLinkToUnrelatedEpisode(t *testing.T) {
	t.Parallel()

	// Wano's specials reuse low episode numbers: this one is episode_number 3.
	got := renderCards(t, []tracker.EpisodeInfo{
		{Number: 3, EpisodeLabel: "SP3", Title: "The Legend of Kozuki Oden!"},
	})

	if strings.Contains(got, "wiki/Episode_3") {
		t.Errorf("special must not link to Episode_3, got %q", got)
	}
	if !strings.Contains(got, "Special:Search") {
		t.Errorf("special should fall back to a title search, got %q", got)
	}
	if !strings.Contains(got, "#SP3") {
		t.Errorf("special should be labeled #SP3, got %q", got)
	}
}
