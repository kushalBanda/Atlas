package fields

import (
	"testing"

	"atlas/pkg/storage"
)

func strp(v string) *string { return &v }

func TestExtractAgentFields_PrimaryKeys(t *testing.T) {
	s := storage.Span{Attributes: map[string]any{
		"session.id":      "sess-1",
		"user.id":         "user-1",
		"agent.run.id":    "run-1",
		"agent.name":      "researcher",
		"agent.step.kind": "tool",
	}}

	ExtractAgentFields(&s)

	for _, tc := range []struct {
		name string
		got  *string
		want string
	}{
		{"SessionID", s.SessionID, "sess-1"},
		{"UserID", s.UserID, "user-1"},
		{"AgentRunID", s.AgentRunID, "run-1"},
		{"AgentName", s.AgentName, "researcher"},
		{"AgentStepKind", s.AgentStepKind, "tool"},
	} {
		if tc.got == nil || *tc.got != tc.want {
			t.Errorf("%s = %v, want %q", tc.name, tc.got, tc.want)
		}
	}
}

func TestExtractAgentFields_GenAIKeys(t *testing.T) {
	s := storage.Span{Attributes: map[string]any{
		"gen_ai.conversation.id": "sess-2",
		"gen_ai.user.id":         "user-2",
		"gen_ai.agent.run.id":    "run-2",
		"gen_ai.agent.name":      "planner",
	}}

	ExtractAgentFields(&s)

	if s.SessionID == nil || *s.SessionID != "sess-2" {
		t.Errorf("SessionID = %v, want sess-2", s.SessionID)
	}
	if s.UserID == nil || *s.UserID != "user-2" {
		t.Errorf("UserID = %v, want user-2", s.UserID)
	}
	if s.AgentRunID == nil || *s.AgentRunID != "run-2" {
		t.Errorf("AgentRunID = %v, want run-2", s.AgentRunID)
	}
	if s.AgentName == nil || *s.AgentName != "planner" {
		t.Errorf("AgentName = %v, want planner", s.AgentName)
	}
}

func TestExtractAgentFields_PrimaryKeyWinsOverGenAI(t *testing.T) {
	s := storage.Span{Attributes: map[string]any{
		"session.id":             "primary",
		"gen_ai.conversation.id": "fallback",
	}}

	ExtractAgentFields(&s)

	if s.SessionID == nil || *s.SessionID != "primary" {
		t.Errorf("SessionID = %v, want primary", s.SessionID)
	}
}

func TestExtractAgentFields_StepKindFallsBackToSpanKind(t *testing.T) {
	s := storage.Span{
		Attributes: map[string]any{"agent.run.id": "run-3"},
		SpanKind:   strp("LLM"),
	}

	ExtractAgentFields(&s)

	if s.AgentStepKind == nil || *s.AgentStepKind != "LLM" {
		t.Errorf("AgentStepKind = %v, want LLM", s.AgentStepKind)
	}
}

func TestExtractAgentFields_NoAgentAttributesLeavesSpanUntouched(t *testing.T) {
	s := storage.Span{Attributes: map[string]any{"http.method": "GET"}}

	ExtractAgentFields(&s)

	if s.SessionID != nil || s.UserID != nil || s.AgentRunID != nil ||
		s.AgentName != nil || s.AgentStepKind != nil {
		t.Errorf("expected all agent fields nil, got %+v", s)
	}
}
