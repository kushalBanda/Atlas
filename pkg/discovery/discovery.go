// Package discovery finds services on the host that might be worth
// ingesting traces from, matches them against curated rules, and reports
// matched + unrecognized targets. It runs on its own timer, separate from
// the request path — see docs/plans/atlas/02-architecture.md "Flow".
// v1 reports only; it does not hot-patch the OTel Collector (see
// docs/plans/atlas/03-program-design.md "Least confident decisions" #3).
package discovery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

// Rule is one curated (port, process/image) -> receiver-config match rule,
// loaded from conf/discovery-rules.yaml.
type Rule struct {
	Port           int    `yaml:"port"`
	ProcessMatch   string `yaml:"process_match"`
	ReceiverConfig string `yaml:"receiver_config"`
}

// Target is one candidate service a Discoverer found. MatchedRule is nil
// for a target that didn't match any curated rule (surfaced as
// "unrecognized", not silently dropped). ResourceAttributes carries
// discoverer-specific metadata (e.g. K8s pod/namespace/node) for reporting
// only — v1 does not use it to auto-wire the OTel Collector (see the
// package doc).
type Target struct {
	Host               string
	Port               int
	ProcessOrImage     string
	MatchedRule        *Rule
	ResourceAttributes map[string]string
}

// Discoverer finds Targets on the host. Each implementation (process-scan,
// Docker, K8s) reports every target it finds, matched or not — matching
// against curated Rules happens inside Discover.
type Discoverer interface {
	Name() string
	Discover(ctx context.Context) ([]Target, error)
}

// RunAll runs every discoverer and merges their results, splitting into
// matched and unrecognized targets. A single discoverer's error is logged
// and its results skipped — it does not stop the others (a broken Docker
// socket must not also take down process-scan reporting).
func RunAll(ctx context.Context, discoverers []Discoverer) (matched, unrecognized []Target, err error) {
	var errs []error

	for _, d := range discoverers {
		targets, derr := d.Discover(ctx)
		if derr != nil {
			slog.ErrorContext(ctx, "discoverer failed", "discoverer", d.Name(), "error", derr)
			errs = append(errs, fmt.Errorf("%s: %w", d.Name(), derr))
			continue
		}
		for _, t := range targets {
			if t.MatchedRule != nil {
				matched = append(matched, t)
			} else {
				unrecognized = append(unrecognized, t)
			}
		}
	}

	return matched, unrecognized, errors.Join(errs...)
}

// matchRule returns the first rule matching port and processOrImage (a
// process name or container image), or nil if none match.
func matchRule(rules []Rule, port int, processOrImage string) *Rule {
	for i := range rules {
		r := &rules[i]
		if r.Port != 0 && r.Port != port {
			continue
		}
		if r.ProcessMatch != "" && !containsFold(processOrImage, r.ProcessMatch) {
			continue
		}
		return r
	}
	return nil
}

func containsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
