package pullrequest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/nsxbet/proliferate/pkg/mygit"
	"github.com/nsxbet/proliferate/pkg/printer"
	"github.com/nsxbet/proliferate/pkg/types"
	"gopkg.in/yaml.v3"
)

type PullRequest = types.PullRequest

type PullRequestSet struct {
	prs            []PullRequest
	git            *mygit.Git
	status         *PRStatusManager
	templateString string
	printer        printer.Printer
}

func NewPullRequestSet(yamlTemplate string, git *mygit.Git, printer printer.Printer) (*PullRequestSet, error) {
	var prs []PullRequest
	decoder := yaml.NewDecoder(bytes.NewBufferString(yamlTemplate))
	for {
		var pr PullRequest
		if err := decoder.Decode(&pr); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to parse template: %v", err)
		}
		if pr.APIVersion != "" {
			prs = append(prs, pr)
		}
	}

	if len(prs) == 0 {
		var pr PullRequest
		if err := yaml.Unmarshal([]byte(yamlTemplate), &pr); err != nil {
			return nil, fmt.Errorf("failed to parse PR template: %v", err)
		}
		if pr.APIVersion != "" {
			prs = append(prs, pr)
		}
	}

	if len(prs) == 0 {
		return nil, fmt.Errorf("no valid pull requests found in template")
	}

	return &PullRequestSet{
		prs:            prs,
		git:            git,
		status:         NewPRStatusManager(".proliferate", printer),
		templateString: yamlTemplate,
		printer:        printer,
	}, nil
}

func (prs *PullRequestSet) GetPRs() []PullRequest {
	return prs.prs
}

func (prs *PullRequestSet) ProcessPR(ctx context.Context, index int, pr PullRequest, dryRun bool) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	prs.printer.PrintNamespaceHeader(fmt.Sprintf("Pull Request %d", index+1))
	prs.printer.PrintPRConfig(pr)

	repoDir, err := prs.git.Clone(pr.Spec.Repo)
	if err != nil {
		return err
	}
	if !dryRun {
		defer os.RemoveAll(repoDir)
	}
	prs.printer.PrintInfo("Cloned repository to: %s", repoDir)

	if err := prs.git.CreateBranch(repoDir, pr.Spec.Branch); err != nil {
		return err
	}

	// Will halt if any script fails
	for _, script := range pr.Spec.Scripts {
		if err := prs.runScript(repoDir, script, pr.Spec.ScriptsContext, pr.Metadata.Name, pr.Metadata.Namespace); err != nil {
			return err
		}
	}

	diffOutput, err := prs.git.Diff(repoDir)
	if err != nil {
		return err
	}
	if len(diffOutput) > 0 {
		prs.printer.PrintDiff(diffOutput)
	} else {
		prs.printer.PrintInfo("No changes in repository")
	}

	if err := prs.git.Add(repoDir); err != nil {
		return err
	}

	if err := prs.git.Commit(repoDir, pr.Spec.CommitMessage); err != nil {
		return err
	}

	if dryRun {
		return nil
	}

	if err := prs.git.Push(repoDir, pr.Spec.Branch); err != nil {
		return err
	}

	owner, repoName, err := prs.git.ParseRepoString(pr.Spec.Repo)
	if err != nil {
		return err
	}

	createdPR, err := prs.git.CreatePR(
		ctx,
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

	commitID, err := prs.git.GetCommitID(repoDir)
	if err != nil {
		return err
	}

	if err := prs.status.UpdatePRStatus(pr.Metadata.Namespace, pr.Metadata.Name, func(status *types.PRStatus) {
		status.Name = pr.Metadata.Name
		status.LastApplied = time.Now()
		status.Branch = pr.Spec.Branch
		status.Repository = pr.Spec.Repo
		status.LastDiff = diffOutput
		status.LastCommit = commitID
		status.PRNumber = createdPR.GetNumber()
		status.PRUrl = createdPR.GetHTMLURL()
	}); err != nil {
		return fmt.Errorf("failed to update PR status: %v", err)
	}

	// Print summary after successful PR creation
	prs.printer.PrintPRSummary(
		pr.Metadata.Namespace,
		pr.Metadata.Name,
		pr.Spec.Repo,
		pr.Spec.Branch,
		createdPR.GetNumber(),
		createdPR.GetHTMLURL(),
		commitID,
		len(diffOutput) > 0,
	)

	return nil
}

func (prs *PullRequestSet) runScript(repoDir string, script string, context map[string]string, prName string, namespace string) error {
	env := os.Environ()
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %v", err)
	}
	for k, v := range context {
		env = append(env, fmt.Sprintf("%s=%s", strings.ToUpper(k), v))
	}
	env = append(env, fmt.Sprintf("PRO_ROOT=%s", currentDir))

	// Print envs
	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = repoDir
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	prs.printer.PrintScriptOutput(fmt.Sprintf("PR(%s) Script %s", prName, script), output, err)
	if err != nil {
		errMsg := fmt.Sprintf("script failed: %v\n%s", err, output)
		if updateErr := prs.status.UpdatePRStatus(namespace, prName, func(status *types.PRStatus) {
			status.LastError = errMsg
			status.LastErrorAt = time.Now()
		}); updateErr != nil {
			prs.printer.PrintError("Failed to update status: %v", updateErr)
		}
		return fmt.Errorf(errMsg)
	}
	return nil
}
