package discovery

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeLister struct {
	procs []listeningProcess
	err   error
}

func (f fakeLister) ListListeningProcesses(_ context.Context) ([]listeningProcess, error) {
	return f.procs, f.err
}

func TestProcessScan_MatchesCuratedRule(t *testing.T) {
	t.Parallel()
	rules := []Rule{{Port: 5432, ProcessMatch: "postgres", ReceiverConfig: "postgresql"}}
	d := &ProcessScanDiscoverer{
		rules:  rules,
		lister: fakeLister{procs: []listeningProcess{{Port: 5432, ProcessName: "postgres"}}},
	}

	targets, err := d.Discover(context.Background())
	require.NoError(t, err)
	require.Len(t, targets, 1)
	require.NotNil(t, targets[0].MatchedRule)
	require.Equal(t, "postgresql", targets[0].MatchedRule.ReceiverConfig)
}

func TestProcessScan_UnmatchedPortSurfacedAsUnrecognized(t *testing.T) {
	t.Parallel()
	rules := []Rule{{Port: 5432, ProcessMatch: "postgres", ReceiverConfig: "postgresql"}}
	d := &ProcessScanDiscoverer{
		rules:  rules,
		lister: fakeLister{procs: []listeningProcess{{Port: 9999, ProcessName: "some-custom-app"}}},
	}

	targets, err := d.Discover(context.Background())
	require.NoError(t, err)
	require.Len(t, targets, 1)
	require.Nil(t, targets[0].MatchedRule)
}

func TestProcessScan_ListerErrorPropagates(t *testing.T) {
	t.Parallel()
	d := &ProcessScanDiscoverer{lister: fakeLister{err: errors.New("lsof failed")}}

	_, err := d.Discover(context.Background())
	require.Error(t, err)
}

func TestProcessScan_Name(t *testing.T) {
	t.Parallel()
	require.Equal(t, "processscan", NewProcessScanDiscoverer(nil).Name())
}

func TestParseLsofOutput_ExtractsPortAndProcessName(t *testing.T) {
	t.Parallel()
	out := []byte(`COMMAND   PID   USER   FD   TYPE DEVICE SIZE/OFF NODE NAME
postgres  123   user   7u   IPv4 0x1234 0t0      TCP  *:5432 (LISTEN)
redis-ser 456   user   6u   IPv4 0x5678 0t0      TCP  127.0.0.1:6379 (LISTEN)
`)

	procs := parseLsofOutput(out)
	require.Len(t, procs, 2)
	require.Equal(t, 5432, procs[0].Port)
	require.Equal(t, "postgres", procs[0].ProcessName)
	require.Equal(t, 6379, procs[1].Port)
}
