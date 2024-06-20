package cmd

import (
	"github.com/charmbracelet/log"
	"github.com/slashtechno/cross-blogger/internal/platforms"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// watchCmd represents the watch command
var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch a source for new content and publish it",
	Long: `Watch a source for new content and publish it.
	Specify the source with the first positional argument.
	The second positional argument and on are treated as destination names.
	Ensure that these are configured in the config file.
	`,
	// Arg 1: Source
	// Arg 2+: Destinations
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		// Load the sources and destinations
		sourceSlice, _, err := platforms.Load(viper.Get("sources"), viper.Get("destinations"), []string{args[0]}, args[1:])
		// sourceSlice, destinationSlice, err := platforms.Load(viper.Get("sources"), viper.Get("destinations"), []string{args[0]}, args[1:])
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
		// Assert that the source is a watchable source
		_, ok := source.(platforms.WatchableSource)
		if !ok {
			log.Fatal("Source is not watchable", "source", args[0])
		}

		// Pull the data from the source
		var options platforms.PushPullOptions
		switch source.GetType() {
		case "blogger":
			_, accessToken, blogId, err := prepareBlogger(source, nil, viper.GetString("google-client-id"), viper.GetString("google-client-secret"), viper.GetString("google-refresh-token"))
			if err != nil {
				log.Fatal(err)
			}
			options = platforms.PushPullOptions{
				// TODO pass the refresh token since I imagine the access token will expire eventually
				AccessToken: accessToken,
				BlogId:      blogId,
			}
		default:
			log.Fatal("Source type not implemented", "source", source.GetType())
		}
		// Waitgroups are used to ensure that the program doesn't exit before the goroutines finish
		// Channels are used to pass data between the goroutines
		postChan := make(chan platforms.PostData)
		errChan := make(chan error)

		blogger := source.(platforms.WatchableSource)
		go blogger.Watch(viper.GetDuration("interval"), options, postChan, errChan)

		for {
			select {
			case post := <-postChan:
				// Handle new post
				log.Info("New post", "post", post)
			case err := <-errChan:
				// Handle error
				log.Error("Error", "error", err)
			}
		}

	},
}

func init() {
	publishCmd.AddCommand(watchCmd)
	// The interval can be parsed with the time.ParseDu	ration function
	watchCmd.Flags().StringP("interval", "i", "1m", "Interval to check for new content")
	viper.BindPFlag("interval", watchCmd.Flags().Lookup("interval"))
}
