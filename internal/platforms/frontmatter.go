package platforms

import (
	"errors"
	"fmt"
)

// This is more just a set of defaults compatible with Hugo's frontmatter
var FrontMatterMappings = map[string]string{"title": "title", "date": "date", "date_updated": "lastmod", "description": "description", "canonical_url": "canonicalURL"}

type Frontmatter struct {
	// TOOD: make frontmatter mappings configurable, somehow
	Title        string
	Date         string
	DateUpdated  string
	Description  string
	CanonicalUrl string
}

type FrontmatterMapping struct {
	Title        string
	Date         string
	LastUpdated  string
	Description  string
	CanonicalURL string
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
	if f.CanonicalUrl != "" && frontmatterMapping.CanonicalURL != "" {
		frontmatterAsMap[frontmatterMapping.CanonicalURL] = f.CanonicalUrl
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
		strValue, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("frontmatter mapping key '%s' is not a string", k)
		}
		// Use the GetMappingOrDefault function to ensure a default value is used if necessary
		frontmatterMapping[k] = GetMappingOrDefault(k, strValue)
	}

	return &FrontmatterMapping{
		Title:        GetMappingOrDefault("title", "defaultTitle"),
		Date:         GetMappingOrDefault("date", "defaultDate"),
		LastUpdated:  GetMappingOrDefault("date_updated", "defaultLastUpdated"),
		Description:  GetMappingOrDefault("description", "defaultDescription"),
		CanonicalURL: GetMappingOrDefault("canonical_url", "defaultCanonicalURL"),
	}, nil
}

// GetMappingOrDefault returns the value for a given key from FrontMatterMappings.
// If the key does not exist, it returns a default value.
func GetMappingOrDefault(key, defaultValue string) string {
	if value, exists := FrontMatterMappings[key]; exists {
		return value
	}
	return defaultValue
}

// Take a map and return a Frontmatter struct, taking FrontmatterMapping into account
func FrontmatterFromMap(m map[string]interface{}, frontmatterMapping FrontmatterMapping) *Frontmatter {
	frontmatterObjet := &Frontmatter{}
	if title, ok := m[frontmatterMapping.Title]; ok {
		frontmatterObjet.Title = title.(string)
	}
	if date, ok := m[frontmatterMapping.Date]; ok {
		frontmatterObjet.Date = date.(string)
	}
	if lastUpdated, ok := m[frontmatterMapping.LastUpdated]; ok {
		frontmatterObjet.DateUpdated = lastUpdated.(string)
	}
	if description, ok := m[frontmatterMapping.Description]; ok {
		frontmatterObjet.Description = description.(string)
	}
	if canonicalURL, ok := m[frontmatterMapping.CanonicalURL]; ok {
		frontmatterObjet.CanonicalUrl = canonicalURL.(string)
	}
	return frontmatterObjet
}
