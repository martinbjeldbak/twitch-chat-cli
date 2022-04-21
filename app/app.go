package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gempir/go-twitch-irc/v3"
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

func Start() error {
	p := tea.NewProgram(model{
		messages: make(chan MessageMsg),
		ms:       make([]MessageMsg, 0),
	},
		tea.WithAltScreen(),
		tea.WithMouseCellMotion())

	return p.Start()
}
