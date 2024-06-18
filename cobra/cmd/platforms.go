package cmd

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/go-resty/resty/v2"
	"github.com/slashtechno/cross-blogger/cobra/pkg/oauth"
)

type Destination interface {
	Push()
}

type Source interface {
	Pull()
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

func (b Blogger) authorize(clientId string, clientSecret string) (string, error) {
	oauthConfig := oauth.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Port:         "8080",
	}
	refreshToken, err := oauth.GetToken(oauthConfig)
	if err != nil {
		return "", err
	}

	accessToken, err := oauth.GetAccessToken(oauthConfig, refreshToken)
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
func (b Blogger) Push() {
	log.Error("not implemented")
}
func (b Blogger) Pull(postAddress string, accessToken string) {

}

type Markdown struct {
	Name       string
	ContentDir string
}

func (m Markdown) Push() {
	log.Error("not implemented")
}

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
