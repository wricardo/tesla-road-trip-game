package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type sessionsModel struct {
	client   *Client
	sessions []Session
	table    table.Model
	loading  bool
	err      error
	deleting bool
	width    int
	height   int
	spinner  spinner.Model
}

func newSessionsModel(c *Client) sessionsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	cols := []table.Column{
		{Title: "Name / ID", Width: 24},
		{Title: "Map", Width: 12},
		{Title: "Battery", Width: 10},
		{Title: "Score", Width: 6},
		{Title: "Status", Width: 12},
		{Title: "Last Accessed", Width: 16},
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(15),
	)
	ts := table.DefaultStyles()
	ts.Header = ts.Header.BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("62")).
		BorderBottom(true).
		Bold(true)
	ts.Selected = ts.Selected.
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("62")).
		Bold(true)
	t.SetStyles(ts)

	return sessionsModel{
		client:  c,
		table:   t,
		loading: true,
		spinner: s,
	}
}

func (m sessionsModel) loadCmd() tea.Cmd {
	return func() tea.Msg {
		sessions, err := m.client.Sessions()
		if err != nil {
			return sessionsErrMsg{err}
		}
		return sessionsLoadedMsg{sessions}
	}
}

func (m sessionsModel) deleteCmd(id string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.DeleteSession(id)
		if err != nil {
			return deleteErrMsg{err}
		}
		return deletedMsg{}
	}
}

func (m sessionsModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.loadCmd())
}

func (m sessionsModel) Update(msg tea.Msg) (sessionsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case sessionsLoadedMsg:
		m.loading = false
		m.err = nil
		m.sessions = msg.sessions
		m.table.SetRows(buildRows(msg.sessions))
		return m, nil

	case sessionsErrMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case deletedMsg:
		m.deleting = false
		return m, m.loadCmd()

	case deleteErrMsg:
		m.deleting = false
		m.err = msg.err
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "n":
			return m, func() tea.Msg { return switchToCreateMsg{} }
		case "r":
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.loadCmd())
		case "d":
			if len(m.sessions) == 0 {
				return m, nil
			}
			row := m.table.Cursor()
			if row < len(m.sessions) {
				m.deleting = true
				id := m.sessions[row].ID
				return m, m.deleteCmd(id)
			}
		case "enter":
			if len(m.sessions) == 0 {
				return m, nil
			}
			row := m.table.Cursor()
			if row < len(m.sessions) {
				id := m.sessions[row].ID
				return m, func() tea.Msg { return switchToPlayMsg{sessionID: id} }
			}
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m sessionsModel) View() string {
	var b strings.Builder

	header := titleStyle.Render(" Tesla Road Trip Game ")
	b.WriteString(header)
	b.WriteString("\n\n")

	if m.loading {
		b.WriteString(fmt.Sprintf("  %s Loading sessions...\n", m.spinner.View()))
	} else if m.err != nil {
		b.WriteString(errStyle.Render("  Error: "+m.err.Error()) + "\n")
		b.WriteString(dimStyle.Render("  Make sure server is running. Press r to retry.\n"))
	} else if len(m.sessions) == 0 {
		b.WriteString(dimStyle.Render("  No sessions yet. Press n to create one.\n"))
	} else {
		b.WriteString(m.table.View())
		b.WriteString("\n")
	}

	b.WriteString("\n")
	keys := "  n new  ·  enter play  ·  d delete  ·  r refresh  ·  q quit"
	if m.deleting {
		keys = "  Deleting..."
	}
	b.WriteString(statusBarStyle.Render(keys))

	return b.String()
}

func buildRows(sessions []Session) []table.Row {
	rows := make([]table.Row, len(sessions))
	for i, s := range sessions {
		idLen := min(8, len(s.ID))
		name := s.ID[:idLen]
		if s.DisplayName != nil && *s.DisplayName != "" {
			n := *s.DisplayName
			if len(n) > 22 {
				n = n[:22]
			}
			name = n
		}

		gs := s.GameState
		battery := fmt.Sprintf("%d/%d", gs.Battery, gs.MaxBattery)

		status := cyanStyle.Render("Playing")
		if gs.Victory {
			status = greenStyle.Render("Victory!")
		} else if gs.GameOver {
			status = redStyle.Render("Game Over")
		}

		accessed := formatTime(s.LastAccessedAt)

		rows[i] = table.Row{name, s.MapName, battery, fmt.Sprintf("%d", gs.Score), status, accessed}
	}
	return rows
}

func formatTime(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		if len(ts) > 16 {
			return ts[:16]
		}
		return ts
	}
	return t.Format("01/02 15:04")
}
