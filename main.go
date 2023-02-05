package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/joho/godotenv"
	"github.com/tidwall/gjson"
)

type PublishCmd struct {
}

type GoogleOauthCmd struct {
	ClientId     string `arg:"--client-id, env:CLIENT_ID" help:"Google OAuth client ID"`
	ClientSecret string `arg:"--client-secret, env:CLIENT_SECRET" help:"Google OAuth client secret"`
}

var args struct {
	GoogleOauth *GoogleOauthCmd `arg:"subcommand:google-oauth"`
	LogLevel    string          `arg:"--log-level, env:LOG_LEVEL" help:"\"debug\", \"info\", \"warning\", \"error\", or \"fatal\"" default:"info"`
	LogColor    bool            `arg:"--log-color, env:LOG_COLOR" help:"Force colored logs" default:"true"`
}

var googleRefreshToken string

func main() {
	godotenv.Load(".env")	// Load the .env file
	arg.MustParse(&args)

	switch {
	case args.GoogleOauth != nil:
		// googleOauth()
	}
}

func storeRefreshToken() error { // Rename to getRefreshToken(), perhaps?
	message := "The following must be set"
	if args.GoogleOauth.ClientId == "" {
		message += "\n- client_id"
	}
	if args.GoogleOauth.ClientSecret == "" {
		message += "\n- client_secret"
	}
	if message != "The following must be set in config.json" {
		return errors.New(message)
	}

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
	return nil
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
	return string(resultBodyBytes), nil
}
