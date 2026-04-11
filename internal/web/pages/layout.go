package pages

import (
	g "maragu.dev/gomponents"
	c "maragu.dev/gomponents/components"
	"maragu.dev/gomponents/html"

	"github.com/yardenshoham/onepiece/internal/web/components"
)

// Layout wraps page content with the shared HTML layout.
func Layout(title, currentPath string, children ...g.Node) g.Node {
	return c.HTML5(c.HTML5Props{
		Title:    title,
		Language: "en",
		Head: []g.Node{
			html.Meta(g.Attr("name", "viewport"), g.Attr("content", "width=device-width, initial-scale=1")),
			html.Link(g.Attr("rel", "stylesheet"), g.Attr("href", "https://cdn.simplecss.org/simple.min.css")),
			html.Script(
				g.Attr("type", "module"),
				g.Attr("src", "https://unpkg.com/@github/relative-time-element@5.0.0/dist/index.js"),
			),
			html.StyleEl(g.Text(`
				.grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 1em; }
				article { margin: 0; }
			`)),
		},
		Body: []g.Node{
			html.Header(
				html.H1(g.Text("\U0001F3F4\u200D☠️ "+title)),
				components.Navigation(currentPath),
			),
			html.Main(g.Group(children)),
			html.Footer(
				html.P(g.Text("One Piece Tracker — Unofficial Crunchyroll watch tracker")),
			),
		},
	})
}
