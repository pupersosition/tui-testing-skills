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
	barWidth := 20
	filled := m.count + 10
	if filled < 0 {
		filled = 0
	}
	if filled > barWidth {
		filled = barWidth
	}
	bar := fmt.Sprintf("[%s%s]", repeat("#", filled), repeat(".", barWidth-filled))

	return fmt.Sprintf(
		"+------------------------------+\n"+
			"| Bubble Tea Counter Fixture   |\n"+
			"+------------------------------+\n\n"+
			"%s\n\n"+
			"Counter: %d\n"+
			"Meter:   %s\n\n"+
			"Controls:\n"+
			"  + / up / k     Increment\n"+
			"  - / down / j   Decrement\n"+
			"  q              Quit\n",
		status,
		m.count,
		bar,
	)
}

func repeat(value string, count int) string {
	if count <= 0 {
		return ""
	}
	result := ""
	for i := 0; i < count; i++ {
		result += value
	}
	return result
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "fixture failed: %v\n", err)
		os.Exit(1)
	}
}
