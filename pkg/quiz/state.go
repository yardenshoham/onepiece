package quiz

import (
	"context"
	"crypto/rand"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/yardenshoham/onepiece/pkg/tracker"
)

// Question holds a single quiz question and its answer state.
type Question struct {
	ID            string
	EpisodeNumber int
	Text          string
	CorrectOption string
	WrongOptions  []string // exactly 3
	Answered      bool
	UserChoice    string
}

// batch is an internal set of questions tied to a specific episode set.
type batch struct {
	questions     []Question
	episodeSig    string
	usedQuestions []string // texts of all answered questions, accumulated across batches
}

// State manages the in-process quiz state. It is safe for concurrent use.
type State struct {
	mu      sync.Mutex
	current *batch
}

// NewState returns an empty State.
func NewState() *State {
	return &State{}
}

// GetOrGenerate returns the current batch's questions, generating a new batch
// when necessary. A new batch is generated when:
//   - there is no current batch,
//   - the episode signature has changed, or
//   - refresh is true and all questions in the current batch are answered.
//
// episodes should be the most recent 5 (or fewer) episodes from the dashboard.
func (s *State) GetOrGenerate(ctx context.Context, gen *Generator, episodes []tracker.EpisodeInfo, refresh bool) ([]Question, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sig := episodeSig(episodes)

	if s.current != nil && s.current.episodeSig != sig {
		// Episode set changed; preserve used-question history but start fresh.
		used := s.current.usedQuestions
		s.current = &batch{episodeSig: sig, usedQuestions: used}
	}

	needGenerate := s.current == nil ||
		len(s.current.questions) == 0 ||
		(refresh && allAnswered(s.current.questions))

	if !needGenerate {
		return s.current.questions, nil
	}

	srcs := episodeSources(episodes)

	var usedTexts []string
	if s.current != nil {
		usedTexts = s.current.usedQuestions
	}

	raw, err := gen.GenerateQuestions(ctx, srcs, usedTexts)
	if err != nil {
		return nil, err
	}

	questions := make([]Question, len(raw))
	for i, r := range raw {
		questions[i] = Question{
			ID:            newID(),
			EpisodeNumber: srcs[i%len(srcs)].Number,
			Text:          r.Question,
			CorrectOption: r.CorrectOption,
			WrongOptions:  r.WrongOptions,
		}
	}

	preserved := usedTexts
	if s.current != nil {
		preserved = s.current.usedQuestions
	}
	s.current = &batch{
		questions:     questions,
		episodeSig:    sig,
		usedQuestions: preserved,
	}
	return questions, nil
}

// Answer marks the question with the given ID as answered with the given choice.
// Returns the updated Question and true, or a zero Question and false if the ID
// is not found.
func (s *State) Answer(questionID, choice string) (Question, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current == nil {
		return Question{}, false
	}

	for i, q := range s.current.questions {
		if q.ID != questionID {
			continue
		}
		if q.Answered {
			// Already answered — return as-is.
			return q, true
		}
		s.current.questions[i].Answered = true
		s.current.questions[i].UserChoice = choice
		s.current.usedQuestions = append(s.current.usedQuestions, q.Text)
		return s.current.questions[i], true
	}
	return Question{}, false
}

// AllAnswered reports whether every question in the current batch has been answered.
func (s *State) AllAnswered() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return false
	}
	return allAnswered(s.current.questions)
}

func allAnswered(qs []Question) bool {
	for _, q := range qs {
		if !q.Answered {
			return false
		}
	}
	return true
}

// episodeSig returns a stable string signature for a slice of episodes.
func episodeSig(episodes []tracker.EpisodeInfo) string {
	nums := make([]int, len(episodes))
	for i, ep := range episodes {
		nums[i] = ep.Number
	}
	sort.Ints(nums)
	parts := make([]string, len(nums))
	for i, n := range nums {
		parts[i] = fmt.Sprintf("%d", n)
	}
	return strings.Join(parts, ",")
}

// episodeSources converts tracker.EpisodeInfo to EpisodeSource, preferring
// LongDescription when available.
func episodeSources(episodes []tracker.EpisodeInfo) []EpisodeSource {
	srcs := make([]EpisodeSource, len(episodes))
	for i, ep := range episodes {
		desc := ep.LongDescription
		if desc == "" {
			desc = ep.Description
		}
		srcs[i] = EpisodeSource{
			Number:      ep.Number,
			Title:       ep.Title,
			Description: desc,
		}
	}
	return srcs
}

// newID generates a random hex ID.
func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%x", b)
}
