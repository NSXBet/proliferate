package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nsxbet/masspr/cmd/masspr/apply"
	"github.com/nsxbet/masspr/cmd/masspr/status"
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

	prCmd.AddCommand(apply.NewCommand())
	prCmd.AddCommand(status.NewCommand())
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

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
