package ui

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/vpoluyaktov/tview"
	"biblio-audiobook-builder-tts/internal/dto"
	"biblio-audiobook-builder-tts/internal/mq"
	parser "biblio-audiobook-builder-tts/internal/parser"
	"biblio-audiobook-builder-tts/internal/tts"
)

type BookPage struct {
	app          *tview.Application
	pages        *tview.Pages
	grid         *tview.Grid
	chapterList  *tview.List
	book         *parser.Book
	providers    []string
	mq           *mq.Dispatcher
	fileButton   *tview.Button
	providerDropDown *tview.DropDown
	settingsButton  *tview.Button
	selectedProvider string
	bookInfoView *tview.TextView
}

var (
	controlsBgColor = tcell.ColorDarkBlue
	controlsFgColor = tcell.ColorWhite
)

func NewBookPage(app *tview.Application, pages *tview.Pages, book *parser.Book, voices []tts.Voice, providers []string, dispatcher *mq.Dispatcher, _ interface{}) *BookPage {
	p := &BookPage{
		app:       app,
		pages:     pages,
		book:      nil,
		providers: providers,
		mq:        dispatcher,

	}

	// Register message handler
	p.mq.RegisterHandler(mq.BookPage, p.dispatchMessage)

	// Create main grid
	p.grid = tview.NewGrid()
	p.grid.SetRows(3, 3, -1)
	p.grid.SetColumns(0, 20)

	// Provider selector at the top left
	p.providerDropDown = tview.NewDropDown()
	p.providerDropDown.SetLabel("Ebook Provider: ")
	providerOptions := []string{"local file", "Project Gutenberg", "OPDS"}
	p.providerDropDown.SetOptions(providerOptions, func(option string, index int) {
		p.selectedProvider = option
		// Optionally: trigger provider-specific logic here
	})
	p.providerDropDown.SetCurrentOption(0)

	// Settings button at the top right
	p.settingsButton = tview.NewButton("Settings")
	p.settingsButton.SetBackgroundColor(controlsBgColor)
	p.settingsButton.SetLabelColor(controlsFgColor)
	p.settingsButton.SetSelectedFunc(p.onSettingsButtonPressed)

	// Ebook selection section frame
selectionSection := tview.NewGrid()
selectionSection.SetBorder(true)
selectionSection.SetTitle(" Ebook Selection ")
selectionSection.SetTitleAlign(tview.AlignLeft)
selectionSection.SetRows(1, 1)
selectionSection.SetColumns(0, 20)

// Top bar: provider selector left, settings button right
 topBar := tview.NewGrid()
topBar.SetColumns(0, 20)
topBar.SetRows(1)
topBar.AddItem(p.providerDropDown, 0, 0, 1, 1, 0, 0, false)
topBar.AddItem(p.settingsButton, 0, 1, 1, 1, 0, 0, false)
selectionSection.AddItem(topBar, 0, 0, 1, 2, 0, 0, false)

// File selection button (centered and small)
p.fileButton = tview.NewButton("Select Ebook...")
p.fileButton.SetBackgroundColor(controlsBgColor)
p.fileButton.SetLabelColor(controlsFgColor)
p.fileButton.SetSelectedFunc(func() {
	ShowFileDialog(p.app, p.pages, ".", func(path string) {
		cmd := &dto.ParseBookCommand{FilePath: path}
		p.mq.SendMessage(mq.BookPage, mq.BookController, cmd, mq.PriorityNormal)
	}, nil)
})
fileButtonGrid := tview.NewGrid()
fileButtonGrid.SetRows(1)
fileButtonGrid.SetColumns(0, 20, 0)
fileButtonGrid.AddItem(tview.NewBox(), 0, 0, 1, 1, 0, 0, false)
fileButtonGrid.AddItem(p.fileButton, 0, 1, 1, 1, 0, 0, true)
fileButtonGrid.AddItem(tview.NewBox(), 0, 2, 1, 1, 0, 0, false)
selectionSection.AddItem(fileButtonGrid, 1, 0, 1, 2, 0, 0, false)

// Add the selection section to the main grid
p.grid.AddItem(selectionSection, 0, 0, 2, 2, 0, 0, false)

// Book Information section (between ebook selection and chapters)
bookInfo := tview.NewTextView()
bookInfo.SetDynamicColors(true)
bookInfo.SetWrap(true)
bookInfoBox := tview.NewGrid()
bookInfoBox.SetBorder(true)
bookInfoBox.SetTitle(" Book Information ")
bookInfoBox.SetTitleAlign(tview.AlignLeft)
bookInfoBox.AddItem(bookInfo, 0, 0, 1, 1, 0, 0, false)
p.grid.AddItem(bookInfoBox, 2, 0, 1, 2, 0, 0, false)
p.bookInfoView = bookInfo

// Chapters section (only shown if a book is loaded)
p.chapterList = tview.NewList()
p.chapterList.ShowSecondaryText(false)
p.chapterList.SetMainTextColor(valuesColor)
p.chapterList.SetSelectedTextColor(black)
p.chapterList.SetSelectedBackgroundColor(yellow)
chaptersSection := tview.NewGrid()
chaptersSection.SetBorder(true)
chaptersSection.SetTitle(" Chapters ")
chaptersSection.SetTitleAlign(tview.AlignLeft)
chaptersSection.AddItem(p.chapterList, 0, 0, 1, 1, 0, 0, true)
p.grid.AddItem(chaptersSection, 3, 0, 1, 2, 0, 0, true)

// Initialize Book Information section
if p.book != nil {
	p.updateBookInfo()
} else {
	bookInfo.SetText("")
}

	return p
}

func (p *BookPage) GetGrid() *tview.Grid {
	return p.grid
}

func (p *BookPage) Draw() {
	p.app.Draw()
}

func (p *BookPage) loadBook(path string) {

}

func (p *BookPage) showError(title string, err error) {
	ShowDialog(p.app, p.pages, title, err.Error(), "OK", nil)
}

func (p *BookPage) updateChapterList() {
	p.chapterList.Clear()
	if p.book == nil {
		return
	}
	for _, chapter := range p.book.Chapters {
		p.chapterList.AddItem(chapter.Title, "", 0, nil)
	}
}

// Update Book Information section
func (p *BookPage) updateBookInfo() {
	if p.book == nil || p.bookInfoView == nil {
		p.bookInfoView.SetText("")
		return
	}
	author := p.book.Author
	title := p.book.Title
	series := ""
	desc := ""
	if p.book.Metadata != nil {
		if s, ok := p.book.Metadata["series"]; ok {
			series = s
		}
		if d, ok := p.book.Metadata["annotation"]; ok {
			desc = d
		} else if d, ok := p.book.Metadata["description"]; ok {
			desc = d
		}
	}
	info := "[white]"
	if title != "" {
		info += "Title: [yellow]" + title + "\n"
	}
	if author != "" {
		info += "Author: [yellow]" + author + "\n"
	}
	if series != "" {
		info += "Series: [yellow]" + series + "\n"
	}
	if desc != "" {
		info += "Description: [gray]" + desc + "\n"
	}
	p.bookInfoView.SetText(info)
}

func (p *BookPage) Show() {
	p.app.SetRoot(p.grid, true)
	// Set focus to the provider dropdown by default
	p.app.SetFocus(p.providerDropDown)
}

func (p *BookPage) Hide() {
	p.app.SetRoot(nil, false)
}

func (p *BookPage) dispatchMessage(m *mq.Message) {
	switch dto := m.Dto.(type) {
	case *dto.BookParsedResult:
		p.app.QueueUpdateDraw(func() {
			if dto.Error != "" {
				ShowErrorDialog(p.app, p.pages, "Book Load Error", dto.Error)
				return
			}
			p.book = dto.Book
			p.updateBookInfo()
			p.updateChapterList()
		})
	case *dto.ConversionComplete:
		p.app.QueueUpdateDraw(func() {
			ShowInfoDialog(p.app, p.pages, "Conversion Complete", "Audio files have been saved to: "+dto.OutputPath)
		})
	case *dto.ConversionError:
		p.app.QueueUpdateDraw(func() {
			ShowErrorDialog(p.app, p.pages, "Conversion Error", "Failed to convert chapter "+dto.Chapter+": "+dto.Error)
		})
	case *dto.ConversionProgress:
		p.app.QueueUpdateDraw(func() {
			p.chapterList.SetTitle(fmt.Sprintf(" Chapters - Converting: %.0f%% ", dto.Progress*100))
		})
	}
}

func (p *BookPage) onSettingsButtonPressed() {
	p.mq.SendMessage(mq.BookPage, mq.ConfigPage, &dto.DisplayConfigCommand{}, mq.PriorityHigh)
	p.mq.SendMessage(mq.BookPage, mq.Frame, &dto.SwitchToPageCommand{Name: "ConfigPage"}, mq.PriorityNormal)
}

func (p *BookPage) onConvertButtonPressed() {
	if p.book == nil {
		return
	}
	// Only trigger conversion for the loaded book
	cmd := &dto.ConvertCommand{
		Book: p.book,
		Provider: p.selectedProvider,
	}
	p.mq.SendMessage(mq.BookPage, mq.TTSController, cmd, mq.PriorityNormal)
}
