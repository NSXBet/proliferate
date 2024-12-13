package pullrequest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/nsxbet/masspr/pkg/mygit"
	"github.com/nsxbet/masspr/pkg/printer"
	"github.com/nsxbet/masspr/pkg/service"
	"github.com/nsxbet/masspr/pkg/types"
	"gopkg.in/yaml.v3"
)

type PullRequest = types.PullRequest

type PullRequestSet struct {
	prs            []PullRequest
	svc            *service.Service
	git            *mygit.Git
	status         *PRStatusManager
	templateString string
	printer        printer.Printer
}

func NewPullRequestSet(yamlTemplate string, git *mygit.Git, svc *service.Service, printer printer.Printer) (*PullRequestSet, error) {
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
		svc:            svc,
		git:            git,
		status:         NewPRStatusManager(".masspr", printer),
		templateString: yamlTemplate,
		printer:        printer,
	}, nil
}

func (prs *PullRequestSet) Process(ctx context.Context, dryRun bool) error {
	prs.printer.PrintNamespaceHeader(fmt.Sprintf("Found %d Pull Request(s)", len(prs.prs)))

	for i, pr := range prs.prs {
		if err := prs.processPR(ctx, i, pr, dryRun); err != nil {
			return err
		}
	}
	return nil
}

func (prs *PullRequestSet) runScript(repoDir string, script string, context map[string]string) error {
	env := os.Environ()
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %v", err)
	}

	for k, v := range context {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	env = append(env, fmt.Sprintf("MASSPR_ROOT=%s", currentDir))

	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = repoDir
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("script failed: %s: %v", output, err)
	}

	if len(output) > 0 {
		prs.printer.PrintInfo("Script output (%s):\n%s", script, output)
	}

	return nil
}

func (prs *PullRequestSet) processPR(ctx context.Context, index int, pr PullRequest, dryRun bool) error {
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

	for _, script := range pr.Spec.Scripts {
		if err := prs.runScript(repoDir, script, pr.Spec.ScriptsContext); err != nil {
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

	createdPR, err := prs.svc.CreatePR(
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

	prStatus := PRStatus{
		Name:        pr.Metadata.Name,
		LastApplied: time.Now(),
		PRNumber:    createdPR.GetNumber(),
		PRUrl:       createdPR.GetHTMLURL(),
		Branch:      pr.Spec.Branch,
		Repository:  pr.Spec.Repo,
		LastDiff:    diffOutput,
		LastCommit:  commitID,
	}
	if err := prs.status.SaveStatus(pr.Metadata.Namespace, prStatus); err != nil {
		return fmt.Errorf("failed to update status: %v", err)
	}

	return nil
}
