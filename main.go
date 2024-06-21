package main

import (
	"io/fs"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/slashtechno/cross-blogger/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

func init() {
	cobra.OnInitialize(initConfig)
	// Load a .env file if it exists
	gotenv.Load()
}

func initConfig() {
	// Tell Viper to use the prefix "CROSS_BLOGGER" for environment variables
	viper.SetEnvPrefix("CROSS_BLOGGER")
	// log.Debug(cfgFile)
	if cmd.ConfigFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cmd.ConfigFile)
	} else {
		// Use config.yaml in the current working directory.
		viper.SetConfigFile("./config.toml")
	}

	if err := viper.ReadInConfig(); err == nil {
		log.Debug("", "config file:", viper.ConfigFileUsed())
	} else {
		// If the config file is not found, create a file, write the default values and exit
		// Since viper.ConfigFileNotFoundError doesn't always work, also use fs.PathError
		if _, ok := err.(*fs.PathError); ok {
			log.Debug("Config file not found, creating a new one")
			// Destinations
			viper.SetDefault("destinations", []map[string]interface{}{
				{
					"name":      "blog",
					"type":      "blogger",
					"blog_url":  "https://example.com",
					"overwrite": false,
				},
				{
					"name":        "otherblog",
					"type":        "markdown",
					"content_dir": "/hugo-site/content/blog",
					"git_dir":     "/hugo-site",
					"overwrite":   false,
				},
			})
			// Sources
			viper.SetDefault("sources", []map[string]interface{}{
				{
					"name":     "someblog",
					"type":     "blogger",
					"blog_url": "https://example.com",
				},
				{
					"name":        "aBlogInMarkdown",
					"type":        "markdown",
					"content_dir": "content",
				},
			})

			if err := viper.WriteConfigAs(cmd.ConfigFile); err != nil {
				log.Fatal("Failed to write config file:", err)
			}
			log.Fatal("Failed to read config file. Created a config file with default values. Please edit the file and run the command again.", "path", cmd.ConfigFile)
		} else {
			log.Fatal("Failed to read config file:", err)
		}
	}
}

func main() {

	switch strings.ToLower(viper.GetString("log_level")) {
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
		log.Info("Invalid log level passed, using InfoLevel", "passed", viper.GetString("log_level"))
	}
	cmd.Execute()
}
