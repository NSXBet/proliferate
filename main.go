package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"context"

	"github.com/google/go-github/v60/github"
	"github.com/k0kubun/pp/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

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

type Config struct {
	GithubToken string `yaml:"github-token"`
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

type NamespacedStatus map[string]map[string]PRStatus // namespace -> name -> status

var (
	cfgFile       string
	valuesFile    string
	prFile        string
	dryRun        bool
	printTemplate bool
	rootCmd       = &cobra.Command{
		Use:   "masspr",
		Short: "Mass PR creation tool",
	}
)

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml)")

	prCmd := &cobra.Command{
		Use:   "pr",
		Short: "Pull request operations",
	}

	applyCmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply values to a pull request definition",
		RunE:  runApply,
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of pull requests",
		RunE:  runStatus,
	}

	applyCmd.Flags().StringVarP(&valuesFile, "values", "f", "", "Path to values YAML file")
	applyCmd.Flags().StringVarP(&prFile, "pr", "p", "", "Path to pull request YAML file")
	applyCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print the parsed pull requests without applying")
	applyCmd.Flags().BoolVar(&printTemplate, "print-template", false, "Print the rendered template")
	applyCmd.MarkFlagRequired("values")
	applyCmd.MarkFlagRequired("pr")

	prCmd.AddCommand(applyCmd)
	prCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(prCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
	}

	// Read environment variables
	viper.SetEnvPrefix("MASSPR")
	viper.AutomaticEnv()

	// Map specific env vars
	viper.BindEnv("github-token", "GHA_PAT", "GITHUB_TOKEN")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Printf("Error reading config file: %v\n", err)
		}
	}
}

var templateFuncs = template.FuncMap{
	"splitLast": func(sep, s string) string {
		parts := strings.Split(s, sep)
		return parts[len(parts)-1]
	},
	"lower": strings.ToLower,
}

func renderTemplate(tmpl string, values interface{}) (string, error) {
	t, err := template.New("pr").Funcs(templateFuncs).Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %v", err)
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, map[string]interface{}{
		"Values": values,
	})
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %v", err)
	}

	return buf.String(), nil
}

func loadGithubToken() (string, error) {
	token := viper.GetString("github-token")
	if token == "" {
		return "", fmt.Errorf("no GitHub token found in environment or config file")
	}
	return token, nil
}

func gitClone(repo, token string) (string, error) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "masspr-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}

	// Format clone URL with token
	cloneURL := fmt.Sprintf("https://%s@%s.git", token, repo)

	// Run git clone
	cmd := exec.Command("git", "clone", cloneURL, tmpDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(tmpDir) // Clean up on error
		return "", fmt.Errorf("failed to clone repository: %s: %v", output, err)
	}

	return tmpDir, nil
}

func gitDiff(dir string) (string, error) {
	// Configure git
	configCmd := exec.Command("git", "-C", dir, "config", "--local", "diff.noprefix", "true")
	if err := configCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to configure git diff: %v", err)
	}

	// Add all changes to see them in diff
	addCmd := exec.Command("git", "-C", dir, "add", "-A")
	if err := addCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to stage changes: %v", err)
	}

	// Run diff against HEAD
	cmd := exec.Command("git", "-C", dir, "diff", "--cached", "--stat")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get git diff: %v", err)
	}

	// Clean up the diff output
	diffOutput := strings.TrimSpace(string(output))
	if diffOutput == "" {
		return "", nil
	}

	// Format the diff output with proper indentation
	lines := strings.Split(diffOutput, "\n")
	formattedDiff := strings.Join(lines, "\n ")

	return formattedDiff, nil
}

func runScript(dir string, script interface{}, context map[string]string) error {
	cmdStr, ok := script.(string)
	if !ok {
		return fmt.Errorf("unsupported script type: %T", script)
	}

	// Create environment variables from context
	env := os.Environ()

	// Add current directory as MASSPR_ROOT
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %v", err)
	}
	env = append(env, fmt.Sprintf("MASSPR_ROOT=%s", currentDir))

	for k, v := range context {
		env = append(env, fmt.Sprintf("%s=%s", strings.ToUpper(k), v))
	}

	// Run command
	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Dir = dir
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("script failed: %s: %v", output, err)
	}

	pp.Printf("\nScript output (%s):\n%s", script, string(output))
	return nil
}

func gitAdd(dir string) error {
	cmd := exec.Command("git", "-C", dir, "add", ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add changes: %s: %v", output, err)
	}
	return nil
}

func gitCommit(dir string, message string) error {
	cmd := exec.Command("git", "-C", dir, "commit", "-m", message)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to commit changes: %s: %v", output, err)
	}
	return nil
}

func createPR(token, owner, repo, branch, base, title, body string, labels []string, assignees []string) (*github.PullRequest, error) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Check if PR already exists with exact branch match
	existingPRs, _, err := client.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{
		Head: fmt.Sprintf("%s:%s", owner, branch), // Use owner:branch format
		Base: base,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list PRs: %v", err)
	}

	var pr *github.PullRequest
	if len(existingPRs) > 0 {
		// Find PR with exact branch match
		var matchingPR *github.PullRequest
		for _, existingPR := range existingPRs {
			if existingPR.GetHead().GetRef() == branch {
				matchingPR = existingPR
				break
			}
		}

		if matchingPR != nil {
			// Update existing PR
			pr, _, err = client.PullRequests.Edit(ctx, owner, repo, matchingPR.GetNumber(), &github.PullRequest{
				Title: github.String(title),
				Body:  github.String(body),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to update PR: %v", err)
			}
			pp.Printf("\nUpdated existing PR: %s\n", pr.GetHTMLURL())
		} else {
			// Create new PR as no exact match found
			newPR := &github.NewPullRequest{
				Title:               github.String(title),
				Head:                github.String(branch),
				Base:                github.String(base),
				Body:                github.String(body),
				MaintainerCanModify: github.Bool(true),
			}
			pr, _, err = client.PullRequests.Create(ctx, owner, repo, newPR)
			if err != nil {
				return nil, fmt.Errorf("failed to create PR: %v", err)
			}
			pp.Printf("\nCreated new PR: %s\n", pr.GetHTMLURL())
		}
	} else {
		// Create new PR as none exist
		newPR := &github.NewPullRequest{
			Title:               github.String(title),
			Head:                github.String(branch),
			Base:                github.String(base),
			Body:                github.String(body),
			MaintainerCanModify: github.Bool(true),
		}
		pr, _, err = client.PullRequests.Create(ctx, owner, repo, newPR)
		if err != nil {
			return nil, fmt.Errorf("failed to create PR: %v", err)
		}
		pp.Printf("\nCreated new PR: %s\n", pr.GetHTMLURL())
	}

	// Update labels
	_, _, err = client.Issues.ReplaceLabelsForIssue(ctx, owner, repo, pr.GetNumber(), labels)
	if err != nil {
		return nil, fmt.Errorf("failed to update labels: %v", err)
	}

	// Update assignees (first remove existing ones)
	if len(pr.Assignees) > 0 {
		var currentAssignees []string
		for _, a := range pr.Assignees {
			currentAssignees = append(currentAssignees, a.GetLogin())
		}
		_, _, err = client.Issues.RemoveAssignees(ctx, owner, repo, pr.GetNumber(), currentAssignees)
		if err != nil {
			return nil, fmt.Errorf("failed to remove existing assignees: %v", err)
		}
	}

	if len(assignees) > 0 {
		_, _, err = client.Issues.AddAssignees(ctx, owner, repo, pr.GetNumber(), assignees)
		if err != nil {
			return nil, fmt.Errorf("failed to add assignees: %v", err)
		}
	}

	return pr, nil
}

func parseRepoString(repoStr string) (owner string, repo string, err error) {
	parts := strings.Split(repoStr, "/")
	if len(parts) != 3 {
		return "", "", fmt.Errorf("invalid repo format, expected 'github.com/owner/repo'")
	}
	return parts[1], parts[2], nil
}

func runApply(cmd *cobra.Command, args []string) error {
	// Load GitHub token
	token, err := loadGithubToken()
	if err != nil {
		return err
	}

	// Read values file
	valuesData, err := os.ReadFile(valuesFile)
	if err != nil {
		return fmt.Errorf("failed to read values file: %v", err)
	}

	// Parse values with anchor support
	var values map[string]interface{}
	decoder := yaml.NewDecoder(bytes.NewBuffer(valuesData))
	decoder.KnownFields(false) // Allow unknown fields
	if err := decoder.Decode(&values); err != nil {
		return fmt.Errorf("failed to parse values file: %v", err)
	}

	// Read PR template
	prTemplate, err := os.ReadFile(prFile)
	if err != nil {
		return fmt.Errorf("failed to read PR template: %v", err)
	}

	// Render template
	rendered, err := renderTemplate(string(prTemplate), values)
	if err != nil {
		return fmt.Errorf("failed to render template: %v", err)
	}

	if printTemplate {
		fmt.Println(rendered)
		return nil
	}

	// Parse rendered YAML into PullRequest structs
	var prs []PullRequest
	decoder = yaml.NewDecoder(bytes.NewBufferString(rendered))
	for {
		var pr PullRequest
		if err := decoder.Decode(&pr); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to parse rendered template: %v", err)
		}
		if pr.APIVersion != "" { // Skip empty documents
			prs = append(prs, pr)
		}
	}

	if len(prs) == 0 {
		// Try parsing as single PR if no documents were found
		var pr PullRequest
		if err := yaml.Unmarshal([]byte(rendered), &pr); err != nil {
			return fmt.Errorf("failed to parse PR template: %v", err)
		}
		if pr.APIVersion != "" {
			prs = append(prs, pr)
		}
	}

	if len(prs) == 0 {
		return fmt.Errorf("no valid pull requests found in template")
	}

	pp.Printf("=== Found %d Pull Request(s) ===\n", len(prs))
	for i, pr := range prs {
		pp.Printf("\n=== Pull Request %d ===\n", i+1)
		pp.Println(pr)

		// 1. Clone repository
		repoDir, err := gitClone(pr.Spec.Repo, token)
		if err != nil {
			return err
		}
		if !dryRun {
			defer os.RemoveAll(repoDir)
		}
		pp.Printf("Cloned repository to: %s\n", repoDir)

		// 2. Create branch
		if err := gitCreateBranch(repoDir, pr.Spec.Branch); err != nil {
			return err
		}

		// 3. Run scripts
		for _, script := range pr.Spec.Scripts {
			if err := runScript(repoDir, script, pr.Spec.ScriptsContext); err != nil {
				return err
			}
		}

		// 4. Show diff
		diffOutput, err := gitDiff(repoDir)
		if err != nil {
			return err
		}
		if len(diffOutput) > 0 {
			pp.Printf("\nRepository changes:\n%s", diffOutput)
		} else {
			pp.Printf("\nNo changes in repository\n")
		}

		// 5. Stage changes
		if err := gitAdd(repoDir); err != nil {
			return err
		}

		// 6. Commit changes
		if err := gitCommit(repoDir, pr.Spec.CommitMessage); err != nil {
			return err
		}

		if dryRun {
			continue
		}

		// 7. Push branch
		if err := gitPush(repoDir, pr.Spec.Branch); err != nil {
			return err
		}

		// 8. Create pull request
		owner, repoName, err := parseRepoString(pr.Spec.Repo)
		if err != nil {
			return err
		}

		createdPR, err := createPR(
			token,
			owner,
			repoName,
			pr.Spec.Branch,
			"main",
			pr.Spec.PRTitle,
			pr.Spec.PRBody,
			pr.Spec.PRLabels,
			pr.Spec.PRAssignees,
		)
		if err != nil {
			return err
		}

		// Get commit ID
		commitCmd := exec.Command("git", "-C", repoDir, "rev-parse", "HEAD")
		commitOutput, err := commitCmd.Output()
		if err != nil {
			return fmt.Errorf("failed to get commit ID: %v", err)
		}
		commitID := strings.TrimSpace(string(commitOutput))

		// Update status with more information
		if err := updateStatus(
			pr.Metadata.Namespace,
			pr.Metadata.Name,
			rendered,
			diffOutput,
			commitID,
			&StatusInfo{
				PRNumber:   createdPR.GetNumber(),
				PRUrl:      createdPR.GetHTMLURL(),
				Branch:     pr.Spec.Branch,
				Repository: pr.Spec.Repo,
			},
		); err != nil {
			return fmt.Errorf("failed to update status: %v", err)
		}
	}

	return nil
}

func gitCreateBranch(dir string, branch string) error {
	cmd := exec.Command("git", "-C", dir, "checkout", "-b", branch)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create branch: %s: %v", output, err)
	}
	return nil
}

func gitPush(dir string, branch string) error {
	cmd := exec.Command("git", "-C", dir, "push", "--force", "origin", branch)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to push branch: %s: %v", output, err)
	}
	return nil
}

func loadStatus() (NamespacedStatus, error) {
	statusFile := ".masspr/status.yaml"
	status := make(NamespacedStatus)

	if _, err := os.Stat(statusFile); os.IsNotExist(err) {
		return status, nil
	}

	data, err := os.ReadFile(statusFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read status file: %v", err)
	}

	if err := yaml.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("failed to parse status file: %v", err)
	}

	return status, nil
}

func saveStatus(status NamespacedStatus) error {
	statusDir := ".masspr"
	if err := os.MkdirAll(statusDir, 0755); err != nil {
		return fmt.Errorf("failed to create status directory: %v", err)
	}

	data, err := yaml.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal status: %v", err)
	}

	if err := os.WriteFile(filepath.Join(statusDir, "status.yaml"), data, 0644); err != nil {
		return fmt.Errorf("failed to write status file: %v", err)
	}

	return nil
}

type StatusInfo struct {
	PRNumber   int
	PRUrl      string
	Branch     string
	Repository string
}

func updateStatus(namespace, name string, rendered string, diff string, commitId string, info *StatusInfo) error {
	status, err := loadStatus()
	if err != nil {
		return err
	}

	if status[namespace] == nil {
		status[namespace] = make(map[string]PRStatus)
	}

	prStatus, exists := status[namespace][name]
	if !exists {
		prStatus = PRStatus{
			Name: name,
		}
	}

	prStatus.LastRendered = rendered
	prStatus.LastApplied = time.Now()
	prStatus.LastDiff = diff
	prStatus.LastCommit = commitId
	if info != nil {
		prStatus.PRNumber = info.PRNumber
		prStatus.PRUrl = info.PRUrl
		prStatus.Branch = info.Branch
		prStatus.Repository = info.Repository
	}

	status[namespace][name] = prStatus
	return saveStatus(status)
}

func getPRStatus(client *github.Client, owner, repo string, number int) (string, error) {
	ctx := context.Background()
	pr, _, err := client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return "", fmt.Errorf("failed to get PR status: %v", err)
	}

	state := pr.GetState()
	merged := pr.GetMerged()
	if merged {
		return "merged", nil
	}
	return state, nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	status, err := loadStatus()
	if err != nil {
		return err
	}

	if len(status) == 0 {
		pp.Println("No pull requests found")
		return nil
	}

	// If no namespace provided, list available namespaces
	if len(args) == 0 {
		pp.Println("=== Available Namespaces ===")
		for namespace, prs := range status {
			pp.Printf("\n%s (%d PRs)\n", namespace, len(prs))
		}
		return nil
	}

	namespace := args[0]
	prs, exists := status[namespace]
	if !exists {
		return fmt.Errorf("namespace %s not found", namespace)
	}

	// Setup GitHub client
	token, err := loadGithubToken()
	if err != nil {
		return err
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	pp.Printf("=== Pull Request Status for %s ===\n", namespace)
	for name, pr := range prs {
		owner, repoName, err := parseRepoString(pr.Repository)
		if err != nil {
			return err
		}

		prState, err := getPRStatus(client, owner, repoName, pr.PRNumber)
		if err != nil {
			pp.Printf("\n PR: %s (Failed to get GitHub status: %v)\n", name, err)
			continue
		}

		// Emoji based on PR state
		stateEmoji := map[string]string{
			"open":   "🟢",
			"closed": "🔴",
			"merged": "🟣",
		}[prState]

		pp.Printf("\n PR: %s %s\n", name, stateEmoji)
		pp.Printf("├── Repository: %s\n", pr.Repository)
		pp.Printf("├── Branch: %s\n", pr.Branch)
		pp.Printf("├── Pull Request: #%d (%s)\n", pr.PRNumber, prState)
		pp.Printf("├── URL: %s\n", pr.PRUrl)
		pp.Printf("├── Last Applied: %s\n", pr.LastApplied.Format(time.RFC3339))
		pp.Printf("├── Last Commit: %s\n", pr.LastCommit)
		if pr.LastDiff != "" {
			pp.Printf("└── Changes:\n    %s\n", strings.ReplaceAll(pr.LastDiff, "\n", "\n    "))
		} else {
			pp.Printf("└── No changes\n")
		}
	}

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
