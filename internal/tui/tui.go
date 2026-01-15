package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"abb_tts/internal/server"
	"abb_tts/internal/storage"
	"abb_tts/internal/tts"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// JobDB defines the database operations needed for TUI
type JobDB interface {
	ListJobs(status string, limit int) ([]*storage.Job, error)
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			Padding(0, 1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))
)

// tickMsg is sent every second to update the display
type tickMsg time.Time

// Model represents the TUI state
type Model struct {
	spinner        spinner.Model
	serverURL      string
	startTime      time.Time
	db             JobDB
	ttsService     tts.Service
	wsHub          *server.Hub
	providersTable table.Model
	jobsTable      table.Model
	width, height  int
	quitting       bool
}

// InitialModel creates a new TUI model
func InitialModel(serverURL string, db JobDB, ttsService tts.Service, wsHub *server.Hub) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))

	// Create providers table
	providerColumns := []table.Column{
		{Title: "#", Width: 3},
		{Title: "Provider", Width: 12},
		{Title: "Type", Width: 8},
		{Title: "Status", Width: 10},
		{Title: "Voices", Width: 8},
	}
	providersTable := table.New(
		table.WithColumns(providerColumns),
		table.WithFocused(false),
		table.WithHeight(6),
	)
	providersTable.SetStyles(defaultTableStyles())

	// Create jobs table
	jobColumns := []table.Column{
		{Title: "#", Width: 3},
		{Title: "Status", Width: 12},
		{Title: "Book Title", Width: 30},
		{Title: "Progress", Width: 12},
		{Title: "Chapter", Width: 10},
		{Title: "Provider", Width: 10},
	}
	jobsTable := table.New(
		table.WithColumns(jobColumns),
		table.WithFocused(true),
		table.WithHeight(8),
	)
	jobsTable.SetStyles(defaultTableStyles())

	m := Model{
		spinner:        s,
		serverURL:      serverURL,
		startTime:      time.Now(),
		db:             db,
		ttsService:     ttsService,
		wsHub:          wsHub,
		providersTable: providersTable,
		jobsTable:      jobsTable,
	}

	m.refreshProviders()
	m.refreshJobs()

	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tickCmd(),
		tea.WindowSize(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "r":
			m.refreshProviders()
			m.refreshJobs()
		}

	case tickMsg:
		m.refreshJobs()
		cmds = append(cmds, tickCmd())

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()
	}

	var cmd tea.Cmd
	m.providersTable, cmd = m.providersTable.Update(msg)
	cmds = append(cmds, cmd)
	m.jobsTable, cmd = m.jobsTable.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.quitting {
		return "Shutting down server...\n"
	}

	width := m.width
	height := m.height
	if width == 0 {
		width = 100
	}
	if height == 0 {
		height = 30
	}

	// Title
	title := titleStyle.Render("🎧  AUDIOBOOK BUILDER TTS SERVER  🎧")

	// Server status line
	uptime := time.Since(m.startTime).Round(time.Second)
	serverStatus := labelStyle.Render("Server: ") + valueStyle.Render(m.serverURL) +
		"  " + m.spinner.View() + " " + successStyle.Render("Running") +
		"  " + labelStyle.Render("Uptime: ") + valueStyle.Render(uptime.String())

	// WebSocket clients count
	clientCount := 0
	if m.wsHub != nil {
		clientCount = m.wsHub.ClientCount()
	}
	serverStatus += "  " + labelStyle.Render("Clients: ") + valueStyle.Render(fmt.Sprintf("%d", clientCount))

	// Calculate layout
	topHeight, bottomHeight := calculateHeights(height)
	serverWidth := int(float64(width) * 0.35)
	if serverWidth < 35 {
		serverWidth = 35
	}
	providersWidth := width - serverWidth - 6
	if providersWidth < 45 {
		providersWidth = 45
	}

	// Server info box
	serverInfoContent := fmt.Sprintf("🖥️  SERVER STATUS\n\n"+
		"URL:     %s\n"+
		"Uptime:  %s\n"+
		"Clients: %d connected\n\n"+
		"📡 API ENDPOINTS\n"+
		"• GET  /api/jobs\n"+
		"• POST /api/upload\n"+
		"• POST /api/preview\n"+
		"• GET  /api/providers\n"+
		"• WS   /api/ws",
		m.serverURL, uptime.String(), clientCount)

	serverInfo := boxStyle.
		Width(serverWidth).
		Height(topHeight).
		Render(serverInfoContent)

	// Providers box
	providersHeader := fmt.Sprintf("🔊 TTS PROVIDERS (%d)", len(m.providersTable.Rows()))
	providersContent := providersHeader + "\n\n" + m.providersTable.View()
	providersBox := boxStyle.
		Width(providersWidth).
		Height(topHeight).
		Render(providersContent)

	// Arrange top row in columns
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, serverInfo, providersBox)

	// Jobs box
	jobCount := len(m.jobsTable.Rows())
	jobsHeader := fmt.Sprintf("📋 ACTIVE JOBS (%d)", jobCount)
	jobsContent := jobsHeader + "\n\n" + m.jobsTable.View()
	jobsBox := boxStyle.
		Width(width - 4).
		Height(bottomHeight).
		Render(jobsContent)

	// Help text
	helpText := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("Press 'r' to refresh • 'q' or 'Ctrl+C' to quit")
	helpBox := boxStyle.
		Width(width - 4).
		Height(3).
		Render(helpText)

	// Combine all sections
	output := fmt.Sprintf("%s\n%s\n%s\n%s\n%s",
		title,
		serverStatus,
		topRow,
		jobsBox,
		helpBox,
	)

	return output
}

func (m *Model) refreshProviders() {
	if m.ttsService == nil {
		return
	}

	providers := m.ttsService.GetAvailableProviders()
	allVoices := m.ttsService.GetAvailableVoices()

	// Count voices per provider
	voicesByProvider := make(map[string]int)
	for _, v := range allVoices {
		voicesByProvider[v.Provider]++
	}

	rows := make([]table.Row, 0, len(providers))

	for i, p := range providers {
		providerType := "Local"
		if p == "google" || p == "azure" {
			providerType = "Cloud"
		}

		voiceCount := voicesByProvider[p]
		status := "✅ OK"
		voiceCountStr := fmt.Sprintf("%d", voiceCount)

		if voiceCount == 0 {
			status = "⚠️ No voices"
			voiceCountStr = "-"
		}

		rows = append(rows, table.Row{
			fmt.Sprintf("%d", i+1),
			p,
			providerType,
			status,
			voiceCountStr,
		})
	}

	m.providersTable.SetRows(rows)
}

func (m *Model) refreshJobs() {
	if m.db == nil {
		return
	}

	dbJobs, err := m.db.ListJobs("", 0)
	if err != nil {
		return
	}

	// Convert to slice for sorting
	jobs := dbJobs

	// Sort by created time (newest first)
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].CreatedAt.After(jobs[j].CreatedAt)
	})

	rows := make([]table.Row, 0, len(jobs)*2) // Extra space for worker rows
	for i, job := range jobs {
		// Use conversion progress, or build progress if conversion is done
		displayProgress := job.ConversionProgress
		if job.ConversionProgress >= 1.0 && job.BuildProgress > 0 {
			displayProgress = job.BuildProgress
		}
		progress := fmt.Sprintf("%d%%", int(displayProgress*100))
		progressBar := renderProgressBar(displayProgress, 8)

		chapter := "-"
		if job.TotalChapters > 0 {
			chapter = fmt.Sprintf("%d/%d", job.CurrentChapterNum, job.TotalChapters)
		}

		title := job.BookTitle
		if title == "" {
			title = job.FileName
		}
		if len(title) > 28 {
			title = title[:25] + "..."
		}

		statusIcon := getStatusIconFromString(job.Status)

		rows = append(rows, table.Row{
			fmt.Sprintf("%d", i+1),
			statusIcon + " " + job.Status,
			title,
			progressBar + " " + progress,
			chapter,
			job.Provider,
		})

		// Add worker progress rows for converting and building jobs
		if (job.Status == "converting" || job.Status == "building") && len(job.WorkerProgress) > 0 {
			for _, wp := range job.WorkerProgress {
				workerStatus := "idle"
				workerProgress := ""
				if wp.Active {
					if job.Status == "converting" {
						workerStatus = fmt.Sprintf("Ch.%d", wp.ChapterIndex+1)
					} else {
						workerStatus = fmt.Sprintf("Part %d", wp.ChapterIndex+1)
					}
					workerProgress = renderProgressBar(wp.Progress, 6) + fmt.Sprintf(" %d/%d", wp.ChunksComplete, wp.ChunksTotal)
				}
				rows = append(rows, table.Row{
					"",
					fmt.Sprintf("  └─ W%d", wp.WorkerID+1),
					workerStatus,
					workerProgress,
					"",
					"",
				})
			}
		}
	}

	m.jobsTable.SetRows(rows)
}

func renderProgressBar(progress float64, width int) string {
	filled := int(progress * float64(width))
	empty := width - filled
	if filled > width {
		filled = width
	}
	if empty < 0 {
		empty = 0
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", empty)
}

func getStatusIconFromString(status string) string {
	switch status {
	case "pending":
		return "⏳"
	case "parsing":
		return "📖"
	case "converting":
		return "🔄"
	case "building":
		return "📦"
	case "uploading":
		return "☁️"
	case "completed":
		return "✅"
	case "failed":
		return "❌"
	case "cancelled":
		return "🚫"
	default:
		return "❓"
	}
}

func defaultTableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	return s
}

func calculateHeights(terminalHeight int) (topHeight, bottomHeight int) {
	titleLines := 1
	serverStatusLines := 1
	newlines := 4
	helpBoxHeight := 3
	reservedLines := titleLines + serverStatusLines + newlines + helpBoxHeight + 5

	availableHeight := terminalHeight - reservedLines
	if availableHeight < 10 {
		availableHeight = 10
	}

	topHeight = availableHeight / 2
	bottomHeight = availableHeight - topHeight
	return topHeight, bottomHeight
}

func (m *Model) updateLayout() {
	if m.height == 0 {
		return
	}
	topHeight, bottomHeight := calculateHeights(m.height)

	topTableHeight := topHeight - 4
	if topTableHeight < 3 {
		topTableHeight = 3
	}
	bottomTableHeight := bottomHeight - 4
	if bottomTableHeight < 3 {
		bottomTableHeight = 3
	}

	m.providersTable.SetHeight(topTableHeight)
	m.jobsTable.SetHeight(bottomTableHeight)
}

// RunTUI starts the TUI application
func RunTUI(serverURL string, db JobDB, ttsService tts.Service, wsHub *server.Hub) error {
	p := tea.NewProgram(
		InitialModel(serverURL, db, ttsService, wsHub),
		tea.WithAltScreen(),
	)
	_, err := p.Run()
	return err
}
