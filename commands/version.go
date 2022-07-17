package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of twitch-chat-cli",
	Long:  "All software has versions. This is Twitch Chat CLI's.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Twitch Chat CLI v0.1 -- HEAD") // TODO: set right version, see https://github.com/gohugoio/hugo/blob/41cc4e4ba3bd849cee7dcb691504ebebbfce680f/common/hugo/version.go#L129
	},
}
