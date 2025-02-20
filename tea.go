package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model holds the table and log data
type model struct {
	tableData [][]string
	logs      []string
}

// Init runs background tasks (like periodic updates)
func (m model) Init() tea.Cmd {
	return tickUpdate()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		fmt.Println(msg)
		// Update table data
		newTableData := [][]string{
			{"ID", "Name", "Status"},
			{"1", "Server-A", fmt.Sprintf("Running %d", time.Now().Second())},
			{"2", "Server-B", fmt.Sprintf("Stopped %d", time.Now().Second())},
			{"3", "Server-C", fmt.Sprintf("Pending %d", time.Now().Second())},
		}

		// Add new log entry
		newLogs := append(m.logs, fmt.Sprintf("Log: Updated at %s", time.Now().Format("15:04:05")))

		// Keep only last 5 logs
		if len(newLogs) > 5 {
			newLogs = newLogs[1:]
		}

		m.tableData = newTableData
		m.logs = newLogs

		return m, tickUpdate() // Continue ticking

	default:
		// Explicitly return unchanged model & no new commands
		return m, nil
	}
}

// View renders the UI
func (m model) View() string {
	// Table Header Styling
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFA500"))

	// Build table
	var tableView string
	for i, row := range m.tableData {
		line := ""
		for _, col := range row {
			line += lipgloss.NewStyle().Padding(0, 2).Render(col)
		}
		if i == 0 {
			line = headerStyle.Render(line) // Highlight header
		}
		tableView += line + "\n"
	}

	// Build logs
	logStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	logView := ""
	for _, log := range m.logs {
		logView += logStyle.Render(log) + "\n"
	}

	// Combine table and logs
	return tableView + "\n" + logView
}

// Message for periodic updates
type tickMsg struct{}

// tickUpdate triggers an update every second
func tickUpdate() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func whatever() {
	m := model{
		tableData: [][]string{
			{"IP Address", "Name", "Status"},
		},
		logs: []string{"Starting logs..."},
	}

	// Start program
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
	}
}
