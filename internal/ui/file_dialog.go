package ui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/vpoluyaktov/tview"
)

type FileDialog struct {
	app          *tview.Application
	pages        *tview.Pages
	grid         *tview.Grid
	dirList      *tview.List
	fileList     *tview.List
	pathText     *tview.TextView
	path         string
	onSelect     func(path string)
	onCancel     func()
}

func NewFileDialog(app *tview.Application, pages *tview.Pages, path string, onSelect func(path string), onCancel func()) *FileDialog {
	d := &FileDialog{
		app:      app,
		pages:    pages,
		path:     path,
		onSelect: onSelect,
		onCancel: onCancel,
	}

	// Create main grid
	d.grid = tview.NewGrid()
	d.grid.SetBorder(true)
	d.grid.SetTitle(" Select eBook File ")
	d.grid.SetTitleAlign(tview.AlignLeft)

	// Create path display
	d.pathText = tview.NewTextView()
	d.pathText.SetDynamicColors(true)
	d.pathText.SetText("[yellow]Current directory: " + d.path)

	// Create directory list
	d.dirList = tview.NewList()
	d.dirList.ShowSecondaryText(false)
	d.dirList.SetMainTextColor(valuesColor)
	d.dirList.SetSelectedTextColor(black)
	d.dirList.SetSelectedBackgroundColor(yellow)
	d.dirList.SetTitle(" Directories ")
	d.dirList.SetBorder(true)

	// Create file list
	d.fileList = tview.NewList()
	d.fileList.ShowSecondaryText(false)
	d.fileList.SetMainTextColor(valuesColor)
	d.fileList.SetSelectedTextColor(black)
	d.fileList.SetSelectedBackgroundColor(yellow)
	d.fileList.SetTitle(" eBook Files ")
	d.fileList.SetBorder(true)

	// Add navigation buttons
	buttonBar := tview.NewFlex()
	buttonBar.SetDirection(tview.FlexColumn)

	selectButton := tview.NewButton("Select")
	selectButton.SetBackgroundColor(footerBgColor)
	selectButton.SetLabelColor(footerFgColor)
	selectButton.SetSelectedFunc(func() {
		// Check if a directory is selected
		if d.app.GetFocus() == d.dirList && d.dirList.GetItemCount() > 0 {
			if item := d.dirList.GetCurrentItem(); item >= 0 {
				text, _ := d.dirList.GetItemText(item)
				if text == ".." {
					d.path = filepath.Dir(d.path)
				} else {
					d.path = filepath.Join(d.path, text)
				}
				d.updateList()
			}
		} else if d.app.GetFocus() == d.fileList && d.fileList.GetItemCount() > 0 {
			// Check if a file is selected
			if item := d.fileList.GetCurrentItem(); item >= 0 {
				text, _ := d.fileList.GetItemText(item)
				path := filepath.Join(d.path, text)
				d.pages.RemovePage("file_dialog")
				d.onSelect(path)
			}
		}
	})

	cancelButton := tview.NewButton("Cancel")
	cancelButton.SetBackgroundColor(footerBgColor)
	cancelButton.SetLabelColor(footerFgColor)
	cancelButton.SetSelectedFunc(func() {
		d.pages.RemovePage("file_dialog")
		if d.onCancel != nil {
			d.onCancel()
		}
	})

	buttonBar.AddItem(nil, 0, 1, false)
	buttonBar.AddItem(selectButton, 8, 0, true)
	buttonBar.AddItem(nil, 1, 0, false)
	buttonBar.AddItem(cancelButton, 8, 0, true)
	buttonBar.AddItem(nil, 0, 1, false)

	// Set up directory navigation on selection
	d.dirList.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		if mainText == ".." {
			d.path = filepath.Dir(d.path)
		} else {
			d.path = filepath.Join(d.path, mainText)
		}
		d.updateList()
	})

	// Set up file selection
	d.fileList.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		path := filepath.Join(d.path, mainText)
		d.pages.RemovePage("file_dialog")
		d.onSelect(path)
	})

	// Create content layout
	content := tview.NewFlex()
	content.SetDirection(tview.FlexRow)
	content.AddItem(d.pathText, 1, 0, false)
	
	// Create lists layout
	lists := tview.NewFlex()
	lists.SetDirection(tview.FlexColumn)
	lists.AddItem(d.dirList, 0, 1, true)  // Start with focus on directories
	lists.AddItem(d.fileList, 0, 1, false)
	
	content.AddItem(lists, 0, 1, true)

	// Layout
	d.grid.SetRows(-1, 1)
	d.grid.SetColumns(0)
	d.grid.AddItem(content, 0, 0, 1, 1, 0, 0, true)
	d.grid.AddItem(buttonBar, 1, 0, 1, 1, 0, 0, false)

	// Update list
	d.updateList()

	return d
}

func (d *FileDialog) updateList() {
	// Update path display
	d.pathText.SetText("[yellow]Current directory: " + d.path)

	// Clear both lists
	d.dirList.Clear()
	d.fileList.Clear()

	// Add parent directory to directory list
	d.dirList.AddItem("..", "", 0, nil)

	// Read directory contents
	entries, err := os.ReadDir(d.path)
	if err != nil {
		return
	}

	// Add directories to directory list
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if entry.IsDir() {
			d.dirList.AddItem(name, "", 0, nil)
		}
	}

	// Add ebook files to file list
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if !entry.IsDir() && isEBook(name) {
			d.fileList.AddItem(name, "", 0, nil)
		}
	}

	// Set focus to directory list if it has items, otherwise to file list
	if d.dirList.GetItemCount() > 0 {
		d.app.SetFocus(d.dirList)
	} else if d.fileList.GetItemCount() > 0 {
		d.app.SetFocus(d.fileList)
	}
}

func isEBook(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".epub" || ext == ".fb2"
}

func (d *FileDialog) GetGrid() *tview.Grid {
	return d.grid
}

func ShowFileDialog(app *tview.Application, pages *tview.Pages, path string, onSelect func(path string), onCancel func()) {
	dialog := NewFileDialog(app, pages, path, onSelect, onCancel)
	pages.AddPage("file_dialog", dialog.GetGrid(), true, true)
}
