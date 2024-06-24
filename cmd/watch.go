package cmd

import (
	"github.com/charmbracelet/log"
	"github.com/slashtechno/cross-blogger/internal"
	"github.com/slashtechno/cross-blogger/internal/platforms"
	"github.com/spf13/cobra"
)

// watchCmd represents the watch command
var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Act as a headless CMS of sorts by watching a source for new content and publishing it to configured destinations.",
	Long: `Act as a headless CMS of sorts by watching a source for new content and publishing it to configured destinations.
	Specify the source with the first positional argument.
	The second positional argument and on are treated as destination names.
	Ensure that these are configured in the config file.
	`,
	// Arg 1: Source
	// Arg 2+: Destinations
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		// Load the sources and destinations
		// sourceSlice, _, err := platforms.Load(viper.Get("sources"), viper.Get("destinations"), []string{args[0]}, args[1:])
		sourceSlice, destinationSlice, err := platforms.Load(internal.ConfigViper.Get("sources"), internal.ConfigViper.Get("destinations"), []string{args[0]}, args[1:])
		if err != nil {
			log.Fatal(err)
		}
		// As with publish.go, iterate over the sources to ensure that the source matches the first argument
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
		// Assert that the source is a WatchableSource
		watcher, ok := source.(platforms.WatchableSource)
		if !ok {
			log.Fatal("Source is not a watcher", "source", args[0])
		}

		// Pull the data from the source
		var options platforms.PushPullOptions
		switch source.GetType() {
		case "blogger":
			_, _, blogId, refreshToken, err := prepareBlogger(source, nil, internal.CredentialViper.GetString("google_client_id"), internal.CredentialViper.GetString("google_client_secret"), internal.CredentialViper.GetString("google_refresh_token"))
			if err != nil {
				log.Fatal(err)
			}
			options = platforms.PushPullOptions{
				BlogId: blogId,
				// Credentials for getting the access token with the refresh token
				RefreshToken: refreshToken,
				ClientId:     internal.CredentialViper.GetString("google_client_id"),
				ClientSecret: internal.CredentialViper.GetString("google_client_secret"),
				// If enabled these details are used for generating a description via an LLM
				LlmProvider:  internal.CredentialViper.GetString("llm_provider"),
				LlmBaseUrl:   internal.CredentialViper.GetString("llm_base_url"),
				LlmApiKey:    internal.CredentialViper.GetString("llm_api_key"),
				LlmModel:     internal.CredentialViper.GetString("llm_model"),
			}
		default:
			log.Fatal("Source type not implemented", "source", source.GetType())
		}
		// Waitgroups are used to ensure that the program doesn't exit before the goroutines finish
		// Channels are used to pass data between the goroutines
		postChan := make(chan platforms.PostData)
		errChan := make(chan error)

		// Start watching the source in a separate goroutine
		// This will send new posts to postChan and errors to errChan
		go watcher.Watch(internal.ConfigViper.GetDuration("interval"), options, postChan, errChan)

		// Start an infinite loop
		for {
			// Wait for something to happen
			select {
			// If a new post arrives
			case post := <-postChan:
				// Log the new post
				log.Info("New post", "post", post)
				err := pushToDestinations(post, destinationSlice, false)

				if err != nil {
					log.Error("Error", "error", err)
				}
			// If an error occurs
			case err := <-errChan:
				// Log the error
				log.Error("Error", "error", err)
			}
		}

	},
}

func init() {
	publishCmd.AddCommand(watchCmd)
	// The interval can be parsed with the time.ParseDu	ration function
	watchCmd.Flags().StringP("interval", "i", "30s", "Interval to check for new content")
	internal.ConfigViper.BindPFlag("interval", watchCmd.Flags().Lookup("interval"))
}
