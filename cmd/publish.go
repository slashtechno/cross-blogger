package cmd

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/redis/go-redis/v9"
	"github.com/slashtechno/cross-blogger/internal"
	"github.com/slashtechno/cross-blogger/internal/platforms"
	"github.com/spf13/cobra"
)

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish to a destination",
	Long: `Publish to a destination from a source. 
	Specify the source with the first positional argument. 
	The second positional argument is the specifier, such as a Blogger post URL or a file path.
	All arguments after the first are treated as destinations.
	Destinations should be the name of the destinations specified in the config file`,
	// Arg 1: Source
	// Arg 2: Specifier
	// Arg 3+: Destinations
	Args: cobra.MinimumNArgs(3),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		var redisOptions *redis.Options
		if redisOptions, err = internal.InitializeRedisOptions(internal.CredentialViper.GetStringMap("db")); err != nil {
			return err
		}
		// https://github.com/spf13/viper?tab=readme-ov-file#accessing-nested-keys
		if !internal.CredentialViper.GetBool("db.enable") {
			log.Debug("DB is disabled")
			return nil
		}
		if err := internal.InitializeDb("redis", redisOptions); err != nil {
			return err
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		destinations := internal.ConfigViper.Get("destinations")
		sources := internal.ConfigViper.Get("sources")
		if destinations == nil {
			log.Fatal("Failed to get destinations from config")
		}

		// Load the sources and destinations
		// For now, since we're only pulling from one source, the first argument is the source
		sourceSlice, destinationSlice, err := platforms.Load(sources, destinations, []string{args[0]}, args[2:])
		if err != nil {
			log.Fatal(err)
		}
		// Whilst this shouldn't happen since args[0] is passed to Load, iterate over the sources to ensure that the source matches the first argument
		var found bool = false
		var source platforms.Source
		for _, s := range sourceSlice {
			if s.GetName() == args[0] {
				source = s
				found = true
				break
			}
		}
		if !found {
			log.Fatal("Source not found", "source", args[0])
		}
		// Pull the data from the source
		var options platforms.PushPullOptions
		switch source.GetType() {
		case "blogger":
			_, accessToken, blogId, _, err := prepareBlogger(source, nil, internal.CredentialViper.GetString("google_client_id"), internal.CredentialViper.GetString("google_client_secret"), internal.CredentialViper.GetString("google_refresh_token"))
			if err != nil {
				log.Fatal(err)
			}

			options = platforms.PushPullOptions{
				AccessToken: accessToken,
				BlogId:      blogId,
				PostUrl:     args[1],
				LlmProvider: internal.CredentialViper.GetString("llm_provider"),
				LlmBaseUrl:  internal.CredentialViper.GetString("llm_base_url"),
				LlmApiKey:   internal.CredentialViper.GetString("llm_api_key"),
				LlmModel:    internal.CredentialViper.GetString("llm_model"),
			}
		case "markdown":
			options = platforms.PushPullOptions{
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
		err = pushToDestinations(postData, destinationSlice, dryRun)
		if err != nil {
			log.Fatal(err)
		}
	},
}

var dryRun bool

func init() {
	RootCmd.AddCommand(publishCmd)
	// publishCmd.Flags().StringP("title", "t", "", "Specify custom title instead of using the default")
	publishCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Dry run - don't actually push the data")
	publishCmd.PersistentFlags().String("google-client-id", "", "Google OAuth client ID")
	publishCmd.PersistentFlags().String("google-client-secret", "", "Google OAuth client secret")
	publishCmd.PersistentFlags().String("google-refresh-token", "", "Google OAuth refresh token")
	publishCmd.PersistentFlags().String("llm-provider", "", "LLM platform (\"openai\" or \"ollama\")")
	publishCmd.PersistentFlags().String("llm-base-url", "", "Base URL")
	publishCmd.PersistentFlags().String("llm-api-key", "", "OpenAI API key")
	publishCmd.PersistentFlags().String("llm-model", "", "LLM model to use for OpenAI-compatible platforms")
	// Allow the OAuth stuff to be set via viper
	internal.CredentialViper.BindPFlag("google_client_id", publishCmd.Flags().Lookup("google-client-id"))
	internal.CredentialViper.BindPFlag("google_client_secret", publishCmd.Flags().Lookup("google-client-secret"))
	internal.CredentialViper.BindPFlag("google_refresh_token", publishCmd.Flags().Lookup("google-refresh-token"))
	// Bind Viper to LLM flags
	internal.CredentialViper.BindPFlag("llm_provider", publishCmd.Flags().Lookup("llm-provider"))
	internal.CredentialViper.BindPFlag("llm_base_url", publishCmd.Flags().Lookup("llm-base-url"))
	internal.CredentialViper.BindPFlag("llm_api_key", publishCmd.Flags().Lookup("llm-api-key"))
	internal.CredentialViper.BindPFlag("llm_model", publishCmd.Flags().Lookup("llm-model"))
}

// Return the Blogger object and a string with the access token, the blog ID, a refresh token, and an error if one occurred
// Take the client ID, client secret, and refresh token as a parameter
func prepareBlogger(source platforms.Source, destination platforms.Destination, clientId string, clientSecret string, refreshToken string) (platforms.Blogger, string, string, string, error) {
	// Check if the user passed a source or destination. Exactly one should be passed.
	var platform interface{}
	if source == nil && destination == nil {
		return platforms.Blogger{}, "", "", "", fmt.Errorf("no source or destination passed")
	} else if source != nil && destination != nil {
		return platforms.Blogger{}, "", "", "", fmt.Errorf("both source and destination passed")
	} else if source != nil {
		platform = source
	} else if destination != nil {
		platform = destination
	} else {
		return platforms.Blogger{}, "", "", "", fmt.Errorf("failed to determine if source or destination was passed")
	}

	// Convert source to Blogger
	var blogger *platforms.Blogger
	if tmpBlogger, ok := platform.(*platforms.Blogger); ok {
		log.Debug("Asserted that source is Blogger successfully")
		blogger = tmpBlogger
	} else {
		return platforms.Blogger{}, "", "", "", fmt.Errorf("failed to assert that source is Blogger - potentially due to being called on a non-Blogger source")
	}
	// If the refresh token exists in Viper, pass that to Blogger.Authorize. Otherwise, pass an empty string
	var accessToken string
	var err error
	if refreshToken == "" {
		log.Warn("No refresh token found in Viper")
		accessToken, refreshToken, err = blogger.Authorize(clientId, clientSecret, "")
		if err != nil {
			return platforms.Blogger{}, "", "", "", err
		}
		// Write the refresh token to the config file
		log.Info("Writing refresh token to Viper")
		internal.CredentialViper.Set("google-refresh-token", refreshToken)
		// The flag viper should be .env only
		err = internal.CredentialViper.WriteConfig()
		if err != nil {
			return platforms.Blogger{}, "", "", "", err
		}
	} else {
		log.Info("Found refresh token in Viper")
		accessToken, _, err = blogger.Authorize(clientId, clientSecret, refreshToken)
	}
	if err != nil {
		return platforms.Blogger{}, "", "", "", err
	}

	blogId, err := blogger.GetBlogId(accessToken)
	if err != nil {
		return platforms.Blogger{}, "", "", "", err
	}
	return *blogger, accessToken, blogId, refreshToken, nil
}

// For each destination, push the data
func pushToDestinations(postData platforms.PostData, destinationSlice []platforms.Destination, dryRun bool) error {
	for _, destination := range destinationSlice {
		var found bool = true
		var options platforms.PushPullOptions
		switch destination.GetType() {
		case "markdown":
			options = platforms.PushPullOptions{}

		case "blogger":
			_, accessToken, blogId, _, err := prepareBlogger(nil, destination, internal.CredentialViper.GetString("google_client_id"), internal.CredentialViper.GetString("google_client_secret"), internal.CredentialViper.GetString("google_refresh_rtoken"))
			if err != nil {
				return err
			}
			options = platforms.PushPullOptions{
				AccessToken: accessToken,
				BlogId:      blogId,
			}
		default:
			found = false
		}
		if found {
			// Check if this is a dry run
			if dryRun {
				log.Info("Skipping push due to dry run")
				continue
			}
			err := destination.Push(postData, options)
			if err != nil {
				return err
			}
		} else {
			log.Error("Destination type not implemented", "type", destination.GetType())
		}
	}
	// This should never be reached unless there are no destinations
	return nil
}
