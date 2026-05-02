package components

import (
	g "maragu.dev/gomponents"
	"maragu.dev/gomponents/html"
)

// Navigation renders the navigation bar. showQuiz controls whether the Quiz
// link is included.
func Navigation(currentPath string, showQuiz bool) g.Node {
	return html.Nav(
		html.A(g.Attr("href", "/"),
			g.If(currentPath == "/", html.Style("font-weight: bold")),
			g.Text("Dashboard"),
		),
		g.If(showQuiz,
			html.A(g.Attr("href", "/quiz"),
				g.If(currentPath == "/quiz", html.Style("font-weight: bold")),
				g.Text("Quiz"),
			),
		),
		html.A(g.Attr("href", "/about"),
			g.If(currentPath == "/about", html.Style("font-weight: bold")),
			g.Text("About"),
		),
	)
}
