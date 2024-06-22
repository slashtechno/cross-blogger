package platforms

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/gosimple/slug"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"gopkg.in/yaml.v2"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/charmbracelet/log"
	"github.com/go-resty/resty/v2"
	"github.com/slashtechno/cross-blogger/pkg/oauth"
	"github.com/slashtechno/cross-blogger/pkg/utils"
	"github.com/spf13/afero"
	"go.abhg.dev/goldmark/frontmatter"
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
type WatchableSource interface {
	Source
	Watch(time.Duration, PushPullOptions, chan<- PostData, chan<- error)
}

type PushPullOptions struct {
	AccessToken  string
	BlogId       string
	PostUrl      string
	Filepath     string
	RefreshToken string
	ClientId     string
	ClientSecret string
}

type PostData struct {
	Title       string
	Html        string
	Markdown    string
	Date        time.Time
	DateUpdated time.Time
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
	Overwrite  bool
	knownPosts []string
}

// Return the access token, refresh token (if one was not provided), and an error (if one occurred).
// The access and refresh tokens are only returned if an error did not occur.
// In Google Cloud, create OAuth client credentials for a desktop app and enable the Blogger API.
func (b Blogger) Authorize(clientId string, clientSecret string, providedRefreshToken string) (string, string, error) {
	oauthConfig := oauth.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Port:         "8080",
	}
	var refreshToken string
	var err error
	if providedRefreshToken != "" {
		log.Debug("Using provided refresh token")
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
	log.Debug("", "access token", accessToken)
	if providedRefreshToken != "" {
		return accessToken, providedRefreshToken, nil
	}
	return accessToken, refreshToken, nil

}
func (b Blogger) GetBlogId(accessToken string) (string, error) {
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
	// TODO: optionally only pull published posts
	// Compile a regex that matches both http and https schemes
	regex, err := regexp.Compile(`^https?:\/\/[^\/]+`)
	if err != nil {
		return PostData{}, fmt.Errorf("regex compilation error: %v", err)
	}
	// Use regex to replace the scheme and domain part of the URL
	postPath := regex.ReplaceAllString(options.PostUrl, "")
	client := resty.New()
	resp, err := client.R().SetHeader("Authorization", fmt.Sprintf("Bearer %s", options.AccessToken)).SetResult(&map[string]interface{}{}).Get("https://www.googleapis.com/blogger/v3/blogs/" + options.BlogId + "/posts/bypath?path=" + postPath)
	if err != nil {
		return PostData{}, err
	}
	if resp.StatusCode() != 200 {
		return PostData{}, fmt.Errorf("failed to get post: %s", resp.String())
	}
	log.Debug("Got response", "response", resp.String())
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
	// Date published is returned like `"published": "2024-06-19T09:37:00-07:00,`
	// Convert the HTML to Markdown
	rfcDate, ok := result["published"].(string)
	if !ok {
		return PostData{}, fmt.Errorf("published date not found in response or is not a string")
	}
	date, err := time.Parse(time.RFC3339, rfcDate)
	if err != nil {
		return PostData{}, err
	}
	rfcDateUpdated, ok := result["updated"].(string)
	if !ok {
		return PostData{}, fmt.Errorf("updated date not found in response or is not a string")
	}
	dateUpdated, err := time.Parse(time.RFC3339, rfcDateUpdated)
	if err != nil {
		return PostData{}, err
	}
	markdown, err := md.NewConverter("", true, nil).ConvertString(html)
	if err != nil {
		return PostData{}, err
	}
	return PostData{
		Title:        title,
		Html:         html,
		Markdown:     markdown,
		Date:         date,
		DateUpdated:  dateUpdated,
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

// Every interval, check for new posts (posts that haven't been seen before) and send them to the postChan channel.
func (b *Blogger) Watch(interval time.Duration, options PushPullOptions, postChan chan<- PostData, errChan chan<- error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		// Fetch new posts from Blogger
		// Refresh the access token
		var err error
		options.AccessToken, _, err = b.Authorize(options.ClientId, options.ClientSecret, options.RefreshToken)
		if err != nil {
			errChan <- err
			return
		}
		posts, err := b.fetchNewPosts(options)
		if err != nil {
			errChan <- err
			// Continue will skip the rest of the loop and go to the next iteration
			// Return will end the goroutine if an error occurs
			// continue
			return
		}

		// Send new posts to the channel
		for _, post := range posts {
			postChan <- post
		}
	}
}

// Get posts that haven't been seen before and return them
func (b *Blogger) fetchNewPosts(options PushPullOptions) ([]PostData, error) {
	// Get the list of posts
	client := resty.New()
	resp, err := client.R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", options.AccessToken)).
		SetResult(&map[string]interface{}{}).
		Get("https://www.googleapis.com/blogger/v3/blogs/" + options.BlogId + "/posts")
	if err != nil {
		return nil, err
	}
	// Assert that posts is a pointer to a map. However, dereference it to get the map
	posts := (*resp.Result().(*map[string]interface{}))["items"].([]interface{})
	// log.Debug("", "posts", posts)
	// Check if the list of known posts is empty
	if len(b.knownPosts) == 0 {
		// Add all the posts to the list of known posts
		for _, p := range posts {
			post := p.(map[string]interface{})
			b.knownPosts = append(b.knownPosts, post["id"].(string))
		}
		// Return an empty slice because there are no new posts
		log.Debug("No known posts", "knownPosts", b.knownPosts)
		return []PostData{}, nil
	} else {
		// Get the new posts
		newPosts := []PostData{}
		// TODO: Make this concurrent
		// TODO: Don't append a post if it's unpublished
		for _, p := range posts {
			post := p.(map[string]interface{})
			// Check if the post is new
			if !utils.ContainsString(b.knownPosts, post["id"].(string)) {
				// Add the post to the list of known posts
				b.knownPosts = append(b.knownPosts, post["id"].(string))
				// Add the post to the list of new posts (run Pull)
				postData, err := b.Pull(PushPullOptions{
					// Watch() passes a fresh access token
					AccessToken: options.AccessToken,
					BlogId:      options.BlogId,
					PostUrl:     post["url"].(string),
				})
				if err != nil {
					return nil, err
				}
				newPosts = append(newPosts, postData)
			}
		}
		return newPosts, nil
	}
	// It is not possible to reach this point because the function will return before this point
	// return nil, errors.New("unreachable")
}
func (b Blogger) GetName() string { return b.Name }
func (b Blogger) GetType() string { return "blogger" }

type Markdown struct {
	Name string
	// ContentDir, for retrieving, should only be used if treating the passed post path as relative results in no file found
	ContentDir string
	GitDir     string
	// Example: []string{"title", "date", "lastmod", "canonicalURL"}
	FrontmatterSelection []string
	FrontmatterMapping
	Overwrite bool
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
	// Add the frontmatter fields that are selected
	postFrontmatter := Frontmatter{
		Title: data.Title,
		// Date:         data.Date.Format(time.RFC3339),
		// DateUpdated:  data.DateUpdated.Format(time.RFC3339),
		CanonicalUrl: data.CanonicalUrl,
	}

	// Only add Date if it's not the zero value
	if !data.Date.IsZero() {
		postFrontmatter.Date = data.Date.Format(time.RFC3339)
	}

	// Only add DateUpdated if it's not the zero value
	if !data.DateUpdated.IsZero() {
		postFrontmatter.DateUpdated = data.DateUpdated.Format(time.RFC3339)
	}
	// Convert the frontmatter to YAML
	frontmatterYaml, err := yaml.Marshal(postFrontmatter.ToMap(m.FrontmatterMapping))
	if err != nil {
		return err
	}
	content := fmt.Sprintf("---\n%s---\n\n%s", frontmatterYaml, data.Markdown)
	log.Debug("Writing content", "content", content, "file", filePath)
	_, err = file.WriteString(content)
	if err != nil {
		return err
	}

	// If the Git directory is set, commit + push the changes
	if m.GitDir != "" {

		// Open the git repository
		// Clean up the Git directory path
		dirPath = filepath.Clean(m.GitDir)
		// No need to check if the directory exits since PlainOpen will return an error if a repos
		// Open the repository
		repo, err := git.PlainOpen(dirPath)
		if err != nil {
			return err
		}
		repoWorktree, err := repo.Worktree()
		if err != nil {
			return err
		}
		// Add the file
		_, err = repoWorktree.Add(filepath.Base(filePath))
		if err != nil {
			return err
		}
		// Commit the changes
		commitHash, err := repoWorktree.Commit("Add "+slug+".md", &git.CommitOptions{})
		if err != nil {
			return err
		}
		log.Info("Committed changes", "hash", commitHash.String())
		// Push the changes
		err = repo.Push(&git.PushOptions{})
		if err != nil {
			return err
		}
	}

	return nil

}
func (m Markdown) Pull(options PushPullOptions) (PostData, error) {
	// Get the file path
	fs := afero.NewOsFs()
	// Treat the post path as relative to the content dir
	// However, if the content dir does not exist or the file is not found, treat the post path as a normal path without the content dir
	filePath := filepath.Join(m.ContentDir, options.Filepath)
	if _, err := fs.Stat(filePath); os.IsNotExist(err) {
		filePath = options.Filepath
	}
	// Read the file
	data, err := afero.ReadFile(fs, filePath)
	if err != nil {
		return PostData{}, err
	}
	markdown := string(data)
	// Convert the markdown to HTML with Goldmark
	// Use the Frontmatter extension to get the frontmatter
	mdParser := goldmark.New(goldmark.WithExtensions(&frontmatter.Extender{}))
	ctx := parser.NewContext()
	var buf bytes.Buffer
	err = mdParser.Convert([]byte(markdown), &buf, parser.WithContext(ctx))
	if err != nil {
		return PostData{}, err
	}
	// Get the frontmatter
	markdownFrontmatter := Frontmatter{}
	frontmatterData := frontmatter.Get(ctx)
	if err := frontmatterData.Decode(&markdownFrontmatter); err != nil {
		return PostData{}, err
	}
	// Check if title and canonical URL are set
	if markdownFrontmatter.Title == "" {
		return PostData{}, fmt.Errorf("title is required in frontmatter")
	}
	if markdownFrontmatter.CanonicalUrl == "" {
		log.Warn("canonical_url is not set in frontmatter")
	}
	// Convert the HTML to Markdown
	html := buf.String()
	// The frontmatter is stripped before converting to HTML
	// Just convert the HTML to Markdown so the Markdown doesn't have the frontmatter (otherwise it would be duplicated)
	markdown, err = md.NewConverter("", true, nil).ConvertString(html)
	if err != nil {
		return PostData{}, err
	}
	return PostData{
		Title:        markdownFrontmatter.Title,
		Html:         html,
		Markdown:     markdown,
		CanonicalUrl: markdownFrontmatter.CanonicalUrl,
	}, nil

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

		return &Blogger{
			Name:      name,
			BlogUrl:   blogUrl,
			Overwrite: overwrite,
		}, nil
	case "markdown":
		contentDir, ok := destMap["content_dir"].(string)
		if !ok || contentDir == "" {
			return nil, fmt.Errorf("content_dir is required for markdown")
		}
		gitDir, _ := destMap["git_dir"].(string) // If not set, defaults to ""
		// Assert that frontmatter_mapping is a map of strings to strings
		frontmatterMapping, err := FrontmatterMappingFromInterface(destMap["frontmatter_mapping"])
		if err != nil {
			log.Warn("Failed to get frontmatter mapping. Using default", "error", err, "default", FrontMatterMappings)
			frontmatterMapping, err = FrontmatterMappingFromInterface(FrontMatterMappings)
			if err != nil {
				return nil, err
			}
		}
		overwrite, _ := destMap["overwrite"].(bool) // If not set or not a bool, defaults to false

		return &Markdown{
			Name:               name,
			ContentDir:         contentDir,
			GitDir:             gitDir,
			FrontmatterMapping: *frontmatterMapping,
			Overwrite:          overwrite,
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

		return &Blogger{
			Name:    name,
			BlogUrl: blogUrl,
		}, nil
	case "markdown":
		// If the content_dir is not set, set it to null as its not required
		contentDir, _ := sourceMap["content_dir"].(string)
		return &Markdown{
			Name:       name,
			ContentDir: contentDir,
		}, nil
	default:
		return nil, fmt.Errorf("unknown source type: %s", sourceMap["type"])
	}
}
