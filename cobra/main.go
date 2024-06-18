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
	cmd.RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "./config.toml", "config file path")
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Use config.yaml in the current working directory.
		viper.SetConfigFile("./config.toml")
	}

	if err := viper.ReadInConfig(); err == nil {
		log.Debug("Using config file:", viper.ConfigFileUsed())
	} else {
		// If the config file is not found, create a file, write the default values and exit
		// Since viper.ConfigFileNotFoundError doesn't always work, also use fs.PathError
		if _, ok := err.(*fs.PathError); ok {
			log.Debug("Config file not found, creating a new one")
			viper.SetDefault("destinations", []map[string]interface{}{
				{
					"name":    "blogger",
					"type":    "blogger",
					"blogUrl": "https://example.com",
					"blogId":  "1234567890",
				},
				{
					"name":       "markdown1",
					"type":       "markdown",
					"contentDir": "content",
				},
			})
			log.Debug("Config file created at:", cfgFile)
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
