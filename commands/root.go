package commands

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gempir/go-twitch-irc/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type MessageMsg struct {
	message string
}

type model struct {
	messages chan MessageMsg // where we will get messages
	ms       []MessageMsg
	err      error
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		connectTwitch(m.messages),  // generate messages
		waitForMessage(m.messages), // wait for a message
	)
}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

func waitForMessage(sub chan MessageMsg) tea.Cmd {
	return func() tea.Msg {
		return MessageMsg(<-sub)
	}
}

func connectTwitch(sub chan MessageMsg) tea.Cmd {
	return func() tea.Msg {
		client := twitch.NewAnonymousClient()

		client.OnPrivateMessage(func(m twitch.PrivateMessage) {
			sub <- MessageMsg{message: m.Message}
		})

		client.Join("nmplol")
		err := client.Connect()

		if err != nil {
			return errMsg{err}
		}
		return nil
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			return m, tea.Quit
		}
	case MessageMsg:
		m.ms = append(m.ms, msg)
		return m, waitForMessage(m.messages)
	case errMsg:
		m.err = msg
		return m, tea.Quit
	}

	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("\nWe had some trouble: %v\n\n", m.err)
	}

	s := "Messages:\n"
	for _, msg := range m.ms {
		s += fmt.Sprintf("\n%v", msg.message)
	}

	s += "\n"

	// Send to UI for rendering
	return s
}

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "twitch-chat-cli",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {

		p := tea.NewProgram(model{
			messages: make(chan MessageMsg),
			ms:       make([]MessageMsg, 0),
		},
			tea.WithAltScreen(),
			tea.WithMouseCellMotion())

		if err := p.Start(); err != nil {
			fmt.Printf("Uh oh, there was an error: %v\n", err)
			os.Exit(1)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.twitch-chat-cli.yaml)")

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
