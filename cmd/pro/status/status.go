package status

import (
	"context"

	"github.com/nsxbet/proliferate/pkg/core"
	"github.com/nsxbet/proliferate/pkg/mygit"
	"github.com/nsxbet/proliferate/pkg/pullrequest"
	"github.com/spf13/cobra"
)

func NewCommand(c core.Core) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [namespace]",
		Short: "Show status of pull requests",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(c, args)
		},
	}
	return cmd
}

func runStatus(c core.Core, args []string) error {
	git := mygit.NewGit(c.Config)
	ctx := context.Background()
	statusMgr := pullrequest.NewPRStatusManager(".proliferate", c.Printer)

	if len(args) == 0 {
		return statusMgr.DisplayNamespacesSummary()
	}

	namespace := args[0]
	return statusMgr.DisplayNamespaceDetails(ctx, namespace, git)
}
