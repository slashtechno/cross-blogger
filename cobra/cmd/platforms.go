package cmd

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/go-resty/resty/v2"
	"github.com/slashtechno/cross-blogger/cobra/pkg/oauth"
)

type Destination interface {
	Push()
	GetName() string
}

type Source interface {
	Pull(SourceOptions)
	GetName() string
	GetType() string
}

type SourceOptions struct {
	AccessToken string
	BlogId      string
	Filepath    string
}

// type PlatformParent struct {
// 	Name string
// }

// func (p PlatformParent) Push() {
// 	log.Error("child class must implement this method")
// }

type Blogger struct {
	Name    string
	BlogUrl string
}

func (b Blogger) authorize(clientId string, clientSecret string, providedRefreshToken string) (string, error) {
	oauthConfig := oauth.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Port:         "8080",
	}
	var refreshToken string
	var err error
	if providedRefreshToken == "" {
		refreshToken, err = oauth.GetGoogleRefreshToken(oauthConfig)
		if err != nil {
			return "", err
		}
	} else {
		refreshToken = providedRefreshToken
	}
	accessToken, err := oauth.GetGoogleAccessToken(oauthConfig, refreshToken)
	if err != nil {
		return "", err
	}
	log.Info("", "access token", accessToken)
	return accessToken, nil
}
func (b Blogger) getBlogId(accessToken string) (string, error) {
	client := resty.New()
	resp, err := client.R().SetHeader("Authorization", fmt.Sprintf("Bearer %s", accessToken)).SetResult(&map[string]interface{}{}).Get("https://www.googleapis.com/blogger/v3/blogs/byurl?url=" + b.BlogUrl)
	if err != nil {
		return "", err
	}
	if resp.StatusCode() != 200 {
		return "", fmt.Errorf("failed to get blog id: %s", resp.String())
	}
	// Get the key "id" from the response
	result := (*resp.Result().(*map[string]interface{}))
	id, ok := result["id"]
	if !ok {
		return "", fmt.Errorf("id not found in response")
	}
	return id.(string), nil
}
func (b Blogger) Pull(options SourceOptions) {
	log.Info("Blogger pull called", "options", options)
}
func (b Blogger) Push()           { log.Error("not implemented") }
func (b Blogger) GetName() string { return b.Name }
func (b Blogger) GetType() string { return "blogger" }

type Markdown struct {
	Name       string
	ContentDir string
}

func (m Markdown) Push()                      { log.Error("not implemented") }
func (m Markdown) Pull(options SourceOptions) { log.Error("not implemented") }
func (m Markdown) GetName() string            { return m.Name }
func (m Markdown) GetType() string            { return "markdown" }

func CreateDestination(destMap map[string]interface{}) (Destination, error) {
	switch destMap["type"] {
	case "blogger":
		return Blogger{
			Name:    destMap["name"].(string),
			BlogUrl: destMap["blog_url"].(string),
		}, nil
	case "markdown":
		return Markdown{
			Name:       destMap["name"].(string),
			ContentDir: destMap["content_dir"].(string),
		}, nil
	default:
		return nil, fmt.Errorf("unknown destination type: %s", destMap["type"])
	}
}

func CreateSource(sourceMap map[string]interface{}) (Source, error) {
	switch sourceMap["type"] {
	case "blogger":
		return Blogger{
			Name:    sourceMap["name"].(string),
			BlogUrl: sourceMap["blog_url"].(string),
		}, nil
	case "file":
		return Markdown{
			Name:       sourceMap["name"].(string),
			ContentDir: sourceMap["content_dir"].(string),
		}, nil
	default:
		return nil, fmt.Errorf("unknown source type: %s", sourceMap["type"])
	}
}
