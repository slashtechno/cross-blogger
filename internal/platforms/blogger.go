package platforms

import (
	"fmt"
	"regexp"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/charmbracelet/log"
	"github.com/go-resty/resty/v2"
	"github.com/slashtechno/cross-blogger/pkg/oauth"
	"github.com/slashtechno/cross-blogger/pkg/utils"
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
		SetQueryParam("fetchBodies", "true").
		SetQueryParam("status", "LIVE").
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

		// Check if there are any posts known to the program but that are no longer live
		// If a post is no longer live, remove it from the list of known posts
		for _, knownPost := range b.knownPosts {
			found := false
			for _, p := range posts {
				post := p.(map[string]interface{})
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