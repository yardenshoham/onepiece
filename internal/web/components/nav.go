package components

import (
	g "maragu.dev/gomponents"
	"maragu.dev/gomponents/html"
)

// Navigation renders the navigation bar.
func Navigation(currentPath string) g.Node {
	return html.Nav(
		html.A(g.Attr("href", "/"),
			g.If(currentPath == "/", html.Style("font-weight: bold")),
			g.Text("Dashboard"),
		),
		html.A(g.Attr("href", "/about"),
			g.If(currentPath == "/about", html.Style("font-weight: bold")),
			g.Text("About"),
		),
	)
}
