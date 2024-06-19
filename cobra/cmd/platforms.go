package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gosimple/slug"
	"gopkg.in/yaml.v2"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/charmbracelet/log"
	"github.com/go-resty/resty/v2"
	"github.com/slashtechno/cross-blogger/cobra/pkg/oauth"
	"github.com/spf13/afero"
)

type Destination interface {
	Push(PostData, PushPullOptions) error
	GetName() string
	GetType() string
}

type Source interface {
	Pull(PushPullOptions) (PostData, error)
	GetName() string
	GetType() string
}

type PushPullOptions struct {
	AccessToken string
	BlogId      string
	PostUrl     string
}

type PostData struct {
	Title    string
	Html     string
	Markdown string
	// Other fields that are probably needed are canonical URL, publish date, and description
	CanonicalUrl string
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
	// https://developers.google.com/blogger/docs/3.0/reference/posts/delete
	Overwrite bool
}

// Return the access token, refresh token (if one was not provided), and an error (if one occurred).
// The access and refresh tokens are only returned if an error did not occur.
// In Google Cloud, create OAuth client credentials for a desktop app and enable the Blogger API.
func (b Blogger) authorize(clientId string, clientSecret string, providedRefreshToken string) (string, string, error) {
	oauthConfig := oauth.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Port:         "8080",
	}
	var refreshToken string
	var err error
	if providedRefreshToken != "" {
		log.Info("Using provided refresh token")
		refreshToken = providedRefreshToken
	} else {
		log.Info("No refresh token provided, starting OAuth flow")
		refreshToken, err = oauth.GetGoogleRefreshToken(oauthConfig)
		if err != nil {
			return "", "", err
		}
	}
	accessToken, err := oauth.GetGoogleAccessToken(oauthConfig, refreshToken)
	if err != nil {
		// Not returning the refresh token because it may have been invalid
		return "", "", err
	}
	log.Info("", "access token", accessToken)
	if providedRefreshToken != "" {
		return accessToken, providedRefreshToken, nil
	}
	return accessToken, refreshToken, nil

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
func (b Blogger) Pull(options PushPullOptions) (PostData, error) {
	log.Info("Blogger pull called", "options", options)
	postPath := strings.Replace(options.PostUrl, b.BlogUrl, "", 1)
	client := resty.New()
	resp, err := client.R().SetHeader("Authorization", fmt.Sprintf("Bearer %s", options.AccessToken)).SetResult(&map[string]interface{}{}).Get("https://www.googleapis.com/blogger/v3/blogs/" + options.BlogId + "/posts/bypath?path=" + postPath)
	if err != nil {
		return PostData{}, err
	}
	if resp.StatusCode() != 200 {
		return PostData{}, fmt.Errorf("failed to get post: %s", resp.String())
	}
	// Get the keys "title" and "content" from the response
	result := (*resp.Result().(*map[string]interface{}))
	title, ok := result["title"].(string)
	if !ok {
		return PostData{}, fmt.Errorf("title not found in response or is not a string")
	}
	html, ok := result["content"].(string)
	if !ok {
		return PostData{}, fmt.Errorf("content not found in response or is not a string")
	}
	canonicalUrl, ok := result["url"].(string)
	if !ok {
		return PostData{}, fmt.Errorf("url not found in response or is not a string")
	}
	// Convert the HTML to Markdown
	markdown, err := md.NewConverter("", true, nil).ConvertString(html)
	if err != nil {
		return PostData{}, err
	}
	return PostData{
		Title:        title,
		Html:         html,
		Markdown:     markdown,
		CanonicalUrl: canonicalUrl,
	}, nil

}
func (b Blogger) Push(data PostData, options PushPullOptions) error {
	// Set the client
	client := resty.New()
	blogId := options.BlogId

	// Delete any post with the same ID
	if b.Overwrite {
		// Get the list of existing posts
		resp, err := client.R().
			SetHeader("Authorization", fmt.Sprintf("Bearer %s", options.AccessToken)).
			SetResult(&map[string]interface{}{}).
			Get("https://www.googleapis.com/blogger/v3/blogs/" + blogId + "/posts")
		if err != nil {
			return err
		}
		posts := (*resp.Result().(*map[string]interface{}))["items"].([]interface{})

		// Check if a post with the same title already exists
		for _, p := range posts {
			post := p.(map[string]interface{})
			if post["title"].(string) == data.Title {
				// Delete the post
				_, err := client.R().
					SetQueryParam("useTrash", "true").
					SetHeader("Authorization", fmt.Sprintf("Bearer %s", options.AccessToken)).
					Delete("https://www.googleapis.com/blogger/v3/blogs/" + blogId + "/posts/" + post["id"].(string))
				if err != nil {
					return err
				}
				log.Info("Moved post with the same title to trash", "title", data.Title)
				// If break is used and there are multiple posts with the same title, only the first one will be deleted
				// break
			}
		}
	}
	log.Warn("Blogger does not support setting the canonical URL")
	// Prepare the request
	req := client.R().SetHeader("Authorization", fmt.Sprintf("Bearer %s", options.AccessToken)).SetBody(map[string]interface{}{
		"title":   data.Title,
		"content": data.Html,
		// "url":     data.CanonicalUrl,
	}).SetResult(&map[string]interface{}{})
	// Make the request
	resp, err := req.Post("https://www.googleapis.com/blogger/v3/blogs/" + blogId + "/posts")
	if err != nil {
		return err
	}
	if resp.StatusCode() != 200 {
		return fmt.Errorf("failed to post: %s", resp.String())
	}
	result := (*resp.Result().(*map[string]interface{}))
	log.Debug("Posted successfully", "result", result)
	return nil
}
func (b Blogger) GetName() string { return b.Name }
func (b Blogger) GetType() string { return "blogger" }

type Markdown struct {
	Name string
	// ContentDir, for retrieving, should only be used if treating the passed post path as relative results in no file found
	ContentDir string
	Overwrite  bool
}

func (m Markdown) GetName() string { return m.Name }
func (m Markdown) GetType() string { return "markdown" }

// Push the data to the contentdir with the title as the filename using gosimple/slug.
// The markdown file should have YAML frontmatter compatible with Hugo.
func (m Markdown) Push(data PostData, options PushPullOptions) error {
	// Create the file, if it exists, log an error and return
	fs := afero.NewOsFs()
	slug := slug.Make(data.Title)
	// Clean the slug to remove any characters that may cause issues with the filesystem
	slug = filepath.Clean(slug)
	filePath := filepath.Join(m.ContentDir, slug+".md")
	// Create parent directories if they don't exist
	dirPath := filepath.Dir(filePath)
	if _, err := fs.Stat(dirPath); os.IsNotExist(err) {
		errDir := fs.MkdirAll(dirPath, 0755)
		if errDir != nil {
			log.Error("failed to create directory", "directory", dirPath)
			return errDir
		}
	}
	// Check if the file already exists
	if _, err := fs.Stat(filePath); err == nil && !m.Overwrite {
		return fmt.Errorf("file already exists and overwrite is false for file: %s", filePath)
	} else if err != nil && !os.IsNotExist(err) { // If the error is not a "file does not exist" error
		return err
	} else if err == nil && m.Overwrite { // If the file exists and overwrite is true, remove the file
		log.Info("Removing file as overwrite is true", "file", filePath)
		err := fs.Remove(filePath)
		if err != nil {
			return err
		}
	}

	// Create the file
	file, err := fs.Create(filePath)
	if err != nil {
		return err
	}
	// After the function returns, close the file
	defer file.Close()
	// Create the frontmatter
	frontmatter := struct {
		Title        string `yaml:"title"`
		CanonicalUrl string `yaml:"canonicalUrl"`
	}{
		Title:        data.Title,
		CanonicalUrl: data.CanonicalUrl,
	}
	// Convert the frontmatter to YAML
	frontmatterYaml, err := yaml.Marshal(frontmatter)
	if err != nil {
		return err
	}
	content := fmt.Sprintf("---\n%s---\n\n%s", frontmatterYaml, data.Markdown)
	log.Debug("Writing content", "content", content, "file", filePath)
	_, err = file.WriteString(content)
	if err != nil {
		return err
	}
	return nil

}
func (m Markdown) Pull(options PushPullOptions) (PostData, error) {
	log.Info("Markdown pull called", "options", options)
	return PostData{}, nil
}

func CreateDestination(destMap map[string]interface{}) (Destination, error) {
	name, ok := destMap["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required")
	}

	switch destMap["type"] {
	case "blogger":
		blogUrl, ok := destMap["blog_url"].(string)
		if !ok || blogUrl == "" {
			return nil, fmt.Errorf("blog_url is required for blogger")
		}

		overwrite, _ := destMap["overwrite"].(bool) // If not set or not a bool, defaults to false

		return Blogger{
			Name:      name,
			BlogUrl:   blogUrl,
			Overwrite: overwrite,
		}, nil
	case "markdown":
		contentDir, ok := destMap["content_dir"].(string)
		if !ok || contentDir == "" {
			return nil, fmt.Errorf("content_dir is required for markdown")
		}

		overwrite, _ := destMap["overwrite"].(bool) // If not set or not a bool, defaults to false

		return Markdown{
			Name:       name,
			ContentDir: contentDir,
			Overwrite:  overwrite,
		}, nil
	default:
		return nil, fmt.Errorf("unknown destination type: %s", destMap["type"])
	}
}

func CreateSource(sourceMap map[string]interface{}) (Source, error) {
	// In Go, ifa type assertion fails, it will return the zero value of the type and false.
	name, ok := sourceMap["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required")
	}

	switch sourceMap["type"] {
	case "blogger":
		blogUrl, ok := sourceMap["blog_url"].(string)
		if !ok || blogUrl == "" {
			return nil, fmt.Errorf("blog_url is required for blogger")
		}

		return Blogger{
			Name:    name,
			BlogUrl: blogUrl,
		}, nil
	case "file":
		// If the content_dir is not set, set it to null as its not required
		contentDir, _ := sourceMap["content_dir"].(string)
		return Markdown{
			Name:       name,
			ContentDir: contentDir,
		}, nil
	default:
		return nil, fmt.Errorf("unknown source type: %s", sourceMap["type"])
	}
}
