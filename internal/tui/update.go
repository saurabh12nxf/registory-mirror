package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/saurabh12nxf/registry-mirror/internal/storage"
	"time"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "tab":
			m.currentView = (m.currentView + 1) % 4
			m.cursor = 0
			return m, nil

		case "1":
			m.currentView = dashboardView
			return m, nil
		case "2":
			m.currentView = imageListView
			return m, nil
		case "3":
			m.currentView = statsView
			return m, nil
		case "4":
			m.currentView = logsView
			return m, nil

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case "down", "j":
			if m.currentView == imageListView && m.cursor < len(m.images)-1 {
				m.cursor++
			}
			return m, nil

		case "r":
			return m, tea.Batch(
				fetchImages(m.db),
				fetchStats(m.db),
			)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		m.lastUpdate = time.Time(msg)
		return m, tea.Batch(
			tickCmd(),
			fetchImages(m.db),
			fetchStats(m.db),
		)

	case imagesMsg:
		m.images = []storage.SyncRecord(msg)
		return m, nil

	case statsMsg:
		m.stats = msg
		return m, nil

	case errMsg:
		m.err = error(msg)
		return m, nil
	}

	return m, nil
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchImages(db *storage.DB) tea.Cmd {
	return func() tea.Msg {
		images, err := db.GetRecentSyncs(50)
		if err != nil {
			return errMsg(err)
		}
		return imagesMsg(images)
	}
}

func fetchStats(db *storage.DB) tea.Cmd {
	return func() tea.Msg {
		stats, err := db.GetAggregatedStats()
		if err != nil {
			return errMsg(err)
		}
		return statsMsg(stats)
	}
}
