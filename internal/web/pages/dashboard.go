package pages

import (
	"fmt"

	g "maragu.dev/gomponents"
	"maragu.dev/gomponents/html"

	"github.com/yardenshoham/onepiece/internal/web/components"
	"github.com/yardenshoham/onepiece/pkg/tracker"
)

// DashboardPage renders the main dashboard.
func DashboardPage(d *tracker.Dashboard, analyticsConfig AnalyticsConfig) g.Node {
	title := fmt.Sprintf("One Piece Tracker — %s's Journey", d.ProfileName)

	return Layout(title, "/", 7200, analyticsConfig,
		html.Div(g.Attr("class", "dashboard-layout"),
			html.Div(g.Attr("class", "grid"),
				components.Card("📺 Progress",
					html.P(g.Textf("%d / %d episodes", d.EpisodesWatched, d.TotalEpisodes)),
					components.ProgressBar(d.ProgressPercent),
				),
				components.Card("📈 Watch Rate",
					html.P(g.Textf("%.1f episodes/day", d.AvgEpisodesPerDay)),
					html.P(g.Textf("Since %s (%d days)", d.FirstWatchDate.Format("Jan 2, 2006"), d.DaysSinceFirst)),
				),
				components.Card("🔥 Streak",
					html.P(g.Textf("Current: %d days", d.CurrentStreak)),
					html.P(g.Textf("Best: %d days", d.LongestStreak)),
				),
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
			),
			components.Card("📍 Now Watching",
				html.P(html.Strong(g.Textf("Episode %d: %s", d.LastEpisode.Number, d.LastEpisode.Title))),
				html.P(g.Textf("Season: %s", d.CurrentSeason)),
				html.P(g.Textf("Watched: %s", d.LastEpisode.WatchedAt.Format("Jan 2, 2006"))),
			),
			html.Article(g.Attr("class", "dashboard-panel"),
				html.H2(g.Text("📊 Episodes Per Day")),
				components.DailyChart(d.DailyEpisodes),
			),
			html.Article(g.Attr("class", "dashboard-panel"),
				html.H2(g.Text("📜 Recent Episodes")),
				g.If(len(d.RecentEpisodes) > 0,
					recentEpisodeCards(d.RecentEpisodes),
				),
				g.If(len(d.RecentEpisodes) == 0, html.P(g.Text("No episodes watched yet."))),
			),
			html.P(
				g.Attr("class", "dashboard-updated"),
				html.Style("color: var(--text-light, #6c757d); font-size: 0.85em;"),
				g.Text("Last updated: "),
				g.El("relative-time",
					g.Attr("datetime", d.LastUpdated.Format("2006-01-02T15:04:05Z07:00")),
					g.Attr("format", "relative"),
					g.Text(d.LastUpdated.Format("Jan 2, 2006 3:04 PM MST")),
				),
			),
		),
	)
}

func recentEpisodeCards(episodes []tracker.EpisodeInfo) g.Node {
	var cards []g.Node
	for _, ep := range episodes {
		wikiURL := fmt.Sprintf("https://onepiece.fandom.com/wiki/Episode_%d", ep.Number)
		duration := fmt.Sprintf("%d min", ep.DurationMS/1000/60)

		var thumb g.Node
		if ep.ThumbnailURL != "" {
			thumb = html.Img(
				g.Attr("src", ep.ThumbnailURL),
				g.Attr("alt", ep.Title),
				g.Attr("loading", "lazy"),
				g.Attr("class", "ep-card__thumb"),
			)
		}

		var desc g.Node
		if ep.Description != "" {
			desc = html.P(g.Attr("class", "ep-card__desc"), g.Text(ep.Description))
		}

		cards = append(cards, html.Article(
			g.Attr("class", "ep-card"),
			html.A(
				g.Attr("href", wikiURL),
				g.Attr("target", "_blank"),
				g.Attr("rel", "noopener noreferrer"),
				g.Attr("class", "ep-card__link"),
				thumb,
			),
			html.Div(g.Attr("class", "ep-card__body"),
				html.Div(g.Attr("class", "ep-card__meta"),
					html.Span(g.Attr("class", "ep-card__num"), g.Textf("#%d", ep.Number)),
					html.Span(g.Attr("class", "ep-card__season"), g.Text(ep.SeasonTitle)),
				),
				html.H3(
					html.A(
						g.Attr("href", wikiURL),
						g.Attr("target", "_blank"),
						g.Attr("rel", "noopener noreferrer"),
						g.Text(ep.Title),
					),
				),
				desc,
				html.Div(g.Attr("class", "ep-card__footer"),
					html.Span(g.Text(ep.WatchedAt.Format("Jan 2, 2006"))),
					html.Span(g.Text(duration)),
				),
			),
		))
	}
	return html.Div(g.Attr("class", "ep-grid"), g.Group(cards))
}

// LoadingPage renders a page shown when data hasn't been fetched yet.
func LoadingPage(analyticsConfig AnalyticsConfig) g.Node {
	return Layout("One Piece Tracker — Loading...", "/", 5, analyticsConfig,
		html.P(g.Text("Loading data from Crunchyroll... This page will refresh automatically.")),
	)
}
