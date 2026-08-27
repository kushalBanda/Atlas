package discovery

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeDiscoverer struct {
	name    string
	targets []Target
	err     error
}

func (f *fakeDiscoverer) Name() string { return f.name }

func (f *fakeDiscoverer) Discover(_ context.Context) ([]Target, error) {
	return f.targets, f.err
}

func TestRunAll_MergesAcrossDiscoverers(t *testing.T) {
	t.Parallel()
	rule := &Rule{Port: 5432, ReceiverConfig: "postgresql"}
	d1 := &fakeDiscoverer{name: "processscan", targets: []Target{
		{Host: "localhost", Port: 5432, ProcessOrImage: "postgres", MatchedRule: rule},
	}}
	d2 := &fakeDiscoverer{name: "docker", targets: []Target{
		{Host: "localhost", Port: 9999, ProcessOrImage: "custom-app"},
	}}

	matched, unrecognized, err := RunAll(context.Background(), []Discoverer{d1, d2})
	require.NoError(t, err)
	require.Len(t, matched, 1)
	require.Equal(t, "postgres", matched[0].ProcessOrImage)
	require.Len(t, unrecognized, 1)
	require.Equal(t, "custom-app", unrecognized[0].ProcessOrImage)
}

func TestRunAll_OneDiscovererErrorDoesNotStopOthers(t *testing.T) {
	t.Parallel()
	broken := &fakeDiscoverer{name: "docker", err: errors.New("docker socket unreachable")}
	working := &fakeDiscoverer{name: "processscan", targets: []Target{
		{Host: "localhost", Port: 1234, ProcessOrImage: "app"},
	}}

	matched, unrecognized, err := RunAll(context.Background(), []Discoverer{broken, working})
	require.Error(t, err)
	require.Empty(t, matched)
	require.Len(t, unrecognized, 1)
}

func TestMatchRule_PortAndProcessMatch(t *testing.T) {
	t.Parallel()
	rules := []Rule{
		{Port: 5432, ProcessMatch: "postgres", ReceiverConfig: "postgresql"},
		{Port: 6379, ProcessMatch: "redis", ReceiverConfig: "redis"},
	}

	require.NotNil(t, matchRule(rules, 5432, "postgres"))
	require.Nil(t, matchRule(rules, 5432, "nginx"))
	require.Nil(t, matchRule(rules, 9999, "postgres"))
}
