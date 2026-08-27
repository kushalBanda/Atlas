// Package ingest receives OTLP trace exports over HTTP and dispatches the
// resulting spans through the plugin registry.
package ingest

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/collector/pdata/ptrace"

	"atlas/pkg/storage"
)

// maxRequestBodyBytes bounds the OTLP export payload accepted per request,
// preventing an unbounded read from exhausting memory (DoS defense).
const maxRequestBodyBytes = 32 << 20 // 32 MiB

// Dispatcher routes a batch of spans to the plugin module(s) that own them.
// Satisfied by *plugin.Registry; declared locally to avoid ingest importing
// plugin (and to keep this handler-testable without a real registry).
type Dispatcher interface {
	Dispatch(ctx context.Context, spans []storage.Span) error
}

// Server receives OTLP/HTTP trace exports and dispatches spans to dispatcher.
type Server struct {
	dispatcher Dispatcher
}

// NewServer returns an ingest Server dispatching through dispatcher.
func NewServer(dispatcher Dispatcher) *Server {
	return &Server{dispatcher: dispatcher}
}

// ServeOTLP implements the OTLP/HTTP traces receiver: POST /v1/traces,
// accepting application/x-protobuf (default) or application/json bodies.
func (s *Server) ServeOTLP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodyBytes+1))
	if err != nil {
		http.Error(w, "reading request body", http.StatusBadRequest)
		return
	}
	if len(body) > maxRequestBodyBytes {
		http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
		return
	}

	traces, err := unmarshalTraces(r.Header.Get("Content-Type"), body)
	if err != nil {
		slog.WarnContext(r.Context(), "rejecting malformed OTLP export", "error", err)
		http.Error(w, "malformed OTLP payload", http.StatusBadRequest)
		return
	}

	spans := tracesToSpans(traces)
	if err := s.dispatcher.Dispatch(r.Context(), spans); err != nil {
		slog.ErrorContext(r.Context(), "dispatching spans failed", "error", err, "span_count", len(spans))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	slog.InfoContext(r.Context(), "ingested spans", "span_count", len(spans))
	w.Header().Set("Content-Type", contentTypeFor(r.Header.Get("Content-Type")))
	w.WriteHeader(http.StatusOK)
}

func contentTypeFor(reqContentType string) string {
	if reqContentType == "application/json" {
		return "application/json"
	}
	return "application/x-protobuf"
}

var errUnsupportedContentType = errors.New("unsupported content type")

func unmarshalTraces(contentType string, body []byte) (ptrace.Traces, error) {
	switch contentType {
	case "", "application/x-protobuf":
		var u ptrace.ProtoUnmarshaler
		traces, err := u.UnmarshalTraces(body)
		if err != nil {
			return ptrace.Traces{}, fmt.Errorf("unmarshaling protobuf traces: %w", err)
		}
		return traces, nil
	case "application/json":
		var u ptrace.JSONUnmarshaler
		traces, err := u.UnmarshalTraces(body)
		if err != nil {
			return ptrace.Traces{}, fmt.Errorf("unmarshaling json traces: %w", err)
		}
		return traces, nil
	default:
		return ptrace.Traces{}, fmt.Errorf("%w: %q", errUnsupportedContentType, contentType)
	}
}

func tracesToSpans(traces ptrace.Traces) []storage.Span {
	var out []storage.Span

	resourceSpansSlice := traces.ResourceSpans()
	for i := 0; i < resourceSpansSlice.Len(); i++ {
		rs := resourceSpansSlice.At(i)
		resourceAttrs := rs.Resource().Attributes().AsRaw()
		serviceName := serviceNameFromResource(resourceAttrs)

		scopeSpansSlice := rs.ScopeSpans()
		for j := 0; j < scopeSpansSlice.Len(); j++ {
			spanSlice := scopeSpansSlice.At(j).Spans()
			for k := 0; k < spanSlice.Len(); k++ {
				span := spanSlice.At(k)

				var parentSpanID string
				if psid := span.ParentSpanID(); !psid.IsEmpty() {
					parentSpanID = hex.EncodeToString(psid[:])
				}

				out = append(out, storage.Span{
					TraceID:            hex.EncodeToString(traceIDBytes(span)),
					SpanID:             hex.EncodeToString(spanIDBytes(span)),
					ParentSpanID:       parentSpanID,
					ServiceName:        serviceName,
					Name:               span.Name(),
					StartTime:          span.StartTimestamp().AsTime(),
					EndTime:            span.EndTimestamp().AsTime(),
					StatusCode:         statusCodeString(span.Status().Code()),
					Attributes:         span.Attributes().AsRaw(),
					ResourceAttributes: resourceAttrs,
				})
			}
		}
	}

	return out
}

func traceIDBytes(span ptrace.Span) []byte {
	id := span.TraceID()
	return id[:]
}

func spanIDBytes(span ptrace.Span) []byte {
	id := span.SpanID()
	return id[:]
}

func serviceNameFromResource(resourceAttrs map[string]any) string {
	if v, ok := resourceAttrs["service.name"]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return "unknown_service"
}

func statusCodeString(code ptrace.StatusCode) string {
	switch code {
	case ptrace.StatusCodeOk:
		return "ok"
	case ptrace.StatusCodeError:
		return "error"
	default:
		return "unset"
	}
}
