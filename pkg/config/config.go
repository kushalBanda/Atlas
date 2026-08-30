// Package config loads atlas-server's runtime configuration: storage path,
// listen addresses, root-cause thresholds, and the discovery rules file
// location. All fields have sane defaults — a config file only needs to
// override what differs from them.
package config

import (
	"fmt"
	"os"
	"time"

	"go.yaml.in/yaml/v3"
)

// Config is atlas-server's full runtime configuration.
type Config struct {
	StoragePath          string
	IngestAddr           string
	APIAddr              string
	TraceCloseTimeout    time.Duration
	RootCauseSelfTimePct float64
	DiscoveryRulesPath   string
	// AgentRunRepeatThreshold is the number of consecutive same-agent,
	// same-step spans that mark a repeat group in a run graph. Purely a
	// display annotation — see pkg/agentrun.
	AgentRunRepeatThreshold int
}

// Default returns Config with every field set to its v1 default.
func Default() Config {
	return Config{
		StoragePath: "./atlas.duckdb",
		IngestAddr:  "127.0.0.1:4318",
		APIAddr:     "127.0.0.1:8080",
		// TraceCloseTimeout is the fallback close trigger only (root span
		// never arrives); matches OTel Collector's tail_sampling
		// decision_wait default.
		TraceCloseTimeout: 30 * time.Second,
		// RootCauseSelfTimePct is an unvalidated v1 default — see
		// docs/plans/atlas/future.md.
		RootCauseSelfTimePct:    0.30,
		DiscoveryRulesPath:      "./conf/discovery-rules.yaml",
		AgentRunRepeatThreshold: 3,
	}
}

// rawConfig mirrors Config's YAML shape with pointer fields, so Load can
// tell "key absent, keep default" apart from "key present with zero value"
// — trace_close_timeout also needs string->Duration parsing, which plain
// yaml.Unmarshal onto a time.Duration field can't do without a custom
// UnmarshalYAML.
type rawConfig struct {
	StoragePath          *string  `yaml:"storage_path"`
	IngestAddr           *string  `yaml:"ingest_addr"`
	APIAddr              *string  `yaml:"api_addr"`
	TraceCloseTimeout    *string  `yaml:"trace_close_timeout"`
	RootCauseSelfTimePct *float64 `yaml:"root_cause_self_time_pct"`
	DiscoveryRulesPath   *string  `yaml:"discovery_rules_path"`

	AgentRunRepeatThreshold *int `yaml:"agent_run_repeat_threshold"`
}

// Load reads a YAML config file at path, applying it on top of Default().
// A field the file omits keeps its default; path must exist — callers that
// want a "no config file yet" fallback should check existence themselves
// (see cmd/atlas-server for the dev-friendly fallback to Default()).
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %q: %w", path, err)
	}

	var raw rawConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing config file %q: %w", path, err)
	}

	cfg := Default()
	if raw.StoragePath != nil {
		cfg.StoragePath = *raw.StoragePath
	}
	if raw.IngestAddr != nil {
		cfg.IngestAddr = *raw.IngestAddr
	}
	if raw.APIAddr != nil {
		cfg.APIAddr = *raw.APIAddr
	}
	if raw.TraceCloseTimeout != nil {
		d, err := time.ParseDuration(*raw.TraceCloseTimeout)
		if err != nil {
			return nil, fmt.Errorf("parsing trace_close_timeout %q: %w", *raw.TraceCloseTimeout, err)
		}
		cfg.TraceCloseTimeout = d
	}
	if raw.RootCauseSelfTimePct != nil {
		cfg.RootCauseSelfTimePct = *raw.RootCauseSelfTimePct
	}
	if raw.DiscoveryRulesPath != nil {
		cfg.DiscoveryRulesPath = *raw.DiscoveryRulesPath
	}
	if raw.AgentRunRepeatThreshold != nil {
		cfg.AgentRunRepeatThreshold = *raw.AgentRunRepeatThreshold
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("validating config file %q: %w", path, err)
	}
	return &cfg, nil
}

func (c Config) validate() error {
	if c.RootCauseSelfTimePct < 0 || c.RootCauseSelfTimePct > 1 {
		return fmt.Errorf("root_cause_self_time_pct must be between 0 and 1, got %v", c.RootCauseSelfTimePct)
	}
	if c.TraceCloseTimeout <= 0 {
		return fmt.Errorf("trace_close_timeout must be positive, got %v", c.TraceCloseTimeout)
	}
	if c.StoragePath == "" {
		return fmt.Errorf("storage_path must not be empty")
	}
	if c.AgentRunRepeatThreshold < 2 {
		return fmt.Errorf("agent_run_repeat_threshold must be at least 2, got %d", c.AgentRunRepeatThreshold)
	}
	return nil
}
