package quiz

import (
	"testing"

	"github.com/yardenshoham/onepiece/pkg/tracker"
)

func TestEpisodeSig(t *testing.T) {
	t.Parallel()

	episodes := []tracker.EpisodeInfo{
		{Number: 1005},
		{Number: 1003},
		{Number: 1001},
	}
	got := episodeSig(episodes)
	want := "1001,1003,1005"
	if got != want {
		t.Errorf("episodeSig = %q, want %q", got, want)
	}
}

func TestEpisodeSigStable(t *testing.T) {
	t.Parallel()

	a := []tracker.EpisodeInfo{{Number: 10}, {Number: 5}}
	b := []tracker.EpisodeInfo{{Number: 5}, {Number: 10}}
	if episodeSig(a) != episodeSig(b) {
		t.Error("episodeSig is not order-independent")
	}
}

func TestEpisodeSourcesPrefersLongDescription(t *testing.T) {
	t.Parallel()

	eps := []tracker.EpisodeInfo{
		{Number: 1, Title: "Romance Dawn", Description: "short", LongDescription: "long"},
		{Number: 2, Title: "Other", Description: "only short"},
	}
	srcs := episodeSources(eps)
	if srcs[0].Description != "long" {
		t.Errorf("expected long description, got %q", srcs[0].Description)
	}
	if srcs[1].Description != "only short" {
		t.Errorf("expected short description fallback, got %q", srcs[1].Description)
	}
}

func TestAllAnswered(t *testing.T) {
	t.Parallel()

	qs := []Question{
		{Answered: true},
		{Answered: true},
		{Answered: false},
	}
	if allAnswered(qs) {
		t.Error("expected not all answered")
	}
	qs[2].Answered = true
	if !allAnswered(qs) {
		t.Error("expected all answered")
	}
}

func TestStateAnswer(t *testing.T) {
	t.Parallel()

	s := NewState()
	s.mu.Lock()
	s.current = &batch{
		questions: []Question{
			{ID: "abc", Text: "question1", CorrectOption: "correct", WrongOptions: []string{"w1", "w2", "w3"}},
		},
	}
	s.mu.Unlock()

	// Answer with wrong choice.
	q, ok := s.Answer("abc", "wrong")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !q.Answered {
		t.Error("expected question to be marked answered")
	}
	if q.UserChoice != "wrong" {
		t.Errorf("got UserChoice %q, want %q", q.UserChoice, "wrong")
	}

	// Answering again returns existing state.
	q2, ok2 := s.Answer("abc", "other")
	if !ok2 {
		t.Fatal("expected ok2=true")
	}
	if q2.UserChoice != "wrong" {
		t.Error("re-answering should not change user choice")
	}

	// Unknown ID returns false.
	_, ok3 := s.Answer("nope", "x")
	if ok3 {
		t.Error("expected ok3=false for unknown question ID")
	}
}

func TestStateUsedQuestionsAccumulate(t *testing.T) {
	t.Parallel()

	s := NewState()
	s.mu.Lock()
	s.current = &batch{
		questions: []Question{
			{ID: "q1", Text: "What did Luffy eat?"},
		},
		episodeSig: "1",
	}
	s.mu.Unlock()

	s.Answer("q1", "Gomu Gomu no Mi")

	s.mu.Lock()
	used := s.current.usedQuestions
	s.mu.Unlock()

	if len(used) != 1 || used[0] != "What did Luffy eat?" {
		t.Errorf("expected used questions to contain answered text, got %v", used)
	}
}
