package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"github.com/nsxbet/proliferate/cmd/pro/apply"
	"github.com/nsxbet/proliferate/cmd/pro/status"
	"github.com/nsxbet/proliferate/pkg/core"
	"github.com/nsxbet/proliferate/pkg/types"
)

// Config holds the application configuration
type Config struct {
	GithubToken string `yaml:"github-token"`
	AuthorEmail string `yaml:"author-email"`
	AuthorName  string `yaml:"author-name"`
}

func (c Config) GetAuthorEmail() string {
	return c.AuthorEmail
}

func (c Config) GetAuthorName() string {
	return c.AuthorName
}

func (c Config) GetGithubToken() string {
	return c.GithubToken
}

var (
	// Version will be replaced during build time
	Version = "dev"
)

// Execute sets up and runs the CLI application
func Execute() {
	rootCmd := &cobra.Command{
		Use:   "pro",
		Short: "Proliferate - A tool for managing multiple pull requests",
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("pro version %s\n", Version)
		},
	}
	rootCmd.AddCommand(versionCmd)

	rootCmd.PersistentFlags().String("config", "", "config file (default is ./config.yaml)")
	cfgFile, _ := rootCmd.PersistentFlags().GetString("config")

	app := fx.New(
		fx.WithLogger(func() fxevent.Logger { return fxevent.NopLogger }),
		fx.Provide(func() (types.Config, error) {
			appConfig, err := loadConfig(cfgFile)
			return appConfig, err
		}),
		core.Module,
		fx.Invoke(func(c core.Core) {
			prCmd := &cobra.Command{
				Use:   "pr",
				Short: "Pull request operations",
			}

			prCmd.AddCommand(apply.NewCommand(c))
			prCmd.AddCommand(status.NewCommand(c))
			rootCmd.AddCommand(prCmd)

			if err := rootCmd.Execute(); err != nil {
				log.Error("failed to execute command", "err", err)
				os.Exit(1)
			}

			os.Exit(0)
		}),
	)

	if err := app.Err(); err != nil {
		log.Error("failed to initialize application", "err", err)
		os.Exit(1)
	}

	app.Run()
}

// loadConfig loads configuration from the specified file or default locations
func loadConfig(cfgFile string) (cfg Config, err error) {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		log.Debug("using config file", "path", cfgFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		log.Debug("using default config path")
	}

	viper.SetEnvPrefix("PRO")
	viper.AutomaticEnv()
	viper.BindEnv("github-token", "GHA_PAT", "GITHUB_TOKEN")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return cfg, err
		}
		log.Debug("no config file found")
	} else {
		log.Debug("config file used", "path", viper.ConfigFileUsed())
	}

	cfg.GithubToken = viper.GetString("github-token")
	cfg.AuthorEmail = viper.GetString("author-email")
	cfg.AuthorName = viper.GetString("author-name")

	log.Debug("config loaded", "config", cfg)

	if cfg.GithubToken == "" {
		return cfg, fmt.Errorf("github token is required but was empty")
	}
	return cfg, nil
}
