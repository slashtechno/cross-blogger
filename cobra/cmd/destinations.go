package cmd

import "fmt"

type Destination struct {
	Name string
	Type string
}

type Blogger struct {
	Destination
	BlogUrl string
	BlogId  string
}
type Markdown struct {
	Destination
	ContentDir string
}

func CreateDestination(destMap map[string]interface{}) (interface{}, error) {
	switch destMap["type"] {
	case "blogger":
		return Blogger{
			Destination: Destination{
				Name: destMap["name"].(string),
				Type: destMap["type"].(string),
			},
			BlogUrl: destMap["blog_url"].(string),
			BlogId:  destMap["blog_id"].(string),
		}, nil
	case "markdown":
		return Markdown{
			Destination: Destination{
				Name: destMap["name"].(string),
				Type: destMap["type"].(string),
			},
			ContentDir: destMap["content_dir"].(string),
		}, nil
	default:
		return nil, fmt.Errorf("unknown destination type: %s", destMap["type"])
	}
}
