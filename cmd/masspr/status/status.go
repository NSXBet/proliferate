package status

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nsxbet/masspr/pkg/mygit"
	"github.com/nsxbet/masspr/pkg/printer"
	"github.com/nsxbet/masspr/pkg/pullrequest"
)

func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status [namespace]",
		Short: "Show status of pull requests",
		RunE:  runStatus,
	}
}

func runStatus(cmd *cobra.Command, args []string) error {
	token := viper.GetString("github-token")
	if token == "" {
		return fmt.Errorf("no GitHub token found in environment or config file")
	}

	ctx := context.Background()
	git := mygit.NewGit(token)
	p := printer.NewConsolePrinter()
	statusMgr := pullrequest.NewPRStatusManager(".masspr", p)

	if len(args) == 0 {
		return statusMgr.DisplayNamespacesSummary()
	}

	namespace := args[0]
	return statusMgr.DisplayNamespaceDetails(ctx, namespace, git)
}
