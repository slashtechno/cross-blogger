package main

import (
	"io/fs"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/slashtechno/cross-blogger/cmd"
	"github.com/slashtechno/cross-blogger/internal"
	"github.com/slashtechno/cross-blogger/internal/platforms"
	"github.com/spf13/cobra"
	"github.com/subosito/gotenv"
)

func init() {
	cobra.OnInitialize(initConfig)
	// Load a .env file if it exists
	gotenv.Load()
}

func initConfig() {
	// Tell Viper to use the prefix "CROSS_BLOGGER" for environment variables
	internal.ConfigViper.SetEnvPrefix("CROSS_BLOGGER")
	internal.CredentialViper.SetEnvPrefix("CROSS_BLOGGER")

	// log.Debug(cfgFile)
	if cmd.CredentialFile != "" {
		// Use config file from the flag.
		internal.CredentialViper.SetConfigFile(cmd.CredentialFile)
	} else {
		// Use config.yaml in the current working directory.
		internal.CredentialViper.SetConfigFile("credentials.yaml")
	}
	if err := internal.CredentialViper.ReadInConfig(); err == nil {
		log.Debug("", "credential file:", internal.CredentialViper.ConfigFileUsed())
	} else {
		// Generate a default .env file null values
		if _, ok := err.(*fs.PathError); ok {
			log.Debug("Credential file not found, creating a new one")
			internal.CredentialViper.SetDefault("google_client_id", "")
			internal.CredentialViper.SetDefault("google_client_secret", "")
			if err := internal.CredentialViper.WriteConfigAs(cmd.CredentialFile); err != nil {
				log.Fatal("Failed to write credential file:", err)
			}
		} else {
			log.Fatal("Failed to read credential file:", err)
		}
	}

	if cmd.ConfigFile != "" {
		// Use config file from the flag.
		internal.ConfigViper.SetConfigFile(cmd.ConfigFile)
	} else {
		// Use config.yaml in the current working directory.
		internal.ConfigViper.SetConfigFile("./config.toml")
	}

	if err := internal.ConfigViper.ReadInConfig(); err == nil {
		log.Debug("", "config file:", internal.ConfigViper.ConfigFileUsed())
	} else {
		// If the config file is not found, create a file, write the default values and exit
		// Since viper.ConfigFileNotFoundError doesn't always work, also use fs.PathError
		if _, ok := err.(*fs.PathError); ok {
			log.Debug("Config file not found, creating a new one")
			// Destinations
			internal.ConfigViper.SetDefault("destinations", []map[string]interface{}{
				{
					"name":      "blog",
					"type":      "blogger",
					"blog_url":  "https://example.com",
					"overwrite": false,
				},
				{
					"name":                "otherblog",
					"type":                "markdown",
					"content_dir":         "/hugo-site/content/blog",
					"git_dir":             "/hugo-site",
					"frontmatter_mapping": platforms.FrontMatterMappings,
					"overwrite":           false,
				},
			})
			// Sources
			internal.ConfigViper.SetDefault("sources", []map[string]interface{}{
				{
					"name":     "someblog",
					"type":     "blogger",
					"blog_url": "https://example.com",
				},
				{
					"name":                "aBlogInMarkdown",
					"type":                "markdown",
					"content_dir":         "content",
					"frontmatter_mapping": platforms.FrontMatterMappings,
				},
			})

			if err := internal.ConfigViper.WriteConfigAs(cmd.ConfigFile); err != nil {
				log.Fatal("Failed to write config file:", err)
			}
			log.Fatal("Failed to read config file. Created a config file with default values. Please edit the file and run the command again.", "path", cmd.ConfigFile)
		} else {
			log.Fatal("Failed to read config file:", err)
		}
	}
}

func main() {

	switch strings.ToLower(internal.ConfigViper.GetString("log_level")) {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
		log.Info("Invalid log level passed, using InfoLevel", "passed", internal.ConfigViper.GetString("log_level"))
	}
	cmd.Execute()
}
