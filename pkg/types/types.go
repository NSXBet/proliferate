package types

import (
	"time"
)

type Printer interface {
	PrintNamespacesSummary(namespaces []string, counts map[string]int)
	PrintNamespaceHeader(namespace string)
	PrintPRStatus(name string, pr PRStatus, state string)
	PrintError(format string, args ...interface{})
	PrintPRConfig(pr interface{})
	PrintInfo(format string, args ...interface{})
	PrintDiff(diff string)
	PrintScriptOutput(script string, output []byte)
}

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

// PullRequest represents the PR configuration
type PullRequest struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Spec struct {
		Repo           string            `yaml:"repo"`
		Branch         string            `yaml:"branch"`
		CommitMessage  string            `yaml:"commitMessage"`
		PRTitle        string            `yaml:"prTitle"`
		PRBody         string            `yaml:"prBody"`
		PRLabels       []string          `yaml:"prLabels"`
		PRAssignees    []string          `yaml:"prAssignees"`
		ScriptsContext map[string]string `yaml:"scriptsContext"`
		Scripts        []string          `yaml:"scripts"`
	} `yaml:"spec"`
}

type Config interface {
	GetGithubToken() string
	GetAuthorEmail() string
	GetAuthorName() string
}
