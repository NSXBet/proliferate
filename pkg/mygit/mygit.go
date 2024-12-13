package mygit

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Git struct {
	token string
}

func NewGit(token string) *Git {
	return &Git{
		token: token,
	}
}

func (g *Git) Clone(repo string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "masspr-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}

	cloneURL := fmt.Sprintf("https://%s@%s.git", g.token, repo)
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
