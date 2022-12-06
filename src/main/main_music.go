package main

import (
	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"github.com/jotitan/fyne_poc/src/panel"
	"os"
)

func main() {
	application := app.New()
	win := application.NewWindow("Music player")

	win.Resize(fyne.Size{800, 600})
	mp := panel.NewMusicPanel(os.Args[1], os.Args[2], application)

	mp.CreateMainPanel(win)

	win.ShowAndRun()
}
