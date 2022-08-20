package commands

import (
	"fmt"
	"os"

	"github.com/martinbjeldbak/twitch-chat-cli/app"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "twitch-chat-cli",
	Short: "Chat on Twitch.tv from your terminal",
	Long: `A Twitch chat client in the terminal.

Allows chatting in multiple Twitch channels from the comfort of your terminal.
Supports connecting anonymously or as an authenticated user.

This application pairs nicely with Streamlink <https://streamlink.github.io/> for
a complete website-free viewing experience!`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return app.Start(viper.GetStringSlice("channels"), viper.GetInt("loglevel"), viper.GetStringSlice("accounts"))
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.twitch-chat-cli.yaml)")
	rootCmd.PersistentFlags().StringSlice("accounts", nil, "accounts and their oauth tokens to use, for example see example.yaml. Fetched via the `auth` command")
	rootCmd.PersistentFlags().StringSlice("channels", []string{"pokimane"}, "channels to join")

	for _, key := range []string{"accounts", "channels"} {
		err := viper.BindPFlag(key, rootCmd.PersistentFlags().Lookup(key))
		if err != nil {
			fmt.Printf("Error binding pflag '%v': %v", key, err)
		}
	}

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".twitch-chat-cli" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".twitch-chat-cli")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
