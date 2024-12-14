package main

import (
	"log"
	"time"
	"os"
	"path"
	"io"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

var globalAudioContext *audio.Context

type Audio struct {
	// Theme music asset file
	themeMusicAssetFile string
	loopedPlay bool
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
	if globalAudioContext == nil {
		globalAudioContext = audio.NewContext(44100)
	}

	var err error
	a.musicFile, err = os.Open(a.themeMusicAssetFile)
	if err != nil {
		log.Fatal(err)
	}
	
	fileExt := path.Ext(a.themeMusicAssetFile)

	var audioStream io.ReadSeeker
	audioStreamLength := int64(0)
	if fileExt == ".mp3" {
		mp3AudioStream, err2 := mp3.DecodeWithSampleRate(globalAudioContext.SampleRate(), a.musicFile)
		audioStreamLength = mp3AudioStream.Length()
		audioStream = mp3AudioStream
		err = err2
	} else if fileExt == ".wav" {
		wavAudioStream, err2 := wav.DecodeWithSampleRate(globalAudioContext.SampleRate(), a.musicFile)
		audioStreamLength = wavAudioStream.Length()
		audioStream = wavAudioStream
		err = err2
	} else {
		log.Fatalf("Unknown auido file extension '%s'", a.themeMusicAssetFile)
	}

	if err != nil {
		log.Fatal(err)
	}

	audioLengthSec := (int)(audioStreamLength) / (globalAudioContext.SampleRate() * 2 * 2) // 16 bits stereo
	log.Printf("Stream '%s' (length %d s) prepared (infinite=%t)", a.themeMusicAssetFile, audioLengthSec, a.loopedPlay)

	if a.loopedPlay {
		loopedStream := audio.NewInfiniteLoop(audioStream, audioStreamLength)
		a.player, err = globalAudioContext.NewPlayer(loopedStream)
	} else  {
		a.player, err = globalAudioContext.NewPlayer(audioStream)
	}

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
func NewAudio(themeMusicAssetFile string, loopedPlay bool) *Audio {
	a := &Audio{
		themeMusicAssetFile: themeMusicAssetFile,
		loopedPlay: loopedPlay,
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

func (a *Audio) SeekPlay(offset time.Duration) {
	a.getPlayer().Rewind()
	a.getPlayer().Seek(offset)
	a.getPlayer().Play()
}
