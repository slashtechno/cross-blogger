package cmd

import (
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
		log.Debug(viper.AllSettings())
		destinations, ok := viper.Get("destinations").([]map[string]interface{})
		if !ok {
			log.Fatal("Failed to get destinations from config file")
		}
		// Make a list of the respective Destination structs
		var destinationSlice []interface{}
		// _ ignores the index. `dest` is the map
		for _, dest := range destinations {
			switch dest["type"] {
			case "blogger":
				destinationSlice = append(destinationSlice, Blogger{
					Destination: Destination{
						Name: dest["name"].(string),
						Type: dest["type"].(string),
					},
					BlogUrl: dest["blogUrl"].(string),
					BlogId:  dest["blogId"].(string),
				})
			case "markdown":
				destinationSlice = append(destinationSlice, Markdown{
					Destination: Destination{
						Name: dest["name"].(string),
						Type: dest["type"].(string),
					},
					ContentDir: dest["contentDir"].(string),
				})
			default:
				log.Fatal("Unknown destination type:", dest["type"])
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(publishCmd)

	publishCmd.Flags().StringP("title", "t", "", "Specify custom title instead of using the default")
	publishCmd.Flags().BoolP("dry-run", "r", false, "Don't actually publish")
	publishCmd.Flags().String("client-id", "", "Google OAuth client ID")
	publishCmd.Flags().String("client-secret", "", "Google OAuth client secret")
	publishCmd.Flags().String("refresh-token", "", "Google OAuth refresh token")
}
