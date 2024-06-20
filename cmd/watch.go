package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
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
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("watch called")
	},
}

func init() {
	publishCmd.AddCommand(watchCmd)

}
