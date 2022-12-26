/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/slashtechno/cross-blogger/cmd/info"
	"github.com/slashtechno/cross-blogger/cmd/net"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "All things config",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(viper.GetViper().GetString("name"))
	},
}

func init() {
	rootCmd.AddCommand(info.InfoCmd)
	rootCmd.AddCommand(net.NetCmd)
	rootCmd.AddCommand(configCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// configCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// configCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
