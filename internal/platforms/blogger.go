package platforms

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/charmbracelet/log"
	"github.com/go-resty/resty/v2"
	"github.com/gosimple/slug"
	"github.com/slashtechno/cross-blogger/pkg/oauth"
	"github.com/slashtechno/cross-blogger/pkg/utils"
	"github.com/spf13/afero"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
)

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
	// If GenerateLlmDescriptions is true, generate descriptions for the post
	var postDescription string
	var llmImplementation llms.Model
	prompt := fmt.Sprintf("The following is a blog post titled \"%s\" with the content:\n\n%s\n\nThe description of the post is:", title, markdown)
	if b.GenerateLlmDescriptions {
		switch strings.ToLower(options.LlmProvider) {
		case "openai":
			// If the API key is not set warn
			// If the base URL or model is not set, return an error
			if options.LlmApiKey == "" {
				log.Warn("No key for an OpenAI-compatible API was provided")
			}
			if options.LlmBaseUrl == "" || options.LlmModel == "" {
				return PostData{}, fmt.Errorf("OpenAI base URL and model are required")
			}
			llmImplementation, err = openai.New(openai.WithBaseURL(options.LlmBaseUrl), openai.WithModel(options.LlmModel), openai.WithToken(options.LlmApiKey))
			if err != nil {
				return PostData{}, err
			}
		case "ollama":
			// If base URL is set, use that. Otherwise, use http://localhost:11434
			baseUrl := options.LlmBaseUrl
			if baseUrl == "" {
				baseUrl = "http://localhost:11434"
			}
			llmImplementation, err = ollama.New(ollama.WithServerURL(baseUrl), ollama.WithModel(options.LlmModel))
			if err != nil {
				return PostData{}, err
			}
		default:
			return PostData{}, fmt.Errorf("invalid LLM platform")
		}
		ctx := context.Background()
		postDescription, err = llms.GenerateFromSinglePrompt(ctx, llmImplementation, prompt)
		if err != nil {
			return PostData{}, err
		}
		postDescription = strings.TrimSpace(postDescription)
		log.Debug("Generated description", "description", postDescription)
	}

	return PostData{
		Title:        title,
		Html:         html,
		Markdown:     markdown,
		Date:         date,
		DateUpdated:  dateUpdated,
		Description:  postDescription,
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
func (b *Blogger) Watch(wg *sync.WaitGroup, interval time.Duration, options PushPullOptions, postChan chan<- PostData, errChan chan<- error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	defer wg.Done()
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

func (b *Blogger) fetchPosts(blogId, accessToken string) ([]map[string]interface{}, error) {
	// Get the list of posts
	client := resty.New()
	resp, err := client.R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", accessToken)).
		SetResult(&map[string]interface{}{}).
		SetQueryParam("fetchBodies", "true").
		SetQueryParam("status", "LIVE").
		Get("https://www.googleapis.com/blogger/v3/blogs/" + blogId + "/posts")
	if err != nil {
		return nil, err
	}
	// Assert that posts is a pointer to a map. However, dereference it to get the map
	posts := (*resp.Result().(*map[string]interface{}))["items"].([]interface{})
	// Make another slice but assert each element to be a map of string to interface
	postsValidated := []map[string]interface{}{}
	for _, p := range posts {
		if post, ok := p.(map[string]interface{}); ok {
			postsValidated = append(postsValidated, post)
		}
		postsValidated = append(postsValidated, p.(map[string]interface{}))
	}
	return postsValidated, nil
}

// Get posts that haven't been seen before and return them
func (b *Blogger) fetchNewPosts(options PushPullOptions) ([]PostData, error) {
	// Get the list of posts
	posts, err := b.fetchPosts(options.BlogId, options.AccessToken)
	if err != nil {
		return nil, err
	}
	if len(b.knownPosts) == 0 {
		// Add all the posts to the list of known posts
		for _, post := range posts {
			b.knownPosts = append(b.knownPosts, post["id"].(string))
		}
		// Return an empty slice because there are no new posts
		log.Debug("Adding all posts as none are currently known", "knownPosts", b.knownPosts)
		return []PostData{}, nil
	} else {
		// Get the new posts
		newPosts := []PostData{}
		// TODO: Make this concurrent

		// Check if there are any posts known to the program but that are no longer live
		// If a post is no longer live, remove it from the list of known posts
		for _, knownPost := range b.knownPosts {
			found := false
			for _, post := range posts {
				if post["id"].(string) == knownPost {
					found = true
					break
				}
			}
			if !found {
				// Remove the post from the list of known posts
				log.Info("Removing post from known posts", "post", knownPost)
				b.knownPosts = utils.RemoveString(b.knownPosts, knownPost)
			}
		}

		// Get the postData for new posts
		for _, post := range posts {
			// Check if the post is new
			if !utils.ContainsString(b.knownPosts, post["id"].(string)) {
				// Add the post to the list of known posts
				b.knownPosts = append(b.knownPosts, post["id"].(string))
				// Add the post to the list of new posts (run Pull)
				optionsToPass := options
				optionsToPass.PostUrl = post["url"].(string)
				postData, err := b.Pull(optionsToPass)
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

// Go through the contentDir of the Markdown struct and delete any posts that are not in the list of known posts.
// Only delete them if Frontmatter.Managed is true, however.
func (b Blogger) CleanMarkdownPosts(wg *sync.WaitGroup, interval time.Duration, markdownDest *Markdown, options PushPullOptions, errChan chan<- error) {
	defer wg.Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		var err error
		options.AccessToken, _, err = b.Authorize(options.ClientId, options.ClientSecret, options.RefreshToken)
		if err != nil {
			errChan <- err
			return
		}
		knownPosts, err := b.fetchPosts(options.BlogId, options.AccessToken)
		if err != nil {
			errChan <- err
			return
		}
		// Get the title of each post and convert it to a slug.
		// Add it to a slice with ".md" appended to it
		knownFiles := []string{}
		for _, post := range knownPosts {
			// Get the post by its ID
			resp, err := resty.New().R().
				SetHeader("Authorization", fmt.Sprintf("Bearer %s", options.AccessToken)).
				SetResult(&map[string]interface{}{}).
				Get("https://www.googleapis.com/blogger/v3/blogs/" + options.BlogId + "/posts/" + post["id"].(string))
			if err != nil {
				errChan <- err
				return
			}
			post := (*resp.Result().(*map[string]interface{}))
			title := post["title"].(string)
			slug := slug.Make(title)
			fileName := slug + ".md"
			knownFiles = append(knownFiles, fileName)
		}
		// List all files in markdownDest.ContentDir
		fs := afero.NewOsFs()
		contentDir := filepath.Clean(markdownDest.ContentDir)
		absContentDir, err := filepath.Abs(contentDir)
		if err != nil {
			errChan <- err
			return
		}
		files, err := afero.ReadDir(fs, absContentDir)
		if err != nil {
			errChan <- err
			return
		}
		// make a list of files that are not in knownFiles
		unkownFiles := []string{}
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			if !utils.ContainsString(knownFiles, file.Name()) {
				// With Hugo at least, `_index.md`
				unkownFiles = append(unkownFiles, file.Name())
			}
		}
		// Pull the frontmatter for each file
		for _, file := range unkownFiles {
			// Get absolute path
			absPath := filepath.Join(absContentDir, file)
			// Read file
			fileBytes, err := afero.ReadFile(fs, absPath)
			if err != nil {
				errChan <- err
				return
			}
			markdownString := string(fileBytes)
			// Get the frontmatter for the file
			_, _, postFrontmatter, err := markdownDest.ParseMarkdown(markdownString)
			if err != nil {
				errChan <- err
				return
			}
			log.Debug("Got frontmatter", "frontmatter", postFrontmatter)
			if postFrontmatter.Managed {
				// Delete the file
				err := fs.Remove(absPath)
				if err != nil {
					errChan <- err
					return
				}
				// Comit and push the changes
				if markdownDest.GitDir != "" {
					slug := strings.TrimSuffix(file, filepath.Ext(file))
					commitHash, err := markdownDest.Commit(slug, true)
					log.Info("Committed and pushed changes", "hash", commitHash)
					if err != nil {
						errChan <- err
						return
					}
				}
			}
		}
	}

}
func (b Blogger) GetName() string { return b.Name }
func (b Blogger) GetType() string { return "blogger" }
