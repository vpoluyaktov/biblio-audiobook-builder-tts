package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/vpoluyaktov/tview"
)

type dialogWindow struct {
	app        *tview.Application
	grid       *tview.Grid
	form       *tview.Form
	shadow     *tview.Grid
	background *tview.Grid
	height     int
	width      int
	focus      tview.Primitive // set focus after the form close
}

func newDialogWindow(app *tview.Application, height int, width int, focus tview.Primitive) *dialogWindow {
	d := &dialogWindow{}
	d.app = app
	d.height = height
	d.width = width
	d.focus = focus

	// shadow background
	d.shadow = tview.NewGrid()
	d.shadow.SetRows(2, -1, d.height, -1)
	d.shadow.SetColumns(4, -1, d.width, -1)
	d.shadow.AddItem(tview.NewBox().SetBackgroundColor(black), 2, 2, 1, 1, 0, 0, false)

	// gray background
	d.background = tview.NewGrid()
	d.background.SetRows(-1, d.height, -1)
	d.background.SetColumns(-1, d.width, -1)
	d.background.AddItem(tview.NewBox().SetBackgroundColor(gray), 1, 1, 1, 1, 0, 0, false)

	// transparent background
	d.grid = tview.NewGrid()
	d.grid.SetRows(-1, d.height, -1)
	d.grid.SetColumns(-1, d.width, -1)

	d.grid.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		return action, event
	})

	d.grid.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			{
				d.Close()
			}
		}
		return event
	})

	return d
}

func (d *dialogWindow) Show() {
	pages := tview.NewPages()
	pages.AddPage("Shadow", d.shadow, true, false)
	pages.AddPage("Background", d.background, true, false)
	pages.AddPage("DialogWindow", d.grid, true, true)
	d.app.SetRoot(pages, true)
	d.app.SetFocus(d.form)
	d.app.Draw()
}

func (d *dialogWindow) Close() {
	if d.focus != nil {
		d.app.SetRoot(d.focus, true)
		d.app.SetFocus(d.focus)
	}
	d.app.Draw()
}

func (d *dialogWindow) setForm(f *tview.Form) {
	d.form = f
	d.setFormAttributes()
	d.grid.AddItem(d.form, 1, 1, 1, 1, 0, 0, true)
}

func (d *dialogWindow) setFormAttributes() {
	d.form.SetBorderColor(black)
	d.form.SetTitleColor(blue)
	d.form.SetLabelColor(blue)
	d.form.SetFieldTextColor(black)
	d.form.SetFieldBackgroundColor(cyan)
	d.form.SetButtonTextColor(white)
	d.form.SetButtonBackgroundColor(blue)
	d.form.SetFieldTextColor(black)
	d.form.SetBackgroundColor(gray)
	d.form.SetBorder(true)
	d.form.SetHorizontal(false)
	d.form.SetButtonsAlign(tview.AlignCenter)
}

type OkFunc func()

func newMessageDialog(app *tview.Application, title string, message string, focus tview.Primitive, okFunc OkFunc) {
	d := newDialogWindow(app, 12, 80, focus)
	f := tview.NewForm()
	f.SetTitle(title)
	tv := tview.NewTextView()
	tv.SetWrap(true)
	tv.SetWordWrap(true)
	tv.SetDynamicColors(true)
	tv.SetText(message)
	tv.SetTextAlign(tview.AlignCenter)
	f.AddFormItem(tv)
	f.AddButton("Ok", func() {
		okFunc()
		d.Close()
	})
	d.setForm(f)
	d.Show()
}

type YesNoFunc func()

func newYesNoDialog(app *tview.Application, title string, message string, focus tview.Primitive, yesFunc YesNoFunc, noFunc YesNoFunc) {
	d := newDialogWindow(app, 11, 60, focus)
	f := tview.NewForm()
	f.SetTitle(title)
	tv := tview.NewTextView()
	tv.SetWrap(true)
	tv.SetWordWrap(true)
	tv.SetDynamicColors(true)
	tv.SetText(message)
	tv.SetTextAlign(tview.AlignCenter)
	f.AddFormItem(tv)
	f.AddButton("Yes", func() {
		yesFunc()
		d.Close()
	})
	f.AddButton("No", func() {
		noFunc()
		d.Close()
	})
	d.setForm(f)
	d.Show()
}

func ShowDialog(app *tview.Application, pages *tview.Pages, title string, text string, buttonLabel string, callback func()) {
	modal := tview.NewModal()
	modal.SetText(text)
	modal.AddButtons([]string{buttonLabel})
	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		pages.RemovePage("dialog")
		if callback != nil {
			callback()
		}
	})
	modal.SetTitle(title)
	modal.SetBorder(true)
	pages.AddPage("dialog", modal, true, true)
}

func ShowInfoDialog(app *tview.Application, pages *tview.Pages, title string, message string) {
	newMessageDialog(app, title, message, pages, func() {})
}

func ShowErrorDialog(app *tview.Application, pages *tview.Pages, title string, message string) {
	newMessageDialog(app, "❌ "+title, message, pages, func() {})
}
