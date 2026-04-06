package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/saurabh12nxf/registry-mirror/internal/storage"
)

type viewMode int

const (
	dashboardView viewMode = iota
	imageListView
	statsView
	logsView
)

type Model struct {
	db           *storage.DB
	currentView  viewMode
	cursor       int
	images       []storage.SyncRecord
	stats        *storage.AggregatedStats
	width        int
	height       int
	err          error
	lastUpdate   time.Time
}

type tickMsg time.Time
type imagesMsg []storage.SyncRecord
type statsMsg *storage.AggregatedStats
type errMsg error

func NewModel() Model {
	db, err := storage.NewDB()
	if err != nil {
		return Model{err: err}
	}

	return Model{
		db:          db,
		currentView: dashboardView,
		cursor:      0,
		lastUpdate:  time.Now(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		fetchImages(m.db),
		fetchStats(m.db),
	)
}
