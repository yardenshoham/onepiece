package components

import (
	"encoding/json"
	"fmt"

	g "maragu.dev/gomponents"
	"maragu.dev/gomponents/html"

	"github.com/yardenshoham/onepiece/pkg/tracker"
)

// DailyChart renders a Chart.js bar chart of daily episode counts.
// Each bar is coloured by whether that day was above (good/green) or below
// (bad/red) the running average of all previous days.
func DailyChart(data []tracker.DailyCount, profileName string) g.Node {
	if len(data) == 0 {
		return html.P(g.Text("No data to display."))
	}

	const (
		goodColor    = "seagreen"
		badColor     = "crimson"
		neutralColor = "steelblue"
	)

	labels := make([]string, len(data))
	counts := make([]int, len(data))
	colors := make([]string, len(data))

	cumSum := 0
	for i, d := range data {
		labels[i] = d.Date[5:] // "MM-DD"
		counts[i] = d.Count

		if i == 0 {
			colors[i] = neutralColor
		} else {
			avg := float64(cumSum) / float64(i)
			switch {
			case float64(d.Count) > avg:
				colors[i] = goodColor
			case float64(d.Count) < avg:
				colors[i] = badColor
			default:
				colors[i] = neutralColor
			}
		}
		cumSum += d.Count
	}

	labelsJSON, err := json.Marshal(labels)
	if err != nil {
		return html.P(g.Textf("Error rendering chart: %v", err))
	}
	countsJSON, err := json.Marshal(counts)
	if err != nil {
		return html.P(g.Textf("Error rendering chart: %v", err))
	}
	colorsJSON, err := json.Marshal(colors)
	if err != nil {
		return html.P(g.Textf("Error rendering chart: %v", err))
	}

	script := fmt.Sprintf(`new Chart(document.getElementById('dailyChart'),{type:'bar',data:{labels:%s,datasets:[{label:'Episodes',data:%s,backgroundColor:%s,borderColor:%s,borderWidth:1}]},options:{responsive:true,maintainAspectRatio:false,scales:{y:{beginAtZero:true,ticks:{precision:0}}},plugins:{legend:{display:false}}}});`,
		labelsJSON, countsJSON, colorsJSON, colorsJSON)

	return html.Div(
		html.Div(
			g.Attr("class", "chart-shell"),
			g.El("canvas", g.Attr("id", "dailyChart")),
			html.Script(g.Raw(script)),
		),
		html.Div(
			g.Attr("class", "chart-legend"),
			legendItem(goodColor, fmt.Sprintf("Good day — above %s's running average", profileName)),
			legendItem(badColor, fmt.Sprintf("Bad day — below %s's running average", profileName)),
			legendItem(neutralColor, "Neutral — first day or on average"),
		),
	)
}

func legendItem(color, label string) g.Node {
	return html.Span(
		g.Attr("class", "chart-legend__item"),
		html.Span(g.Attr("class", "chart-legend__swatch"), g.Attr("style", "background:"+color)),
		g.Text(label),
	)
}
