package apply

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/nsxbet/proliferate/pkg/core"
	"github.com/nsxbet/proliferate/pkg/pullrequest"
)

type applyCommand struct {
	valuesFile string
	prFile     string
	dryRun     bool
	core       core.Core
}

func NewCommand(c core.Core) *cobra.Command {
	ac := &applyCommand{core: c}
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply values to a pull request definition",
		RunE:  ac.run,
	}

	cmd.Flags().StringVarP(&ac.valuesFile, "values", "f", "", "Path to values YAML file")
	cmd.Flags().StringVarP(&ac.prFile, "pr", "p", "", "Path to pull request YAML file")
	cmd.Flags().BoolVar(&ac.dryRun, "dry-run", false, "Print the parsed pull requests without applying")
	cmd.MarkFlagRequired("pr")

	return cmd
}

func (ac *applyCommand) run(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	token := viper.GetString("github-token")
	if token == "" {
		return fmt.Errorf("no GitHub token found in environment or config file")
	}

	valuesData, err := os.ReadFile(ac.valuesFile)
	if err != nil {
		fmt.Printf("failed to read values file: %v", err)
	}

	values := make(map[string]interface{})
	if len(valuesData) > 0 {
		decoder := yaml.NewDecoder(bytes.NewBuffer(valuesData))
		decoder.KnownFields(false)
		if err := decoder.Decode(&values); err != nil {
			return fmt.Errorf("failed to parse values file: %v", err)
		}
	}

	prTemplate, err := os.ReadFile(ac.prFile)
	if err != nil {
		return fmt.Errorf("failed to read PR template: %v", err)
	}

	templateString, err := renderTemplate("pr", string(prTemplate), values)
	if err != nil {
		return fmt.Errorf("failed to render template: %v", err)
	}

	prSet, err := pullrequest.NewPullRequestSet(templateString, ac.core.Git, ac.core.Printer)
	if err != nil {
		return err
	}

	prs := prSet.GetPRs()

	// Create a worker pool
	workers := 3
	if workers > len(prs) {
		workers = len(prs)
	}

	jobs := make(chan int, len(prs))
	results := make(chan error, len(prs))

	// Start workers
	for w := 0; w < workers; w++ {
		go func() {
			for i := range jobs {
				err := prSet.ProcessPR(ctx, i, prs[i], ac.dryRun)
				results <- err
				if err != nil {
					cancel() // Cancel other workers on error
				}
			}
		}()
	}

	// Send jobs
	for i := range prs {
		jobs <- i
	}
	close(jobs)

	// Collect results
	for i := 0; i < len(prs); i++ {
		if err := <-results; err != nil {
			return err
		}
	}

	return nil
}
