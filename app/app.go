package app

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gempir/go-twitch-irc/v3"
	"github.com/muesli/reflow/wordwrap"
	"github.com/nicklaw5/helix"
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
	Badges      map[string]int
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

type twitchAccount struct {
	Username    string
	UserId      int
	ClientId    string
	AccessToken string
}

func (a twitchAccount) isAnonymous() bool {
	return a.Username == "" || a.ClientId == "" || a.AccessToken == ""
}

func (a twitchAccount) isAuthenticated() bool {
	return !a.isAnonymous()
}

func (a *twitchAccount) UnmarshalString(encodedKvs string) error {
	kvs := strings.Split(encodedKvs, ";")

	for _, kv := range kvs {
		entry := strings.Split(kv, "=")

		switch entry[0] {
		case "username":
			a.Username = entry[1]
		case "user_id":
			userId, err := strconv.Atoi(entry[1])
			if err != nil {
				return err
			}

			a.UserId = userId
		case "client_id":
			a.ClientId = entry[1]
		case "oauth_token":
			a.AccessToken = entry[1]
		}
	}
	return nil
}

type (
	twitchChannelInfos []*twitchChannel
	accountInfos       []twitchAccount
)

type model struct {
	currChannel  int
	channels     twitchChannelInfos
	accounts     accountInfos
	client       *twitch.Client
	helixClient  *helix.Client
	logger       *zap.SugaredLogger
	viewport     viewport.Model
	ready        bool
	messageStyle map[string]lipgloss.Style

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

func handleChatMessage(ci twitchChannelInfos) func(m twitch.PrivateMessage) {
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
		client.OnPrivateMessage(handleChatMessage(ci))

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

func sayMsg(client *twitch.Client, channel *twitchChannel, message string) tea.Cmd {
	return func() tea.Msg {
		client.Say(channel.name, message)

		channel.textInput.Reset()

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

			cmds = append(cmds, sayMsg(m.client, currentChannel, currentChannel.textInput.Value()))
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
			for _, c := range m.channels {
				c.textInput.Width = m.viewport.Width
			}
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight

			for _, c := range m.channels {
				c.textInput.Width = m.viewport.Width
			}
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
		style, ok := m.messageStyle[msg.user.Color]

		if !ok {
			m.messageStyle[msg.user.Color] = lipgloss.NewStyle().
				Inline(true).
				Bold(true).
				Foreground(lipgloss.Color(msg.user.Color))
			style = m.messageStyle[msg.user.Color]
		}

		userName := style.MaxWidth(m.viewport.Width).Render(msg.user.DisplayName)

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

	// Send to UI for rendering
	return fmt.Sprintf("%s\n%s", wordwrap.String(m.viewport.View(), m.viewport.Width), m.footerView())
}

func setupTwitchApiClient(u twitchAccount) (*helix.Client, error) {
	hclient, err := helix.NewClient(&helix.Options{
		ClientID:       u.ClientId,
		AppAccessToken: u.AccessToken,
	})
	if err != nil {
		return nil, err
	}

	return hclient, err
}

func getChannelInformation(authedClient *helix.Client, channelNames []string) (*helix.GetChannelInformationResponse, error) {
	usersResp, err := authedClient.GetUsers(&helix.UsersParams{Logins: channelNames})
	if err != nil {
		return nil, err
	}
	if usersResp.Error != "" {
		return nil, fmt.Errorf("error requesting channels: %v", usersResp.Error)
	}

	ids := make([]string, len(usersResp.Data.Users))
	for i, u := range usersResp.Data.Users {
		ids[i] = u.ID
	}

	liveInfos, err := authedClient.GetChannelInformation(&helix.GetChannelInformationParams{
		BroadcasterIDs: ids,
	})
	if err != nil {
		return nil, err
	}
	if liveInfos.Error != "" {
		return nil, fmt.Errorf("error requesting channels: %v", usersResp.Error)
	}

	return liveInfos, nil
}

func initialModel(sugar *zap.SugaredLogger, encodedAccountsInfo []string, initChannels []string) (model, error) {
	var ircClient *twitch.Client
	var liveInfos *helix.GetChannelInformationResponse
	var twitchClient *helix.Client

	as := make(accountInfos, 0, len(encodedAccountsInfo)+1) // +1 for anon account
	as = append(as, twitchAccount{Username: "anonymous"})   // TODO: we shouldn't connect to helix for extra infos for anon

	for _, kvs := range encodedAccountsInfo {
		var a twitchAccount

		err := a.UnmarshalString(kvs)
		if err != nil {
			return model{}, err
		}

		as = append(as, a)
	}

	initialAccount := as[len(as)-1]

	if initialAccount.isAnonymous() {
		ircClient = twitch.NewAnonymousClient()
	} else {
		ircClient = twitch.NewClient(initialAccount.Username, fmt.Sprintf("oauth:%v", initialAccount.AccessToken))
		twitchClient, err := setupTwitchApiClient(initialAccount)
		if err != nil {
			return model{}, err
		}

		liveInfos, err = getChannelInformation(twitchClient, initChannels)
		if err != nil {
			return model{}, err
		}
	}

	channelInfo := make(twitchChannelInfos, len(initChannels))
	for i, c := range initChannels {
		gameName := ""

		ti := textinput.New()
		ti.CharLimit = 156
		ti.Width = 20
		ti.Placeholder = fmt.Sprintf("Authenticate to send messages in %v", c)

		if initialAccount.isAuthenticated() {
			cInfo := liveInfos.Data.Channels[i]
			gameName = fmt.Sprintf("(streaming %v)", cInfo.GameName)
			ti.Placeholder = fmt.Sprintf("Send a message in %v %v as %v", c, gameName, initialAccount.Username)
			ti.Focus()
		}

		channelInfo[i] = &twitchChannel{
			name:           c,
			messageChannel: make(chan twitchMessage),
			textInput:      ti,
		}
	}

	return model{
		currChannel:  0,
		channels:     channelInfo,
		logger:       sugar,
		client:       ircClient,
		helixClient:  twitchClient,
		messageStyle: make(map[string]lipgloss.Style),
		accounts:     as,
	}, nil
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
	initModel, err := initialModel(sugar, accounts, channels)
	if err != nil {
		return err
	}

	p := tea.NewProgram(initModel,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion())

	return p.Start()
}
