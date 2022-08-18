package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/tidwall/gjson"
)

type Configuration struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	// Scope             string   `json:"scope"`
	AuthorizationCode string   `json:"authorization_code"`
	Blogs             []string `json:"blogs"`
}

var configuration Configuration
var currentDirectory, _ = os.Getwd()
var configPath = filepath.Join(currentDirectory, "config.json")

func main() {
	// Set the default logger to have the default flags
	log.SetFlags(log.LstdFlags)

	if _, err := os.Stat(configPath); err == nil {
		log.Println("Configuration file found")
		// If config.json exists, load it as a struct
	} else {
		log.Println("No configuration found, creating new configuration file")
		configFile, err := os.OpenFile(configPath, os.O_WRONLY|os.O_CREATE, 0644)
		checkNilErr(err)
		configFile.WriteString(`
{
	"client_id": "",
	"client_secret": "",
	"blogs": []
}
		
`)
		fmt.Println(`Please add the appropriate values to config.json
client_id and client_secret can be retrieved from Google Cloud Console
blogs should be an array of blogs in the following format: [“example.com", “2.example.com”]`)
		os.Exit(0)
	}

	configuration := loadConfiguration()
	emptyCode := 0
	if configuration.ClientID == "" {
		emptyCode = 1
	} else if configuration.ClientSecret == "" {
		emptyCode = 2
	} else if configuration.ClientSecret == "" && configuration.ClientID == "" {
		emptyCode = 3
	}
	switch emptyCode {
	case 1:
		log.Fatalln("Please set client_id in config.json")
	case 2:
		log.Fatalln("Please set client_secret in config.json")
	case 3:
		log.Fatalln("Please set client_id and client_secret in config.json")
	}
	if configuration.RefreshToken == "" {
		log.Println("No refresh token found")
		getRefreshToken()
	}

	log.Println(getAccessToken())
}

func getAccessToken() string {
	// Get access token using the refresh token
	url := "https://oauth2.googleapis.com/token?client_id=" + configuration.ClientID + "&client_secret=" + configuration.ClientSecret + "&refresh_token=" + configuration.RefreshToken + "&redirect_uri=https%3A%2F%2Foauthcodeviewer.netlify.app&grant_type=refresh_token"
	// Send a POST request, to the URL above ^ and convert the result body to bytes, which is then converted to a string
	req, err := http.NewRequest("POST", url, nil)
	checkNilErr(err)
	res, err := http.DefaultClient.Do(req)
	checkNilErr(err)
	defer res.Body.Close()
	resultBodyBytes, err := io.ReadAll(res.Body)
	checkNilErr(err)
	resultBody := string(resultBodyBytes)
	// Get the authorization token
	configuration.AccessToken = gjson.Get(resultBody, "access_token").String()
	return configuration.AccessToken
}

func getRefreshToken() {

	fmt.Println("Please go to the following link in your browser:")
	fmt.Println("\nhttps://accounts.google.com/o/oauth2/v2/auth?client_id=" + configuration.ClientID + "&redirect_uri=https%3A%2F%2Foauthcodeviewer.netlify.app&scope=https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fblogger&response_type=code&access_type=offline&prompt=consent\n")
	fmt.Println("Input the authorization code below")
	reader := bufio.NewReader(os.Stdin)
	authorizationCode, err := reader.ReadString('\n')
	checkNilErr(err)
	if strings.TrimSpace(authorizationCode) == "!exit!" {
		os.Exit(0)
	}
	configuration.AuthorizationCode = strings.TrimSpace(authorizationCode)
	// log.Println(configuration.AuthorizationCode)

	// Get refresh token using the authorization code given by the user
	url := "https://oauth2.googleapis.com/token?client_id=" + configuration.ClientID + "&client_secret=" + configuration.ClientSecret + "&code=" + configuration.AuthorizationCode + "&redirect_uri=https%3A%2F%2Foauthcodeviewer.netlify.app&grant_type=authorization_code"
	// Send a POST request, to the URL above ^ and convert the result body to bytes, which is then converted to a string
	req, err := http.NewRequest("POST", url, nil)
	checkNilErr(err)
	res, err := http.DefaultClient.Do(req)
	checkNilErr(err)
	defer res.Body.Close()
	resultBodyBytes, err := io.ReadAll(res.Body)
	checkNilErr(err)
	resultBody := string(resultBodyBytes)

	// Get the refresh token
	configuration.RefreshToken = gjson.Get(resultBody, "refresh_token").String()
	log.Println(configuration.RefreshToken) // Print refresh token as string
	writeConfiguration()
}

func checkNilErr(err any) {
	if err != nil {
		// log.Fatalln("Error:\n%v\n", err)
		log.Fatalln(err)
	}
}

func loadConfiguration() Configuration {
	configFile, err := os.OpenFile(configPath, os.O_RDONLY, 0644)
	checkNilErr(err)
	json.NewDecoder(configFile).Decode(&configuration)
	if configuration.RefreshToken == "" {
	}
	return configuration
}

func writeConfiguration() {
	configFile, err := os.OpenFile(configPath, os.O_WRONLY|os.O_CREATE, 0644)
	checkNilErr(err)
	configJsonBytes, err := json.MarshalIndent(configuration, "", "    ")
	checkNilErr(err)
	configFile.Write(configJsonBytes)
}
