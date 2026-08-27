package discovery

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// listeningProcess is one local process found listening on a TCP port.
type listeningProcess struct {
	Port        int
	ProcessName string
}

// processLister lists locally listening TCP processes. Abstracted so
// ProcessScanDiscoverer is testable without shelling out to lsof.
type processLister interface {
	ListListeningProcesses(ctx context.Context) ([]listeningProcess, error)
}

// ProcessScanDiscoverer finds local processes listening on TCP ports and
// matches them against curated rules.
type ProcessScanDiscoverer struct {
	rules  []Rule
	lister processLister
}

// NewProcessScanDiscoverer returns a process-scan Discoverer matching
// against rules, using lsof to enumerate listening processes.
func NewProcessScanDiscoverer(rules []Rule) *ProcessScanDiscoverer {
	return &ProcessScanDiscoverer{rules: rules, lister: lsofLister{}}
}

// Name implements Discoverer.
func (d *ProcessScanDiscoverer) Name() string { return "processscan" }

// Discover implements Discoverer: every listening process found is
// reported, matched or not.
func (d *ProcessScanDiscoverer) Discover(ctx context.Context) ([]Target, error) {
	procs, err := d.lister.ListListeningProcesses(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing listening processes: %w", err)
	}

	targets := make([]Target, 0, len(procs))
	for _, p := range procs {
		targets = append(targets, Target{
			Host:           "localhost",
			Port:           p.Port,
			ProcessOrImage: p.ProcessName,
			MatchedRule:    matchRule(d.rules, p.Port, p.ProcessName),
		})
	}
	return targets, nil
}

// lsofLister enumerates listening TCP processes via `lsof` (macOS/Linux).
type lsofLister struct{}

func (lsofLister) ListListeningProcesses(ctx context.Context) ([]listeningProcess, error) {
	cmd := exec.CommandContext(ctx, "lsof", "-iTCP", "-sTCP:LISTEN", "-P", "-n")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running lsof: %w", err)
	}
	return parseLsofOutput(out), nil
}

// parseLsofOutput parses `lsof -iTCP -sTCP:LISTEN -P -n` output. Example line:
//
//	COMMAND   PID   USER   FD   TYPE DEVICE SIZE/OFF NODE NAME
//	postgres  123   user   7u   IPv4 0x...  0t0      TCP  *:5432 (LISTEN)
func parseLsofOutput(out []byte) []listeningProcess {
	var procs []listeningProcess
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	first := true
	for scanner.Scan() {
		if first {
			first = false
			continue // header row
		}
		fields := strings.Fields(scanner.Text())
		if len(fields) < 9 {
			continue
		}
		name := fields[0]
		nameField := fields[8] // e.g. "*:5432" or "127.0.0.1:5432"
		idx := strings.LastIndex(nameField, ":")
		if idx == -1 {
			continue
		}
		port, err := strconv.Atoi(nameField[idx+1:])
		if err != nil {
			continue
		}
		procs = append(procs, listeningProcess{Port: port, ProcessName: name})
	}
	return procs
}
