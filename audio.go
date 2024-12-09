package main

import (
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
)

type Audio struct {
	// Theme music asset file
	themeMusicAssetFile string
	// player
	player *audio.Player
	// music file
	musicFile *os.File
}

// createMusicPlayer initializes the audio context and creates a music player
// for the theme music. It opens the theme music file, decodes it as an MP3
// stream, and creates a new player for the audio context. If any error occurs
// during these steps, the function logs the error and terminates the program.
func (a *Audio) createMusicPlayer() {
	audioContext := audio.NewContext(44100)
	var err error
	a.musicFile, err = os.Open(a.themeMusicAssetFile)
	if err != nil {
		log.Fatal(err)
	}
	themeMusicStream, err := mp3.DecodeWithSampleRate(audioContext.SampleRate(), a.musicFile)
	if err != nil {
		log.Fatal(err)
	}
	a.player, err = audioContext.NewPlayer(themeMusicStream)
	if err != nil {
		log.Fatal(err)
	}
}

// NewAudio creates a new Audio instance with the provided theme music asset file.
// It initializes the music player for the audio.
//
// Parameters:
//   - themeMusicAssetFile: The file path to the theme music asset.
//
// Returns:
//   - *Audio: A pointer to the newly created Audio instance.
func NewAudio(themeMusicAssetFile string) *Audio {
	a := &Audio{
		themeMusicAssetFile: themeMusicAssetFile,
	}
	a.createMusicPlayer()
	return a
}

// getPlayer returns the audio player associated with the Audio instance.
// It provides access to the underlying *audio.Player.
func (a *Audio) getPlayer() *audio.Player {
	return a.player
}

// Play starts the playback of the audio. It first rewinds the audio to the
// beginning and then plays it from the start.
func (a *Audio) Play() {
	a.getPlayer().Rewind()
	a.getPlayer().Play()
}

// Pause pauses the audio playback by calling the Pause method on the underlying player.
func (a *Audio) Pause() {
	a.getPlayer().Pause()
}
