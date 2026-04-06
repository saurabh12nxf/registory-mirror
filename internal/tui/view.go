package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFA500"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			MarginTop(1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2).
			MarginBottom(1)
)

func (m Model) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err))
	}

	var content string

	switch m.currentView {
	case dashboardView:
		content = m.renderDashboard()
	case imageListView:
		content = m.renderImageList()
	case statsView:
		content = m.renderStats()
	case logsView:
		content = m.renderLogs()
	}

	header := m.renderHeader()
	help := m.renderHelp()

	return fmt.Sprintf("%s\n\n%s\n\n%s", header, content, help)
}

func (m Model) renderHeader() string {
	tabs := []string{
		"[1] Dashboard",
		"[2] Images",
		"[3] Stats",
		"[4] Logs",
	}

	var renderedTabs []string
	for i, tab := range tabs {
		if viewMode(i) == m.currentView {
			renderedTabs = append(renderedTabs, selectedStyle.Render(tab))
		} else {
			renderedTabs = append(renderedTabs, tab)
		}
	}

	title := titleStyle.Render("🐳 Registry Mirror TUI")
	tabBar := strings.Join(renderedTabs, "  ")

	return fmt.Sprintf("%s\n%s", title, tabBar)
}

func (m Model) renderDashboard() string {
	if m.stats == nil {
		return "Loading..."
	}

	overview := boxStyle.Render(fmt.Sprintf(
		"%s\n\n"+
			"Unique Images:    %s\n"+
			"Total Syncs:      %s\n"+
			"Data Transferred: %s\n"+
			"Time Saved:       %s",
		headerStyle.Render(" Overview "),
		successStyle.Render(fmt.Sprintf("%d", m.stats.UniqueImages)),
		successStyle.Render(fmt.Sprintf("%d", m.stats.TotalCount)),
		successStyle.Render(formatBytes(m.stats.TotalBytes)),
		successStyle.Render(fmt.Sprintf("%.1f min", m.stats.TotalDuration/60)),
	))

	recentActivity := m.renderRecentActivity(5)

	return fmt.Sprintf("%s\n%s", overview, recentActivity)
}

func (m Model) renderImageList() string {
	if len(m.images) == 0 {
		return boxStyle.Render("No images synced yet.\n\nRun: registry-mirror sync <image>")
	}

	var items []string
	items = append(items, headerStyle.Render(" Cached Images "))
	items = append(items, "")

	for i, img := range m.images {
		cursor := " "
		if i == m.cursor {
			cursor = "▶"
		}

		status := "✅"
		statusColor := successStyle
		if img.Status != "completed" {
			status = "❌"
			statusColor = errorStyle
		}

		line := fmt.Sprintf("%s %s %s - %s (%s)",
			cursor,
			status,
			img.Image,
			statusColor.Render(img.Status),
			formatBytes(img.Bytes),
		)

		if i == m.cursor {
			line = selectedStyle.Render(line)
		}

		items = append(items, line)
	}

	return boxStyle.Render(strings.Join(items, "\n"))
}

func (m Model) renderStats() string {
	if m.stats == nil {
		return "Loading statistics..."
	}

	avgDuration := 0.0
	if m.stats.TotalCount > 0 {
		avgDuration = m.stats.TotalDuration / float64(m.stats.TotalCount)
	}

	avgSize := int64(0)
	if m.stats.TotalCount > 0 {
		avgSize = m.stats.TotalBytes / int64(m.stats.TotalCount)
	}

	content := fmt.Sprintf(
		"%s\n\n"+
			"Total Syncs:          %d\n"+
			"Successful:           %d\n"+
			"Unique Images:        %d\n\n"+
			"Total Data:           %s\n"+
			"Average Size:         %s\n\n"+
			"Total Duration:       %.1f min\n"+
			"Average Duration:     %.2f sec\n\n"+
			"Estimated Bandwidth Saved: %s\n"+
			"Estimated Time Saved:      %.1f min",
		headerStyle.Render(" Detailed Statistics "),
		m.stats.TotalCount,
		m.stats.TotalCount,
		m.stats.UniqueImages,
		formatBytes(m.stats.TotalBytes),
		formatBytes(avgSize),
		m.stats.TotalDuration/60,
		avgDuration,
		successStyle.Render(formatBytes(m.stats.TotalBytes*2)),
		m.stats.TotalDuration/60*3,
	)

	return boxStyle.Render(content)
}

func (m Model) renderLogs() string {
	return m.renderRecentActivity(20)
}

func (m Model) renderRecentActivity(limit int) string {
	if len(m.images) == 0 {
		return boxStyle.Render("No recent activity")
	}

	var items []string
	items = append(items, headerStyle.Render(" Recent Activity "))
	items = append(items, "")

	count := limit
	if len(m.images) < count {
		count = len(m.images)
	}

	for i := 0; i < count; i++ {
		img := m.images[i]

		status := successStyle.Render("✅ " + img.Status)
		if img.Status != "completed" {
			status = errorStyle.Render("❌ " + img.Status)
		}

		timeAgo := formatTimeAgo(img.Timestamp)

		line := fmt.Sprintf("%s | %s | %s | %s",
			img.Image,
			status,
			formatBytes(img.Bytes),
			timeAgo,
		)

		items = append(items, line)
	}

	return boxStyle.Render(strings.Join(items, "\n"))
}

func (m Model) renderHelp() string {
	help := []string{
		"[↑/k] Up",
		"[↓/j] Down",
		"[Tab] Next View",
		"[1-4] Jump to View",
		"[r] Refresh",
		"[q] Quit",
	}
	return helpStyle.Render(strings.Join(help, " • "))
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return fmt.Sprintf("%ds ago", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(duration.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(duration.Hours()/24))
}
