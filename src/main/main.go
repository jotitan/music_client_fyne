package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/container"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/widget"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func main() {
	application := app.New()
	win := application.NewWindow("Play with fyne")

	win.Resize(fyne.Size{800, 600})
	win.SetContent(createHomeView(win))

	win.ShowAndRun()
}

func createHomeView(win fyne.Window) *fyne.Container {
	label := widget.NewLabel("Welcome home")
	button := widget.NewButton("Manual progresser", func() {
		displayContainer(win, createFirstPage())
	})

	buttonProgresser := widget.NewButton("Créer progresser", func() {
		displayContainer(win, createProgresserSelection(win))
	})

	buttonFile := widget.NewButton("Voir fichier", func() {
		createViewFile(win)
	})

	buttonFileApi := widget.NewButton("Voir fichier web", func() {
		loadFileFromUrl(win)
	})

	buttonImages := widget.NewButton("Images web", func() {
		showImages(win)
	})

	return container.NewCenter(container.NewVBox(label, button, buttonProgresser, buttonFile, buttonFileApi, buttonImages))
}

func displayContainer(win fyne.Window, c fyne.CanvasObject) {
	win.SetContent(c)
	c.Show()
}

func showImages(win fyne.Window) {
	c := container.NewAdaptiveGrid(5)
	data, _ := getData()

	for i := 0; i < 72; i++ {
		img := canvas.NewImageFromResource(fyne.NewStaticResource("img.jpg", data))
		//img.Resize(fyne.NewSize(40, 40))
		c.Add(img)
	}

	displayContainer(win, container.NewScroll(c))
}

func loadFileFromUrl(win fyne.Window) {
	img := loadImageFromUrl()

	if img != nil {
		button := widget.NewButton("", func() {
			fmt.Println("SUPER CA TAPE")
		})
		c := container.NewPadded(button, img)
		win.SetContent(c)
		c.Show()
	} else {
		fmt.Println("Impossible to get image")
	}
}

func getData() ([]byte, error) {
	// Force mode insecure
	url := "blabla"
	req, _ := http.NewRequest("GET", url, nil)
	req.AddCookie(&http.Cookie{
		Name:  "token_name",
		Value: "token_value",
	})
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := http.Client{Transport: tr}
	r, err := client.Do(req)
	if err == nil && r.StatusCode == 200 {
		data, _ := io.ReadAll(r.Body)
		return data, nil
	}
	data, _ := io.ReadAll(r.Body)
	fmt.Println(err, r.StatusCode, string(data))

	return []byte{}, errors.New("impossible")
}

func loadImageFromUrl() *canvas.Image {
	data, err := getData()
	if err == nil {
		return canvas.NewImageFromResource(fyne.NewStaticResource("img.jpg", data))
	}
	return nil
}

func createViewFile(win fyne.Window) {
	dialog.ShowFileOpen(func(closer fyne.URIReadCloser, err error) {
		if err == nil {
			fmt.Println("LOAD LOCAL", closer.URI().Name())
			path := strings.ReplaceAll(closer.URI().Name(), "file://", "")
			img := canvas.NewImageFromFile(path)
			img.FillMode = canvas.ImageFillContain
			displayContainer(win, img)
		}
	}, win)
}

func createProgresserSelection(win fyne.Window) *fyne.Container {

	e := widget.NewEntry()
	e.PlaceHolder = "Durée"

	submit := widget.NewButton("Créer progresser", func() {
		timeInMillis, _ := strconv.Atoi(e.Text)
		displayContainer(win, createAutoProgress(timeInMillis, win))
	})

	return container.NewVBox(e, submit)
}

func createAutoProgress(timeInMillis int, win fyne.Window) *fyne.Container {
	progresser := widget.NewProgressBar()
	progresser.Min = 0
	progresser.Max = 100
	progresser.Resize(fyne.NewSize(200, 50))
	go func() {
		for {
			if progresser.Value >= 100 {
				displayContainer(win, createHomeView(win))
				return
			}
			progresser.SetValue(progresser.Value + 1)
			time.Sleep(time.Duration(timeInMillis/100) * time.Millisecond)
		}
	}()
	return container.NewCenter(progresser)
}

func createFirstPage() *fyne.Container {

	progresser := widget.NewProgressBar()
	progresser.Min = 0
	progresser.Max = 100

	button := widget.NewButton("Boutton Add", func() {
		progresser.SetValue(progresser.Value + 1)
	})

	button2 := widget.NewButton("Boutton Remove", func() {
		progresser.SetValue(progresser.Value - 1)
	})

	label := widget.NewLabel("Mon label")

	vContainer := container.NewVBox(button, button2, progresser)
	return container.NewCenter(container.NewHBox(label, vContainer))
}
