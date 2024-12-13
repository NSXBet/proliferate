package masspr

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v60/github"
	"github.com/nsxbet/masspr/pkg/mygit"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

type Service struct {
	githubClient *github.Client
	githubToken  string
	statusDir    string
	git          *mygit.Git
}

func New(token string) *Service {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	return &Service{
		githubToken:  token,
		githubClient: github.NewClient(tc),
		statusDir:    ".masspr",
		git:          mygit.NewGit(token),
	}
}

func (s *Service) RunScript(dir string, script string, context map[string]string) error {
	env := os.Environ()
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %v", err)
	}
	env = append(env, fmt.Sprintf("MASSPR_ROOT=%s", currentDir))

	for k, v := range context {
		env = append(env, fmt.Sprintf("%s=%s", strings.ToUpper(k), v))
	}

	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = dir
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("script failed: %s: %v", output, err)
	}

	return nil
}

func (s *Service) CreatePR(ctx context.Context, owner, repo, branch, base, title, body string, labels []string, assignees []string) (*github.PullRequest, error) {
	existingPRs, _, err := s.githubClient.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{
		Head: fmt.Sprintf("%s:%s", owner, branch),
		Base: base,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list PRs: %v", err)
	}

	var pr *github.PullRequest
	if len(existingPRs) > 0 {
		var matchingPR *github.PullRequest
		for _, existingPR := range existingPRs {
			if existingPR.GetHead().GetRef() == branch {
				matchingPR = existingPR
				break
			}
		}

		if matchingPR != nil {
			pr, _, err = s.githubClient.PullRequests.Edit(ctx, owner, repo, matchingPR.GetNumber(), &github.PullRequest{
				Title: github.String(title),
				Body:  github.String(body),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to update PR: %v", err)
			}
		} else {
			newPR := &github.NewPullRequest{
				Title:               github.String(title),
				Head:                github.String(branch),
				Base:                github.String(base),
				Body:                github.String(body),
				MaintainerCanModify: github.Bool(true),
			}
			pr, _, err = s.githubClient.PullRequests.Create(ctx, owner, repo, newPR)
			if err != nil {
				return nil, fmt.Errorf("failed to create PR: %v", err)
			}
		}
	} else {
		newPR := &github.NewPullRequest{
			Title:               github.String(title),
			Head:                github.String(branch),
			Base:                github.String(base),
			Body:                github.String(body),
			MaintainerCanModify: github.Bool(true),
		}
		pr, _, err = s.githubClient.PullRequests.Create(ctx, owner, repo, newPR)
		if err != nil {
			return nil, fmt.Errorf("failed to create PR: %v", err)
		}
	}

	_, _, err = s.githubClient.Issues.ReplaceLabelsForIssue(ctx, owner, repo, pr.GetNumber(), labels)
	if err != nil {
		return nil, fmt.Errorf("failed to update labels: %v", err)
	}

	if len(pr.Assignees) > 0 {
		var currentAssignees []string
		for _, a := range pr.Assignees {
			currentAssignees = append(currentAssignees, a.GetLogin())
		}
		_, _, err = s.githubClient.Issues.RemoveAssignees(ctx, owner, repo, pr.GetNumber(), currentAssignees)
		if err != nil {
			return nil, fmt.Errorf("failed to remove existing assignees: %v", err)
		}
	}

	if len(assignees) > 0 {
		_, _, err = s.githubClient.Issues.AddAssignees(ctx, owner, repo, pr.GetNumber(), assignees)
		if err != nil {
			return nil, fmt.Errorf("failed to add assignees: %v", err)
		}
	}

	return pr, nil
}

func (s *Service) ParseRepoString(repoStr string) (owner string, repo string, err error) {
	parts := strings.Split(repoStr, "/")
	if len(parts) != 3 {
		return "", "", fmt.Errorf("invalid repo format, expected 'github.com/owner/repo'")
	}
	return parts[1], parts[2], nil
}

func (s *Service) GetPRStatus(ctx context.Context, owner, repo string, number int) (string, error) {
	pr, _, err := s.githubClient.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return "", fmt.Errorf("failed to get PR status: %v", err)
	}

	if pr.GetMerged() {
		return "merged", nil
	}
	return pr.GetState(), nil
}

func (s *Service) SaveStatus(namespace string, status PRStatus) error {
	allStatus, err := s.loadAll()
	if err != nil {
		return err
	}

	if allStatus[namespace] == nil {
		allStatus[namespace] = make(map[string]PRStatus)
	}

	allStatus[namespace][status.Name] = status
	return s.saveAll(allStatus)
}

func (s *Service) GetNamespaces() ([]string, error) {
	status, err := s.loadAll()
	if err != nil {
		return nil, err
	}

	var namespaces []string
	for ns := range status {
		namespaces = append(namespaces, ns)
	}
	return namespaces, nil
}

func (s *Service) GetByNamespace(namespace string) (map[string]PRStatus, error) {
	status, err := s.loadAll()
	if err != nil {
		return nil, err
	}

	return status[namespace], nil
}

func (s *Service) loadAll() (NamespacedStatus, error) {
	statusFile := filepath.Join(s.statusDir, "status.yaml")
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

func (s *Service) saveAll(status NamespacedStatus) error {
	if err := os.MkdirAll(s.statusDir, 0755); err != nil {
		return fmt.Errorf("failed to create status directory: %v", err)
	}

	data, err := yaml.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal status: %v", err)
	}

	if err := os.WriteFile(filepath.Join(s.statusDir, "status.yaml"), data, 0644); err != nil {
		return fmt.Errorf("failed to write status file: %v", err)
	}

	return nil
}
