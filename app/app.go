package app

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gempir/go-twitch-irc/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// You generally won't need this unless you're processing stuff with
// complicated ANSI escape sequences. Turn it on if you notice flickering.
//
// Also keep in mind that high performance rendering only works for programs
// that use the full size of the terminal. We're enabling that below with
// tea.EnterAltScreen().
const useHighPerformanceRenderer = false

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

type twitchChannelInfos []*twitchChannel

type model struct {
	currChannel int
	client      *twitch.Client
	channels    twitchChannelInfos
	logger      *zap.SugaredLogger
	viewport    viewport.Model
	ready       bool

	err error
}

type channelMessageMsg struct {
	channelName string
	message     twitchMessage
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

func (m *model) NextChannel() {
	if !m.OnLastPage() {
		m.currChannel++
	}
}

func (m *model) PrevChannel() {
	if m.currChannel > 0 {
		m.currChannel--
	}
}

func (m model) OnLastPage() bool {
	return m.currChannel == len(m.channels)-1
}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

func waitForMessage(c *twitchChannel) tea.Cmd {
	return func() tea.Msg {
		return channelMessageMsg{channelName: c.name, message: <-c.messageChannel}
	}
}

func handleChatMessage(client *twitch.Client, ci twitchChannelInfos) func(m twitch.PrivateMessage) {
	return func(m twitch.PrivateMessage) {
		channel, ok := channelByName(m.Channel, ci)

		if !ok {
			fmt.Printf("TODO: Got message for unknown channel, need to handle (create a new channel and insert in channel slice)")
			os.Exit(1)
		}
		channel.messageChannel <- twitchMessage{
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
		for _, k := range ci {
			channelNames = append(channelNames, k.name)
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
	var cmds []tea.Cmd

	m.logger.Debugf("Got message %v", msg)

	currentChannel := m.currentChannel()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "left":
			m.PrevChannel()
			m.viewport.SetContent(m.RenderMessages())
		case "right":
			m.NextChannel()
			m.viewport.SetContent(m.RenderMessages())
		case "enter":
			m.logger.Infof("Saying '%v' in %v", currentChannel.textInput.Value(), currentChannel.name)

			m.client.Say(currentChannel.name, currentChannel.textInput.Value())

			currentChannel.textInput.Reset()
		}

	case channelMessageMsg:
		if channel, ok := channelByName(msg.channelName, m.channels); ok {
			channel.messagesToRender = append(channel.messagesToRender, msg.message)

			if channel == currentChannel {
				m.viewport.SetContent(m.RenderMessages())
				m.viewport.GotoBottom()
			}

			cmds = append(cmds, waitForMessage(channel))
		}

	case tea.WindowSizeMsg:
		// TODO: Handle console window change similar to
		// https://github.com/charmbracelet/bubbletea/blob/master/examples/pager/main.go#L61
		// https://github.com/charmbracelet/bubbletea/blob/master/examples/package-manager/main.go#L55
		// headerHeight := lipgloss.Height(m.headerView())
		headerHeight := 0
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.HighPerformanceRendering = useHighPerformanceRenderer
			m.ready = true

			// This is only necessary for high performance rendering, which in
			// most cases you won't need.
			//
			// Render the viewport one line below the header.
			m.viewport.YPosition = headerHeight + 1
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

		if useHighPerformanceRenderer {
			// Render (or re-render) the whole viewport. Necessary both to
			// initialize the viewport and when the window is resized.
			//
			// This is needed for high-performance rendering only.
			cmds = append(cmds, viewport.Sync(m.viewport))
		}
	case errMsg:
		m.err = msg
		return m, tea.Quit
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	currentChannel.textInput, cmd = currentChannel.textInput.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) currentChannel() *twitchChannel {
	return m.channels[m.currChannel]
}

func (m model) footerView() string {
	// info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	// line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	// return lipgloss.JoinHorizontal(lipgloss.Center, line, info)

	return fmt.Sprintf("%s\n", m.currentChannel().textInput.View())
}

func channelByName(name string, channels twitchChannelInfos) (*twitchChannel, bool) {
	for _, v := range channels {
		if v.name == name {
			return v, true
		}
	}
	return nil, false
}

func (m model) RenderMessages() string {
	var b strings.Builder
	for _, msg := range m.currentChannel().messagesToRender {
		userName := lipgloss.NewStyle().
			Inline(true).
			Bold(true).
			Foreground(lipgloss.Color(msg.user.Color)).
			Render(msg.user.DisplayName)

		b.WriteString(fmt.Sprintf("%v: %v\n", userName, msg.message))
	}
	return b.String()
}

func (m model) View() string {
	if m.err != nil {
		m.logger.Debugf("We had some trouble: %v\n", m.err)

		return fmt.Sprintf("\nWe had some trouble: %v\n\n", m.err)
	}

	if !m.ready {
		return "\n Initializing..."
	}

	// var b strings.Builder

	// Send to UI for rendering
	return fmt.Sprintf("%s\n%s", m.viewport.View(), m.footerView())
}

func initialModel(sugar *zap.SugaredLogger, client *twitch.Client, initChannels []string, initUsername string) model {
	channelInfo := make(twitchChannelInfos, len(initChannels))
	for i, c := range initChannels {
		ti := textinput.New()
		ti.Placeholder = fmt.Sprintf("Send a message in %v as %v", c, initUsername)
		ti.Focus()
		ti.CharLimit = 156
		ti.Width = 20

		channelInfo[i] = &twitchChannel{name: c, messageChannel: make(chan twitchMessage), textInput: ti}
	}

	return model{
		currChannel: 0,
		channels:    channelInfo,
		logger:      sugar,
		client:      client,
	}
}

func safeSync(logger *zap.Logger, err *error) {
	if serr := logger.Sync(); serr != nil && *err == nil {
		*err = serr
	}
}

func Start(channels []string, loglevel int, accounts []string) error {
	cfg := zap.NewDevelopmentConfig()
	cfg.OutputPaths = []string{
		"log.log",
	}
	cfg.Development = true
	cfg.Level.SetLevel(zapcore.Level(loglevel))

	logger, err := cfg.Build()
	if err != nil {
		return err
	}

	defer safeSync(logger, &err) // flushes buffer, if any
	sugar := logger.Sugar()

	if len(channels) == 0 {
		return errors.New("unable to load any channels, check config")
	}

	var client *twitch.Client

	username := "anonymous"

	sugar.Infof("Accounts: %v", accounts)

	if len(accounts) > 1 {
		return errors.New("don't yet have support for multi-account")
	}

	if len(accounts) == 0 {
		client = twitch.NewAnonymousClient() // support connecting anonymously by default
	} else {
		userToken := strings.Split(accounts[0], ":")
		username = userToken[0]
		token := userToken[1]

		client = twitch.NewClient(username, fmt.Sprintf("oauth:%v", token))
	}

	p := tea.NewProgram(initialModel(sugar, client, channels, username),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion())

	return p.Start()
}
