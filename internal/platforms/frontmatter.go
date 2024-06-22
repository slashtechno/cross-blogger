package platforms

import "errors"

var FrontmatterOptions = []string{"title", "date", "lastmod", "canonicalURL"}

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
	return map[string]interface{}{
		frontmatterMapping.Title:        f.Title,
		frontmatterMapping.Date:         f.Date,
		frontmatterMapping.LastUpdated:  f.DateUpdated,
		frontmatterMapping.CanonicalURL: f.CanonicalUrl,
	}
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
func FrontmatterFromMap(m map[string]interface{}) *Frontmatter {
	return &Frontmatter{
		Title:        m[FrontMatterMappings["title"]].(string),
		Date:         m[FrontMatterMappings["date"]].(string),
		DateUpdated:  m[FrontMatterMappings["date_updated"]].(string),
		CanonicalUrl: m[FrontMatterMappings["canonical_url"]].(string),
	}
}
