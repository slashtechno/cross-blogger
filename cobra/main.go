/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"github.com/charmbracelet/log"
	"github.com/slashtechno/cross-blogger/cobra/cmd"
)

func main() {
	log.SetLevel(log.DebugLevel)
	cmd.Execute()
}
