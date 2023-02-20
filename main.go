package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	htmltomd "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/alexflint/go-arg"
	mdlib "github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/parser"
	"github.com/imdario/mergo"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/skratchdot/open-golang/open"
	"github.com/tidwall/gjson"
)

type BloggerCmd struct {
	BlogAddress string `arg:"positional, required" help:"Blog address to get content from"`
	PostAddress string `arg:"positional, required" help:"Post slug to get content from"`
}

type FileCmd struct {
	Filepath string `arg:"positional, required" help:"Filepath to get content from"`
}

type PublishCmd struct {
	File    *FileCmd    `arg:"subcommand:file" help:"Publish from a file"`
	Blogger *BloggerCmd `arg:"subcommand:blogger" help:"Publish from Blogger"`
	// Perhaps instead of needing both a key and a value for destinations, parse a single value
	// For example, check if it's a file, and if so, check the file ending to determine the type
	// Maybe check if it contains blogger.com or dev.to to determine the type
	// Of course, an override would be nice
	Destinations map[string]string `arg:"--destinations, required" help:"Destination(s) to publish to\nAvailable destinations: blogger, dev.to, markdown, html\nMake sure to specify with <platform>=<Filepath, blog address, etc>"`
	Title        string            `arg:"-t,--title" help:"Specify custom title instead of using the default"`
	DryRun       bool              `arg:"-d,--dry-run" help:"Don't actually publish"`
}

type GoogleOauthCmd struct {
}

var args struct {
	// Subcommands
	GoogleOauth *GoogleOauthCmd `arg:"subcommand:google-oauth" help:"Store Google OAuth refresh token"`
	Publish     *PublishCmd     `arg:"subcommand:publish" help:"Publish to a destination"`

	// Google OAuth flags
	ClientId     string `arg:"--client-id, env:CLIENT_ID" help:"Google OAuth client ID"`
	ClientSecret string `arg:"--client-secret, env:CLIENT_SECRET" help:"Google OAuth client secret"`
	RefreshToken string `arg:"--refresh-token, env:REFRESH_TOKEN" help:"Google OAuth refresh token" default:""`

	// Misc flags
	LogLevel string `arg:"--log-level, env:LOG_LEVEL" help:"\"debug\", \"info\", \"warning\", \"error\", or \"fatal\"" default:"info"`
	LogColor bool   `arg:"--log-color, env:LOG_COLOR" help:"Force colored logs" default:"false"`
}

func main() {
	godotenv.Load(".env")
	arg.MustParse(&args)

	logrus.SetOutput(os.Stdout)
	logrus.SetFormatter(&logrus.TextFormatter{PadLevelText: true, DisableQuote: true, ForceColors: args.LogColor, DisableColors: !args.LogColor})
	if args.LogLevel == "debug" {
		logrus.SetLevel(logrus.DebugLevel)
		// Enable line numbers in debug logs - Doesn't help too much since a fatal error still needs to be debugged
		logrus.SetReportCaller(true)
	} else if args.LogLevel == "info" {
		logrus.SetLevel(logrus.InfoLevel)
	} else if args.LogLevel == "warning" {
		logrus.SetLevel(logrus.WarnLevel)
	} else if args.LogLevel == "error" {
		logrus.SetLevel(logrus.ErrorLevel)
	} else if args.LogLevel == "fatal" {
		logrus.SetLevel(logrus.FatalLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	switch {
	case args.GoogleOauth != nil:
		_, err := getAccessToken()
		if err != nil {
			logrus.Fatal(err)
		}
	case args.Publish != nil:
		var (
			title    string
			html     string
			markdown string
			err      error
		)
		switch {
		case args.Publish.Blogger != nil:
			title, html, markdown, err = getBloggerPost(args.Publish.Blogger.BlogAddress, args.Publish.Blogger.PostAddress)
			if err != nil {
				logrus.Fatal(err)
			}
		case args.Publish.File != nil:
			title, html, markdown, err = getFilePost(args.Publish.File.Filepath, args.Publish.Title)
			if err != nil {
				logrus.Fatal(err)
			}
		default:
			// Could add an interactive mode here for user-friendliness
			logrus.Fatal("No subcommand specified")
		}
		if args.Publish.DryRun {
			logrus.Debugf("Title: %s | HTML: %s | Markdown: %s", title, html, markdown)
		} else {
			err = publishPost(title, html, markdown, args.Publish.Destinations)
		}
		if err != nil {
			logrus.Fatal(err)
		}
	}
}

func publishPost(title string, html string, markdown string, destinations map[string]string) error {
	for destination, destinationSpecifier := range destinations {
		switch destination {
		case "blogger":
			// Get the blog ID
			blogId, err := getBlogId(destinationSpecifier)
			if err != nil {
				return err
			}
			// Publish to Blogger
			logrus.Info("Publishing to Blogger")
			url := "https://www.googleapis.com/blogger/v3/blogs/" + blogId + "/posts/"
			payloadMap := map[string]interface{}{
				"kind": "blogger#post",
				"blog": map[string]string{
					"id": blogId,
				},
				"title":   title,
				"content": html,
			}
			accessToken, err := getAccessToken()
			if err != nil {
				return err
			}
			_, err = request(url, "POST", accessToken, payloadMap)
			if err != nil {
				return err
			}
		case "markdown":
			logrus.Info("Publishing to Markdown")
			cleanPathToFile := filepath.Clean(destinationSpecifier)
			// Open the file in write-only mode (600) with append and create
			file, err := os.OpenFile(cleanPathToFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
			if err != nil {
				return err
			}
			// Write markdown to file
			_, err = file.WriteString(markdown)
			if err != nil {
				return err
			}
			// Close the file
			err = file.Close()
			if err != nil {
				return err
			}
			// Put dev.to stuff here

		}
	}
	return nil
}
func storeRefreshToken() (string, error) { // Rename to getRefreshToken(), perhaps?
	err := checkNeededFlags(map[string]string{"clientId": args.ClientId, "clientSecret": args.ClientSecret})
	if err != nil {
		return "", err
	}
	// Get the authorization code from the user
	url := "https://accounts.google.com/o/oauth2/v2/auth?client_id=" + args.ClientId + "&redirect_uri=https%3A%2F%2Foauthcodeviewer.netlify.app&scope=https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fblogger&response_type=code&access_type=offline&prompt=consent"
	// Open the URL in the default browser
	err = open.Run(url)
	fmt.Println("If the link didn't open, please manually go to the following link in your browser:")
	// Print the URL
	fmt.Printf("\n%v\n\n", url)
	if err != nil {
		return "", err
	}
	fmt.Println("Input the authorization code below")
	authorizationCode, err := singleLineInput()
	if err != nil {
		return "", err
	}

	// Get refresh token using the authorization code given by the user
	url = "https://oauth2.googleapis.com/token?client_id=" + args.ClientId + "&client_secret=" + args.ClientSecret + "&code=" + authorizationCode + "&redirect_uri=https%3A%2F%2Foauthcodeviewer.netlify.app&grant_type=authorization_code"
	// Send a POST request to the URL with no authorization headers
	resultBody, err := request(url, "POST", "", nil)
	if err != nil {
		return "", err
	}
	googleRefreshToken := gjson.Get(resultBody, "refresh_token").String()
	logrus.Debugf("Refresh token: %s", googleRefreshToken)
	// Merge the new environment variable with the existing environment variables using Mergo
	env := map[string]string{"REFRESH_TOKEN": googleRefreshToken}
	oldEnv, err := godotenv.Read()
	if err != nil {
		return "", err
	}
	err = mergo.Merge(&env, oldEnv)
	// for key, value := range oldEnv {
	// 		env[key] = value
	// }

	if err != nil {
		return "", err
	}

	logrus.Debugf("New environment variables: %v | Old enviroment variables %v", env, oldEnv)
	// May want to use filepath.Join() here
	err = godotenv.Write(env, ".env")
	if err != nil {
		return "", err
	}
	return googleRefreshToken, nil
}

func getAccessToken() (string, error) {
	err := checkNeededFlags(map[string]string{"clientId": args.ClientId, "clientSecret": args.ClientSecret})
	if err != nil {
		return "", err
	}
	var googleRefreshToken string
	// Check if there is a refresh token present
	if args.RefreshToken == "" {
		logrus.Print("No refresh token found. Please input the following information to get a refresh token.\n")
		googleRefreshToken, err = storeRefreshToken()
		if err != nil {
			return "", err
		}
	} else {
		googleRefreshToken = args.RefreshToken

	}

	// Get access token using the refresh token
	url := "https://oauth2.googleapis.com/token?client_id=" + args.ClientId + "&client_secret=" + args.ClientSecret + "&refresh_token=" + googleRefreshToken + "&redirect_uri=https%3A%2F%2Foauthcodeviewer.netlify.app&grant_type=refresh_token"
	// Send a POST request to the URL with no authorization headers
	resultBody, err := request(url, "POST", "", nil)
	if err != nil {
		return "", err
	}
	// Get the authorization token
	accessToken := gjson.Get(resultBody, "access_token").String()
	if accessToken == "" {
		return "", errors.New("no access token found")
	} else {
		logrus.Debugf("Access token: %s", accessToken)

	}
	return accessToken, nil
}

func singleLineInput() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	input = strings.TrimSpace(input)
	// fmt.Print("\n")
	return input, nil
}

func request(url string, requestType string, bearerAuth string, payloadMap map[string]interface{}) (string, error) {
	// Send a request to the URL, with the URL which was passed to the function
	var req *http.Request
	var err error
	// If payloadMap is nil, don't send a payload
	if payloadMap == nil {
		req, err = http.NewRequest(requestType, url, nil)
		if err != nil {
			return "", err
		}
	} else {
		// logrus.Debugf("Payload map: %v", payloadMap)
		payloadBytes, err := json.Marshal(payloadMap)
		if err != nil {
			return "", err
		}
		payload := strings.NewReader(string(payloadBytes))
		req, err = http.NewRequest(requestType, url, payload)
		if err != nil {
			return "", err
		}
	}

	// If the bearerAuth parameter is true, set the Authorization header with an access token
	if bearerAuth != "" {
		req.Header.Add("Authorization", "Bearer "+bearerAuth)
	}
	// Make the actual request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	// Convert the result body to a string and then return it
	resultBodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	err = res.Body.Close()
	if err != nil {
		return "", err
	}
	// Debug the status code
	// logrus.Debugf("Sending %s request to %s with payload %v, bearer authorization %s. Got status code %d", requestType, url, payloadMap, bearerAuth, res.StatusCode)
	return string(resultBodyBytes), nil
}

func getBlogId(blogAddress string) (string, error) {
	// Send a GET request to the URL with bearer authorization
	accessToken, err := getAccessToken()
	logrus.Debugf("Blog address: %s", blogAddress)
	url := "https://www.googleapis.com/blogger/v3/blogs/byurl?url=https%3A%2F%2F" + blogAddress
	if err != nil {
		return "", err
	}
	resultBody, err := request(url, "GET", accessToken, nil)
	if err != nil {
		return "", err
	}
	// logrus.Debugf("Result body: %s", resultBody)
	// Get the blog ID
	id := gjson.Get(resultBody, "id").String()
	if id == "" {
		return "", errors.New("no blog ID found")
	}
	return id, nil
}

func getBloggerPost(blogAddress string, postAddress string) (string, string, string, error) {
	path := strings.Replace(postAddress, blogAddress, "", 1)
	blogID, err := getBlogId(blogAddress)
	logrus.Debugf("Blog ID: %s | Path: %s", blogID, path)
	if err != nil {
		return "", "", "", err
	}
	accessToken, err := getAccessToken()
	if err != nil {
		return "", "", "", err
	}
	// https://www.googleapis.com/blogger/v3/blogs/[BLOG_ID]/posts/bypath?path=/{YEAR}/{MONTH}/{POST}.html
	url := "https://www.googleapis.com/blogger/v3/blogs/" + blogID + "/posts/bypath?path=" + path
	resultBody, err := request(url, "GET", accessToken, nil)
	if err != nil {
		return "", "", "", err
	}
	html := gjson.Get(resultBody, "content").String()
	title := gjson.Get(resultBody, "title").String()
	markdown, err := htmltomd.NewConverter("", true, nil).ConvertString(html)
	if title == "" && html == "" && markdown == "" {
		logrus.Debug(url)
		return "", "", "", errors.New("no post found")
	}
	if err != nil {
		return "", "", "", err
	}
	return title, html, markdown, nil
}

func getFilePost(pathToFile string, defaultTitle string) (string, string, string, error) {
	// If the file path is empty, ask for it (might be a good idea to remove this)
	if pathToFile == "" {
		fmt.Println("Enter the path to the file")
		var err error
		pathToFile, err = singleLineInput()
		if err != nil {
			return "", "", "", err
		}
	}

	cleanPathToFile := filepath.Clean(pathToFile)

	// Check if the file exists
	_, err := os.Stat(cleanPathToFile)
	if errors.Is(err, os.ErrNotExist) {
		return "", "", "", errors.New("file does not exist")
	}

	// Read the file
	fileBytes, err := os.ReadFile(cleanPathToFile)
	if err != nil {
		return "", "", "", err
	}
	fileContent := string(fileBytes)
	logrus.Debug("The file was read successfully")
	var (
		title    string
		html     string
		markdown string
	)

	// Check file extension
	fileExtension := filepath.Ext(pathToFile)
	switch fileExtension {
	case ".html", ".htm":
		// Set HTML to the file content
		html = fileContent
		// Convert to Markdown
		markdown, err = htmltomd.NewConverter("", true, nil).ConvertString(fileContent)
		if err != nil {
			return "", "", "", err
		}

	case ".md", ".markdown":
		// Set Markdown to the file content
		markdown = fileContent
		// Convert to HTML
		extensions := parser.CommonExtensions | parser.AutoHeadingIDs
		parser := parser.NewWithExtensions(extensions)
		html = string(mdlib.ToHTML([]byte(fileContent), parser, nil))

	case ".txt":
		// Not sure if plain text should be supported but it can easily be removed later

		// Set Markdown to the file content
		markdown = fileContent
		// Convert to HTML
		extensions := parser.CommonExtensions | parser.AutoHeadingIDs
		parser := parser.NewWithExtensions(extensions)
		html = string(mdlib.ToHTML([]byte(fileContent), parser, nil))
	default:
		return "", "", "", errors.New("file extension not supported")
	}
	if defaultTitle != "" {
		title = defaultTitle
	} else {
		// Get the file name without the extension
		fileName := filepath.Base(pathToFile)
		fileNameWithoutExtension := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		// Replace underscores with spaces (Might be a good idea to make this optional)
		title = strings.ReplaceAll(fileNameWithoutExtension, "_", " ")
	}
	return title, html, markdown, nil
}

func checkNeededFlags(flags map[string]string) error {
	message := "The following must be set"
	for name, value := range flags {
		if value == "" {
			message += "\n- " + name
		}
		if message != "The following must be set" {
			return errors.New(message)
		}
	}
	return nil
}

// Check if the file exists -> Check file extension -> Convert to the other formats -> Check if user-defined title is set -> If user-defined title is not set, use the file name as the title -> Return the title, HTML, and Markdown and hopefully no errors.
