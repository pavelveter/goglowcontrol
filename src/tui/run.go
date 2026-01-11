package tui

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"
)

// Run launches the TUI using provided dependencies.
func Run(deps Deps) error {
	deps.defaults()

	logCh := make(chan string, 32)
	log.SetOutput(logForwarder{ch: logCh})
	log.SetFlags(0)

	p := tea.NewProgram(newModel(deps, logCh),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}
