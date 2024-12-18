package printer

import (
	"github.com/nsxbet/proliferate/pkg/types"
)

type Printer interface {
	PrintNamespacesSummary(namespaces []string, counts map[string]int)
	PrintNamespaceHeader(namespace string)
	PrintPRStatus(name string, pr types.PRStatus, state string)
	PrintError(format string, args ...interface{})
	PrintPRConfig(pr interface{})
	PrintInfo(format string, args ...interface{})
	PrintDiff(diff string)
	PrintScriptOutput(script string, output []byte, err error)
	PrintPRSummary(namespace, name, repo, branch string, prNumber int, prURL, commit string, hasChanges bool)
}

type ConsolePrinter struct{}

func NewConsolePrinter() *ConsolePrinter {
	return &ConsolePrinter{}
}
