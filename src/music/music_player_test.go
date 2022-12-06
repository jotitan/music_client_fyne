package music

import (
	"fmt"
	"testing"
)

func TestTest(t *testing.T) {
	//player := NewMusicPlayerWrapper("ip_player")
	//server := NewMusicServerWrapper("ip_server")
	server := NewMusicServerWrapper("url with both")
	//fmt.Println(NewMusicWrapper(server, player).GetPlaylist())

	artists, _ := server.HybridSearch(":artist jean gold")
	fmt.Println(artists)
	//server.SearchArtists("jean gold")
}
