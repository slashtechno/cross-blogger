package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	htmltomd "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/TwiN/go-color"
	"github.com/cheynewallace/tabby"
	mdlib "github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/parser"
	"github.com/tidwall/gjson"
)

type Configuration struct {
	RefreshToken string `json:"refresh_token"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	// Scope             string   `json:"scope"`
	Blog     string `json:"blog"`
	BlogID   string `json:"blog_id"`
	DevtoAPI string `json:"devto_api_key"`
}
type BloggerPostPayload struct {
	Kind string `json:"kind"`
	Blog struct {
		ID string `json:"id"`
	} `json:"blog"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type Destination struct {
	DestinationType      string
	DestinationSpecifier string
}

type DevtoPostPayload struct {
	Article struct {
		Title        string   `json:"title"`
		BodyMarkdown string   `json:"body_markdown"`
		Published    bool     `json:"published"`
		Tags         []string `json:"tags"`
	} `json:"article"`
}

/* Possible flags
* Get refresh token
* Title (Overrides title from source)
* Source (String)
* Source specifier (String)
* Destination (Attempt to allow multiple destination flags?)
* JSON
 */

// Source flags
var (
	sourceFlag          = flag.String("source", "", "What source to use\nAvailable sources: blogger, dev.to, markdown, html\ndev.to, markdown, and html work with source-specifier")
	sourceSpecifierFlag = flag.String("source-specifier", "", "Specify a source location\nCan be used with sources: dev.to, markdown, html")
	titleFlag           = flag.String("title", "", "Specify custom title instead of using the default\nAlso, if the title is not specified, using files as a source will require a title to be inputted later")
)

// Destination flags
var (
	devtoDestinationFlag      = flag.Bool("post-to-devto", false, "Post to dev.to")
	bloggerDestinationFlag    = flag.Bool("post-to-blogger", false, "Post to Blogger")
	markdownDestinationFlag   = flag.String("post-to-markdown", "", "Post to a markdown file\nPath to file must be specified")
	htmlDestinationFlag       = flag.String("post-to-html", "", "Post to an HTML file\nPath to file must be specified")
	skipDestinationPromptFlag = flag.Bool("skip-destination-prompt", false, "Don't prompt for additional destinations\nUseful when specifying destinations via CLI")
)

var configuration Configuration
var currentDirectory, _ = os.Getwd()
var configPath = filepath.Join(currentDirectory, "config.json")

func main() {
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), color.InBold("Usage:"))
		fmt.Fprintln(flag.CommandLine.Output(), color.InBold(color.InRed("When passing a value containing whitespace, use quotes")))
		flag.PrintDefaults()
	}
	flag.Parse()
	// Set the default logger to have the default flags
	log.SetFlags(log.LstdFlags)

	if _, err := os.Stat(configPath); err == nil {
		log.Println("Configuration file found")
		// If config.json exists, load it as a struct
		configuration = loadConfiguration()
	} else {
		log.Println("No configuration found, creating new configuration file")
		configFile, err := os.OpenFile(configPath, os.O_WRONLY|os.O_CREATE, 0644)
		checkNilErr(err)
		configFile.WriteString(`
{
	"client_id": "",
	"client_secret": "",
	"blog": "",
	"devto_api_key": ""
}
		
`)
		fmt.Println(`Please add the appropriate values to config.json
client_id and client_secret can be retrieved from Google Cloud Console
blog should be your blog's URL. For example, https://example.blogspot.com
devto_api_key should be your dev.to API key`)
		os.Exit(0)
	}

	chooseSource()
}

func chooseSource() {
	flag.Parse()
	source := *sourceFlag
	if *sourceFlag == "" {
		fmt.Println(`Choose a source (input numeric selection):
1) Dev.to
2) Blogger
3) Markdown file
4) HTML File`)
		source = singleLineInput()
	}
	var title, html, markdown string
	title = *titleFlag
	if source == "1" || source == "dev.to" {
		var article string
		if *sourceSpecifierFlag == "" {
			fmt.Print("dev.to article URL: ")
			article = singleLineInput()
		} else {
			article = *sourceSpecifierFlag
		}
		index := 15
		api_article := article[:index] + "api/articles/" + article[index:]
		resultBody := request(api_article, "GET", false)
		title = gjson.Get(resultBody, "title").String()
		html = gjson.Get(resultBody, "body_html").String()
		markdown = gjson.Get(resultBody, "body_markdown").String()
	} else if source == "2" || source == "blogger" {
		// Check if BlogID is present
		if configuration.BlogID == "" {
			log.Println("No blog ID found")
			getBlogID()
		}

		fmt.Print("Blogger post URL: ")
		path := strings.Replace(singleLineInput(), configuration.Blog, "", 1)
		url := "https://www.googleapis.com/blogger/v3/blogs/" + configuration.BlogID + "/posts/bypath?path=" + path
		resultBody := request(url, "GET", true)
		html = gjson.Get(resultBody, "content").String()
		title = gjson.Get(resultBody, "title").String()
		var err error
		markdown, err = htmltomd.NewConverter("", true, nil).ConvertString(html)
		checkNilErr(err)
	} else if source == "3" || source == "markdown" {
		var filepath string
		if *sourceSpecifierFlag == "" {
			fmt.Print("Path to Markdown file: ")
			filepath = singleLineInput()
		} else {
			filepath = *sourceSpecifierFlag
		}
		if *titleFlag == "" {
			fmt.Print("Title: ")
			title = singleLineInput()
		}
		markdownBytes, err := os.ReadFile(filepath)
		checkNilErr(err)
		// Setup markdown parser extensions
		extensions := parser.CommonExtensions | parser.AutoHeadingIDs
		mdparser := parser.NewWithExtensions(extensions)
		markdown = string(markdownBytes)
		html = string(mdlib.ToHTML(markdownBytes, mdparser, nil))
	} else if source == "4" || source == "html" {
		var filepath string
		if *sourceSpecifierFlag == "" {
			fmt.Print("Path to HTML file: ")
			filepath = singleLineInput()
		} else {
			filepath = *sourceSpecifierFlag
		}
		if *titleFlag == "" {
			fmt.Print("Title: ")
			title = singleLineInput()
		}
		htmlBytes, err := os.ReadFile(filepath)
		checkNilErr(err)
		html = string(htmlBytes)
		markdown, err = htmltomd.NewConverter("", true, nil).ConvertString(html)
		checkNilErr(err)
	} else {
		log.Fatalln("Invalid option")
	}
	if *titleFlag != "" {
		title = *titleFlag
	}
	selectDestinations(title, html, markdown)
}

func selectDestinations(title string, html string, markdown string) {
	flag.Parse()
	destinations := []Destination{}
	if !*skipDestinationPromptFlag {
		for {
			fmt.Println(`Select a destination, and press enter (input numeric selection)
1) Dev.to
2) Blogger
3) Markdown file
4) HTML file
5) Stop adding`)

			destinationSelection := singleLineInput()
			if destinationSelection == "1" || destinationSelection == "dev.to" {
				destinations = append(destinations, Destination{DestinationType: "dev.to"})
			} else if destinationSelection == "2" || destinationSelection == "blogger" {
				destinations = append(destinations, Destination{DestinationType: "blogger"})
			} else if destinationSelection == "3" || destinationSelection == "markdown" {
				fmt.Print("File creation path: ")
				path := singleLineInput()
				destinations = append(destinations, Destination{DestinationType: "markdown", DestinationSpecifier: path})
			} else if destinationSelection == "4" || destinationSelection == "html" {
				fmt.Print("File creation path: ")
				path := singleLineInput()
				destinations = append(destinations, Destination{DestinationType: "html", DestinationSpecifier: path})
			} else if destinationSelection == "5" || destinationSelection == "stop" {
				break
			} else {
				log.Fatalln("Invalid option")
			}
		}
	}
	// CLI Destination Flag Conditionals
	if *devtoDestinationFlag {
		destinations = append(destinations, Destination{DestinationType: "dev.to"})
	}
	if *bloggerDestinationFlag {
		destinations = append(destinations, Destination{DestinationType: "blogger"})
	}
	if *markdownDestinationFlag != "" {
		destinations = append(destinations, Destination{DestinationType: "markdown", DestinationSpecifier: *markdownDestinationFlag})
	}
	if *htmlDestinationFlag != "" {
		destinations = append(destinations, Destination{DestinationType: "html", DestinationSpecifier: *htmlDestinationFlag})
	}

	destinationsTable := tabby.New()
	destinationsTable.AddHeader("Destination", "Specifier")
	for _, destination := range destinations {
		destinationsTable.AddLine(destination.DestinationType, destination.DestinationSpecifier)
	}
	destinationsTable.Print()
	pushPost(title, html, markdown, destinations)
}
func pushPost(title string, html string, markdown string, destinations []Destination) {
	for _, destination := range destinations {
		if destination.DestinationType == "blogger" {
			// Check if BlogID is present
			if configuration.BlogID == "" {
				log.Println("No blog ID found")
				getBlogID()
			}

			log.Println("Pushing to Blogger")
			url := "https://www.googleapis.com/blogger/v3/blogs/" + configuration.BlogID + "/posts/"
			payloadStruct := BloggerPostPayload{Kind: "blogger#post", Blog: struct {
				ID string `json:"id"`
			}{ID: getBlogID()}, Title: title, Content: html}
			payload, err := json.Marshal(payloadStruct)
			checkNilErr(err)
			req, err := http.NewRequest("POST", url, strings.NewReader(string(payload)))
			checkNilErr(err)
			req.Header.Add("content-type", "application/json")
			req.Header.Add("Authorization", "Bearer "+getAccessToken())
			_, err = http.DefaultClient.Do(req)
			checkNilErr(err)
		} else if destination.DestinationType == "dev.to" {
			log.Println("Pushing to dev.to")
			if configuration.DevtoAPI == "" {
				log.Fatalln("\"devto_api_key\" must be set in config.json")
			}
			url := "https://dev.to/api/articles"
			payloadStruct := DevtoPostPayload{Article: struct {
				Title        string   `json:"title"`
				BodyMarkdown string   `json:"body_markdown"`
				Published    bool     `json:"published"`
				Tags         []string `json:"tags"`
			}{Title: title, BodyMarkdown: markdown, Published: true, Tags: []string{}}}
			payload, err := json.MarshalIndent(payloadStruct, "", "    ")
			checkNilErr(err)
			req, err := http.NewRequest("POST", url, strings.NewReader(string(payload)))
			checkNilErr(err)
			req.Header.Add("api-key", configuration.DevtoAPI)
			req.Header.Add("content-type", "application/json")
			_, err = http.DefaultClient.Do(req)
			checkNilErr(err)
		} else if destination.DestinationType == "markdown" {
			file, err := os.OpenFile(destination.DestinationSpecifier, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
			checkNilErr(err)
			file.WriteString(markdown)
		} else if destination.DestinationType == "html" {
			file, err := os.OpenFile(destination.DestinationSpecifier, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
			checkNilErr(err)
			file.WriteString(html)
		} else {
			log.Printf("Destination \"%v\" not yet implemented\n", destination)
		}
	}

}

func getAccessToken() string {
	// Check if client_id and client_secret are set
	message := "The following must be set in config.json"
	if configuration.ClientID == "" {
		message += "\n- client_id"
	}
	if configuration.ClientSecret == "" {
		message += "\n- client_secret"
	}
	if message != "The following must be set in config.json" {
		log.Fatalln(message)
	}
	// Check if there is a refresh token present
	if configuration.RefreshToken == "" {
		log.Println("No refresh token found")
		getRefreshToken()
	}

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
	if configuration.ClientID == "" {
		log.Fatalln("\"client_id\" must be set in config.json")
	}
	fmt.Println("\nhttps://accounts.google.com/o/oauth2/v2/auth?client_id=" + configuration.ClientID + "&redirect_uri=https%3A%2F%2Foauthcodeviewer.netlify.app&scope=https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fblogger&response_type=code&access_type=offline&prompt=consent\n")
	fmt.Println("Input the authorization code below")
	authorizationCode := singleLineInput()

	// Get refresh token using the authorization code given by the user
	url := "https://oauth2.googleapis.com/token?client_id=" + configuration.ClientID + "&client_secret=" + configuration.ClientSecret + "&code=" + authorizationCode + "&redirect_uri=https%3A%2F%2Foauthcodeviewer.netlify.app&grant_type=authorization_code"
	// Send a POST request to the URL with no authorization headers
	resultBody := request(url, "POST", false)
	configuration.RefreshToken = gjson.Get(resultBody, "refresh_token").String()
	writeConfiguration()
}

func getBlogID() string {
	if configuration.Blog == "" {
		log.Fatalln("\"blog\" must be set in config.json")
	}
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
	// fmt.Print("\n")
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
