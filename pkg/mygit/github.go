package mygit

import (
	"context"
	"fmt"
	"strings"
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
