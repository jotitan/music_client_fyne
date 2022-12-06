package panel

import (
	"errors"
	"fmt"
	"fyne.io/fyne"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/container"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"
	"github.com/jotitan/fyne_poc/src/music"
	"sync"
	"time"
)

type MusicPanel struct {
	musicWrapper music.MusicWrapper
	updateChanel chan struct{}
	searchPanel  fyne.Window
}

func NewMusicPanel(urlServer, urlPlayer string, app fyne.App) MusicPanel {
	server := music.NewMusicServerWrapper(urlServer)
	player := music.NewMusicPlayerWrapper(urlPlayer)
	mp := MusicPanel{
		musicWrapper: music.NewMusicWrapper(server, player),
		updateChanel: make(chan struct{}, 10),
	}
	mp.searchPanel = mp.createSearchMusic(app)
	return mp
}

func (mp MusicPanel) CreateMainPanel(win fyne.Window) {
	mp.musicWrapper.SearchArtist("goldm")

	musics, _ := mp.musicWrapper.GetPlaylist()
	list := widget.NewList(
		func() int {
			return len(musics)
		},
		func() fyne.CanvasObject {
			del := createIcon(theme.DeleteIcon())
			play := createIcon(theme.MediaPlayIcon())

			return container.NewHBox(
				widget.NewLabel("template"),
				layout.NewSpacer(),
				container.NewPadded(widget.NewButton("", func() {}), play),
				container.NewPadded(widget.NewButton("", func() {}), del))
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i >= len(musics) {
				return
			}
			o.(*fyne.Container).Objects[0].(*widget.Label).SetText(fmt.Sprintf("%d - %s - %s", i+1, musics[i].Title, musics[i].Artist))
			o.(*fyne.Container).Objects[2].(*fyne.Container).Objects[0].(*widget.Button).OnTapped = func() {
				mp.musicWrapper.Play(i)
			}
			o.(*fyne.Container).Objects[3].(*fyne.Container).Objects[0].(*widget.Button).OnTapped = func() {
				if err := mp.musicWrapper.Delete(i + 1); err != nil {
					fmt.Println("ERROR", err)
				} else {
					mp.updateChanel <- struct{}{}
				}
			}
		})
	go func() {
		// Update current position
		timer := time.NewTicker(10 * time.Second)
		for {
			<-timer.C
			if pos, err := mp.musicWrapper.Current(); err == nil && pos < list.Length() {
				list.Select(pos)
			}
		}
	}()

	button := widget.NewButton("Ajouter", func() {
		mp.searchPanel.Show()
	})
	widget.NewToolbarAction(theme.MediaPlayIcon(), func() {})

	toolbar := mp.createMusicToolbar()
	border := layout.NewBorderLayout(toolbar, button, nil, nil)
	panel := fyne.NewContainerWithLayout(border, toolbar, list, button)

	go func() {
		for {
			<-mp.updateChanel
			musics, _ = mp.musicWrapper.GetPlaylist()
			list.Refresh()
		}
	}()

	win.SetContent(panel)
	panel.Show()
}

func (mp MusicPanel) createMusicToolbar() *widget.Toolbar {

	pause := widget.NewToolbarAction(theme.MediaPlayIcon(), func() { mp.musicWrapper.UnPause() })
	play := widget.NewToolbarAction(theme.MediaPauseIcon(), func() { mp.musicWrapper.Pause() })
	previous := widget.NewToolbarAction(theme.MediaSkipPreviousIcon(), func() { mp.musicWrapper.Previous() })
	next := widget.NewToolbarAction(theme.MediaSkipNextIcon(), func() { mp.musicWrapper.Next() })
	vup := widget.NewToolbarAction(theme.VolumeUpIcon(), func() { mp.musicWrapper.VolumeUp() })
	vdown := widget.NewToolbarAction(theme.VolumeDownIcon(), func() { mp.musicWrapper.VolumeDown() })
	toolbar := widget.NewToolbar(
		pause,
		play,
		widget.NewToolbarSeparator(),
		previous,
		next,
		widget.NewToolbarSeparator(),
		vdown,
		vup,
	)

	return toolbar
}

func (mp MusicPanel) createSearchMusic(application fyne.App) fyne.Window {
	locker := sync.Mutex{}
	win := application.NewWindow("Ajouter musique")

	chanArtist := make(chan music.Music, 1)
	results := make([]music.Music, 0)
	var kindSearch music.Kind

	list := widget.NewList(
		func() int {
			return len(results)
		},
		func() fyne.CanvasObject {
			switch kindSearch {
			case music.SongKind:
				return createSongLine()
			case music.ArtistKind, music.AlbumKind:
				return createArtistLine()
			default:
				return createSongLine()
			}
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i >= len(results) {
				return
			}
			switch kindSearch {
			case music.SongKind:
				showSongLine(o, results[i], mp)
			case music.ArtistKind, music.AlbumKind:
				showArtistLine(o, results[i], chanArtist, mp, kindSearch)
			}
		})

	input := widget.NewEntry()
	input.PlaceHolder = "Rechercher..."

	timeWaiter := time.NewTimer(2000)
	timeWaiter.Stop()

	updateMusics := func() {
		locker.Lock()
		results, kindSearch = updateSearchResults(mp.musicWrapper, input.Text)
		list.Refresh()
		locker.Unlock()
	}

	updateMusicsByArtists := func(id music.Music) {
		locker.Lock()
		results, kindSearch = updateResults(mp.musicWrapper, id, kindSearch)
		list.Refresh()
		locker.Unlock()
	}

	// Detect search to launch, wait 300ms before launch to avoid many request
	go func() {
		for {
			<-timeWaiter.C
			updateMusics()
		}
	}()

	go func() {
		for {
			updateMusicsByArtists(<-chanArtist)
		}
	}()

	input.OnChanged = func(value string) {
		if len(value) >= 3 {
			timeWaiter.Reset(time.Millisecond * 300)
		}
	}

	border := layout.NewBorderLayout(input, nil, nil, nil)

	win.SetContent(fyne.NewContainerWithLayout(border, input, list))
	win.Resize(fyne.Size{600, 600})
	win.Hide()
	return win
}

func showArtistLine(o fyne.CanvasObject, line music.Music, c chan music.Music, mp MusicPanel, kind music.Kind) {
	fields := o.(*fyne.Container).Objects
	fields[0].(*fyne.Container).Objects[0].(*widget.Label).SetText(line.Artist)
	fields[0].(*fyne.Container).Objects[1].(*widget.Label).SetText("")
	fields[2].(*widget.Button).SetText("Show")
	fields[2].(*widget.Button).OnTapped = func() {
		c <- line
	}
	fields[3].(*widget.Button).SetText("Add all")
	fields[3].(*widget.Button).Show()
	fields[3].(*widget.Button).OnTapped = func() {
		err := errors.New("no kind")
		switch kind {
		case music.ArtistKind:
			err = mp.musicWrapper.AddAllArtist(line)
		case music.AlbumKind:
			err = mp.musicWrapper.AddAllAlbum(line)
		}
		if err != nil {
			fmt.Println("ERROR", err)
		} else {
			mp.updateChanel <- struct{}{}
		}
	}
}

func showSongLine(o fyne.CanvasObject, line music.Music, mp MusicPanel) {
	fields := o.(*fyne.Container).Objects
	fields[0].(*fyne.Container).Objects[0].(*widget.Label).SetText(line.Title)
	fields[0].(*fyne.Container).Objects[1].(*widget.Label).SetText(fmt.Sprintf("%s (%s)", line.Artist, line.Album))
	fields[2].(*widget.Button).SetText("Add")
	fields[2].(*widget.Button).OnTapped = func() {
		// Add
		if err := mp.musicWrapper.Add(line); err != nil {
			fmt.Println("ERROR", err)
		} else {
			mp.updateChanel <- struct{}{}
		}
	}
	fields[3].(*widget.Button).Hide()
}

func createArtistLine() fyne.CanvasObject {
	title := widget.NewLabel("artist")
	title.TextStyle = fyne.TextStyle{Bold: true}
	return container.NewHBox(
		container.NewVBox(
			title,
			widget.NewLabel(""),
		),
		layout.NewSpacer(),
		widget.NewButton("Show", func() {}),
		widget.NewButton("Add all", func() {}))
}

func createSongLine() fyne.CanvasObject {
	title := widget.NewLabel("artist")
	title.TextStyle = fyne.TextStyle{Bold: true}
	return container.NewHBox(
		container.NewVBox(
			title,
			widget.NewLabel("artist,album"),
		),
		layout.NewSpacer(),
		widget.NewButton("Add", func() {}),
		widget.NewButton("", func() {}))

}

func updateSearchResults(musicWrapper music.MusicWrapper, value string) ([]music.Music, music.Kind) {
	musics, kind := musicWrapper.HybridSearch(value)
	results := make([]music.Music, len(musics))
	for i, m := range musics {
		results[i] = m
	}
	return results, kind
}

func updateResults(musicWrapper music.MusicWrapper, m music.Music, kind music.Kind) ([]music.Music, music.Kind) {
	var musics []*music.Music
	switch kind {
	case music.ArtistKind:
		musics = musicWrapper.ShowArtist(m)
	case music.AlbumKind:
		musics = musicWrapper.ShowAlbum(m)
	}
	results := make([]music.Music, len(musics))
	for i, m := range musics {
		results[i] = *m
	}
	return results, music.SongKind
}

func createIcon(res fyne.Resource) *canvas.Image {
	img := canvas.NewImageFromResource(res)
	img.FillMode = canvas.ImageFillOriginal

	return img
}
