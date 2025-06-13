package ui

import (
	"github.com/vpoluyaktov/tview"
	"github.com/vpoluyaktov/abb_tts/internal/dto"
	"github.com/vpoluyaktov/abb_tts/internal/mq"
	"github.com/vpoluyaktov/abb_tts/internal/parser"
	"github.com/vpoluyaktov/abb_tts/internal/tts"
)

type TUI struct {
	app      *tview.Application
	pages    *tview.Pages
	mq       *mq.Dispatcher
	service  tts.Service
	bookPage *BookPage
}

func NewTUI() *TUI {
	// Set color theme
	setColorTheme()

	t := &TUI{
		app:   tview.NewApplication(),
		pages: tview.NewPages(),
	}

	return t
}

func (t *TUI) SetDispatcher(dispatcher *mq.Dispatcher) {
	t.mq = dispatcher
}



func (t *TUI) SetService(s tts.Service) {
	t.service = s
}

func (t *TUI) initialize() {
	// Create book page with no book loaded initially
	book := (*parser.Book)(nil)

	// Get available voices and providers
	voices := t.service.GetAvailableVoices()
	providers := t.service.GetAvailableProviders()

	// Create and show book page (no parser argument)
	t.bookPage = NewBookPage(t.app, t.pages, book, voices, providers, t.mq, nil)
	t.pages.AddPage("book", t.bookPage.GetGrid(), true, true)

	// Create header
	header := tview.NewTextView()
	header.SetText("Audiobook Builder TTS")
	header.SetTextAlign(tview.AlignCenter)
	header.SetTextColor(headerFgColor)
	header.SetBackgroundColor(headerBGColor)

	// Create footer
	footer := tview.NewTextView()
	footer.SetText("")
	footer.SetTextAlign(tview.AlignLeft)
	footer.SetTextColor(footerFgColor)
	footer.SetBackgroundColor(footerBgColor)

	// Create main grid
	mainGrid := tview.NewGrid()
	mainGrid.SetRows(1, -1, 1)
	mainGrid.SetColumns(0)
	mainGrid.AddItem(header, 0, 0, 1, 1, 0, 0, false)

	// Add pages container
	mainGrid.AddItem(t.pages, 1, 0, 1, 1, 0, 0, true)

	// Add footer
	mainGrid.AddItem(footer, 2, 0, 1, 1, 0, 0, false)

	// Register footer message handler
	t.mq.RegisterHandler(mq.Footer, func(m *mq.Message) {
		switch dto := m.Dto.(type) {
		case *dto.UpdateStatus:
			t.app.QueueUpdateDraw(func() {
				footer.SetText(dto.Message)
			})
		case *dto.SetBusyIndicator:
			t.app.QueueUpdateDraw(func() {
				if dto.Busy {
					footer.SetBackgroundColor(busyIndicatorBgColor)
					footer.SetTextColor(busyIndicatorFgColor)
				} else {
					footer.SetBackgroundColor(footerBgColor)
					footer.SetTextColor(footerFgColor)
				}
			})
		}
	})

	// Set root
	t.app.SetRoot(mainGrid, true)
}

func (t *TUI) Run() error {
	t.initialize()
	return t.app.EnableMouse(true).Run()
}

func (t *TUI) GetPages() *tview.Pages {
	return t.pages
}

func (t *TUI) GetApplication() *tview.Application {
	return t.app
}

func (t *TUI) Draw() {
	t.app.Draw()
}

func (t *TUI) Stop() {
	t.app.Stop()
}
