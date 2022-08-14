package app

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gempir/go-twitch-irc/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type user struct {
	Name        string
	DisplayName string
	Color       string
	Badges      map[string]int // TODO: consider model for this
}

type twitchMessage struct {
	message string
	user    user
}

type twitchChannel struct {
	name             string
	messageChannel   chan twitchMessage // where we will get messages
	messagesToRender []twitchMessage
	textInput        textinput.Model
}

type twitchChannelInfos map[string]*twitchChannel

type model struct {
	client         *twitch.Client
	currentChannel *twitchChannel
	channels       twitchChannelInfos
	logger         *zap.SugaredLogger

	err error
}

type channelMessageMsg struct {
	name    string
	message twitchMessage
}

func (m model) Init() tea.Cmd {
	var initCmds []tea.Cmd

	// Connect to desired twitch channels
	initCmds = append(initCmds, connectTwitch(m.client, m.channels))

	// Emit Tea messages for each new Twitch message
	for _, channel := range m.channels {
		initCmds = append(initCmds, waitForMessage(channel))
	}

	initCmds = append(initCmds, textinput.Blink)

	return tea.Batch(initCmds...)
}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

func waitForMessage(c *twitchChannel) tea.Cmd {
	return func() tea.Msg {
		return channelMessageMsg{name: c.name, message: <-c.messageChannel}
	}
}

func handleChatMessage(client *twitch.Client, ci twitchChannelInfos) func(m twitch.PrivateMessage) {
	return func(m twitch.PrivateMessage) {
		ci[m.Channel].messageChannel <- twitchMessage{
			message: m.Message,
			user: user{
				Name:        m.User.Name,
				DisplayName: m.User.DisplayName,
				Color:       m.User.Color,
				Badges:      m.User.Badges,
			},
		}
	}
}

func connectTwitch(client *twitch.Client, ci twitchChannelInfos) tea.Cmd {
	return func() tea.Msg {
		client.OnPrivateMessage(handleChatMessage(client, ci))

		channelNames := make([]string, 0, len(ci))
		for k := range ci {
			channelNames = append(channelNames, k)
		}
		client.Join(channelNames...)

		if err := client.Connect(); err != nil {
			return errMsg{err}
		}

		return nil
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// m.logger.Debugf("Got message %v\n", msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "up":
			m.currentChannel = m.channels[m.currentChannel.name] // TODO: fix change channel
		case "enter":
			m.logger.Infof("Saying '%v' in %v", m.currentChannel.textInput.Value(), m.currentChannel.name)

			m.client.Say(m.currentChannel.name, m.currentChannel.textInput.Value())

			m.currentChannel.textInput.Reset()
		default:
			m.currentChannel.textInput, cmd = m.currentChannel.textInput.Update(msg)
		}

	case channelMessageMsg:
		if channel, ok := m.channels[msg.name]; ok {
			// Throw away old messages to improve performance
			if len(channel.messagesToRender) > 1000 { // TODO: paramterise this
				channel.messagesToRender = channel.messagesToRender[50:]
			}

			channel.messagesToRender = append(channel.messagesToRender, msg.message)
			return m, waitForMessage(channel)
		}

	case errMsg:
		m.err = msg
		return m, tea.Quit
	}

	return m, cmd
}

func (m model) View() string {
	if m.err != nil {
		m.logger.Debugf("We had some trouble: %v\n", m.err)

		return fmt.Sprintf("\nWe had some trouble: %v\n\n", m.err)
	}

	s := ""
	for _, msg := range m.currentChannel.messagesToRender {
		userMsg := lipgloss.NewStyle().
			Inline(true).
			Bold(true).
			Foreground(lipgloss.Color(msg.user.Color)).
			Render(msg.user.DisplayName)

		userMsg += fmt.Sprintf(": %v\n", msg.message)

		s += userMsg
	}

	s += fmt.Sprintf("%s", m.currentChannel.textInput.View())
	s += "\n"

	// Send to UI for rendering
	return s
}

func initialModel(sugar *zap.SugaredLogger, c *twitch.Client, initChannels []string) model {
	channelInfo := make(twitchChannelInfos)
	for _, c := range initChannels {
		ti := textinput.New()
		ti.Placeholder = fmt.Sprintf("Send a message in %v", c)
		ti.Focus()
		ti.CharLimit = 156
		ti.Width = 20

		channelInfo[c] = &twitchChannel{name: c, messageChannel: make(chan twitchMessage), textInput: ti}
	}

	return model{
		currentChannel: channelInfo[initChannels[0]],
		channels:       channelInfo,
		logger:         sugar,
		client:         c,
	}
}

func Start(channels []string, loglevel int) error {
	cfg := zap.NewDevelopmentConfig()
	cfg.OutputPaths = []string{
		"log.log",
	}
	cfg.Development = true
	cfg.Level.SetLevel(zapcore.Level(loglevel))

	logger, _ := cfg.Build()

	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	if len(channels) == 0 {
		return errors.New("unable to load any channels, check config")
	}

	// TODO: extract auth to oauth2 pkg https://pkg.go.dev/golang.org/x/oauth2#AuthStyle, https://dev.twitch.tv/docs/authentication/getting-tokens-oauth/#implicit-grant-flow
	c := twitch.NewClient("maartinbm", "oauth:secret")

	p := tea.NewProgram(initialModel(sugar, c, channels),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion())

	return p.Start()
}
