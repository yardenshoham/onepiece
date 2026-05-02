package pages

import (
	"fmt"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	"maragu.dev/gomponents/html"

	"github.com/yardenshoham/onepiece/pkg/quiz"
)

// QuizPage renders the quiz shell — a static page with an HTMX placeholder
// that immediately triggers POST /quiz/questions on load.
//
// The trigger is placed on an inner child div (not on #quiz-shell itself) so
// that when the swap replaces #quiz-shell's innerHTML the trigger element is
// gone and cannot re-fire, preventing an infinite reload loop.
func QuizPage(analyticsConfig AnalyticsConfig, profileName string) g.Node {
	subtitle := fmt.Sprintf("Test your knowledge of the episodes %s watched!", profileName)
	return Layout("Quiz — One Piece Tracker", "/quiz", 0, analyticsConfig,
		html.H2(g.Text("🧠 One Piece Quiz")),
		html.P(g.Text(subtitle)),
		html.Div(
			g.Attr("id", "quiz-shell"),
			// This inner div fires the initial load. It is replaced by the
			// swap response so it will not trigger again.
			html.Div(
				hx.Post("/quiz/questions"),
				hx.Trigger("load"),
				hx.Target("#quiz-shell"),
				hx.Swap("innerHTML"),
				hx.Indicator("#quiz-loading"),
				html.P(
					g.Attr("id", "quiz-loading"),
					g.Attr("class", "htmx-indicator"),
					g.Text("Generating questions…"),
				),
			),
		),
	)
}

// QuizQuestionsFragment renders the quiz questions as an HTML fragment
// (returned by POST /quiz/questions).
// Already-answered questions are skipped so a page refresh only shows what remains.
// If there are no unanswered questions (full-page navigation back to an
// already-completed batch), an auto-trigger div is returned so the next batch
// loads immediately. Otherwise a hidden #quiz-load-more placeholder is included
// as the OOB swap target for QuizAnswerFragment.
func QuizQuestionsFragment(questions []quiz.Question) g.Node {
	var cards []g.Node
	for _, q := range questions {
		if !q.Answered {
			cards = append(cards, questionCard(q))
		}
	}

	if len(cards) == 0 {
		// All questions already answered — auto-trigger the next batch.
		return autoLoadDiv()
	}

	// Placeholder — replaced via OOB when all questions are answered.
	placeholder := html.Div(g.Attr("id", "quiz-load-more"))
	return g.Group(append(cards, placeholder))
}

// QuizAnswerFragment renders a single answered question card.
// If allAnswered is true, an out-of-band swap reveals the "Load 3 more" button.
func QuizAnswerFragment(q quiz.Question, allAnswered bool) g.Node {
	nodes := []g.Node{answeredCard(q)}
	if allAnswered {
		nodes = append(nodes, oobLoadMoreDiv())
	}
	return g.Group(nodes)
}

// QuizErrorFragment renders an error message as an HTML fragment.
func QuizErrorFragment(message string) g.Node {
	return html.P(
		g.Attr("class", "quiz-error"),
		g.Text("⚠️ "+message),
	)
}

// questionCard renders an unanswered question with answer buttons.
func questionCard(q quiz.Question) g.Node {
	options := buildOptions(q)
	var buttons []g.Node
	for _, opt := range options {
		buttons = append(buttons, answerButton(q.ID, opt))
	}

	return html.Article(
		g.Attr("id", fmt.Sprintf("quiz-q-%s", q.ID)),
		g.Attr("class", "quiz-question"),
		html.P(html.Strong(g.Text(q.Text))),
		html.Div(g.Attr("class", "quiz-options"), g.Group(buttons)),
	)
}

// answeredCard renders a question that has been answered, showing feedback.
func answeredCard(q quiz.Question) g.Node {
	correct := q.UserChoice == q.CorrectOption
	resultClass := "quiz-result quiz-result--correct"
	resultText := "✅ Correct!"
	if !correct {
		resultClass = "quiz-result quiz-result--wrong"
		resultText = "❌ Incorrect!"
	}

	options := buildOptions(q)
	var optionNodes []g.Node
	for _, opt := range options {
		class := "quiz-option quiz-option--answered"
		if opt == q.CorrectOption {
			class += " quiz-option--correct"
		} else if opt == q.UserChoice && !correct {
			class += " quiz-option--chosen"
		}
		optionNodes = append(optionNodes, html.Button(
			g.Attr("class", class),
			g.Attr("disabled", ""),
			g.Text(opt),
		))
	}

	return html.Article(
		g.Attr("id", fmt.Sprintf("quiz-q-%s", q.ID)),
		g.Attr("class", "quiz-question quiz-question--answered"),
		html.P(html.Strong(g.Text(q.Text))),
		html.Div(g.Attr("class", "quiz-options"), g.Group(optionNodes)),
		html.P(g.Attr("class", resultClass), g.Text(resultText)),
		g.If(!correct,
			html.P(
				g.Attr("class", "quiz-correct-answer"),
				g.Text("The correct answer was: "),
				html.Strong(g.Text(q.CorrectOption)),
			),
		),
	)
}

// answerButton renders a single answer option button.
func answerButton(questionID, option string) g.Node {
	return html.Button(
		g.Attr("class", "quiz-option"),
		hx.Post("/quiz/answer"),
		hx.Target(fmt.Sprintf("#quiz-q-%s", questionID)),
		hx.Swap("outerHTML"),
		hx.Vals(fmt.Sprintf(`{"question_id":%q,"answer":%q}`, questionID, option)),
		g.Text(option),
	)
}

// oobLoadMoreDiv OOB-replaces the hidden #quiz-load-more placeholder with the
// visible "Load 3 more" button.
func oobLoadMoreDiv() g.Node {
	return html.Div(
		g.Attr("id", "quiz-load-more"),
		hx.SwapOOB("true"),
		loadMoreContents(),
	)
}

// loadMoreContents is the inner content of the "Load 3 more" button area.
func loadMoreContents() g.Node {
	return g.Group([]g.Node{
		html.Button(
			g.Attr("class", "quiz-load-more"),
			hx.Post("/quiz/questions"),
			hx.Target("#quiz-shell"),
			hx.Swap("innerHTML"),
			hx.Vals(`{"refresh":"1"}`),
			hx.Indicator("#quiz-load-more-spinner"),
			g.Attr("hx-disabled-elt", "this"),
			g.Text("Load 3 more"),
		),
		html.Span(
			g.Attr("id", "quiz-load-more-spinner"),
			g.Attr("class", "quiz-load-more-spinner htmx-indicator"),
			g.Attr("aria-label", "Loading…"),
		),
	})
}

// autoLoadDiv renders a div that immediately fires POST /quiz/questions (refresh)
// via hx-trigger="load", placed directly inside #quiz-shell. Mirrors the initial
// page-load trigger in QuizPage and is destroyed when the swap response replaces
// #quiz-shell's innerHTML, preventing a re-fire.
func autoLoadDiv() g.Node {
	return html.Div(
		hx.Post("/quiz/questions"),
		hx.Trigger("load"),
		hx.Target("#quiz-shell"),
		hx.Swap("innerHTML"),
		hx.Vals(`{"refresh":"1"}`),
		html.P(g.Text("Generating questions…")),
	)
}

// buildOptions returns all answer options in a randomised order.
// We use a deterministic shuffle keyed to the question ID so the order is
// stable across re-renders of the same question (no crypto randomness needed here).
func buildOptions(q quiz.Question) []string {
	opts := make([]string, 0, 4)
	opts = append(opts, q.CorrectOption)
	opts = append(opts, q.WrongOptions...)

	// Simple deterministic shuffle based on question ID bytes.
	seed := int(q.ID[0]) + int(q.ID[1])*256
	for i := len(opts) - 1; i > 0; i-- {
		seed = seed*1664525 + 1013904223
		j := ((seed >> 16) & 0x7fff) % (i + 1)
		opts[i], opts[j] = opts[j], opts[i]
	}
	return opts
}
