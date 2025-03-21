package mygit

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/go-github/github"
)

func (g *Git) ParseRepoString(repoStr string) (owner string, repo string, err error) {
	parts := strings.Split(repoStr, "/")
	if len(parts) != 3 {
		return "", "", fmt.Errorf("invalid repo format, expected 'github.com/owner/repo'")
	}
	return parts[1], parts[2], nil
}

func (g *Git) GetPRStatus(ctx context.Context, owner, repo string, number int) (string, error) {
	pr, _, err := g.gh.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return "", fmt.Errorf("failed to get PR status: %v", err)
	}

	if pr.GetMerged() {
		return "merged", nil
	}
	return pr.GetState(), nil
}

// FilterRepositoriesByOrg fetches repositories from a GitHub organization matching a regex pattern
func (g *Git) FilterRepositoriesByOrg(ctx context.Context, org, pattern string) ([]string, error) {
	var allRepos []string

	// List repositories for the organization
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := g.gh.Repositories.ListByOrg(ctx, org, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to list repositories: %v", err)
		}

		// Filter repositories based on pattern
		for _, repo := range repos {
			// Skip forks, archived repositories
			if repo.GetFork() || repo.GetArchived() {
				continue
			}

			repoName := repo.GetName()
			matched, err := regexp.MatchString(pattern, repoName)
			if err != nil {
				return nil, fmt.Errorf("invalid repository filter pattern: %v", err)
			}

			if matched {
				repoURL := fmt.Sprintf("github.com/%s/%s", org, repoName)
				allRepos = append(allRepos, repoURL)
			}
		}

		// Check if there are more pages
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	if len(allRepos) == 0 {
		return nil, fmt.Errorf("no repositories match the filter pattern '%s' in organization '%s'",
			pattern, org)
	}

	return allRepos, nil
}
