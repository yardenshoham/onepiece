package components

import (
	"fmt"

	g "maragu.dev/gomponents"
	"maragu.dev/gomponents/html"

	"github.com/yardenshoham/onepiece/pkg/tracker"
)

const (
	chartWidth  = 800
	chartHeight = 300
	chartPadX   = 60
	chartPadY   = 40
	maxDays     = 60
)

// DailyChart renders an SVG bar chart of daily episode counts.
func DailyChart(data []tracker.DailyCount) g.Node {
	if len(data) == 0 {
		return html.P(g.Text("No data to display."))
	}

	// Limit to last maxDays days
	if len(data) > maxDays {
		data = data[len(data)-maxDays:]
	}

	// Find max count for Y-axis scaling
	maxCount := 0
	for _, d := range data {
		if d.Count > maxCount {
			maxCount = d.Count
		}
	}
	if maxCount == 0 {
		maxCount = 1
	}

	plotW := float64(chartWidth - 2*chartPadX)
	plotH := float64(chartHeight - 2*chartPadY)
	barW := plotW / float64(len(data))
	gap := barW * 0.2
	actualBarW := barW - gap

	var bars []g.Node

	// Y-axis line
	bars = append(bars, el("line",
		attr("x1", fmt.Sprintf("%d", chartPadX)),
		attr("y1", fmt.Sprintf("%d", chartPadY)),
		attr("x2", fmt.Sprintf("%d", chartPadX)),
		attr("y2", fmt.Sprintf("%d", chartHeight-chartPadY)),
		attr("stroke", "currentColor"),
		attr("stroke-width", "1"),
	))

	// X-axis line
	bars = append(bars, el("line",
		attr("x1", fmt.Sprintf("%d", chartPadX)),
		attr("y1", fmt.Sprintf("%d", chartHeight-chartPadY)),
		attr("x2", fmt.Sprintf("%d", chartWidth-chartPadX)),
		attr("y2", fmt.Sprintf("%d", chartHeight-chartPadY)),
		attr("stroke", "currentColor"),
		attr("stroke-width", "1"),
	))

	// Y-axis labels
	for i := 0; i <= maxCount; i++ {
		y := float64(chartHeight-chartPadY) - (float64(i)/float64(maxCount))*plotH
		bars = append(bars, el("text",
			attr("x", fmt.Sprintf("%d", chartPadX-5)),
			attr("y", fmt.Sprintf("%.1f", y+4)),
			attr("text-anchor", "end"),
			attr("font-size", "12"),
			attr("fill", "currentColor"),
			g.Text(fmt.Sprintf("%d", i)),
		))
	}

	// Bars and X-axis labels
	labelInterval := max(len(data)/7, 1)

	for i, d := range data {
		x := float64(chartPadX) + float64(i)*barW + gap/2
		barH := (float64(d.Count) / float64(maxCount)) * plotH
		y := float64(chartHeight-chartPadY) - barH

		if d.Count > 0 {
			bars = append(bars, el("rect",
				attr("x", fmt.Sprintf("%.1f", x)),
				attr("y", fmt.Sprintf("%.1f", y)),
				attr("width", fmt.Sprintf("%.1f", actualBarW)),
				attr("height", fmt.Sprintf("%.1f", barH)),
				attr("fill", "var(--accent, #0d6efd)"),
			))
		}

		// X-axis labels at intervals
		if i%labelInterval == 0 || i == len(data)-1 {
			labelX := x + actualBarW/2
			// Show short month-day format
			label := d.Date[5:] // "MM-DD"
			bars = append(bars, el("text",
				attr("x", fmt.Sprintf("%.1f", labelX)),
				attr("y", fmt.Sprintf("%d", chartHeight-chartPadY+16)),
				attr("text-anchor", "middle"),
				attr("font-size", "10"),
				attr("fill", "currentColor"),
				g.Text(label),
			))
		}
	}

	return html.Div(
		html.Style("width: 100%; max-width: 800px; overflow-x: auto;"),
		el("svg",
			attr("viewBox", fmt.Sprintf("0 0 %d %d", chartWidth, chartHeight)),
			attr("xmlns", "http://www.w3.org/2000/svg"),
			html.Style("width: 100%; height: auto;"),
			g.Group(bars),
		),
	)
}

func el(name string, children ...g.Node) g.Node {
	return g.El(name, children...)
}

func attr(name, value string) g.Node {
	return g.Attr(name, value)
}
