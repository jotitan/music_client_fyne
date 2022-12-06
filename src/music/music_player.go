package music

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
)

type Kind string

const SongKind = Kind("song")
const ArtistKind = Kind("artist")
const AlbumKind = Kind("album")

type Music struct {
	Artist string `json:"artist"`
	Album  string `json:"album"`
	Title  string `json:"title"`
	Id     string `json:"id"`
	Path   string `json:"path"`
}

type musicBy struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}

type MusicServerWrapper struct {
	url          string
	artistTokens tokens
	albumTokens  tokens
	artistDico   map[string]string
	albumDico    map[string]string
}

func NewMusicServerWrapper(url string) MusicServerWrapper {
	msw := &MusicServerWrapper{url: url}
	msw.loadArtists()
	msw.loadAlbums()

	return *msw
}

func (nsw *MusicServerWrapper) loadArtists() error {
	var err error
	err, nsw.artistTokens, nsw.artistDico = nsw.loadSome("listByArtist")
	return err
}

func (nsw *MusicServerWrapper) loadAlbums() error {
	var err error
	err, nsw.albumTokens, nsw.albumDico = nsw.loadSome("listByOnlyAlbums")
	return err
}

func (nsw *MusicServerWrapper) loadSome(url string) (error, tokens, map[string]string) {
	resp, err := http.Get(fmt.Sprintf("%s/%s", nsw.url, url))
	if err != nil {
		return err, nil, nil
	}
	if resp.StatusCode != 200 {
		return errors.New("impossible"), nil, nil
	}
	data, _ := io.ReadAll(resp.Body)
	var entities []musicBy
	err = json.Unmarshal(data, &entities)
	if err != nil {
		return err, nil, nil
	}
	return nil, createTokensTable(entities), createDico(entities)
}

type token struct {
	value string
	ids   []string
}

type tokens []token

func (tok tokens) searchPosition(text string) []int {
	pos := tok.searchIn(tok, text, 0)
	// If pos != -1, search before and after other results
	if pos != -1 {
		results := []int{pos}
		for i := pos + 1; i < len(tok) && strings.HasPrefix(tok[i].value, text); i++ {
			results = append(results, i)
		}
		for i := pos - 1; i >= 0 && strings.HasPrefix(tok[i].value, text); i-- {
			results = append(results, i)
		}
		return results
	}
	return []int{}
}

func (tok tokens) searchIn(subs tokens, text string, pos int) int {
	if len(subs) == 0 {
		return -1
	}
	center := len(subs) / 2
	t := subs[center]
	if strings.HasPrefix(t.value, text) {
		return pos + center
	}
	if len(subs) == 1 {
		return -1
	}
	if t.value < text {
		return tok.searchIn(subs[center:], text, center+pos)
	}
	return tok.searchIn(subs[:center], text, pos)
}

func createDico(values []musicBy) map[string]string {
	m := make(map[string]string)
	for _, value := range values {
		m[value.Url] = value.Name
	}
	return m
}

// Return sorted tokens
func createTokensTable(values []musicBy) tokens {
	m := make(map[string][]string)
	for _, value := range values {
		for _, sub := range strings.Split(strings.ReplaceAll(strings.ToLower(value.Name), "-", " "), " ") {
			list, exist := m[sub]
			if !exist {
				list = make([]string, 0)
			}
			m[sub] = append(list, value.Url)
		}
	}
	asList := make([]token, 0, len(m))
	for key, value := range m {
		asList = append(asList, token{value: key, ids: value})
	}
	sort.Slice(asList, func(i, j int) bool {
		return asList[i].value < asList[j].value
	})
	return asList
}

func (nsw *MusicServerWrapper) getIdsFromPositions(positions []int) []string {
	var results []string
	for _, pos := range positions {
		results = append(results, nsw.artistTokens[pos].ids...)
	}
	return results
}

type responseBy struct {
	Title string `json:"name"`
	Id    string `json:"id"`
	Infos struct {
		Album  string `json:"album"`
		Artist string `json:"artist"`
	} `json:"infos"`
}

func (nsw MusicServerWrapper) GetMusicsByAlbum(idArtist string) []*Music {
	return nsw.getMusicsBy(fmt.Sprintf("%s/listByOnlyAlbums?%s", nsw.url, idArtist))
}

func (nsw MusicServerWrapper) GetMusicsByArtist(idArtist string) []*Music {
	return nsw.getMusicsBy(fmt.Sprintf("%s/listByArtist?%s", nsw.url, idArtist))
}

func (nsw MusicServerWrapper) getMusicsBy(url string) []*Music {
	tempMusics := doSearch[responseBy](url)
	musics := make([]*Music, len(tempMusics))
	for i, m := range tempMusics {
		musics[i] = &Music{
			Id:     m.Id,
			Title:  m.Title,
			Artist: m.Infos.Artist,
			Album:  m.Infos.Album,
		}
	}
	return musics
}

func (nsw MusicServerWrapper) SearchArtists(text string) []Music {
	return nsw.searchSome(text, nsw.artistTokens, nsw.artistDico)
}

func (nsw MusicServerWrapper) SearchAlbums(text string) []Music {
	return nsw.searchSome(text, nsw.albumTokens, nsw.albumDico)
}

func (nsw MusicServerWrapper) searchSome(text string, tks tokens, dico map[string]string) []Music {
	var results []string
	for idx, t := range strings.Split(text, " ") {
		positions := tks.searchPosition(t)
		if len(positions) == 0 {
			return []Music{}
		}
		subResults := make([]string, 0)
		for _, pos := range positions {
			subResults = append(subResults, tks[pos].ids...)
		}
		if idx == 0 {
			results = subResults
		} else {
			// Do intersection
			results = intersect(results, subResults)
			if len(results) == 0 {
				return nil
			}
		}
	}
	musics := make([]Music, len(results))
	for i, id := range results {
		some := dico[id]
		musics[i] = Music{some, some, "", id, ""}
	}
	return musics
}

func intersect(a, b []string) []string {
	if len(a) == 0 || len(b) == 0 {
		return []string{}
	}
	sort.Strings(a)
	sort.Strings(b)
	results := make([]string, 0, int(math.Min(float64(len(a)), float64(len(b)))))
	j := 0
	for i := 0; i < len(a); i++ {
		for {
			stop := false
			switch {
			case j >= len(b) || a[i] < b[j]:
				stop = true
			case a[i] > b[j]:
				j++
			case a[i] == b[j]:
				results = append(results, a[i])
				j++
				stop = true
			}
			if stop {
				break
			}
		}
		if j >= len(b) {
			break
		}
	}
	return results
}

func (nsw MusicServerWrapper) HybridSearch(term string) ([]Music, Kind) {
	fmt.Println("Search")
	if strings.HasPrefix(term, ":") {
		// specific case
		if strings.HasPrefix(term, ":artist ") {
			return nsw.SearchArtists(term[8:]), ArtistKind
		}
		if strings.HasPrefix(term, ":album ") {
			return nsw.SearchAlbums(term[7:]), AlbumKind
		}

		if strings.HasPrefix(term, ":album ") {
			return []Music{}, AlbumKind
		}
	}
	return nsw.Search(term), SongKind
}

func (nsw MusicServerWrapper) Search(term string) []Music {
	return doSearch[Music](fmt.Sprintf("%s/search?term=%s&size=30", nsw.url, strings.ReplaceAll(term, " ", "%20")))
}

func (nsw MusicServerWrapper) doSearch(url string) []Music {
	resp, err := http.Get(url)
	if err != nil {
		return []Music{}
	}
	data, _ := io.ReadAll(resp.Body)
	musics := make([]Music, 0)
	err = json.Unmarshal(data, &musics)
	if err != nil {
		return []Music{}
	}
	return musics
}

func doSearch[R Music | responseBy](url string) []R {
	resp, err := http.Get(url)
	if err != nil {
		return []R{}
	}
	data, _ := io.ReadAll(resp.Body)
	musics := make([]R, 0)
	err = json.Unmarshal(data, &musics)
	if err != nil {
		return []R{}
	}
	return musics
}

func (nsw MusicServerWrapper) GetMusics(ids []int) ([]Music, error) {
	strIds := make([]string, len(ids))
	for i, id := range ids {
		strIds[i] = fmt.Sprintf("%d", id)
	}

	resp, err := http.Get(fmt.Sprintf("%s/musicsInfo?ids=[%s]", nsw.url, strings.Join(strIds, ",")))
	if err != nil {
		return nil, err
	}
	data, _ := io.ReadAll(resp.Body)
	musics := make([]Music, 0)
	err = json.Unmarshal(data, &musics)
	return musics, err
}

func (nsw MusicServerWrapper) FindPath(id string) (string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/pathOfMusic?id=%s", nsw.url, id))
	if err == nil && resp.StatusCode == 200 {
		data, err := io.ReadAll(resp.Body)
		return string(data), err
	}
	return "", err
}

type MusicPlayerWrapper struct {
	url string
}

func NewMusicPlayerWrapper(url string) MusicPlayerWrapper {
	return MusicPlayerWrapper{url: url}
}

func (mpw MusicPlayerWrapper) GetState() ([]int, error) {
	resp, err := http.Get(fmt.Sprintf("%s/playlist/state", mpw.url))
	if err == nil && resp.StatusCode == 200 {
		data, _ := io.ReadAll(resp.Body)
		value := struct {
			Ids []int `json:"ids"`
		}{}
		if err := json.Unmarshal(data, &value); err != nil {
			return nil, err
		}
		return value.Ids, nil
	} else {
		return nil, err
	}
}

func (mpw MusicPlayerWrapper) Play(index int) error {
	_, err := http.Get(fmt.Sprintf("%s/music/play?index=%d", mpw.url, index))
	return err
}

func (mpw MusicPlayerWrapper) Add(m Music, path string) error {
	request := []map[string]string{{"id": m.Id,
		"path": path,
	}}
	dataRequest, _ := json.Marshal(request)
	postUrl := fmt.Sprintf("%s/playlist/add", mpw.url)
	_, err := http.Post(postUrl, "application/json", bytes.NewBuffer(dataRequest))
	return err
}

func (mpw MusicPlayerWrapper) AddMany(listMusics []*Music) error {
	request := make([]map[string]string, len(listMusics))
	for i, m := range listMusics {
		request[i] = map[string]string{"id": m.Id,
			"path": m.Path,
		}
	}
	dataRequest, _ := json.Marshal(request)
	postUrl := fmt.Sprintf("%s/playlist/add", mpw.url)
	_, err := http.Post(postUrl, "application/json", bytes.NewBuffer(dataRequest))
	return err
}

func (mpw MusicPlayerWrapper) Delete(index int) error {
	_, err := http.Get(fmt.Sprintf("%s/playlist/remove?index=%d", mpw.url, index))
	return err
}

func (mpw MusicPlayerWrapper) UnPause() error {
	_, err := http.Get(fmt.Sprintf("%s/music/play", mpw.url))
	return err
}

func (mpw MusicPlayerWrapper) Pause() error {
	_, err := http.Get(fmt.Sprintf("%s/music/pause", mpw.url))
	return err
}

func (mpw MusicPlayerWrapper) Next() error {
	_, err := http.Get(fmt.Sprintf("%s/music/next", mpw.url))
	return err
}

func (mpw MusicPlayerWrapper) Previous() error {
	_, err := http.Get(fmt.Sprintf("%s/music/previous", mpw.url))
	return err
}

func (mpw MusicPlayerWrapper) VolumeUp() error {
	_, err := http.Get(fmt.Sprintf("%s/control/volumeUp", mpw.url))
	return err
}

func (mpw MusicPlayerWrapper) VolumeDown() error {
	_, err := http.Get(fmt.Sprintf("%s/control/volumeDown", mpw.url))
	return err
}

func (mpw MusicPlayerWrapper) Current() (int, error) {
	resp, err := http.Get(fmt.Sprintf("%s/playlist/current", mpw.url))
	if err == nil && resp.StatusCode == 200 {
		data, err := io.ReadAll(resp.Body)
		response := struct {
			Current int `json:"current"`
		}{0}
		if err == nil {
			err = json.Unmarshal(data, &response)
			return response.Current, err
		}
		return 0, err
	}
	return 0, err
}

type MusicWrapper struct {
	server MusicServerWrapper
	player MusicPlayerWrapper
}

func NewMusicWrapper(server MusicServerWrapper, player MusicPlayerWrapper) MusicWrapper {
	return MusicWrapper{server, player}
}

func (mw MusicWrapper) GetPlaylist() ([]Music, error) {
	ids, err := mw.player.GetState()
	if err == nil {
		// Reorder musics according to original state
		musics, err := mw.server.GetMusics(ids)
		if err != nil {
			return nil, err
		}
		musicsById := getMusicsAsMap(musics)
		orderedMusics := make([]Music, len(ids))
		for index, id := range ids {
			orderedMusics[index] = musicsById[fmt.Sprintf("%d", id)]
		}
		return orderedMusics, nil
	}
	return nil, err
}

func getMusicsAsMap(musics []Music) map[string]Music {
	m := make(map[string]Music)
	for _, music := range musics {
		m[music.Id] = music
	}
	return m
}

func (mw MusicWrapper) Play(index int) error {
	return mw.player.Play(index)
}

func (mw MusicWrapper) Search(term string) []Music {
	return mw.server.Search(term)
}

func (mw MusicWrapper) HybridSearch(term string) ([]Music, Kind) {
	return mw.server.HybridSearch(term)
}

func (mw MusicWrapper) SearchArtist(term string) ([]Music, error) {
	return mw.server.SearchArtists(term), nil
}

func (mw MusicWrapper) Add(m Music) error {
	// Extract path before
	path, err := mw.server.FindPath(m.Id)
	if err == nil {
		return mw.player.Add(m, path)
	}
	return err
}

func (mw MusicWrapper) addMany(musics []*Music) error {
	waiter := sync.WaitGroup{}
	for _, m := range musics {
		waiter.Add(1)
		go func(mus *Music) {
			path, _ := mw.server.FindPath(mus.Id)
			mus.Path = path
			waiter.Done()
		}(m)
	}
	waiter.Wait()
	return mw.player.AddMany(musics)
}

func (mw MusicWrapper) Delete(index int) error {
	return mw.player.Delete(index)
}

func (mw MusicWrapper) Pause() error {
	return mw.player.Pause()
}

func (mw MusicWrapper) Current() (int, error) {
	return mw.player.Current()
}

func (mw MusicWrapper) UnPause() error {
	return mw.player.UnPause()
}

func (mw MusicWrapper) VolumeUp() error {
	return mw.player.VolumeUp()
}

func (mw MusicWrapper) VolumeDown() error {
	return mw.player.VolumeDown()
}

func (mw MusicWrapper) Previous() error {
	return mw.player.Previous()
}

func (mw MusicWrapper) Next() error {
	return mw.player.Next()
}

func (mw MusicWrapper) AddAllArtist(m Music) error {
	return mw.addMany(mw.server.GetMusicsByArtist(m.Id))
}

func (mw MusicWrapper) AddAllAlbum(m Music) error {
	return mw.addMany(mw.server.GetMusicsByAlbum(m.Id))
}

func (mw MusicWrapper) ShowArtist(m Music) []*Music {
	return mw.server.GetMusicsByArtist(m.Id)
}

func (mw MusicWrapper) ShowAlbum(m Music) []*Music {
	return mw.server.GetMusicsByAlbum(m.Id)
}
