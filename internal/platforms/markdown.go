package platforms

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/charmbracelet/log"
	"github.com/go-git/go-git/v5"
	"github.com/goccy/go-yaml"
	"github.com/gosimple/slug"
	"github.com/slashtechno/cross-blogger/pkg/utils"
	"github.com/spf13/afero"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
	goldmarkfrontmatter "go.abhg.dev/goldmark/frontmatter"
)

type Markdown struct {
	Name string
	// ContentDir, for retrieving, should only be used if treating the passed post path as relative results in no file found
	ContentDir string
	GitDir     string
	// Example: []string{"title", "date", "lastmod", "canonicalURL"}
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
		Description:  data.Description,
		Categories:   data.Categories,
		Tags:         data.Tags,
		CanonicalUrl: data.CanonicalUrl,
		Managed:      true,
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
		commitHash, err := m.Commit(slug, true)
		if err != nil {
			return err
		}
		log.Info("Committed and pushed changes", "hash", commitHash)

	}
	return nil

}

// Commit and optionally push the changes to the Git repository.
// If contentDir is not a subdirectory of the gitDir, error.
func (m Markdown) Commit(slug string, push bool) (hash string, err error) {
	contentDir := m.ContentDir
	gitDir := m.GitDir
	filePath := filepath.Join(m.ContentDir, slug+".md")

	// Clean the Git directory path
	dirPath := filepath.Clean(gitDir)
	// Clean the content directory path
	contentDir = filepath.Clean(contentDir)
	// Clean the file path
	filePath = filepath.Clean(filePath)
	// Make sure contentDir and gitDir are absolute paths
	contentDir, err = filepath.Abs(contentDir)
	if err != nil {
		return "", err
	}
	gitDir, err = filepath.Abs(gitDir)
	if err != nil {
		return "", err
	}
	// Check if the contentDir is a subdirectory of the gitDir
	// if !filepath.HasPrefix(contentDir, gitDir) {
	// 	return "", fmt.Errorf("contentDir is not a subdirectory of gitDir")
	// }
	isSubdir, err := utils.IsSubdirectory(gitDir, contentDir)
	if err != nil {
		return "", err
	}
	if !isSubdir {
		return "", fmt.Errorf("contentDir is not a subdirectory of gitDir")
	}
	// Open the repository
	repo, err := git.PlainOpen(dirPath)
	if err != nil {
		return "", err
	}
	repoWorktree, err := repo.Worktree()
	if err != nil {
		return "", err
	}
	// Get the relative path of filePath to dirPath
	relativePath, err := filepath.Rel(dirPath, filePath)
	if err != nil {
		// Handle error, for example, return it
		return "", err
	}

	// Add the file
	_, err = repoWorktree.Add(relativePath)
	if err != nil {
		return "", err
	}
	// Commit the changes
	commitHash, err := repoWorktree.Commit("Update "+slug+".md", &git.CommitOptions{})
	if err != nil {
		return "", err
	}
	if push {
		// Push the changes
		err = repo.Push(&git.PushOptions{})
		if err != nil {
			return "", err
		}
	}
	return commitHash.String(), nil
}

func (m Markdown) ParseMarkdown(markdown string) (markdownWithoutFrontmatter string, html string, frontmatterObject *Frontmatter, err error) {
	err = nil
	// Convert the markdown to HTML with Goldmark
	// Use the Frontmatter extension to get the frontmatter
	mdParser := goldmark.New(goldmark.WithExtensions(&goldmarkfrontmatter.Extender{
		Mode: goldmarkfrontmatter.SetMetadata,
	}))
	var buf bytes.Buffer
	parsedDoc := mdParser.Parser().Parse(text.NewReader([]byte(markdown)))
	err = mdParser.Renderer().Render(&buf, []byte(markdown), parsedDoc)
	if err != nil {
		return "", "", nil, err
	}
	// Get the frontmatter
	frontmatterObject, err = FrontmatterFromMap(parsedDoc.OwnerDocument().Meta(), m.FrontmatterMapping)
	if err != nil {
		return "", "", nil, err
	}
	// Check if title and canonical URL are set
	if frontmatterObject.Title == "" {
		return "", "", nil, err
	}
	if frontmatterObject.CanonicalUrl == "" {
		log.Debug("canonical_url is not set in frontmatter")
	}
	// Convert the HTML to Markdown
	html = buf.String()
	// The frontmatter is stripped before converting to HTML
	// Just convert the HTML to Markdown so the Markdown doesn't have the frontmatter (otherwise it would be duplicated)
	markdownWithoutFrontmatter, err = md.NewConverter("", true, nil).ConvertString(html)
	if err != nil {
		return "", "", nil, err
	}
	return
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
	// Parse the markdown
	markdownWithoutFrontmatter, html, frontmatterObject, err := m.ParseMarkdown(markdown)
	if err != nil {
		return PostData{}, err
	}
	return PostData{
		Title:        frontmatterObject.Title,
		Html:         html,
		Markdown:     markdownWithoutFrontmatter,
		Description:  frontmatterObject.Description,
		CanonicalUrl: frontmatterObject.CanonicalUrl,
	}, nil

}
