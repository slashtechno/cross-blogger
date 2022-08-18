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

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/parser"
	"github.com/tidwall/gjson"
)

type Configuration struct {
	RefreshToken string `json:"refresh_token"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	// Scope             string   `json:"scope"`
	AuthorizationCode string `json:"authorization_code"`
	Blog              string `json:"blog"`
	BlogID            string `json:"blog_id"`
}
type PostJson struct {
	Kind string `json:"kind"`
	Blog struct {
		ID string `json:"id"`
	} `json:"blog"`
	Title   string `json:"title"`
	Content string `json:"content"`
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
	"blog": ""
}
		
`)
		fmt.Println(`Please add the appropriate values to config.json
client_id and client_secret can be retrieved from Google Cloud Console
blog should be your blog's URL. For example, https://example.blogspot.com`)
		os.Exit(0)
	}

	configuration := loadConfiguration()

	message := "Please set the following in config.json"
	if configuration.ClientID == "" {
		message += "\n- client_id"
	}
	if configuration.ClientSecret == "" {
		message += "\n- client_secret"
	}
	if configuration.Blog == "" {
		message += "\n- blog"
	}

	if message != "Please set the following in config.json" {
		log.Fatalln(message)
	}
	if configuration.RefreshToken == "" {
		log.Println("No refresh token found")
		getRefreshToken()
	}
	if configuration.BlogID == "" {
		log.Println("No blog ID found")
		getBlogID()
	}

	genPost()
}

func genPost() {
	fmt.Println(`Post formatting (input numeric selection):
1) Raw HTML
2) Markdown (will get converted to HTML automatically)`)
	reader := bufio.NewReader(os.Stdin)
	formatting, err := reader.ReadString('\n')
	checkNilErr(err)
	formatting = strings.TrimSpace(formatting)
	fmt.Print("\n")
	fmt.Print("Title: ")
	title := singleLineInput()
	if formatting == "1" {
		fmt.Print("COMPLETE filepath to HTML file: ")
		filepath := singleLineInput()
		htmlBytes, err := os.ReadFile(filepath)
		html := string(htmlBytes)
		checkNilErr(err)
		fmt.Print("Type \"Yes\" and press enter to confirm ")
		confirmation := singleLineInput()
		if confirmation == "Yes" {
			pushPost(html, title)
		} else {
			log.Fatalln("Confirmation was not given")
		}
	} else if formatting == "2" {
		fmt.Print("COMPLETE filepath to Markdown file: ")
		filepath := singleLineInput()
		markdownBytes, err := os.ReadFile(filepath)
		checkNilErr(err)
		// Setup markdown parser extensions
		extensions := parser.CommonExtensions | parser.AutoHeadingIDs
		mdparser := parser.NewWithExtensions(extensions)
		htmlBytes := markdown.ToHTML(markdownBytes, mdparser, nil)
		fmt.Print("Type \"Yes\" and press enter to confirm ")
		confirmation := singleLineInput()
		if confirmation == "Yes" {
			pushPost(string(htmlBytes), title)
		} else {
			log.Fatalln("Confirmation was not given")
		}
	} else {
		log.Fatalln("Invalid option")
	}

}

func pushPost(content string, title string) {
	url := "https://www.googleapis.com/blogger/v3/blogs/" + configuration.BlogID + "/posts/"
	payloadStruct := PostJson{Kind: "blogger#post", Blog: struct {
		ID string `json:"id"`
	}{ID: getBlogID()}, Title: title, Content: content}
	payload, err := json.Marshal(payloadStruct)
	checkNilErr(err)
	req, err := http.NewRequest("POST", url, strings.NewReader(string(payload)))
	checkNilErr(err)
	req.Header.Add("content-type", "application/json")
	req.Header.Add("Authorization", "Bearer "+getAccessToken())
	_, err = http.DefaultClient.Do(req)
	// res, err := http.DefaultClient.Do(req)
	checkNilErr(err)
	// defer res.Body.Close()
	// resultBodyBytes, err := io.ReadAll(res.Body)
	// checkNilErr(err)
	// resultBody := string(resultBodyBytes)
	// log.Println(resultBody)

}

func getAccessToken() string {
	// Get access token using the refresh token
	url := "https://oauth2.googleapis.com/token?client_id=" + configuration.ClientID + "&client_secret=" + configuration.ClientSecret + "&refresh_token=" + configuration.RefreshToken + "&redirect_uri=https%3A%2F%2Foauthcodeviewer.netlify.app&grant_type=refresh_token"
	// Send a POST request to the URL with no authorization headers
	resultBody := request(url, "POST", false)
	// Get the authorization token
	accessToken := gjson.Get(resultBody, "access_token").String()
	return accessToken
}

func getRefreshToken() {
	// Get the authorization code from the user
	fmt.Println("Please go to the following link in your browser:")
	fmt.Println("\nhttps://accounts.google.com/o/oauth2/v2/auth?client_id=" + configuration.ClientID + "&redirect_uri=https%3A%2F%2Foauthcodeviewer.netlify.app&scope=https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fblogger&response_type=code&access_type=offline&prompt=consent\n")
	fmt.Println("Input the authorization code below")
	configuration.AuthorizationCode = singleLineInput()

	// Get refresh token using the authorization code given by the user
	url := "https://oauth2.googleapis.com/token?client_id=" + configuration.ClientID + "&client_secret=" + configuration.ClientSecret + "&code=" + configuration.AuthorizationCode + "&redirect_uri=https%3A%2F%2Foauthcodeviewer.netlify.app&grant_type=authorization_code"
	// Send a POST request to the URL with no authorization headers
	resultBody := request(url, "POST", false)
	configuration.RefreshToken = gjson.Get(resultBody, "refresh_token").String()
	writeConfiguration()
}

func getBlogID() string {
	url := "https://www.googleapis.com/blogger/v3/blogs/byurl?url=" + configuration.Blog + "%2F"
	// Send a GET request to the URL with bearer authorization
	resultBody := request(url, "GET", true)
	configuration.BlogID = gjson.Get(resultBody, "id").String()
	writeConfiguration()
	return configuration.BlogID
}

func request(url string, requestType string, bearerAuth bool) string {
	// Send a request to the URL, with the URL which was passed to the function
	req, err := http.NewRequest(requestType, url, nil)
	checkNilErr(err)
	// If the bearerAuth parameter is true, set the Authorization header with an access token
	if bearerAuth {
		req.Header.Add("Authorization", "Bearer "+getAccessToken())
	}
	// Make the actual request
	res, err := http.DefaultClient.Do(req)
	checkNilErr(err)
	// Convert the result body to a string and then return it
	defer res.Body.Close()
	resultBodyBytes, err := io.ReadAll(res.Body)
	checkNilErr(err)
	return string(resultBodyBytes)
}

func checkNilErr(err any) {
	if err != nil {
		// log.Fatalln("Error:\n%v\n", err)
		log.Fatalln(err)
	}
}

func singleLineInput() string {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	checkNilErr(err)
	input = strings.TrimSpace(input)
	fmt.Print("\n")
	return input
}

func loadConfiguration() Configuration {
	configFile, err := os.OpenFile(configPath, os.O_RDONLY, 0644)
	checkNilErr(err)
	json.NewDecoder(configFile).Decode(&configuration)
	return configuration
}

func writeConfiguration() {
	configFile, err := os.OpenFile(configPath, os.O_WRONLY|os.O_CREATE, 0644)
	checkNilErr(err)
	configJsonBytes, err := json.MarshalIndent(configuration, "", "    ")
	checkNilErr(err)
	configFile.Write(configJsonBytes)
}
