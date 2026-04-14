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
			html.Meta(g.Attr("http-equiv", "refresh"), g.Attr("content", "7200")), // Auto-refresh every 2 hours
			html.Link(g.Attr("rel", "icon"), g.Attr("type", "image/svg+xml"), g.Attr("href", "/static/favicon.svg")),
			html.Link(g.Attr("rel", "stylesheet"), g.Attr("href", "https://cdn.jsdelivr.net/npm/simpledotcss@2.3.7/simple.min.css")),
			html.Script(g.Attr("src", "https://cdn.jsdelivr.net/npm/chart.js@4.5.1/dist/chart.umd.min.js")),
			html.Script(
				g.Attr("type", "module"),
				g.Attr("src", "https://unpkg.com/@github/relative-time-element@5.0.0/dist/index.js"),
			),
			html.Script(
				g.Raw(`    !function(t,e){var o,n,p,r;e.__SV||(window.posthog && window.posthog.__loaded)||(window.posthog=e,e._i=[],e.init=function(i,s,a){function g(t,e){var o=e.split(".");2==o.length&&(t=t[o[0]],e=o[1]),t[e]=function(){t.push([e].concat(Array.prototype.slice.call(arguments,0)))}}(p=t.createElement("script")).type="text/javascript",p.crossOrigin="anonymous",p.async=!0,p.src=s.api_host.replace(".i.posthog.com","-assets.i.posthog.com")+"/static/array.js",(r=t.getElementsByTagName("script")[0]).parentNode.insertBefore(p,r);var u=e;for(void 0!==a?u=e[a]=[]:a="posthog",u.people=u.people||[],u.toString=function(t){var e="posthog";return"posthog"!==a&&(e+="."+a),t||(e+=" (stub)"),e},u.people.toString=function(){return u.toString(1)+".people (stub)"},o="init Dr qr Ci Br Zr Pr capture calculateEventProperties Ur register register_once register_for_session unregister unregister_for_session Xr getFeatureFlag getFeatureFlagPayload getFeatureFlagResult isFeatureEnabled reloadFeatureFlags updateFlags updateEarlyAccessFeatureEnrollment getEarlyAccessFeatures on onFeatureFlags onSurveysLoaded onSessionId getSurveys getActiveMatchingSurveys renderSurvey displaySurvey cancelPendingSurvey canRenderSurvey canRenderSurveyAsync Jr identify setPersonProperties group resetGroups setPersonPropertiesForFlags resetPersonPropertiesForFlags setGroupPropertiesForFlags resetGroupPropertiesForFlags reset setIdentity clearIdentity get_distinct_id getGroups get_session_id get_session_replay_url alias set_config startSessionRecording stopSessionRecording sessionRecordingStarted captureException captureLog startExceptionAutocapture stopExceptionAutocapture loadToolbar get_property getSessionProperty Wr Hr createPersonProfile setInternalOrTestUser Gr Fr tn opt_in_capturing opt_out_capturing has_opted_in_capturing has_opted_out_capturing get_explicit_consent_status is_capturing clear_opt_in_out_capturing $r debug ki Yr getPageViewId captureTraceFeedback captureTraceMetric Rr".split(" "),n=0;n<o.length;n++)g(u,o[n]);e._i.push([i,s,a])},e.__SV=1)}(document,window.posthog||[]);
				posthog.init('phc_D7HbhfqwyhK8xRP24nHnwajTLq9qKWnxLQGkS9aSBi7M', {
					api_host: 'https://eu.i.posthog.com',
					defaults: '2026-01-30',
					person_profiles: 'always',
				})`)),
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
