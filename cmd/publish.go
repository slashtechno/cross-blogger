package cmd

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish to a destination",
	Long: `Publish to a destination from a source. 
	Specify the source with the first positional argument. 
	The second positional argument is the specifier, such as a Blogger post URL or a file path.
	All arguments after the first are treated as destinations.
	Destinations should be the name of the destinations specified in the config file`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		destinations := viper.Get("destinations")
		sources := viper.Get("sources")
		if destinations == nil {
			log.Fatal("Failed to get destinations from config")
		}

		// Make a list of the Destination structs if the destination name is in the args
		var destinationSlice []Destination
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
			// Check if the destination name is in the third argument or onwards
			for _, arg := range args[2:] {
				if destination.GetName() == arg {
					log.Info("Adding destination", "destination", destination.GetName())
					destinationSlice = append(destinationSlice, destination)
				} else {
					log.Info("Not adding destination as it isn't specified in args", "destination", destination.GetName())
				}
			}
		}
		log.Debug("Destination slice", "destinations", destinationSlice)
		// Create a list of all the sources. If the source name is the first arg, use that source
		var source Source
		var found bool = false
		for _, src := range sources.([]interface{}) {
			sourceMap, ok := src.(map[string]interface{})
			if !ok {
				log.Fatal("Failed to convert source to map")
			}
			src, err := CreateSource(sourceMap)
			if err != nil {
				log.Fatal(err)
			}
			if src.GetName() == args[0] {
				source = src
				found = true
				log.Info("Found source", "source", source.GetName())
			}
		}
		if !found {
			log.Fatal("Failed to find source in config")
		}

		// Check if OAuth works (remove this later!)
		// if blogger, ok := destinationSlice[0].(Blogger); ok {
		// 	token, err := blogger.authorize(viper.GetString("google-client-id"), viper.GetString("google-client-secret"))
		// 	if err != nil {
		// 		log.Fatal(err)
		// 	}
		// 	log.Info("", "token", token)
		// 	blogId, err := blogger.getBlogId(token)
		// 	if err != nil {
		// 		log.Fatal(err)
		// 	}
		// 	log.Info("", "blog id", blogId)
		// }

		// Pull the data from the source
		var options PushPullOptions
		switch source.GetType() {
		case "blogger":
			_, accessToken, blogId, err := prepareBlogger(source, nil)
			if err != nil {
				log.Fatal(err)
			}

			options = PushPullOptions{
				AccessToken: accessToken,
				BlogId:      blogId,
				PostUrl:     args[1],
			}
		case "markdown":
			options = PushPullOptions{
				Filepath: args[1],
			}
		}
		// Pull the data from the source
		postData, err := source.Pull(options)
		if err != nil {
			log.Fatal(err)
		}
		log.Info("Successfully pulled data", "title", postData.Title, "url", postData.CanonicalUrl, "markdown", postData.Markdown)

		// For each destination, push the data
		for _, destination := range destinationSlice {
			var found bool = true
			switch destination.GetType() {
			case "markdown":
				options = PushPullOptions{}

			case "blogger":
				_, accessToken, blogId, err := prepareBlogger(nil, destination)
				if err != nil {
					log.Fatal(err)
				}
				options = PushPullOptions{
					AccessToken: accessToken,
					BlogId:      blogId,
				}
			default:
				found = false
			}
			if found {
				// Check if this is a dry run
				if viper.GetBool("dry-run") {
					log.Info("Dry run - not pushing data")
					continue
				}
				err := destination.Push(postData, options)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				log.Error("Destination type not found", "type", destination.GetType())
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(publishCmd)
	// publishCmd.Flags().StringP("title", "t", "", "Specify custom title instead of using the default")
	publishCmd.Flags().BoolP("dry-run", "r", false, "Don't actually publish")
	publishCmd.Flags().String("google-client-id", "", "Google OAuth client ID")
	publishCmd.Flags().String("google-client-secret", "", "Google OAuth client secret")
	publishCmd.Flags().String("google-refresh-token", "", "Google OAuth refresh token")
	// Keep in mind that if the refresh token is not set in the config file, the program will request one
	// It will then write the refresh token to the config file, along with any flags or env vars that have been set.
	// You could always go back and remove those lines and continue using environment variables or flags as it won't write to the config file as long as the refresh token is set
	// Allow the OAuth stuff to be set via viper
	viper.BindPFlag("google-client-id", publishCmd.Flags().Lookup("google-client-id"))
	viper.BindPFlag("google-client-secret", publishCmd.Flags().Lookup("google-client-secret"))
	viper.BindPFlag("google-refresh-token", publishCmd.Flags().Lookup("google-refresh-token"))
	// Keep in mind that these should be prefixed with CROSS_BLOGGER
	viper.BindEnv("google-client-id", "CROSS_BLOGGER_GOOGLE_CLIENT_ID")
	viper.BindEnv("google-client-secret", "CROSS_BLOGGER_GOOGLE_CLIENT_SECRET")
	viper.BindEnv("google-refresh-token", "GOOGLE_REFRESH_TOKEN")
}

// Return the Blogger object and a string with the access token, the blog ID, and an error (if one occurred)
func prepareBlogger(source Source, destination Destination) (Blogger, string, string, error) {
	// Check if the user passed a source or destination. Exactly one should be passed.
	var platform interface{}
	if source == nil && destination == nil {
		return Blogger{}, "", "", fmt.Errorf("no source or destination passed")
	} else if source != nil && destination != nil {
		return Blogger{}, "", "", fmt.Errorf("both source and destination passed")
	} else if source != nil {
		platform = source
	} else if destination != nil {
		platform = destination
	} else {
		return Blogger{}, "", "", fmt.Errorf("failed to determine if source or destination was passed")
	}

	// Convert source to Blogger
	var blogger Blogger
	if tmpBlogger, ok := platform.(Blogger); ok {
		log.Debug("Asserted that source is Blogger successfully")
		blogger = tmpBlogger
	} else {
		return Blogger{}, "", "", fmt.Errorf("failed to assert that source is Blogger - potentially due to being called on a non-Blogger source")
	}
	// If the refresh token exists in Viper, pass that to Blogger.Authorize. Otherwise, pass an empty string
	refreshToken := viper.GetString("google-refresh-token")
	var accessToken string
	var err error
	if refreshToken == "" {
		log.Warn("No refresh token found in Viper")
		accessToken, refreshToken, err = blogger.authorize(viper.GetString("google-client-id"), viper.GetString("google-client-secret"), "")
		if err != nil {
			return Blogger{}, "", "", err
		}
		// Write the refresh token to the config file
		log.Info("Writing refresh token to Viper")
		viper.Set("google-refresh-token", refreshToken)
		err = viper.WriteConfig()
		if err != nil {
			return Blogger{}, "", "", err
		}
	} else {
		log.Info("Found refresh token in Viper")
		accessToken, _, err = blogger.authorize(viper.GetString("google-client-id"), viper.GetString("google-client-secret"), refreshToken)
	}
	if err != nil {
		return Blogger{}, "", "", err
	}

	blogId, err := blogger.getBlogId(accessToken)
	if err != nil {
		return Blogger{}, "", "", err
	}
	return blogger, accessToken, blogId, nil
}
