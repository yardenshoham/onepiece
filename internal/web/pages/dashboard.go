package pages

import (
	"fmt"

	g "maragu.dev/gomponents"
	"maragu.dev/gomponents/html"

	"github.com/yardenshoham/onepiece/internal/web/components"
	"github.com/yardenshoham/onepiece/pkg/tracker"
)

// DashboardPage renders the main dashboard.
func DashboardPage(d *tracker.Dashboard) g.Node {
	title := fmt.Sprintf("One Piece Tracker — %s's Journey", d.ProfileName)

	return Layout(title, "/",
		// Metric cards grid
		html.Div(g.Attr("class", "grid"),
			// Progress card
			components.Card("📺 Progress",
				html.P(g.Textf("%d / %d episodes", d.EpisodesWatched, d.TotalEpisodes)),
				components.ProgressBar(d.ProgressPercent),
			),

			// Watch rate card
			components.Card("📈 Watch Rate",
				html.P(g.Textf("%.1f episodes/day", d.AvgEpisodesPerDay)),
				html.P(g.Textf("Since %s (%d days)", d.FirstWatchDate.Format("Jan 2, 2006"), d.DaysSinceFirst)),
			),

			// Streak card
			components.Card("🔥 Streak",
				html.P(g.Textf("Current: %d days", d.CurrentStreak)),
				html.P(g.Textf("Best: %d days", d.LongestStreak)),
			),

			// Catch-up card
			components.Card("📅 Estimated Catch-up",
				g.If(d.AvgEpisodesPerDay > 0,
					g.Group([]g.Node{
						html.P(g.Textf("~%s", d.EstimatedCatchUpDate.Format("January 2006"))),
						html.P(g.Textf("%d episodes remaining", d.EpisodesRemaining)),
					}),
				),
				g.If(d.AvgEpisodesPerDay == 0,
					html.P(g.Text("N/A")),
				),
			),

			// Now watching card
			components.Card("📍 Now Watching",
				html.P(html.Strong(g.Textf("Episode %d: %s", d.LastEpisode.Number, d.LastEpisode.Title))),
				html.P(g.Textf("Season: %s", d.CurrentSeason)),
				html.P(g.Textf("Watched: %s", d.LastEpisode.WatchedAt.Format("Jan 2, 2006"))),
			),
		),

		// Daily chart
		html.H2(g.Text("📊 Episodes Per Day")),
		components.DailyChart(d.DailyEpisodes),

		// Recent episodes
		html.H2(g.Text("📜 Recent Episodes")),
		g.If(len(d.RecentEpisodes) > 0, recentTable(d.RecentEpisodes)),
		g.If(len(d.RecentEpisodes) == 0, html.P(g.Text("No episodes watched yet."))),

		// Last updated
		html.P(
			html.Style("color: var(--text-light, #6c757d); font-size: 0.85em; margin-top: 2em;"),
			g.Textf("Last updated: %s", d.LastUpdated.Format("Jan 2, 2006 3:04 PM UTC")),
		),
	)
}

func recentTable(episodes []tracker.EpisodeInfo) g.Node {
	var rows []g.Node
	for _, ep := range episodes {
		rows = append(rows, html.Tr(
			html.Td(g.Textf("#%d", ep.Number)),
			html.Td(g.Text(ep.Title)),
			html.Td(g.Text(ep.SeasonTitle)),
			html.Td(g.Text(ep.WatchedAt.Format("Jan 2, 2006"))),
		))
	}

	return html.Table(
		html.THead(
			html.Tr(
				html.Th(g.Text("#")),
				html.Th(g.Text("Title")),
				html.Th(g.Text("Season")),
				html.Th(g.Text("Watched")),
			),
		),
		html.TBody(g.Group(rows)),
	)
}

// LoadingPage renders a page shown when data hasn't been fetched yet.
func LoadingPage() g.Node {
	return Layout("One Piece Tracker — Loading...", "/",
		html.Meta(g.Attr("http-equiv", "refresh"), g.Attr("content", "5")),
		html.P(g.Text("Loading data from Crunchyroll... This page will refresh automatically.")),
	)
}
