package components

import (
	"fmt"

	g "maragu.dev/gomponents"
	"maragu.dev/gomponents/html"
)

// Card renders a metric card with a title and content.
func Card(title string, children ...g.Node) g.Node {
	return html.Article(
		g.Attr("class", "metric-card"),
		html.H3(g.Text(title)),
		g.Group(children),
	)
}

// ProgressBar renders a progress bar with a percentage label.
func ProgressBar(percent float64) g.Node {
	return html.Div(
		html.Style("background: var(--bg-secondary, #e9ecef); border-radius: 4px; overflow: hidden; height: 1.5em; margin: 0.5em 0;"),
		html.Div(
			html.Style(fmt.Sprintf("background: var(--accent, #0d6efd); height: 100%%; width: %.1f%%; min-width: 2em; border-radius: 4px; text-align: center; color: white; line-height: 1.5em; font-size: 0.85em;", percent)),
			g.Textf("%.1f%%", percent),
		),
	)
}
