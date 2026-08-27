package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadRules_ParsesYAML(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "rules.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
rules:
  - port: 5432
    process_match: postgres
    receiver_config: postgresql
  - port: 6379
    process_match: redis-server
    receiver_config: redis
`), 0o644))

	rules, err := LoadRules(path)
	require.NoError(t, err)
	require.Len(t, rules, 2)
	require.Equal(t, 5432, rules[0].Port)
	require.Equal(t, "postgres", rules[0].ProcessMatch)
	require.Equal(t, "postgresql", rules[0].ReceiverConfig)
}

func TestLoadRules_MissingFileErrors(t *testing.T) {
	t.Parallel()
	_, err := LoadRules(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	require.Error(t, err)
}

func TestLoadRules_InvalidYAMLErrors(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "bad.yaml")
	require.NoError(t, os.WriteFile(path, []byte("not: valid: yaml: [structure"), 0o644))

	_, err := LoadRules(path)
	require.Error(t, err)
}
