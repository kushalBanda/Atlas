package fields

import "atlas/pkg/storage"

// Attribute keys read into the agent-run fields. Two spellings per field:
// the plain Atlas/OpenInference-style key first, the OTel gen_ai semantic
// convention key second. First non-empty value wins, same precedence rule
// stringAttr already applies for the LLM fields.
const (
	attrSessionID     = "session.id"
	attrGenAISession  = "gen_ai.conversation.id"
	attrUserID        = "user.id"
	attrGenAIUserID   = "gen_ai.user.id"
	attrAgentRunID    = "agent.run.id"
	attrGenAIRunID    = "gen_ai.agent.run.id"
	attrAgentName     = "agent.name"
	attrGenAIAgent    = "gen_ai.agent.name"
	attrAgentStepKind = "agent.step.kind"
)

// ExtractAgentFields populates s's agent-run columns from the attribute
// keys above, if present. A span carrying none of them is left untouched.
//
// AgentStepKind falls back to s.SpanKind, so an existing OpenInference
// span (LLM/TOOL/CHAIN/RETRIEVER) already types its own graph node without
// needing a second attribute. This makes ordering matter: ExtractAgentFields
// must run after ExtractSpanKind in All.
func ExtractAgentFields(s *storage.Span) {
	if v := stringAttr(s.Attributes, attrSessionID, attrGenAISession); v != "" {
		s.SessionID = &v
	}
	if v := stringAttr(s.Attributes, attrUserID, attrGenAIUserID); v != "" {
		s.UserID = &v
	}
	if v := stringAttr(s.Attributes, attrAgentRunID, attrGenAIRunID); v != "" {
		s.AgentRunID = &v
	}
	if v := stringAttr(s.Attributes, attrAgentName, attrGenAIAgent); v != "" {
		s.AgentName = &v
	}
	if v := stringAttr(s.Attributes, attrAgentStepKind); v != "" {
		s.AgentStepKind = &v
	} else if s.SpanKind != nil {
		kind := *s.SpanKind
		s.AgentStepKind = &kind
	}
}
