// Package agentrun builds the decision graph for one agent run. A run is
// the set of spans sharing an agent_run_id, which may cross traces. The
// graph is derived at request time from those spans — nothing here is
// stored. See docs/superpowers/specs/2026-08-29-agent-run-debugging-design.md.
package agentrun

import (
	"sort"

	"atlas/pkg/storage"
)

// Node is one span in the run.
type Node struct {
	SpanID       string `json:"span_id"`
	TraceID      string `json:"trace_id"`
	Name         string `json:"name"`
	StepKind     string `json:"step_kind"`
	AgentName    string `json:"agent_name"`
	ServiceName  string `json:"service_name"`
	StatusCode   string `json:"status_code"`
	StartTime    string `json:"start_time"`
	DurationNano int64  `json:"duration_nano"`
	// RepeatGroup indexes into Graph.Repeats when this node is part of a
	// detected repeat group; nil otherwise.
	RepeatGroup *int `json:"repeat_group"`
}

// Edge is a causal link between two nodes.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	// CrossTrace marks an edge inferred across a trace boundary rather
	// than read from parent_span_id.
	CrossTrace bool `json:"cross_trace"`
}

// Repeat is a run of consecutive same-agent, same-name steps at or above
// the configured threshold — a tool loop or retry storm. It is a graph
// annotation only, never a root-cause verdict.
type Repeat struct {
	Index     int      `json:"index"`
	AgentName string   `json:"agent_name"`
	Name      string   `json:"name"`
	Count     int      `json:"count"`
	SpanIDs   []string `json:"span_ids"`
}

// Graph is one run's decision graph.
type Graph struct {
	RunID   string   `json:"run_id"`
	Nodes   []Node   `json:"nodes"`
	Edges   []Edge   `json:"edges"`
	Repeats []Repeat `json:"repeats"`
}

// startTimeLayout keeps node timestamps at nanosecond precision, so the
// frontend can order steps that share a millisecond.
const startTimeLayout = "2006-01-02T15:04:05.000000000Z"

func str(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// Build derives the decision graph for runID from its spans.
//
// Edges come from parent_span_id within a trace. A span that roots its own
// trace is attached to the latest-starting span in a different trace that
// began at or before it — the handoff case, where an agent step starts a
// new trace. When no such span exists, the trace stays its own connected
// component; that is a valid render, not an error.
//
// repeatThreshold is the minimum number of consecutive same-agent,
// same-name steps that count as a repeat group. A value below 2 disables
// repeat detection.
func Build(runID string, spans []storage.Span, repeatThreshold int) Graph {
	g := Graph{RunID: runID, Nodes: []Node{}, Edges: []Edge{}, Repeats: []Repeat{}}
	if len(spans) == 0 {
		return g
	}

	ordered := make([]storage.Span, len(spans))
	copy(ordered, spans)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].StartTime.Equal(ordered[j].StartTime) {
			return ordered[i].SpanID < ordered[j].SpanID
		}
		return ordered[i].StartTime.Before(ordered[j].StartTime)
	})

	index := make(map[string]int, len(ordered))
	for i, s := range ordered {
		index[s.SpanID] = i
		g.Nodes = append(g.Nodes, Node{
			SpanID:       s.SpanID,
			TraceID:      s.TraceID,
			Name:         s.Name,
			StepKind:     str(s.AgentStepKind),
			AgentName:    str(s.AgentName),
			ServiceName:  s.ServiceName,
			StatusCode:   s.StatusCode,
			StartTime:    s.StartTime.UTC().Format(startTimeLayout),
			DurationNano: s.EndTime.Sub(s.StartTime).Nanoseconds(),
		})
	}

	for i, s := range ordered {
		if s.ParentSpanID != "" {
			if _, ok := index[s.ParentSpanID]; ok {
				g.Edges = append(g.Edges, Edge{From: s.ParentSpanID, To: s.SpanID})
			}
			continue
		}
		// Trace root: look for a handoff parent in another trace.
		if from, ok := crossTraceParent(ordered, i); ok {
			g.Edges = append(g.Edges, Edge{From: from, To: s.SpanID, CrossTrace: true})
		}
	}

	g.Repeats, g.Nodes = detectRepeats(g.Nodes, repeatThreshold)
	return g
}

// crossTraceParent returns the span ID of the latest-starting span in a
// different trace that began at or before ordered[i], if any. ordered is
// sorted ascending by start time, so scanning backward finds it first.
func crossTraceParent(ordered []storage.Span, i int) (string, bool) {
	for j := i - 1; j >= 0; j-- {
		if ordered[j].TraceID != ordered[i].TraceID {
			return ordered[j].SpanID, true
		}
	}
	return "", false
}

// detectRepeats annotates consecutive same-agent, same-name node groups of
// at least threshold members. Nodes are already ordered by start time.
func detectRepeats(nodes []Node, threshold int) ([]Repeat, []Node) {
	repeats := []Repeat{}
	if threshold < 2 {
		return repeats, nodes
	}

	for start := 0; start < len(nodes); {
		end := start + 1
		for end < len(nodes) &&
			nodes[end].Name == nodes[start].Name &&
			nodes[end].AgentName == nodes[start].AgentName {
			end++
		}
		if count := end - start; count >= threshold {
			idx := len(repeats)
			r := Repeat{
				Index:     idx,
				AgentName: nodes[start].AgentName,
				Name:      nodes[start].Name,
				Count:     count,
				SpanIDs:   make([]string, 0, count),
			}
			for k := start; k < end; k++ {
				group := idx
				nodes[k].RepeatGroup = &group
				r.SpanIDs = append(r.SpanIDs, nodes[k].SpanID)
			}
			repeats = append(repeats, r)
		}
		start = end
	}
	return repeats, nodes
}
