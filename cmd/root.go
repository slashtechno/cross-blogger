package cmd

import (
	"os"

	"github.com/slashtechno/cross-blogger/internal"
	"github.com/spf13/cobra"
)

var ConfigFile string
var CredentialFile string

// rootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "cross-blogger",
	Short: "A utility to cross-publish content between different platforms",
	Long: `cross-blogger is a utility to cross-publish content between different platforms.
	Viper's AutomaticEnv is enabled, so you can set environment variables with the prefix "CROSS_BLOGGER" to override config values.
	By default, the files for storing credentials and configuration are separate.
	Environment variables can be set (via .env as well) but the credentials configuration will be written to the file when the refresh token is stored. This shouldn't matter too much as environment variable support is mostly meant for running in an environment such as Docker.`,

	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cobra.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	RootCmd.PersistentFlags().StringVar(&ConfigFile, "config", "config.toml", "config file path")
	RootCmd.MarkPersistentFlagFilename("config", "toml", "yaml", "json")
	RootCmd.PersistentFlags().StringVar(&CredentialFile, "credentials-file", "credentials.yaml", "credentials file path")
	RootCmd.MarkPersistentFlagFilename("credentials-file", "yaml", "json", "toml")
	// Log level
	RootCmd.PersistentFlags().String("log-level", "", "Set the log level")
	internal.CredentialViper.BindPFlag("log_level", RootCmd.PersistentFlags().Lookup("log-level"))
	internal.ConfigViper.BindEnv("log_level", "CROSS_BLOGGER_LOG_LEVEL")
	internal.ConfigViper.SetDefault("log_level", "info")

}
