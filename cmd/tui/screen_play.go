package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const pollInterval = 3 * time.Second

type pollTickMsg struct{}
type historyLoadedMsg struct{ trail map[string]bool }

type playModel struct {
	client    *Client
	sessionID string
	state     *GameState
	trail     map[string]bool
	loading   bool
	err       error
	width     int
	height    int
	spinner   spinner.Model
	progress  progress.Model
	vp        viewport.Model
}

func newPlayModel(c *Client, sessionID string) playModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	prog := progress.New(
		progress.WithGradient("#d00000", "#ffd60a"),
		progress.WithoutPercentage(),
	)

	return playModel{
		client:    c,
		sessionID: sessionID,
		loading:   true,
		trail:     make(map[string]bool),
		spinner:   s,
		progress:  prog,
	}
}

func trailKey(x, y int) string {
	return fmt.Sprintf("%d,%d", x, y)
}

func (m playModel) loadHistoryCmd() tea.Cmd {
	return func() tea.Msg {
		entries, err := m.client.AllHistory(m.sessionID)
		if err != nil {
			return historyLoadedMsg{trail: make(map[string]bool)}
		}
		trail := make(map[string]bool, len(entries)+1)
		for _, e := range entries {
			if e.Success {
				trail[trailKey(e.FromPosition.X, e.FromPosition.Y)] = true
				trail[trailKey(e.ToPosition.X, e.ToPosition.Y)] = true
			}
		}
		return historyLoadedMsg{trail: trail}
	}
}

func (m playModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchStateCmd(), m.loadHistoryCmd())
}

func (m playModel) fetchStateCmd() tea.Cmd {
	return func() tea.Msg {
		gs, err := m.client.GameState(m.sessionID)
		if err != nil {
			return gameStateErrMsg{err}
		}
		return gameStateMsg{gs}
	}
}

func (m playModel) moveCmd(dir string) tea.Cmd {
	return func() tea.Msg {
		gs, err := m.client.Move(m.sessionID, dir)
		if err != nil {
			return moveErrMsg{err}
		}
		return moveMsg{gs}
	}
}

func (m playModel) resetCmd() tea.Cmd {
	return func() tea.Msg {
		gs, err := m.client.Reset(m.sessionID)
		if err != nil {
			return gameStateErrMsg{err}
		}
		return gameStateMsg{gs}
	}
}

func pollCmd() tea.Cmd {
	return tea.Tick(pollInterval, func(time.Time) tea.Msg {
		return pollTickMsg{}
	})
}

func (m playModel) Update(msg tea.Msg) (playModel, tea.Cmd) {
	switch msg := msg.(type) {
	case historyLoadedMsg:
		m.trail = msg.trail
		if m.state != nil {
			m.trail[trailKey(m.state.PlayerPos.X, m.state.PlayerPos.Y)] = true
		}
		m.updateViewport()
		return m, nil

	case gameStateMsg:
		m.loading = false
		m.err = nil
		m.state = msg.state
		m.trail[trailKey(msg.state.PlayerPos.X, msg.state.PlayerPos.Y)] = true
		m.updateViewport()
		return m, pollCmd()

	case gameStateErrMsg:
		m.loading = false
		m.err = msg.err
		return m, pollCmd()

	case moveMsg:
		m.loading = false
		m.err = nil
		m.state = msg.state
		m.trail[trailKey(msg.state.PlayerPos.X, msg.state.PlayerPos.Y)] = true
		m.updateViewport()
		return m, pollCmd()

	case moveErrMsg:
		m.err = msg.err
		return m, nil

	case pollTickMsg:
		return m, m.fetchStateCmd()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc", "q":
			return m, func() tea.Msg { return switchToSessionsMsg{} }
		case "r":
			m.loading = true
			m.trail = make(map[string]bool)
			return m, tea.Batch(m.spinner.Tick, m.resetCmd())
		// movement
		case "up", "w", "k":
			return m, m.moveCmd("UP")
		case "down", "s", "j":
			return m, m.moveCmd("DOWN")
		case "left", "a", "h":
			return m, m.moveCmd("LEFT")
		case "right", "d", "l":
			return m, m.moveCmd("RIGHT")
		}
		// viewport scrolling
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *playModel) updateViewport() {
	if m.state == nil {
		return
	}
	gridStr := renderGrid(m.state.Grid, m.state.PlayerPos, m.trail)
	gridW := lipgloss.Width(strings.SplitN(gridStr, "\n", 2)[0])
	gridH := lipgloss.Height(gridStr)

	vpW := gridW + 2
	vpH := gridH
	if m.height > 0 {
		// leave room for header (~5 lines) and footer (~3 lines)
		available := m.height - 8
		if vpH > available && available > 4 {
			vpH = available
		}
	}
	if vpW < 10 {
		vpW = 10
	}
	m.vp = viewport.New(vpW, vpH)
	m.vp.SetContent(gridStr)
}

func (m playModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(" Tesla Road Trip ") + "\n")

	if m.loading && m.state == nil {
		b.WriteString(fmt.Sprintf("\n  %s Loading...\n", m.spinner.View()))
		return b.String()
	}

	if m.state != nil {
		gs := m.state
		b.WriteString("\n")

		// Battery bar
		pct := 0.0
		if gs.MaxBattery > 0 {
			pct = float64(gs.Battery) / float64(gs.MaxBattery)
		}
		battLabel := fmt.Sprintf(" Battery %d/%d ", gs.Battery, gs.MaxBattery)
		riskLabel := ""
		if gs.BatteryRisk == "HIGH" {
			riskLabel = redStyle.Render(" ⚠ LOW")
		} else if gs.BatteryRisk == "MEDIUM" {
			riskLabel = yellowStyle.Render(" ~ caution")
		}
		m.progress.Width = 20
		b.WriteString("  " + boldStyle.Render(battLabel) + m.progress.ViewAs(pct) + riskLabel + "\n")

		// Stats row
		score := fmt.Sprintf("Score: %s", boldStyle.Render(fmt.Sprintf("%d", gs.Score)))
		moves := fmt.Sprintf("Moves: %d", gs.TotalMoves)
		pos := fmt.Sprintf("Pos: (%d,%d)", gs.PlayerPos.X, gs.PlayerPos.Y)
		mapName := fmt.Sprintf("Map: %s", gs.MapName)
		b.WriteString("  " + cyanStyle.Render(score) + "   " + dimStyle.Render(moves) + "   " + dimStyle.Render(pos) + "   " + dimStyle.Render(mapName) + "\n\n")

		// Status banner
		if gs.Victory {
			banner := greenStyle.Render("  🏆 VICTORY! You collected all parks!")
			b.WriteString(banner + "\n\n")
		} else if gs.GameOver {
			banner := redStyle.Render("  💀 GAME OVER — out of battery")
			b.WriteString(banner + "\n\n")
		}

		// Grid (updateViewport already called in Update; call here for View-time safety)
		m.updateViewport()
		b.WriteString(m.vp.View())
		b.WriteString("\n")

		// Message
		if gs.Message != "" {
			b.WriteString("\n  " + yellowStyle.Render(gs.Message) + "\n")
		}
	}

	if m.err != nil {
		b.WriteString("\n" + errStyle.Render("  Error: "+m.err.Error()) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(statusBarStyle.Render("  ↑↓←→ / wasd move  ·  r reset  ·  esc sessions  ·  ctrl+c quit"))

	return b.String()
}

// --- Grid rendering ---

var (
	cellBuilding    = lipgloss.NewStyle().Background(lipgloss.Color("235")).Foreground(lipgloss.Color("240"))
	cellWater       = lipgloss.NewStyle().Background(lipgloss.Color("27")).Foreground(lipgloss.Color("39"))
	cellHome        = lipgloss.NewStyle().Background(lipgloss.Color("196")).Foreground(lipgloss.Color("255")).Bold(true)
	cellSupercharge = lipgloss.NewStyle().Background(lipgloss.Color("220")).Foreground(lipgloss.Color("232")).Bold(true)
	cellPark        = lipgloss.NewStyle().Background(lipgloss.Color("28")).Foreground(lipgloss.Color("82"))
	cellParkVisited = lipgloss.NewStyle().Background(lipgloss.Color("239")).Foreground(lipgloss.Color("247"))
	cellRoad        = lipgloss.NewStyle().Background(lipgloss.Color("238")).Foreground(lipgloss.Color("244"))
	cellRoadVisited = lipgloss.NewStyle().Background(lipgloss.Color("238")).Foreground(lipgloss.Color("33"))
	cellPlayer      = lipgloss.NewStyle().Background(lipgloss.Color("255")).Foreground(lipgloss.Color("196")).Bold(true)
)

func renderGrid(grid [][]Cell, playerPos Position, trail map[string]bool) string {
	var sb strings.Builder
	for y, row := range grid {
		for x, cell := range row {
			isPlayer := x == playerPos.X && y == playerPos.Y
			inTrail := trail[trailKey(x, y)]
			sb.WriteString(renderCell(cell, isPlayer, inTrail))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// directionalGlyph returns a 2-char glyph for a road cell with direction constraints.
func directionalGlyph(dirs []string) string {
	set := make(map[string]bool, len(dirs))
	for _, d := range dirs {
		set[d] = true
	}
	n, s, e, w := set["north"], set["south"], set["east"], set["west"]
	switch {
	case n && s && !e && !w:
		return " ↕"
	case e && w && !n && !s:
		return " ↔"
	case n && !s && !e && !w:
		return " ↑"
	case s && !n && !e && !w:
		return " ↓"
	case e && !n && !s && !w:
		return " →"
	case w && !n && !s && !e:
		return " ←"
	case n && e && !s && !w:
		return " ↗"
	case n && w && !s && !e:
		return " ↖"
	case s && e && !n && !w:
		return " ↘"
	case s && w && !n && !e:
		return " ↙"
	default:
		return " ?"
	}
}

var cellDirectional = lipgloss.NewStyle().Background(lipgloss.Color("238")).Foreground(lipgloss.Color("214")).Bold(true)

func renderCell(cell Cell, isPlayer bool, inTrail bool) string {
	if isPlayer {
		return cellPlayer.Render(" T")
	}
	switch cell.Type {
	case "building":
		return cellBuilding.Render("##")
	case "water":
		return cellWater.Render("~~")
	case "home":
		return cellHome.Render(" H")
	case "supercharger":
		return cellSupercharge.Render(" S")
	case "park":
		if cell.Visited {
			return cellParkVisited.Render(" *")
		}
		return cellPark.Render(" P")
	default: // road
		if len(cell.AllowedDirections) > 0 {
			return cellDirectional.Render(directionalGlyph(cell.AllowedDirections))
		}
		if inTrail {
			return cellRoadVisited.Render(" ·")
		}
		return cellRoad.Render("  ")
	}
}
