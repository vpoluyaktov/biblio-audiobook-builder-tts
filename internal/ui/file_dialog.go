package ui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/vpoluyaktov/tview"
)

type FileDialog struct {
	app       *tview.Application
	pages     *tview.Pages
	grid      *tview.Grid
	list      *tview.List
	path      string
	onSelect  func(path string)
	onCancel  func()
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

	// Create file list
	d.list = tview.NewList()
	d.list.ShowSecondaryText(false)
	d.list.SetMainTextColor(valuesColor)
	d.list.SetSelectedTextColor(black)
	d.list.SetSelectedBackgroundColor(yellow)

	// Add navigation buttons
	buttonBar := tview.NewFlex()
	buttonBar.SetDirection(tview.FlexColumn)

	selectButton := tview.NewButton("Select")
	selectButton.SetBackgroundColor(footerBgColor)
	selectButton.SetLabelColor(footerFgColor)
	selectButton.SetSelectedFunc(func() {
		if item := d.list.GetCurrentItem(); item >= 0 {
			text, _ := d.list.GetItemText(item)
			if text == ".." {
				d.path = filepath.Dir(d.path)
			} else {
				path := filepath.Join(d.path, text)
				if info, err := os.Stat(path); err == nil && !info.IsDir() {
					d.pages.RemovePage("file_dialog")
					d.onSelect(path)
					return
				}
				d.path = path
			}
			d.updateList()
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

	// Layout
	d.grid.SetRows(-1, 1)
	d.grid.SetColumns(0)
	d.grid.AddItem(d.list, 0, 0, 1, 1, 0, 0, true)
	d.grid.AddItem(buttonBar, 1, 0, 1, 1, 0, 0, false)

	// Update list
	d.updateList()

	return d
}

func (d *FileDialog) updateList() {
	d.list.Clear()

	// Add parent directory
	d.list.AddItem("..", "", 0, nil)

	// Read directory contents
	entries, err := os.ReadDir(d.path)
	if err != nil {
		return
	}

	// Add directories first
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if entry.IsDir() {
			d.list.AddItem(name, "", 0, nil)
		}
	}

	// Add files
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if !entry.IsDir() && isEBook(name) {
			d.list.AddItem(name, "", 0, nil)
		}
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
