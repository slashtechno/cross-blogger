package platforms

import (
	"errors"
	"fmt"
	"time"
)

// This is more just a set of defaults compatible with Hugo's frontmatter
var FrontMatterMappings = map[string]string{
	"title":         "title",
	"date":          "date",
	"date_updated":  "lastmod",
	"description":   "description",
	"categories":    "categories",
	"tags":          "tags",
	"canonical_url": "canonicalURL",
	"managed":       "managedByCrossBlogger"}

type Frontmatter struct {
	// TOOD: make frontmatter mappings configurable, somehow
	Title        string
	Date         string
	DateUpdated  string
	Description  string
	Categories   []string
	Tags         []string
	CanonicalUrl string
	Managed      bool
}

type FrontmatterMapping struct {
	Title        string
	Date         string
	LastUpdated  string
	Description  string
	Categories   string
	Tags         string
	CanonicalURL string
	Managed      string
}

// Take a Frontmatter struct and taking FrontmatterMapping into account, return a map ready to be marshaled into YAML
func (f *Frontmatter) ToMap(frontmatterMapping FrontmatterMapping) map[string]interface{} {
	// frontmatterAsMap := map[string]interface{}{
	// 	frontmatterMapping.Title:        f.Title,
	// 	frontmatterMapping.Date:         f.Date,
	// 	frontmatterMapping.LastUpdated:  f.DateUpdated,
	// 	frontmatterMapping.CanonicalURL: f.CanonicalUrl,
	// }
	// As long as BOTH the key and value are not empty, add them to the map
	frontmatterAsMap := make(map[string]interface{})
	if f.Title != "" && frontmatterMapping.Title != "" {
		frontmatterAsMap[frontmatterMapping.Title] = f.Title
	}
	if f.Date != "" && frontmatterMapping.Date != "" {
		frontmatterAsMap[frontmatterMapping.Date] = f.Date
	}
	if f.DateUpdated != "" && frontmatterMapping.LastUpdated != "" {
		frontmatterAsMap[frontmatterMapping.LastUpdated] = f.DateUpdated
	}
	if f.Description != "" && frontmatterMapping.Description != "" {
		frontmatterAsMap[frontmatterMapping.Description] = f.Description
	}
	if len(f.Categories) > 0 && frontmatterMapping.Categories != "" {
		frontmatterAsMap[frontmatterMapping.Categories] = f.Categories
	}
	if len(f.Tags) > 0 && frontmatterMapping.Tags != "" {
		frontmatterAsMap[frontmatterMapping.Tags] = f.Tags
	}
	if f.CanonicalUrl != "" && frontmatterMapping.CanonicalURL != "" {
		frontmatterAsMap[frontmatterMapping.CanonicalURL] = f.CanonicalUrl
	}
	if frontmatterMapping.Managed != "" {
		frontmatterAsMap[frontmatterMapping.Managed] = f.Managed
	}
	return frontmatterAsMap
}

// Convert a frontmatter_mapping (interface{} due to how Viper works) to a FrontmatterMapping struct
func FrontmatterMappingFromInterface(m interface{}) (*FrontmatterMapping, error) {
	frontmatterMapping, ok := m.(map[string]interface{})
	if !ok {
		return nil, errors.New("failed to convert frontmatter mapping to map")
	}
	// Assert that each key is a string. If any are not, return an error
	for k, v := range frontmatterMapping {
		_, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("frontmatter mapping key '%s' is not a string", k)
		}
	}
	// Add any missing keys from the default FrontMatterMappings
	for k, v := range FrontMatterMappings {
		if _, ok := frontmatterMapping[k]; !ok {
			frontmatterMapping[k] = v
		}
	}

	return &FrontmatterMapping{
		Title:        frontmatterMapping["title"].(string),
		Date:         frontmatterMapping["date"].(string),
		LastUpdated:  frontmatterMapping["date_updated"].(string),
		CanonicalURL: frontmatterMapping["canonical_url"].(string),
		Description:  frontmatterMapping["description"].(string),
		Categories:   frontmatterMapping["categories"].(string),
		Tags:         frontmatterMapping["tags"].(string),
		Managed:      frontmatterMapping["managed"].(string),
	}, nil
}

// Take a map and return a Frontmatter struct, taking FrontmatterMapping into account
func FrontmatterFromMap(m map[string]interface{}, frontmatterMapping FrontmatterMapping) (*Frontmatter, error) {
	frontmatterObject := &Frontmatter{}
	if title, ok := m[frontmatterMapping.Title]; ok {
		frontmatterObject.Title = title.(string)
	}
	if date, ok := m[frontmatterMapping.Date]; ok {
		// Convert the time.time to a string
		if dateObject, ok := date.(time.Time); ok {
			frontmatterObject.Date = dateObject.Format(time.RFC3339)
		} else {
			// Check if it's a string. If it's not a string or time.Time, return an error
			if date, ok := date.(string); ok {
				frontmatterObject.Date = date
			} else {
				return nil, errors.New("date is not a string or time.Time")
			}
		}
	}
	if lastUpdated, ok := m[frontmatterMapping.LastUpdated]; ok {
		// Convert the time.time to a string
		if lastUpdatedObject, ok := lastUpdated.(time.Time); ok {
			frontmatterObject.DateUpdated = lastUpdatedObject.Format(time.RFC3339)
		} else {
			// Check if it's a string. If it's not a string or time.Time, return an error
			if lastUpdated, ok := lastUpdated.(string); ok {
				frontmatterObject.DateUpdated = lastUpdated
			} else {
				return nil, errors.New("date_updated is not a string or time.Time")
			}
		}
	}

	if description, ok := m[frontmatterMapping.Description]; ok {
		frontmatterObject.Description = description.(string)
	}
	if categories, ok := m[frontmatterMapping.Categories]; ok {
		// Convert the interface{} to a []string
		categoriesSlice, ok := categories.([]string)
		if !ok {
			return nil, errors.New("categories is not a []string")
		}
		frontmatterObject.Categories = categoriesSlice
	}
	if tags, ok := m[frontmatterMapping.Tags]; ok {
		// Convert the interface{} to a []string
		tagsSlice, ok := tags.([]string)
		if !ok {
			return nil, errors.New("tags is not a []string")
		}
		frontmatterObject.Tags = tagsSlice
	}
	if canonicalURL, ok := m[frontmatterMapping.CanonicalURL]; ok {
		frontmatterObject.CanonicalUrl = canonicalURL.(string)
	}
	if managed, ok := m[frontmatterMapping.Managed]; ok {
		frontmatterObject.Managed = managed.(bool)
	}
	return frontmatterObject, nil
}
