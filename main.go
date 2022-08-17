package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
)

type Configuration struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	Blogs        []string `json:"blogs"`
}

var configuration Configuration
var currentDirectory, _ = os.Getwd()
var configPath = filepath.Join(currentDirectory, "config.json")

func main() {
	// Set the default logger to have the default flags
	log.SetFlags(log.LstdFlags)
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrExist) {
		// If config.json exists, load it as a struct
		configuration := loadConfiguration()
		log.Println(configuration)
	} else {
		log.Println("No configuration found, creating new configuration file")
		configFile, err := os.OpenFile("config.json", os.O_WRONLY|os.O_CREATE, 0644)
		checkNilErr(err)
		configJsonBytes, err := json.MarshalIndent(configuration, "", "    ")
		checkNilErr(err)
		configFile.Write(configJsonBytes)
	}
}

func checkNilErr(err any) {
	if err != nil {
		// log.Fatalln("Error:\n%v\n", err)
		log.Fatalln(err)
	}
}

func loadConfiguration() Configuration {
	configFile, err := os.OpenFile("config.json", os.O_RDONLY, 0644)
	checkNilErr(err)
	json.NewDecoder(configFile).Decode(&configuration)
	return configuration
}
