package printer

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/nsxbet/proliferate/pkg/types"
	"gopkg.in/yaml.v3"
)

var (
	// Style definitions
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#989898")).
			MarginBottom(1)

	stateStyles = map[string]lipgloss.Style{
		"open":   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FF9F")),
		"closed": lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF6B8B")),
		"merged": lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A682FF")),
	}

	treeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#BCBCBC"))

	urlStyle = lipgloss.NewStyle().
			Underline(true).
			Foreground(lipgloss.Color("#00B4FF"))

	diffStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF9F")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#404040")).
			Padding(1, 2)

	prConfigStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF9F")). // Bright green
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#404040")).
			Padding(1, 2)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#BCBCBC")). // Same color as treeStyle
			MarginLeft(2)

	scriptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF9F")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#404040")).
			Padding(1, 2).
			Width(100)

	scriptHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF"))

	tabStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("#404040")).
			Padding(0, 1).
			MarginRight(2)

	activeTabStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("#00FF9F")).
			Foreground(lipgloss.Color("#00FF9F")).
			Padding(0, 1).
			MarginRight(2)

	tabContentStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#404040")).
			Padding(1, 2).
			Width(100)
)

func (p *ConsolePrinter) PrintNamespacesSummary(namespaces []string, counts map[string]int) {
	if len(namespaces) == 0 {
		fmt.Println(subtitleStyle.Render("No pull requests found"))
		return
	}

	fmt.Println(titleStyle.Render("Pull Requests"))
	for _, namespace := range namespaces {
		stateStyle := stateStyles["open"]
		fmt.Printf("%s %s\n",
			titleStyle.Render(namespace),
			stateStyle.Render("🟢"))
	}
}

func (p *ConsolePrinter) PrintNamespaceHeader(namespace string) {
	fmt.Println(titleStyle.Render(namespace))
}

func (p *ConsolePrinter) PrintPRStatus(name string, pr types.PRStatus, state string) {
	stateStyle := stateStyles[state]
	stateEmoji := map[string]string{
		"open":   "🟢",
		"closed": "🔴",
		"merged": "🟣",
	}[state]

	// Build the tree structure
	tree := []string{
		titleStyle.Render(name),
		fmt.Sprintf("├── Repository: %s", pr.Repository),
		fmt.Sprintf("├── Branch: %s", pr.Branch),
		fmt.Sprintf("├── Pull Request: #%d (%s %s)", pr.PRNumber, stateStyle.Render(state), stateStyle.Render(stateEmoji)),
		fmt.Sprintf("├── URL: %s", urlStyle.Render(pr.PRUrl)),
		fmt.Sprintf("├── Last Applied: %s", pr.LastApplied.Format(time.RFC3339)),
		fmt.Sprintf("├── Last Commit: %s", pr.LastCommit),
	}

	if pr.LastDiff != "" {
		tree = append(tree, "└── Changes:")
		fmt.Printf("%s\n", treeStyle.Render(strings.Join(tree, "\n")))
		fmt.Printf("%s\n", diffStyle.Render(pr.LastDiff))
	} else {
		tree = append(tree, "└── No changes")
		fmt.Printf("%s\n", treeStyle.Render(strings.Join(tree, "\n")))
	}
}

func (p *ConsolePrinter) PrintError(format string, args ...interface{}) {
	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ED567A"))
	fmt.Print(errorStyle.Render(fmt.Sprintf(format, args...)))
}

func (p *ConsolePrinter) PrintPRConfig(pr interface{}) {
	prStruct := pr.(types.PullRequest)
	yamlBytes, err := yaml.Marshal(prStruct)
	if err != nil {
		p.PrintError("Failed to marshal PR config: %v", err)
		return
	}

	fmt.Printf("\n%s\n", prConfigStyle.Render(string(yamlBytes)))
}

func (p *ConsolePrinter) PrintInfo(format string, args ...interface{}) {
	fmt.Printf("%s\n", infoStyle.Render(fmt.Sprintf(format, args...)))
}

func (p *ConsolePrinter) PrintDiff(diff string) {
	fmt.Printf("\n%s\n", diffStyle.Render(strings.TrimSpace("Repository changes:\n"+diff)))
}

func (p *ConsolePrinter) PrintScriptOutput(script string, output []byte, err error) {
	if len(output) == 0 {
		return
	}

	// Create tab and content with script info
	tabStyle := activeTabStyle
	if err != nil {
		tabStyle = tabStyle.Foreground(lipgloss.Color("#FF0000"))
	}
	activeTab := tabStyle.Render(fmt.Sprintf("Script %s", script))
	tabs := []string{activeTab}

	// Format the output
	outputStr := strings.TrimSpace(string(output))
	outputLines := strings.Split(outputStr, "\n")

	// Build the content
	content := []string{
		"", // Empty line for spacing
		strings.Join(outputLines, "\n"),
	}

	// Render tabs and content
	tabRow := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	renderedContent := tabContentStyle.Render(strings.Join(content, "\n"))

	fmt.Printf("\n%s\n%s\n", tabRow, renderedContent)
}

func (p *ConsolePrinter) PrintPRSummary(namespace, name, repo, branch string, prNumber int, prURL, commit string, hasChanges bool) {
	summaryStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575")).
		Bold(true)

	tree := []string{
		summaryStyle.Render("Pull Request created/updated successfully"),
		fmt.Sprintf("├── Namespace: %s", namespace),
		fmt.Sprintf("├── Name: %s", name),
		fmt.Sprintf("├── Repository: %s", repo),
		fmt.Sprintf("├── Branch: %s", branch),
		fmt.Sprintf("├── PR: #%d", prNumber),
		fmt.Sprintf("├── URL: %s", urlStyle.Render(prURL)),
		fmt.Sprintf("├── Commit: %s", commit),
		fmt.Sprintf("└── Changes: %v", hasChanges),
	}

	fmt.Printf("%s\n", treeStyle.Render(strings.Join(tree, "\n")))
}
