/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"github.com/charmbracelet/log"
	"github.com/slashtechno/cross-blogger/cobra/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

func init() {
	cobra.OnInitialize(initConfig)
	cmd.RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cobra.yaml)")
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Use config.yaml in the current working directory.
		viper.SetConfigFile("./config.yaml")
	}

	if err := viper.ReadInConfig(); err == nil {
		log.Debug("Using config file:", viper.ConfigFileUsed())
	} else {
		log.Fatal("Failed to read config file:", err)
	}
}

func main() {
	log.SetLevel(log.DebugLevel)
	cmd.Execute()
}
