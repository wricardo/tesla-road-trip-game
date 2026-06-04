// Command tui is a terminal UI client for the Tesla Road Trip Game.
// Run the game server first, then launch this TUI to play or watch sessions.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	var server string
	var session string
	flag.StringVar(&server, "server", "http://localhost:8080", "game server URL")
	flag.StringVar(&session, "session", "", "jump directly to this session ID")
	flag.Parse()

	client := NewClient(server)
	model := newRootModel(client)
	if session != "" {
		model.screen = screenPlay
		model.play = newPlayModel(client, session)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
