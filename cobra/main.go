/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"io/fs"

	"github.com/charmbracelet/log"
	"github.com/slashtechno/cross-blogger/cobra/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

func init() {
	cobra.OnInitialize(initConfig)
	cmd.RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "config.toml", "config file path")
}

func initConfig() {
	// log.Debug(cfgFile)
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
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
			viper.SetDefault("destinations", []map[string]interface{}{
				{
					"name":     "blogger",
					"type":     "blogger",
					"blog_url": "https://example.com",
					"blog_id":  "1234567890",
				},
				{
					"name":        "markdown1",
					"type":        "markdown",
					"content_dir": "content",
				},
			})
			log.Fatal("Failed to read config file. Created a config file with default values. Please edit the file and run the command again.", "path", cfgFile)
			if err := viper.WriteConfigAs(cfgFile); err != nil {
				log.Fatal("Failed to write config file:", err)
			}
		} else {
			log.Fatal("Failed to read config file:", err)
		}
	}
}

func main() {
	log.SetLevel(log.DebugLevel)
	cmd.Execute()
}
