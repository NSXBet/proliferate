package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/k0kubun/pp/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nsxbet/masspr/cmd/masspr/apply"
	"github.com/nsxbet/masspr/pkg/masspr"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "masspr",
		Short: "Mass PR creation tool",
	}
)

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml)")

	prCmd := &cobra.Command{
		Use:   "pr",
		Short: "Pull request operations",
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of pull requests",
		RunE:  runStatus,
	}

	prCmd.AddCommand(apply.NewCommand())
	prCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(prCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
	}

	viper.SetEnvPrefix("MASSPR")
	viper.AutomaticEnv()
	viper.BindEnv("github-token", "GHA_PAT", "GITHUB_TOKEN")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Printf("Error reading config file: %v\n", err)
		}
	}
}

func runStatus(cmd *cobra.Command, args []string) error {
	token := viper.GetString("github-token")
	if token == "" {
		return fmt.Errorf("no GitHub token found in environment or config file")
	}

	svc := masspr.New(token)

	if len(args) == 0 {
		namespaces, err := svc.GetNamespaces()
		if err != nil {
			return err
		}

		if len(namespaces) == 0 {
			pp.Println("No pull requests found")
			return nil
		}

		pp.Println("=== Available Namespaces ===")
		for _, namespace := range namespaces {
			prs, err := svc.GetByNamespace(namespace)
			if err != nil {
				return err
			}
			pp.Printf("\n%s (%d PRs)\n", namespace, len(prs))
		}
		return nil
	}

	namespace := args[0]
	prs, err := svc.GetByNamespace(namespace)
	if err != nil {
		return err
	}
	if prs == nil {
		return fmt.Errorf("namespace %s not found", namespace)
	}

	ctx := context.Background()
	pp.Printf("=== Pull Request Status for %s ===\n", namespace)

	for name, pr := range prs {
		owner, repoName, err := svc.ParseRepoString(pr.Repository)
		if err != nil {
			return err
		}

		prState, err := svc.GetPRStatus(ctx, owner, repoName, pr.PRNumber)
		if err != nil {
			pp.Printf("\n PR: %s (Failed to get GitHub status: %v)\n", name, err)
			continue
		}

		stateEmoji := map[string]string{
			"open":   "🟢",
			"closed": "🔴",
			"merged": "🟣",
		}[prState]

		pp.Printf("\n PR: %s %s\n", name, stateEmoji)
		pp.Printf("├── Repository: %s\n", pr.Repository)
		pp.Printf("├── Branch: %s\n", pr.Branch)
		pp.Printf("├── Pull Request: #%d (%s)\n", pr.PRNumber, prState)
		pp.Printf("├── URL: %s\n", pr.PRUrl)
		pp.Printf("├── Last Applied: %s\n", pr.LastApplied.Format(time.RFC3339))
		pp.Printf("├── Last Commit: %s\n", pr.LastCommit)
		if pr.LastDiff != "" {
			pp.Printf("└── Changes:\n    %s\n", strings.ReplaceAll(pr.LastDiff, "\n", "\n    "))
		} else {
			pp.Printf("└── No changes\n")
		}
	}

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
