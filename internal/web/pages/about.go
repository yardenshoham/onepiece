package pages

import (
	g "maragu.dev/gomponents"
	"maragu.dev/gomponents/html"
)

// AboutPage renders the about page.
func AboutPage(analyticsConfig AnalyticsConfig) g.Node {
	return Layout("About — One Piece Tracker", "/about", 7200, analyticsConfig,
		html.H2(g.Text("What is this?")),
		html.P(g.Text("One Piece Tracker is a web dashboard that connects to the Crunchyroll API, fetches your One Piece watch history, and presents interesting metrics about your viewing journey.")),

		html.H2(g.Text("Features")),
		html.Ul(
			html.Li(g.Text("Track episodes watched vs. total episodes")),
			html.Li(g.Text("View your average watch rate and estimated catch-up date")),
			html.Li(g.Text("See current and longest viewing streaks")),
			html.Li(g.Text("Visualize daily episode counts in a chart")),
			html.Li(g.Text("Auto-refreshing data every hour")),
		),

		html.H2(g.Text("Credits")),
		html.Ul(
			html.Li(
				g.Text("Crunchyroll API (unofficial) — "),
				html.A(g.Attr("href", "https://github.com/Crunchyroll-Plus/crunchyroll-docs"), g.Text("Documentation")),
			),
			html.Li(
				html.A(g.Attr("href", "https://www.gomponents.com"), g.Text("gomponents")),
				g.Text(" — HTML components in Go"),
			),
			html.Li(
				html.A(g.Attr("href", "https://cobra.dev"), g.Text("Cobra")),
				g.Text(" — CLI framework"),
			),
			html.Li(
				html.A(g.Attr("href", "https://simplecss.org"), g.Text("Simple.css")),
				g.Text(" — Classless CSS framework"),
			),
		),

		html.H2(g.Text("Source Code")),
		html.P(
			html.A(g.Attr("href", "https://github.com/yardenshoham/onepiece"), g.Text("github.com/yardenshoham/onepiece")),
		),

		html.H2(g.Text("Disclaimer")),
		html.P(g.Text("This project is not affiliated with Crunchyroll. It uses unofficial API endpoints. You need a Crunchyroll account to use this application.")),
	)
}
