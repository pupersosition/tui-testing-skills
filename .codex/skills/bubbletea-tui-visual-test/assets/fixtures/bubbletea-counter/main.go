package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	count int
	ready bool
}

func initialModel() model {
	return model{count: 0, ready: true}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k", "+":
			m.count++
		case "down", "j", "-":
			m.count--
		}
	}

	return m, nil
}

func (m model) View() string {
	status := "STATUS: READY"
	return fmt.Sprintf(
		"Bubble Tea Fixture\n\n%s\n\nCounter: %d\n\nKeys: + / - (or up/down)\nPress q to quit.\n",
		status,
		m.count,
	)
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "fixture failed: %v\n", err)
		os.Exit(1)
	}
}
