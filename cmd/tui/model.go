package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screenID int

const (
	screenSessions screenID = iota
	screenCreate
	screenPlay
)

// async message types
type sessionsLoadedMsg struct{ sessions []Session }
type sessionsErrMsg struct{ err error }
type mapsLoadedMsg struct{ maps []MapInfo }
type mapsErrMsg struct{ err error }
type sessionCreatedMsg struct{ id string }
type createErrMsg struct{ err error }
type gameStateMsg struct{ state *GameState }
type gameStateErrMsg struct{ err error }
type moveMsg struct{ state *GameState }
type moveErrMsg struct{ err error }
type deletedMsg struct{}
type deleteErrMsg struct{ err error }

// root model routes to sub-screens
type rootModel struct {
	client   *Client
	screen   screenID
	sessions sessionsModel
	create   createModel
	play     playModel
	width    int
	height   int
}

func newRootModel(c *Client) rootModel {
	m := rootModel{
		client: c,
		screen: screenSessions,
	}
	m.sessions = newSessionsModel(c)
	m.create = newCreateModel(c)
	return m
}

func (m rootModel) Init() tea.Cmd {
	if m.screen == screenPlay {
		return m.play.Init()
	}
	return m.sessions.loadCmd()
}

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.sessions.width = msg.Width
		m.sessions.height = msg.Height
		m.play.width = msg.Width
		m.play.height = msg.Height
		return m, nil

	case switchToCreateMsg:
		m.screen = screenCreate
		m.create = newCreateModel(m.client)
		return m, m.create.Init()

	case switchToPlayMsg:
		m.screen = screenPlay
		m.play = newPlayModel(m.client, msg.sessionID)
		m.play.width = m.width
		m.play.height = m.height
		return m, m.play.Init()

	case switchToSessionsMsg:
		m.screen = screenSessions
		m.sessions.width = m.width
		m.sessions.height = m.height
		return m, m.sessions.loadCmd()
	}

	switch m.screen {
	case screenSessions:
		var cmd tea.Cmd
		m.sessions, cmd = m.sessions.Update(msg)
		return m, cmd
	case screenCreate:
		var cmd tea.Cmd
		m.create, cmd = m.create.Update(msg)
		return m, cmd
	case screenPlay:
		var cmd tea.Cmd
		m.play, cmd = m.play.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m rootModel) View() string {
	switch m.screen {
	case screenSessions:
		return m.sessions.View()
	case screenCreate:
		return m.create.View()
	case screenPlay:
		return m.play.View()
	}
	return ""
}

// navigation messages
type switchToCreateMsg struct{}
type switchToPlayMsg struct{ sessionID string }
type switchToSessionsMsg struct{}

// --- shared styles ---

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Padding(0, 2)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	errStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	boldStyle = lipgloss.NewStyle().Bold(true)

	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	redStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	cyanStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
)
