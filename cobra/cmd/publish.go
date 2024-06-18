package cmd

import (
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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
		var destinationSlice []Platform
		// _ ignores the index. `dest` is the map
		for _, dest := range destinations.([]interface{}) {
			destMap, ok := dest.(map[string]interface{})
			if !ok {
				log.Fatal("Failed to convert destination to map")
			}
			destination, err := CreateDestination(destMap)
			if err != nil {
				log.Fatal(err)
			}
			destinationSlice = append(destinationSlice, destination)
		}
		log.Info("Destination slice", "destinations", destinationSlice)
		// Check if OAuth works (remove this later!)
		if blogger, ok := destinationSlice[0].(Blogger); ok {
			blogger.authorize(viper.GetString("google-client-id"), viper.GetString("google-client-secret"))
		}
	},
}

func init() {
	RootCmd.AddCommand(publishCmd)

	publishCmd.Flags().StringP("title", "t", "", "Specify custom title instead of using the default")
	publishCmd.Flags().BoolP("dry-run", "r", false, "Don't actually publish")
	publishCmd.Flags().String("google-client-id", "", "Google OAuth client ID")
	publishCmd.Flags().String("google-client-secret", "", "Google OAuth client secret")
	publishCmd.Flags().String("google-refresh-token", "", "Google OAuth refresh token")
	// Allow the OAuth stuff to be set via viper
	viper.BindPFlag("google-client-id", publishCmd.Flags().Lookup("google-client-id"))
	viper.BindPFlag("google-client-secret", publishCmd.Flags().Lookup("google-client-secret"))
	// viper.BindPFlag("google-refresh-token", publishCmd.Flags().Lookup("google-refresh-token"))
	// Keep in mind that these should be prefixed with CROSS_BLOGGER
	viper.BindEnv("google-client-id", "CROSS_BLOGGER_GOOGLE_CLIENT_ID")
	viper.BindEnv("google-client-secret", "CROSS_BLOGGER_GOOGLE_CLIENT_SECRET")
	// viper.BindEnv("google-refresh-token", "GOOGLE_REFRESH_TOKEN")
}
