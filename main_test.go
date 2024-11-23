package main

import (
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// TestGetRotationTheta tests the getRotationTheta function.
func TestGetRotationTheta(t *testing.T) {
	theta := getRotationTheta(90)
	expected := 1.5707963267948966 // 90 degrees in radians
	if theta != expected {
		t.Errorf("Expected %v, got %v", expected, theta)
	}
}
}
}

// TestHighScore tests the high score functionality.
func TestHighScore(t *testing.T) {
	game := NewGame()

	// Ensure the high score file is removed before testing
	os.Remove("highscore.txt")

	// Test saving a high score
	game.saveHighScore(100)
	highScore := game.loadHighScore()
	if highScore != 100 {
		t.Errorf("Expected high score to be 100, got %d", highScore)
	}

	// Test updating the high score
	game.saveHighScore(200)
	highScore = game.loadHighScore()
	if highScore != 200 {
		t.Errorf("Expected high score to be 200, got %d", highScore)
	}

	// Test not updating if the score is lower
	game.saveHighScore(150)
	highScore = game.loadHighScore()
	if highScore != 200 {
		t.Errorf("Expected high score to remain 200, got %d", highScore)
	}
}

// TestGameOver tests the game over functionality.
func TestGameOver(t *testing.T) {
	game := NewGame()

	// Simulate game over
	game.endGame()
	if !game.gameOver {
		t.Error("Expected gameOver to be true, got false")
	}

	// Ensure high score is saved on game over
	game.score = 300
	game.endGame()
	highScore := game.loadHighScore()
	if highScore != 300 {
		t.Errorf("Expected high score to be 300 after game over, got %d", highScore)
	}

// TestLoadImage tests the LoadImage function.
func TestLoadImage(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("LoadImage panicked with error: %v", r)
		}
	}()
	LoadImage("assets/head.png")
}

// TestNewGame tests the NewGame function.
func TestNewGame(t *testing.T) {
	game := NewGame()
	if game == nil {
		t.Error("Expected new game instance, got nil")
	}
}

// TestGameReset tests the Reset method of Game.
func TestGameReset(t *testing.T) {
	game := NewGame()
	game.Reset()
	if game == nil {
		t.Error("Expected game to be reset, got nil")
	}
}

// TestGameUpdate tests the Update method of Game.
func TestGameUpdate(t *testing.T) {
	game := NewGame()
	err := game.Update()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

// TestGameDraw tests the Draw method of Game.
func TestGameDraw(t *testing.T) {
	game := NewGame()
	screen := ebiten.NewImage(screenWidth, screenHeight)
	game.Draw(screen)
}

// TestGameLayout tests the Layout method of Game.
func TestGameLayout(t *testing.T) {
	game := NewGame()
	width, height := game.Layout(0, 0)
	if width != screenWidth || height != screenHeight {
		t.Errorf("Expected (%d, %d), got (%d, %d)", screenWidth, screenHeight, width, height)
	}
}

// TestGameCanMove tests the canMove method of Game.
func TestGameCanMove(t *testing.T) {
	game := NewGame()
	if !game.canMove(0, 1) {
		t.Error("Expected piece to be able to move down")
	}
}

// TestGameLockPiece tests the lockPiece method of Game.
func TestGameLockPiece(t *testing.T) {
	game := NewGame()
	game.lockPiece()
	if len(game.lockedPieces) != 1 {
		t.Errorf("Expected 1 locked piece, got %d", len(game.lockedPieces))
	}
}

// TestGameSpawnNewPiece tests the spawnNewPiece method of Game.
func TestGameSpawnNewPiece(t *testing.T) {
	game := NewGame()
	game.spawnNewPiece()
	if game.activePiece == nil {
		t.Error("Expected new active piece, got nil")
	}
}
