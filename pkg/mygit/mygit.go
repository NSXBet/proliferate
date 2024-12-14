package mygit

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/nsxbet/proliferate/pkg/types"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type Git struct {
	config types.Config
	gh     *github.Client
}

func NewGit(cfg types.Config) *Git {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cfg.GetGithubToken()})
	tc := oauth2.NewClient(ctx, ts)
	gh := github.NewClient(tc)

	return &Git{
		config: cfg,
		gh:     gh,
	}
}

func (g *Git) Clone(repo string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "proliferate-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}

	cloneURL := fmt.Sprintf("https://%s@github.com/%s.git", g.config.GetGithubToken(), repo)
	fmt.Printf("Debug - Cloning repository: %s\n", repo)

	cmd := exec.Command("git", "clone", cloneURL, tmpDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to clone repository: %s: %v", output, err)
	}

	return tmpDir, nil
}

func (g *Git) Diff(dir string) (string, error) {
	configCmd := exec.Command("git", "-C", dir, "config", "--local", "diff.noprefix", "true")
	if err := configCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to configure git diff: %v", err)
	}

	addCmd := exec.Command("git", "-C", dir, "add", "-A")
	if err := addCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to stage changes: %v", err)
	}

	cmd := exec.Command("git", "-C", dir, "diff", "--cached", "--stat")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get git diff: %v", err)
	}

	diffOutput := strings.TrimSpace(string(output))
	if diffOutput == "" {
		return "", nil
	}

	lines := strings.Split(diffOutput, "\n")
	return strings.Join(lines, "\n "), nil
}

func (g *Git) Add(dir string) error {
	cmd := exec.Command("git", "-C", dir, "add", ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add changes: %s: %v", output, err)
	}
	return nil
}

func (g *Git) Commit(dir string, message string) error {
	// Set the local git config for this repository
	emailCmd := exec.Command("git", "-C", dir, "config", "user.email", g.config.GetAuthorEmail())
	if output, err := emailCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set git email: %s: %v", output, err)
	}

	nameCmd := exec.Command("git", "-C", dir, "config", "user.name", g.config.GetAuthorName())
	if output, err := nameCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set git name: %s: %v", output, err)
	}

	// Perform the commit
	cmd := exec.Command("git", "-C", dir, "commit", "-m", message)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to commit changes: %s: %v", output, err)
	}
	return nil
}

func (g *Git) CreateBranch(dir string, branch string) error {
	cmd := exec.Command("git", "-C", dir, "checkout", "-b", branch)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create branch: %s: %v", output, err)
	}
	return nil
}

func (g *Git) Push(dir string, branch string) error {
	// Configure the remote URL with the token
	remoteURL := fmt.Sprintf("https://%s@github.com/%s.git",
		g.config.GetGithubToken(),
		strings.TrimPrefix(strings.TrimPrefix(g.config.GetGithubToken(), "https://"), "github.com/"))

	setRemoteCmd := exec.Command("git", "-C", dir, "remote", "set-url", "origin", remoteURL)
	if err := setRemoteCmd.Run(); err != nil {
		return fmt.Errorf("failed to set remote URL: %v", err)
	}

	cmd := exec.Command("git", "-C", dir, "push", "--force", "origin", branch)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to push branch: %s: %v", output, err)
	}

	return nil
}

func (g *Git) GetCommitID(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commit ID: %v", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (g *Git) CreatePR(ctx context.Context, owner, repo, branch, base, title, body string, labels []string, assignees []string) (*github.PullRequest, error) {
	existingPRs, _, err := g.gh.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{
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
			pr, _, err = g.gh.PullRequests.Edit(ctx, owner, repo, matchingPR.GetNumber(), &github.PullRequest{
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
			pr, _, err = g.gh.PullRequests.Create(ctx, owner, repo, newPR)
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
		pr, _, err = g.gh.PullRequests.Create(ctx, owner, repo, newPR)
		if err != nil {
			return nil, fmt.Errorf("failed to create PR: %v", err)
		}
	}

	_, _, err = g.gh.Issues.ReplaceLabelsForIssue(ctx, owner, repo, pr.GetNumber(), labels)
	if err != nil {
		return nil, fmt.Errorf("failed to update labels: %v", err)
	}

	if len(pr.Assignees) > 0 {
		var currentAssignees []string
		for _, a := range pr.Assignees {
			currentAssignees = append(currentAssignees, a.GetLogin())
		}
		_, _, err = g.gh.Issues.RemoveAssignees(ctx, owner, repo, pr.GetNumber(), currentAssignees)
		if err != nil {
			return nil, fmt.Errorf("failed to remove existing assignees: %v", err)
		}
	}

	if len(assignees) > 0 {
		_, _, err = g.gh.Issues.AddAssignees(ctx, owner, repo, pr.GetNumber(), assignees)
		if err != nil {
			return nil, fmt.Errorf("failed to add assignees: %v", err)
		}
	}

	return pr, nil
}
