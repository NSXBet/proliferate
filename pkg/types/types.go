package types

import "time"

type PRStatus struct {
	Name         string    `yaml:"name"`
	LastRendered string    `yaml:"lastRendered"`
	LastApplied  time.Time `yaml:"lastApplied"`
	PRNumber     int       `yaml:"prNumber"`
	PRUrl        string    `yaml:"prUrl"`
	Branch       string    `yaml:"branch"`
	Repository   string    `yaml:"repository"`
	LastDiff     string    `yaml:"lastDiff"`
	LastCommit   string    `yaml:"lastCommit"`
}

type NamespacedStatus map[string]map[string]PRStatus
