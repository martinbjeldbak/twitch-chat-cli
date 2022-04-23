package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gempir/go-twitch-irc/v3"
)

type message struct {
	message string
}

type channel struct {
	name             string
	messageChannel   chan message // where we will get messages
	messagesToRender []message
}

type twitchChannelInfos map[string]*channel

type model struct {
	channels twitchChannelInfos

	err error
}

type channelMessageMsg struct {
	name    string
	message message
}

func (m model) Init() tea.Cmd {
	var initCmds []tea.Cmd

	// Connect to desired twitch channels
	initCmds = append(initCmds, connectTwitch(m.channels))

	// Emit Tea messages for each new Twitch message
	for _, channel := range m.channels {
		initCmds = append(initCmds, waitForMessage(channel))
	}

	return tea.Batch(initCmds...)
}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

func waitForMessage(c *channel) tea.Cmd {
	return func() tea.Msg {
		return channelMessageMsg{name: c.name, message: <-c.messageChannel}
	}
}

func connectTwitch(ci twitchChannelInfos) tea.Cmd {
	return func() tea.Msg {
		client := twitch.NewAnonymousClient()

		client.OnPrivateMessage(func(m twitch.PrivateMessage) {
			ci[m.Channel].messageChannel <- message{message: m.Message}
		})

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
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			return m, tea.Quit
		}
	case channelMessageMsg:
		channel, ok := m.channels[msg.name]
		if ok {
			channel.messagesToRender = append(channel.messagesToRender, msg.message)
			return m, waitForMessage(channel)
		}
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

	s := fmt.Sprintf("Messages for channel %v:\n", m.channels["nmplol"].name)
	for _, msg := range m.channels["nmplol"].messagesToRender {
		s += fmt.Sprintf("\n%v", msg.message)
	}

	s += "\n"

	// Send to UI for rendering
	return s
}

func Start(channels []string) error {
	channelInfo := make(twitchChannelInfos)
	for _, c := range channels {
		channelInfo[c] = &channel{name: c, messageChannel: make(chan message)}
	}

	p := tea.NewProgram(model{
		channels: channelInfo,
	},
		tea.WithAltScreen(),
		tea.WithMouseCellMotion())

	return p.Start()
}
