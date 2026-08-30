package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o644))
	return path
}

func TestLoad_AppliesDefaults(t *testing.T) {
	t.Parallel()
	path := writeConfigFile(t, `storage_path: /data/atlas.duckdb`)

	cfg, err := Load(path)
	require.NoError(t, err)
	require.Equal(t, "/data/atlas.duckdb", cfg.StoragePath)
	// Everything else keeps Default()'s values.
	require.Equal(t, "127.0.0.1:4318", cfg.IngestAddr)
	require.Equal(t, "127.0.0.1:8080", cfg.APIAddr)
	require.Equal(t, 30*time.Second, cfg.TraceCloseTimeout)
	require.InDelta(t, 0.30, cfg.RootCauseSelfTimePct, 0.001)
	require.Equal(t, "./conf/discovery-rules.yaml", cfg.DiscoveryRulesPath)
}

func TestLoad_ParsesAllFields(t *testing.T) {
	t.Parallel()
	path := writeConfigFile(t, `
storage_path: /data/atlas.duckdb
ingest_addr: 0.0.0.0:4318
api_addr: 0.0.0.0:8080
trace_close_timeout: 45s
root_cause_self_time_pct: 0.5
discovery_rules_path: /etc/atlas/rules.yaml
`)

	cfg, err := Load(path)
	require.NoError(t, err)
	require.Equal(t, "/data/atlas.duckdb", cfg.StoragePath)
	require.Equal(t, "0.0.0.0:4318", cfg.IngestAddr)
	require.Equal(t, "0.0.0.0:8080", cfg.APIAddr)
	require.Equal(t, 45*time.Second, cfg.TraceCloseTimeout)
	require.InDelta(t, 0.5, cfg.RootCauseSelfTimePct, 0.001)
	require.Equal(t, "/etc/atlas/rules.yaml", cfg.DiscoveryRulesPath)
}

func TestLoad_RejectsInvalidYAML(t *testing.T) {
	t.Parallel()
	path := writeConfigFile(t, "not: valid: yaml: [structure")

	_, err := Load(path)
	require.Error(t, err)
}

func TestLoad_MissingFile_ReturnsError(t *testing.T) {
	t.Parallel()
	_, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	require.Error(t, err)
}

func TestLoad_InvalidDuration_ReturnsError(t *testing.T) {
	t.Parallel()
	path := writeConfigFile(t, `trace_close_timeout: not-a-duration`)

	_, err := Load(path)
	require.Error(t, err)
}

func TestLoad_RejectsOutOfRangeSelfTimePct(t *testing.T) {
	t.Parallel()
	path := writeConfigFile(t, `root_cause_self_time_pct: 1.5`)

	_, err := Load(path)
	require.Error(t, err)
}

func TestLoad_RejectsNonPositiveTraceCloseTimeout(t *testing.T) {
	t.Parallel()
	path := writeConfigFile(t, `trace_close_timeout: 0s`)

	_, err := Load(path)
	require.Error(t, err)
}

func TestDefault_ReturnsSaneValues(t *testing.T) {
	t.Parallel()
	cfg := Default()
	require.NoError(t, cfg.validate())
}

func TestDefault_AgentRunRepeatThreshold(t *testing.T) {
	require.Equal(t, 3, Default().AgentRunRepeatThreshold)
}

func TestLoad_RejectsRepeatThresholdBelowTwo(t *testing.T) {
	path := writeConfigFile(t, "agent_run_repeat_threshold: 1\n")

	_, err := Load(path)
	require.Error(t, err, "expected an error for a repeat threshold below 2")
}

func TestLoad_OverridesRepeatThreshold(t *testing.T) {
	path := writeConfigFile(t, "agent_run_repeat_threshold: 5\n")

	cfg, err := Load(path)
	require.NoError(t, err)
	require.Equal(t, 5, cfg.AgentRunRepeatThreshold)
}
