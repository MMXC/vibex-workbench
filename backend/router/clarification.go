package router

import (
	"fmt"
	"time"
)

type ClarificationState string

const (
	StateIdle              ClarificationState = "idle"
	StateReceiving         ClarificationState = "receiving"
	StateClarificationLoop ClarificationState = "clarification_loop"
	StateTerminating       ClarificationState = "terminating"
	StateResolved          ClarificationState = "resolved"
)

type ClarificationSession struct {
	Id          string                `json:"id"`
	ThreadId    string                `json:"threadId"`
	Route       string                `json:"route"` // "goal" | "feature" | "bug"
	State       ClarificationState    `json:"state"`
	SubState    string                `json:"subState,omitempty"`
	Questions   []ClarificationItem   `json:"questions"`
	Answers     []ClarificationItem   `json:"answers"`
	Suggestions []SuggestionItem      `json:"suggestions,omitempty"`
	IsResolved  bool                  `json:"isResolved"`
	CreatedAt   string                `json:"createdAt"`
}

type ClarificationItem struct {
	Id        string `json:"id"`
	Type      string `json:"type"` // "ask" | "answer"
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

type SuggestionItem struct {
	Id       string `json:"id"`
	Content  string `json:"content"`
	Type     string `json:"type"` // "style" | "direction" | "detail" | "boundary"
	Accepted *bool  `json:"accepted,omitempty"`
}

func NewClarificationSession(threadId, route string) *ClarificationSession {
	return &ClarificationSession{
		Id:        threadId + "-clarify",
		ThreadId:  threadId,
		Route:     route,
		State:     StateIdle,
		Questions: []ClarificationItem{},
		Answers:   []ClarificationItem{},
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

// Transition advances the state machine based on incoming events
func (s *ClarificationSession) Transition(event string) {
	switch s.State {
	case StateIdle:
		if event == "user_input_received" {
			s.State = StateReceiving
		}
	case StateReceiving:
		if event == "needs_clarification" {
			s.State = StateClarificationLoop
		} else if event == "input_clear" {
			s.State = StateTerminating
		}
	case StateClarificationLoop:
		if event == "user_confirmed" {
			s.State = StateTerminating
		} else if event == "user_rejected" {
			s.SubState = "rejected"
			s.State = StateTerminating
		}
	case StateTerminating:
		if event == "confirmed_io_recorded" {
			s.State = StateResolved
			s.IsResolved = true
		}
	}
}

// AddQuestion adds a clarification question
func (s *ClarificationSession) AddQuestion(content string) {
	s.Questions = append(s.Questions, ClarificationItem{
		Id:        fmt.Sprintf("q-%d", len(s.Questions)+1),
		Type:      "ask",
		Content:   content,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// AddAnswer records a user answer
func (s *ClarificationSession) AddAnswer(content string) {
	s.Answers = append(s.Answers, ClarificationItem{
		Id:        fmt.Sprintf("a-%d", len(s.Answers)+1),
		Type:      "answer",
		Content:   content,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// AddSuggestion adds a suggestion item
func (s *ClarificationSession) AddSuggestion(content, suggestionType string) {
	s.Suggestions = append(s.Suggestions, SuggestionItem{
		Id:      fmt.Sprintf("s-%d", len(s.Suggestions)+1),
		Content: content,
		Type:    suggestionType,
	})
}
