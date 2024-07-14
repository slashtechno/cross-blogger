package platforms

import (
	"fmt"
	"sync"
	"time"

	"github.com/charmbracelet/log"
)

type Destination interface {
	Push(PostData, PushPullOptions) error
	// Push(*redis.Client, PostData, PushPullOptions) error
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
	// Watch(time.Duration, PushPullOptions, chan<- PostData, chan<- error)
	Watch(*sync.WaitGroup, time.Duration, PushPullOptions, chan<- PostData, chan<- error)
	CleanMarkdownPosts(*sync.WaitGroup, time.Duration, *Markdown, PushPullOptions, chan<- error)
}

type PushPullOptions struct {
	AccessToken  string
	BlogId       string
	PostUrl      string
	Filepath     string
	RefreshToken string
	ClientId     string
	ClientSecret string
	LlmProvider  string
	LlmBaseUrl   string
	LlmApiKey    string
	LlmModel     string
}

type PostData struct {
	Title       string
	Html        string
	Markdown    string
	Date        time.Time
	DateUpdated time.Time
	// TODO: Add frontmatter descriptions
	Description string
	Categories  []string
	Tags        []string
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
	Name           string
	BlogUrl        string
	CategoryPrefix string
	// https://developers.google.com/blogger/docs/3.0/reference/posts/delete
	Overwrite               bool
	GenerateLlmDescriptions bool
	knownPosts              []string
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
		// Optionally, enable LLM generated descriptions
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
		// Check if category_prefix is set, if not, set it to null and move on
		// If the value is not a string, it will be set to "category::"
		categoryPrefix, ok := sourceMap["category_prefix"].(string)
		if !ok || categoryPrefix == "" {
			log.Warn("category_prefix is not a string or is empty. Using default", "default", "category::")
			categoryPrefix = "category::"
		}
		generateLlmDescriptions, _ := sourceMap["generate_llm_descriptions"].(bool)
		return &Blogger{
			Name:                    name,
			BlogUrl:                 blogUrl,
			GenerateLlmDescriptions: generateLlmDescriptions,
			CategoryPrefix:          categoryPrefix,
		}, nil
	case "markdown":
		// If the content_dir is not set, set it to null as its not required
		contentDir, _ := sourceMap["content_dir"].(string)
		// Assert that frontmatter_mapping is a map of strings to strings
		frontmatterMapping, err := FrontmatterMappingFromInterface(sourceMap["frontmatter_mapping"])
		if err != nil {
			log.Warn("Failed to get frontmatter mapping. Using default", "error", err, "default", FrontMatterMappings)
			frontmatterMapping, err = FrontmatterMappingFromInterface(FrontMatterMappings)
			if err != nil {
				return nil, err
			}
		}
		return &Markdown{
			Name:               name,
			ContentDir:         contentDir,
			FrontmatterMapping: *frontmatterMapping,
		}, nil
	default:
		return nil, fmt.Errorf("unknown source type: %s", sourceMap["type"])
	}
}
