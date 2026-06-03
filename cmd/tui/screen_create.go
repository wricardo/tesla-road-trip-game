package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type createState int

const (
	createLoading   createState = iota
	createPickMap               // select map with arrow keys
	createNameInput             // type display name
	createCreating
)

type createModel struct {
	client    *Client
	state     createState
	maps      []MapInfo
	cursor    int
	nameInput textinput.Model
	err       error
	spinner   spinner.Model
}

func newCreateModel(c *Client) createModel {
	ti := textinput.New()
	ti.Placeholder = "Leave empty to skip"
	ti.CharLimit = 64
	ti.Width = 40

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	return createModel{
		client:    c,
		state:     createLoading,
		nameInput: ti,
		spinner:   s,
	}
}

func (m createModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.loadMapsCmd())
}

func (m createModel) loadMapsCmd() tea.Cmd {
	return func() tea.Msg {
		maps, err := m.client.Maps()
		if err != nil {
			return mapsErrMsg{err}
		}
		return mapsLoadedMsg{maps}
	}
}

func (m createModel) createSessionCmd(mapID, name string) tea.Cmd {
	return func() tea.Msg {
		id, err := m.client.CreateSession(mapID)
		if err != nil {
			return createErrMsg{err}
		}
		if name != "" {
			_ = m.client.UpdateSession(id, name)
		}
		return sessionCreatedMsg{id}
	}
}

func (m createModel) Update(msg tea.Msg) (createModel, tea.Cmd) {
	switch msg := msg.(type) {
	case mapsLoadedMsg:
		m.maps = msg.maps
		m.state = createPickMap
		m.cursor = 0
		return m, nil

	case mapsErrMsg:
		m.err = msg.err
		return m, nil

	case sessionCreatedMsg:
		return m, func() tea.Msg { return switchToPlayMsg{sessionID: msg.id} }

	case createErrMsg:
		m.err = msg.err
		m.state = createPickMap
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch m.state {
		case createPickMap:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "esc":
				return m, func() tea.Msg { return switchToSessionsMsg{} }
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.maps)-1 {
					m.cursor++
				}
			case "enter":
				m.state = createNameInput
				return m, m.nameInput.Focus()
			}

		case createNameInput:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.state = createPickMap
				m.nameInput.Blur()
				return m, nil
			case "enter":
				if len(m.maps) == 0 {
					return m, nil
				}
				m.state = createCreating
				mapID := m.maps[m.cursor].MapId
				name := strings.TrimSpace(m.nameInput.Value())
				return m, tea.Batch(m.spinner.Tick, m.createSessionCmd(mapID, name))
			}
			var cmd tea.Cmd
			m.nameInput, cmd = m.nameInput.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m createModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(" New Session "))
	b.WriteString("\n\n")

	switch m.state {
	case createLoading:
		b.WriteString(fmt.Sprintf("  %s Loading maps...\n", m.spinner.View()))

	case createPickMap:
		b.WriteString(boldStyle.Render("  Choose a map:") + "\n\n")
		for i, mp := range m.maps {
			cursor := "  "
			line := fmt.Sprintf("%-16s %s (grid: %d, battery: %d)", mp.Name, mp.Description, mp.GridSize, mp.MaxBattery)
			if i == m.cursor {
				cursor = cyanStyle.Render("> ")
				line = boldStyle.Render(line)
			} else {
				line = dimStyle.Render(line)
			}
			b.WriteString(cursor + line + "\n")
		}
		b.WriteString("\n")
		if m.err != nil {
			b.WriteString(errStyle.Render("  Error: "+m.err.Error()) + "\n\n")
		}
		b.WriteString(statusBarStyle.Render("  ↑/↓ select  ·  enter confirm  ·  esc back"))

	case createNameInput:
		selectedMap := ""
		if m.cursor < len(m.maps) {
			selectedMap = m.maps[m.cursor].Name
		}
		b.WriteString(boldStyle.Render("  Map: ") + cyanStyle.Render(selectedMap) + "\n\n")
		b.WriteString(boldStyle.Render("  Session name (optional):\n"))
		b.WriteString("  " + m.nameInput.View() + "\n\n")
		b.WriteString(statusBarStyle.Render("  enter create  ·  esc back to maps"))

	case createCreating:
		b.WriteString(fmt.Sprintf("  %s Creating session...\n", m.spinner.View()))
	}

	return b.String()
}
