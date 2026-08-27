package discovery

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

// ruleFile is the shape of conf/discovery-rules.yaml.
type ruleFile struct {
	Rules []Rule `yaml:"rules"`
}

// LoadRules reads curated (port, process/image) -> receiver-config rules
// from an external YAML file (not compiled into the binary), matching
// netdata's own service-discovery config pattern — editable without a
// rebuild. See docs/plans/atlas/03-program-design.md "Least confident
// decisions" #5.
func LoadRules(path string) ([]Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading discovery rules file %q: %w", path, err)
	}

	var f ruleFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing discovery rules file %q: %w", path, err)
	}

	return f.Rules, nil
}
