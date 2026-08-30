package agentrun

import (
	"testing"
	"time"

	"atlas/pkg/storage"
)

func sp(traceID, spanID, parent, name, agent, kind string, offset int) storage.Span {
	base := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	runID := "run-a"
	s := storage.Span{
		TraceID: traceID, SpanID: spanID, ParentSpanID: parent,
		ServiceName: "svc", Name: name, StatusCode: "ok",
		StartTime:  base.Add(time.Duration(offset) * time.Second),
		EndTime:    base.Add(time.Duration(offset+1) * time.Second),
		AgentRunID: &runID,
	}
	if agent != "" {
		s.AgentName = &agent
	}
	if kind != "" {
		s.AgentStepKind = &kind
	}
	return s
}

func edgeSet(g Graph) map[string]bool {
	out := map[string]bool{}
	for _, e := range g.Edges {
		out[e.From+"->"+e.To] = true
	}
	return out
}

func TestBuild_LinearRun(t *testing.T) {
	spans := []storage.Span{
		sp("t1", "s1", "", "plan", "researcher", "chain", 0),
		sp("t1", "s2", "s1", "llm", "researcher", "LLM", 1),
		sp("t1", "s3", "s2", "search", "researcher", "tool", 2),
	}

	g := Build("run-a", spans, 3)

	if len(g.Nodes) != 3 {
		t.Fatalf("got %d nodes, want 3", len(g.Nodes))
	}
	if g.Nodes[0].SpanID != "s1" {
		t.Errorf("nodes not ordered by start time: first = %s", g.Nodes[0].SpanID)
	}
	if g.Nodes[1].StepKind != "LLM" {
		t.Errorf("StepKind = %q, want LLM", g.Nodes[1].StepKind)
	}
	edges := edgeSet(g)
	if !edges["s1->s2"] || !edges["s2->s3"] {
		t.Errorf("edges = %+v, want s1->s2 and s2->s3", g.Edges)
	}
	if len(g.Repeats) != 0 {
		t.Errorf("got %d repeats, want 0", len(g.Repeats))
	}
}

func TestBuild_FanOut(t *testing.T) {
	spans := []storage.Span{
		sp("t1", "s1", "", "plan", "researcher", "chain", 0),
		sp("t1", "s2", "s1", "search", "researcher", "tool", 1),
		sp("t1", "s3", "s1", "fetch", "researcher", "tool", 1),
	}

	g := Build("run-a", spans, 3)

	edges := edgeSet(g)
	if !edges["s1->s2"] || !edges["s1->s3"] {
		t.Errorf("edges = %+v, want both children attached to s1", g.Edges)
	}
}

func TestBuild_CrossTraceEdge(t *testing.T) {
	spans := []storage.Span{
		sp("t1", "s1", "", "plan", "researcher", "chain", 0),
		sp("t1", "s2", "s1", "handoff", "researcher", "tool", 1),
		// Root of a second trace, started after s2: attaches to s2.
		sp("t2", "s3", "", "writer-run", "writer", "chain", 2),
	}

	g := Build("run-a", spans, 3)

	var cross *Edge
	for i := range g.Edges {
		if g.Edges[i].CrossTrace {
			cross = &g.Edges[i]
		}
	}
	if cross == nil {
		t.Fatalf("no cross-trace edge in %+v", g.Edges)
	}
	if cross.From != "s2" || cross.To != "s3" {
		t.Errorf("cross edge = %s->%s, want s2->s3", cross.From, cross.To)
	}
}

func TestBuild_CrossTraceWithoutOverlapLeavesComponentsSeparate(t *testing.T) {
	// t2's root starts BEFORE anything in t1, so there is no earlier span
	// in another trace to attach it to.
	spans := []storage.Span{
		sp("t2", "s3", "", "writer-run", "writer", "chain", 0),
		sp("t1", "s1", "", "plan", "researcher", "chain", 1),
		sp("t1", "s2", "s1", "handoff", "researcher", "tool", 2),
	}

	g := Build("run-a", spans, 3)

	for _, e := range g.Edges {
		if e.To == "s3" {
			t.Errorf("unexpected edge into s3: %+v", e)
		}
	}
}

func TestBuild_RepeatLoopDetected(t *testing.T) {
	spans := []storage.Span{
		sp("t1", "s1", "", "plan", "researcher", "chain", 0),
		sp("t1", "s2", "s1", "search", "researcher", "tool", 1),
		sp("t1", "s3", "s1", "search", "researcher", "tool", 2),
		sp("t1", "s4", "s1", "search", "researcher", "tool", 3),
	}

	g := Build("run-a", spans, 3)

	if len(g.Repeats) != 1 {
		t.Fatalf("got %d repeats, want 1", len(g.Repeats))
	}
	r := g.Repeats[0]
	if r.Count != 3 || r.Name != "search" || r.AgentName != "researcher" {
		t.Errorf("repeat = %+v, want 3x researcher/search", r)
	}
	for _, n := range g.Nodes {
		if n.SpanID == "s3" && (n.RepeatGroup == nil || *n.RepeatGroup != 0) {
			t.Errorf("s3 RepeatGroup = %v, want 0", n.RepeatGroup)
		}
		if n.SpanID == "s1" && n.RepeatGroup != nil {
			t.Errorf("s1 RepeatGroup = %v, want nil", n.RepeatGroup)
		}
	}
}

func TestBuild_BelowThresholdIsNotARepeat(t *testing.T) {
	spans := []storage.Span{
		sp("t1", "s1", "", "search", "researcher", "tool", 0),
		sp("t1", "s2", "", "search", "researcher", "tool", 1),
	}

	g := Build("run-a", spans, 3)

	if len(g.Repeats) != 0 {
		t.Errorf("got %d repeats, want 0", len(g.Repeats))
	}
}

func TestBuild_NoAgentAttributesStillBuildsNodes(t *testing.T) {
	spans := []storage.Span{
		{TraceID: "t1", SpanID: "s1", Name: "GET /x", ServiceName: "svc",
			StatusCode: "ok",
			StartTime:  time.Now().UTC(), EndTime: time.Now().UTC()},
	}

	g := Build("run-a", spans, 3)

	if len(g.Nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(g.Nodes))
	}
	if g.Nodes[0].StepKind != "" {
		t.Errorf("StepKind = %q, want empty", g.Nodes[0].StepKind)
	}
}

func TestBuild_EmptyRun(t *testing.T) {
	g := Build("run-a", nil, 3)

	if g.RunID != "run-a" {
		t.Errorf("RunID = %q, want run-a", g.RunID)
	}
	if len(g.Nodes) != 0 || len(g.Edges) != 0 || len(g.Repeats) != 0 {
		t.Errorf("expected empty graph, got %+v", g)
	}
}
