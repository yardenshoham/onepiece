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
		avgColor     = "darkorange"
	)

	labels := make([]string, len(data))
	counts := make([]int, len(data))
	colors := make([]string, len(data))
	runningAverages := make([]float64, len(data))

	cumSum := 0
	for i, d := range data {
		labels[i] = d.Date[5:] // "MM-DD"
		counts[i] = d.Count

		if i == 0 {
			colors[i] = neutralColor
		} else {
			avg := float64(cumSum) / float64(i)
			diff := float64(d.Count) - avg
			switch {
			case diff > 0.1:
				colors[i] = goodColor
			case diff < -0.1:
				colors[i] = badColor
			default:
				colors[i] = neutralColor
			}
		}
		cumSum += d.Count
		runningAverages[i] = float64(cumSum) / float64(i+1)
	}

	// These are plain []string/[]int/[]float64 built above, and the running
	// average always divides by i+1 ≥ 1, so no NaN or Inf can reach here —
	// json.Marshal cannot fail on any of them.
	labelsJSON, _ := json.Marshal(labels)
	countsJSON, _ := json.Marshal(counts)
	colorsJSON, _ := json.Marshal(colors)
	avgsJSON, _ := json.Marshal(runningAverages)

	script := fmt.Sprintf(`new Chart(document.getElementById('dailyChart'),{type:'bar',data:{labels:%s,datasets:[{label:'Episodes',data:%s,backgroundColor:%s,borderColor:%s,borderWidth:1},{type:'line',label:'Running avg',data:%s,borderColor:'darkorange',backgroundColor:'transparent',borderWidth:2,pointRadius:0,tension:0.3}]},options:{responsive:true,maintainAspectRatio:false,scales:{y:{beginAtZero:true,ticks:{precision:0}}},plugins:{legend:{display:false}}}});`,
		labelsJSON, countsJSON, colorsJSON, colorsJSON, avgsJSON)

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
			legendItem(avgColor, "Running average"),
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
