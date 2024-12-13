package printer

import (
	"strings"
	"time"

	"github.com/k0kubun/pp/v3"
	"github.com/nsxbet/masspr/pkg/types"
)

type Printer interface {
	PrintNamespacesSummary(namespaces []string, counts map[string]int)
	PrintNamespaceHeader(namespace string)
	PrintPRStatus(name string, pr types.PRStatus, state string)
	PrintError(format string, args ...interface{})
	PrintPRConfig(pr interface{})
	PrintInfo(format string, args ...interface{})
	PrintDiff(diff string)
}

type ConsolePrinter struct{}

func NewConsolePrinter() *ConsolePrinter {
	return &ConsolePrinter{}
}

func (p *ConsolePrinter) PrintNamespacesSummary(namespaces []string, counts map[string]int) {
	if len(namespaces) == 0 {
		pp.Println("No pull requests found")
		return
	}

	pp.Println("=== Available Namespaces ===")
	for _, namespace := range namespaces {
		pp.Printf("\n%s (%d PRs)\n", namespace, counts[namespace])
	}
}

func (p *ConsolePrinter) PrintNamespaceHeader(namespace string) {
	pp.Printf("=== Pull Request Status for %s ===\n", namespace)
}

func (p *ConsolePrinter) PrintPRStatus(name string, pr types.PRStatus, state string) {
	stateEmoji := map[string]string{
		"open":   "🟢",
		"closed": "🔴",
		"merged": "🟣",
	}[state]

	pp.Printf("\n PR: %s %s\n", name, stateEmoji)
	pp.Printf("├── Repository: %s\n", pr.Repository)
	pp.Printf("├── Branch: %s\n", pr.Branch)
	pp.Printf("├── Pull Request: #%d (%s)\n", pr.PRNumber, state)
	pp.Printf("├── URL: %s\n", pr.PRUrl)
	pp.Printf("├── Last Applied: %s\n", pr.LastApplied.Format(time.RFC3339))
	pp.Printf("├── Last Commit: %s\n", pr.LastCommit)
	if pr.LastDiff != "" {
		pp.Printf("└── Changes:\n    %s\n", strings.ReplaceAll(pr.LastDiff, "\n", "\n    "))
	} else {
		pp.Printf("└── No changes\n")
	}
}

func (p *ConsolePrinter) PrintError(format string, args ...interface{}) {
	pp.Printf(format, args...)
}

func (p *ConsolePrinter) PrintPRConfig(pr interface{}) {
	pp.Println(pr)
}

func (p *ConsolePrinter) PrintInfo(format string, args ...interface{}) {
	pp.Printf(format+"\n", args...)
}

func (p *ConsolePrinter) PrintDiff(diff string) {
	pp.Printf("\nRepository changes:\n%s", diff)
}
