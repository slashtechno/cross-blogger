package platforms

import "errors"


// This is more just a set of defaults compatible with Hugo's frontmatter
var FrontMatterMappings = map[string]string{"title": "title", "date": "date", "date_updated": "lastmod", "canonical_url": "canonicalURL"}

type Frontmatter struct {
	// TOOD: make frontmatter mappings configurable, somehow
	Title        string `yaml:"title"`
	Date         string `yaml:"date"`
	DateUpdated  string `yaml:"lastmod"`
	CanonicalUrl string `yaml:"canonicalURL"`
}

type FrontmatterMapping struct {
	Title        string `toml:"title"`
	Date         string `toml:"date"`
	LastUpdated  string `toml:"lastUpdated"`
	CanonicalURL string `toml:"canonicalURL"`
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
	for k := range frontmatterMapping {
		if _, ok := frontmatterMapping[k].(string); !ok {
			return nil, errors.New("frontmatter mapping key is not a string")
		}
	}
	return &FrontmatterMapping{
		Title:        frontmatterMapping["title"].(string),
		Date:         frontmatterMapping["date"].(string),
		LastUpdated:  frontmatterMapping["date_updated"].(string),
		CanonicalURL: frontmatterMapping["canonical_url"].(string),
	}, nil
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
	if canonicalURL, ok := m[frontmatterMapping.CanonicalURL]; ok {
		frontmatterObjet.CanonicalUrl = canonicalURL.(string)
	}
	return frontmatterObjet
}
