package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"github.com/nsxbet/masspr/cmd/masspr/apply"
	"github.com/nsxbet/masspr/cmd/masspr/status"
	"github.com/nsxbet/masspr/pkg/core"
	"github.com/nsxbet/masspr/pkg/types"
)

type cmdConfig struct {
	GithubToken string `yaml:"github-token"`
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "masspr",
		Short: "Mass PR creation tool",
	}

	rootCmd.PersistentFlags().String("config", "", "config file (default is ./config.yaml)")
	cfgFile, _ := rootCmd.PersistentFlags().GetString("config")

	app := fx.New(
		fx.WithLogger(func() fxevent.Logger { return fxevent.NopLogger }),
		fx.Provide(func() (types.Config, error) {
			appConfig, err := NewConfig(cfgFile)
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

func NewConfig(cfgFile string) (cfg cmdConfig, err error) {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		log.Debug("using config file", "path", cfgFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		log.Debug("using default config path")
	}

	viper.SetEnvPrefix("MASSPR")
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
	log.Debug("config loaded", "config", cfg)

	if cfg.GithubToken == "" {
		return cfg, fmt.Errorf("github token is required but was empty")
	}
	return cfg, nil
}

func (c cmdConfig) GetGithubToken() string {
	return c.GithubToken
}
