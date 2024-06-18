package cmd

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish to a destination",
	Long: `Publish to a destination from a source. 
	Specify the source with the first positional argument. All arguments after the first are treated as destinations.
	Destinations should be the name of the destinations specified in the config file`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Get the list of objects `destinations` from Viper and make a list of Destination structs
		destinations := viper.Get("destinations")
		if destinations == nil {
			log.Fatal("Failed to get destinations from config")
		}

		// Make a list of the respective Destination structs
		var destinationSlice []interface{}
		// _ ignores the index. `dest` is the map
		for _, dest := range destinations.([]interface{}) {
			destMap, ok := dest.(map[string]interface{})
			if !ok {
				log.Fatal("Failed to convert destination to map")
			}
			destination, err := createDestination(destMap)
			if err != nil {
				log.Fatal(err)
			}
			destinationSlice = append(destinationSlice, destination)
		}
		// Use destinationSlice here
	},
}

func createDestination(destMap map[string]interface{}) (interface{}, error) {
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
		return nil, fmt.Errorf("Unsupported destination type")
	}
}

func init() {
	RootCmd.AddCommand(publishCmd)

	publishCmd.Flags().StringP("title", "t", "", "Specify custom title instead of using the default")
	publishCmd.Flags().BoolP("dry-run", "r", false, "Don't actually publish")
	publishCmd.Flags().String("client-id", "", "Google OAuth client ID")
	publishCmd.Flags().String("client-secret", "", "Google OAuth client secret")
	publishCmd.Flags().String("refresh-token", "", "Google OAuth refresh token")
}
