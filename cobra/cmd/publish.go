package cmd

import (
	"strings"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish to a destination",
	Long:  `Publish to a destination. Available destinations: blogger, markdown, html. Make sure to specify with <platform>,<key1>=<value1>,<key2>=<value2>,...`,
	Run: func(cmd *cobra.Command, args []string) {
		destinations, _ := cmd.Flags().GetStringSlice("destination")
		// Make a map of platforms and their attributes
		platformMap := make(map[string]map[string]string)

		// Iterate over the destinations
		for _, destination := range destinations {
			// Split the destination into platform and attributes
			// This splits the destination into two parts: the platform and the attributes
			// Anything after the first semicolon is considered attributes
			destinationParts := strings.SplitN(destination, ";", 2)
			platform := destinationParts[0]
			// Create a new map for the platform if it doesn't exist
			if _, ok := platformMap[platform]; !ok {
				platformMap[platform] = make(map[string]string)
			}
			// Check if there are any attributes
			if len(destinationParts) > 1 {
				// Split the attributes part on semicolon
				attributes := strings.Split(destinationParts[1], ";")
				// Populate the map with the attributes
				for _, attribute := range attributes {
					keyValue := strings.Split(attribute, "=")
					if len(keyValue) == 2 {
						platformMap[platform][keyValue[0]] = keyValue[1]
					}
				}
			}
		}

		// Debug log the destination map
		log.Debugf("Platform map: %v", platformMap)
		// Iterate over the platforms
		for platform, attributes := range platformMap {
			// Debug log the platform and attributes
			log.Debugf("Platform: %s, Attributes: %v", platform, attributes)
		}

	},
}

func init() {
	// example command: go run . publish --destination "blogger;blogAddress=example.com;postAddress=example-post" --destination "markdown;filepath=example.md" --title "Example Title" --dry-run
	rootCmd.AddCommand(publishCmd)

	publishCmd.Flags().StringSliceP("destination", "d", nil, "Destination(s) to publish to\nAvailable destinations: blogger, markdown, html\nMake sure to specify with <platform>,<key1>=<value1>,<key2>=<value2>,...")
	publishCmd.Flags().StringP("title", "t", "", "Specify custom title instead of using the default")
	publishCmd.Flags().BoolP("dry-run", "r", false, "Don't actually publish")
	publishCmd.Flags().String("client-id", "", "Google OAuth client ID")
	publishCmd.Flags().String("client-secret", "", "Google OAuth client secret")
	publishCmd.Flags().String("refresh-token", "", "Google OAuth refresh token")
}
