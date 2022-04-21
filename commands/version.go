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
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Twitch Chat CLI v0.0 -- HEAD") // TODO: set right version
	},
}
