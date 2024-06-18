package cmd

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/slashtechno/cross-blogger/cobra/pkg/oauth"
	"github.com/spf13/viper"
)

type Platform interface {
	Push()
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
	BlogId  string
}

func (b Blogger) authorize() (string, error) {
	oauthConfig := oauth.Config{
		ClientID:     viper.GetString("client-id"),
		ClientSecret: viper.GetString("client-secret"),
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

type Markdown struct {
	Name       string
	ContentDir string
}

func CreateDestination(destMap map[string]interface{}) (interface{}, error) {
	switch destMap["type"] {
	case "blogger":
		return Blogger{
			Name:    destMap["name"].(string),
			BlogUrl: destMap["blog_url"].(string),
			BlogId:  destMap["blog_id"].(string),
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
