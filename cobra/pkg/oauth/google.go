package oauth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/charmbracelet/log"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Config holds the configuration for the Google OAuth2 client.
type Config struct {
	ClientID     string
	ClientSecret string
	Port         string
}

// GetToken starts the OAuth2 flow, opens a temporary web server to handle the callback,
// and returns the refresh token as a string.
func GetToken(cfg Config) (string, error) {
	// Configure the Google OAuth2 client.
	googleOauthConfig := &oauth2.Config{
		RedirectURL:  "http://localhost:8080/callback",
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/blogger"},
		Endpoint:     google.Endpoint,
	}

	// This channel will receive the refresh token when the auth flow is done.
	tokenCh := make(chan string)

	// This is the handler for the login route.
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		url := googleOauthConfig.AuthCodeURL("state", oauth2.AccessTypeOffline)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	})

	// This is the handler for the callback route.
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.FormValue("code")
		token, err := googleOauthConfig.Exchange(context.Background(), code)
		if err != nil {
			log.Fatal(err)
		}
		
		// Send the refresh token to the channel.
		tokenCh <- token.RefreshToken
	})

	// Start the server in a separate goroutine.
	go http.ListenAndServe(":"+cfg.Port, nil)
	fmt.Println("Go to http://localhost:" + cfg.Port + "/login to get the refresh token.")

	// Wait for the refresh token and return it.
	return <-tokenCh, nil
}

// Return an access token from a refresh token
func GetAccessToken(cfg Config, refreshToken string) (string, error) {
	// Configure the Google OAuth2 client.
	googleOauthConfig := &oauth2.Config{
		RedirectURL:  "http://localhost:" + cfg.Port + "/callback",
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/blogger"},
		Endpoint:     google.Endpoint,
	}

	// Create a new token from the refresh token
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	// Get a new access token
	tokenSource := googleOauthConfig.TokenSource(context.Background(), token)
	newToken, err := tokenSource.Token()
	if err != nil {
		log.Fatal(err)
	}

	return newToken.AccessToken, nil
}
