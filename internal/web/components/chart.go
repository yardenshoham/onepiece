package components

import (
	"encoding/json"
	"fmt"

	g "maragu.dev/gomponents"
	"maragu.dev/gomponents/html"

	"github.com/yardenshoham/onepiece/pkg/tracker"
)

// DailyChart renders a Chart.js bar chart of daily episode counts.
func DailyChart(data []tracker.DailyCount) g.Node {
	if len(data) == 0 {
		return html.P(g.Text("No data to display."))
	}

	labels := make([]string, len(data))
	counts := make([]int, len(data))
	for i, d := range data {
		labels[i] = d.Date[5:] // "MM-DD"
		counts[i] = d.Count
	}

	labelsJSON, _ := json.Marshal(labels)
	countsJSON, _ := json.Marshal(counts)

	script := fmt.Sprintf(`new Chart(document.getElementById('dailyChart'),{type:'bar',data:{labels:%s,datasets:[{label:'Episodes',data:%s,backgroundColor:'rgba(13,110,253,0.7)',borderColor:'rgba(13,110,253,1)',borderWidth:1}]},options:{responsive:true,scales:{y:{beginAtZero:true,ticks:{precision:0}}},plugins:{legend:{display:false}}}});`, labelsJSON, countsJSON)

	return html.Div(
		html.Style("width: 100%; max-width: 800px;"),
		g.El("canvas", g.Attr("id", "dailyChart")),
		html.Script(g.Raw(script)),
	)
}
