package queue

import (
	"errors"
	"fmt"
	"sync"
)

var ErrPlaylistNotFound = errors.New("playlist not found")
var ErrVideoNotFound = errors.New("video not found")
var ErrVideoAlreadyInPlaylist = errors.New("video already in playlist")

type Video struct {
	AddedBy string `json:"addedBy"`
	VideoID string `json:"videoID"`
}

type PlaylistHub struct {
	playlists map[string][]Video
	mu        *sync.RWMutex
}

func NewPlaylistHub() PlaylistHub {
	return PlaylistHub{
		playlists: map[string][]Video{},
		mu:        &sync.RWMutex{},
	}
}

func (hub *PlaylistHub) AddVideo(username string, video Video) error {
	defer hub.mu.Unlock()
	hub.mu.Lock()

	playlist, ok := hub.playlists[username]

	if !ok {
		playlist = []Video{}
		hub.playlists[username] = playlist
	}

	for _, val := range playlist {
		if val.VideoID == video.VideoID {
			return ErrVideoAlreadyInPlaylist
		}
	}

	hub.playlists[username] = append(playlist, video)

	fmt.Println(username)
	fmt.Println(hub.playlists[username])
	return nil
}

func (hub *PlaylistHub) RemoveVideo(username string, videoID string) error {
	defer hub.mu.Unlock()
	hub.mu.Lock()

	playlist, ok := hub.playlists[username]

	if !ok {
		return ErrPlaylistNotFound
	}

	position := -1

	for i, val := range playlist {
		if val.VideoID == videoID {
			position = i
			break
		}
	}

	if position == -1 {
		return ErrVideoNotFound
	}

	hub.playlists[username] = append(playlist[:position], playlist[position+1:]...)

	return nil
}

func (hub *PlaylistHub) GetPlaylist(username string) []Video {

	defer hub.mu.RUnlock()
	hub.mu.RLock()

	playlist, ok := hub.playlists[username]

	if !ok {
		return []Video{}
	}

	return playlist

}
