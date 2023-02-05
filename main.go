package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

type BloggerCmd struct {
	ClientId     string `arg:"--client-id, env:CLIENT_ID, required" help:"Google OAuth client ID"`
	ClientSecret string `arg:"--client-secret, env:CLIENT_SECRET, required" help:"Google OAuth client secret"`
	RefreshToken string `arg:"--refresh-token, env:REFRESH_TOKEN" help:"Google OAuth refresh token"`

	BlogAddress string `arg:"positional, required" help:"Blog address to publish to"`
}

type PublishCmd struct {
	// Source          string `arg:"-s,--source" help:"What source to use\nAvailable sources: blogger, dev.to, markdown, html\ndev.to, markdown, and html work with source-specifier"`
	Blogger      *BloggerCmd       `arg:"subcommand:blogger" help:"Publish to Blogger"`
	Destinations map[string]string `arg:"--destinations, required" help:"Destination(s) to publish to\nAvailable destinations: blogger, dev.to, markdown, html\nMake sure to specify with <platform>=<Filepath, blog address, etc>"` // TODO: Make this a map
	Title        string            `arg:"-t,--title" help:"Specify custom title instead of using the default"`
}

type GoogleOauthCmd struct {
	ClientId     string `arg:"--client-id, env:CLIENT_ID, required" help:"Google OAuth client ID"`
	ClientSecret string `arg:"--client-secret, env:CLIENT_SECRET, required" help:"Google OAuth client secret"`
	RefreshToken string `arg:"--refresh-token, env:REFRESH_TOKEN" help:"Google OAuth refresh token"`
}

var args struct {
	GoogleOauth *GoogleOauthCmd `arg:"subcommand:google-oauth" help:"Store Google OAuth refresh token"`
	Publish     *PublishCmd     `arg:"subcommand:publish" help:"Publish to a destination"`
	LogLevel    string          `arg:"--log-level, env:LOG_LEVEL" help:"\"debug\", \"info\", \"warning\", \"error\", or \"fatal\"" default:"info"`
	LogColor    bool            `arg:"--log-color, env:LOG_COLOR" help:"Force colored logs" default:"false"`
}

var googleRefreshToken string

func main() {
	godotenv.Load(".env")
	arg.MustParse(&args)
	googleRefreshToken = args.GoogleOauth.RefreshToken

	logrus.SetOutput(os.Stdout)
	logrus.SetFormatter(&logrus.TextFormatter{PadLevelText: true, DisableQuote: true, ForceColors: args.LogColor, DisableColors: !args.LogColor})

	switch {
	case args.GoogleOauth != nil:
		_, err := getAccessToken()
		if err != nil {
			logrus.Fatal(err)
		}
	}
}

func storeRefreshToken() error { // Rename to getRefreshToken(), perhaps?

	// Get the authorization code from the user
	fmt.Println("Please go to the following link in your browser:")
	fmt.Println("\nhttps://accounts.google.com/o/oauth2/v2/auth?client_id=" + args.GoogleOauth.ClientId + "&redirect_uri=https%3A%2F%2Foauthcodeviewer.netlify.app&scope=https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fblogger&response_type=code&access_type=offline&prompt=consent\n")
	fmt.Println("Input the authorization code below")
	authorizationCode, err := singleLineInput()
	if err != nil {
		return err
	}

	// Get refresh token using the authorization code given by the user
	url := "https://oauth2.googleapis.com/token?client_id=" + args.GoogleOauth.ClientId + "&client_secret=" + args.GoogleOauth.ClientSecret + "&code=" + authorizationCode + "&redirect_uri=https%3A%2F%2Foauthcodeviewer.netlify.app&grant_type=authorization_code"
	// Send a POST request to the URL with no authorization headers
	resultBody, err := request(url, "POST", "")
	if err != nil {
		return err
	}
	googleRefreshToken = gjson.Get(resultBody, "refresh_token").String()
	env, _ := godotenv.Unmarshal("REFRESH_TOKEN=" + googleRefreshToken)
	// May want to use filepath.Join() here
	err = godotenv.Write(env, ".env")
	if err != nil {
		return err
	}
	return nil
}

func getAccessToken() (string, error) {

	// Check if there is a refresh token present
	if googleRefreshToken == "" {
		logrus.Print("No refresh token found. Please input the following information to get a refresh token.\n")
		storeRefreshToken()
	}

	// Get access token using the refresh token
	url := "https://oauth2.googleapis.com/token?client_id=" + args.GoogleOauth.ClientId + "&client_secret=" + args.GoogleOauth.ClientSecret + "&refresh_token=" + googleRefreshToken + "&redirect_uri=https%3A%2F%2Foauthcodeviewer.netlify.app&grant_type=refresh_token"
	// Send a POST request to the URL with no authorization headers
	resultBody, err := request(url, "POST", "")
	if err != nil {
		return "", err
	}
	// Get the authorization token
	accessToken := gjson.Get(resultBody, "access_token").String()
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

func request(url string, requestType string, bearerAuth string) (string, error) {
	// Send a request to the URL, with the URL which was passed to the function
	req, err := http.NewRequest(requestType, url, nil)
	if err != nil {
		return "", err
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
	defer res.Body.Close()
	resultBodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(resultBodyBytes), nil
}

func getBlogID() (string, error) {
	url := "https://www.googleapis.com/blogger/v3/blogs/byurl?url=" + args.Publish.Blogger.BlogAddress + "%2F"
	// Send a GET request to the URL with bearer authorization
	accessToken, err := getAccessToken()
	if err != nil {
		return "", err
	}
	resultBody, err := request(url, "GET", accessToken)
	if err != nil {
		return "", err
	}
	// Get the blog ID
	return gjson.Get(resultBody, "id").String(), nil
}
