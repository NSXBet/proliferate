package service

import (
	"context"
	"fmt"

	"github.com/google/go-github/v60/github"
	"github.com/nsxbet/masspr/pkg/mygit"
	"golang.org/x/oauth2"
)

type Service struct {
	githubClient *github.Client
	githubToken  string
	git          *mygit.Git
}

func New(token string) *Service {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	return &Service{
		githubToken:  token,
		githubClient: github.NewClient(tc),
		git:          mygit.NewGit(token),
	}
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
