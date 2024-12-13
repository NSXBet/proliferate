package apply

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/nsxbet/masspr/pkg/mygit"
	"github.com/nsxbet/masspr/pkg/printer"
	"github.com/nsxbet/masspr/pkg/pullrequest"
	"github.com/nsxbet/masspr/pkg/service"
)

type applyCommand struct {
	valuesFile string
	prFile     string
	dryRun     bool
}

func NewCommand() *cobra.Command {
	ac := &applyCommand{}
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply values to a pull request definition",
		RunE:  ac.run,
	}

	cmd.Flags().StringVarP(&ac.valuesFile, "values", "f", "", "Path to values YAML file")
	cmd.Flags().StringVarP(&ac.prFile, "pr", "p", "", "Path to pull request YAML file")
	cmd.Flags().BoolVar(&ac.dryRun, "dry-run", false, "Print the parsed pull requests without applying")
	cmd.MarkFlagRequired("values")
	cmd.MarkFlagRequired("pr")

	return cmd
}

func (ac *applyCommand) run(cmd *cobra.Command, args []string) error {
	token := viper.GetString("github-token")
	if token == "" {
		return fmt.Errorf("no GitHub token found in environment or config file")
	}

	valuesData, err := os.ReadFile(ac.valuesFile)
	if err != nil {
		return fmt.Errorf("failed to read values file: %v", err)
	}

	var values map[string]interface{}
	decoder := yaml.NewDecoder(bytes.NewBuffer(valuesData))
	decoder.KnownFields(false)
	if err := decoder.Decode(&values); err != nil {
		return fmt.Errorf("failed to parse values file: %v", err)
	}

	prTemplate, err := os.ReadFile(ac.prFile)
	if err != nil {
		return fmt.Errorf("failed to read PR template: %v", err)
	}

	templateString, err := renderTemplate("pr", string(prTemplate), values)
	if err != nil {
		return fmt.Errorf("failed to render template: %v", err)
	}

	svc := service.New(token)
	git := mygit.NewGit(token)
	p := printer.NewConsolePrinter()

	prSet, err := pullrequest.NewPullRequestSet(templateString, git, svc, p)
	if err != nil {
		return err
	}

	ctx := context.Background()

	if err := prSet.Process(ctx, ac.dryRun); err != nil {
		return err
	}

	return nil
}
