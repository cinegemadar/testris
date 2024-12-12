package main

import (
	"os"
	"testing"
	"slices"
	"strings"

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

// TestHighScore tests the high score functionality.
func TestHighScore(t *testing.T) {
	game := NewGame()

	// Ensure the high score file is removed before testing
	_ = os.Remove("highscore.txt")

	// Test saving a high score
	game.saveScore(100)
	highScore := game.loadHighScore()
	if highScore != 100 {
		t.Errorf("Expected high score to be 100, got %d", highScore)
	}

	// Test updating the high score
	game.saveScore(200)
	highScore = game.loadHighScore()
	if highScore != 200 {
		t.Errorf("Expected high score to be 200, got %d", highScore)
	}

	// Test not updating if the score is lower
	game.saveScore(250)
	highScore = game.loadHighScore()
	if highScore < 250 {
		t.Errorf("Expected high score to remain 200, got %d", highScore)
	}
}

// TestGameOver tests the game over functionality.
func TestGameOver(t *testing.T) {
	game := NewGame()

	// Simulate game over
	game.endGame()
	if game.gameOver.getState() != StateBlocking {
		t.Errorf("Expected state of gameOver component to be StateBlocking(%d), got %d", StateBlocking, game.gameOver.getState())
	}

	// Ensure high score is saved on game over
	game.score = 300
	game.endGame()
	highScore := game.loadHighScore()
	if highScore != 300 {
		t.Errorf("Expected high score to be 300 after game over, got %d", highScore)
	}
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
	game.score = 342
	game.Reset()
	if game == nil {
		t.Error("Expected game to be reset, got nil")
	}
	if game.score != 0 {
		t.Error("Expected game score reset to 0")
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
	game.sideBar.setValues(game.nextPiece, game.score, game.speedLevelIdx+1, game.loadTopScores())

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
	if !game.grid.canMove(game.activePiece, 0, 1) {
		t.Error("Expected piece to be able to move down")
	}
}

// TestGameLockPiece tests the lockPiece method of Game.
func TestGameLockPiece(t *testing.T) {
	game := NewGame()
	piece := game.activePiece

	game.grid.lockPiece(game.activePiece)
	if len(game.grid.lockedPieces) != 1 {
		t.Errorf("Expected 1 locked piece, got %d", len(game.grid.lockedPieces))
	}
	if game.grid.content[gridSize.w/2][0] != piece {
		t.Errorf("Expected Game.grid refers to the locked piece")
	}
}

// TestGameSpawnNewPiece tests the spawnNewPiece method of Game.
func TestGameSpawnNewPiece(t *testing.T) {
	game := NewGame()
	game.activePiece = game.nextPiece
	game.spawnNewPiece()
	if game.activePiece == nil {
		t.Error("Expected new active piece, got nil")
	}
}

func fillGrid(g *Game, gridRows []string) [][]*Piece {
	headIdx := slices.IndexFunc(allPieces, func(p Piece) bool { return p.pieceType == "Head" })
	torsoIdx := slices.IndexFunc(allPieces, func(p Piece) bool { return p.pieceType == "Torso" })
	legIdx := slices.IndexFunc(allPieces, func(p Piece) bool { return p.pieceType == "Leg" })
	bombIdx := slices.IndexFunc(allPieces, func(p Piece) bool { return p.pieceType == "Bomb" })

	s := allPieces[headIdx].size.w // for simplicity consider all pieces have the same w+h size
	bottom := gridSize.h - 1

	var piecesMatrix [][]*Piece

	piecePos := Pos{1, bottom - len(gridRows) * s}
	for _, rowDesc := range gridRows {
		var piecesRow []*Piece

		piecePos.x = 1
		pieceDescList := strings.Split(rowDesc, " ")
		for _, pieceDesc := range pieceDescList {
			var piece Piece

			if pieceDesc == "" {
				continue
			}
			
			if pieceDesc != "_" {
				piece = Piece{}
				switch t := pieceDesc[1]; t {
					case 'H': piece = allPieces[headIdx]
					case 'T': piece = allPieces[torsoIdx]
					case 'L': piece = allPieces[legIdx]
					case 'B': piece = allPieces[bombIdx]
				}

				switch r := pieceDesc[0]; r {
					case '^':  piece.currentRotation = 0
					case '<': piece.currentRotation = 90
					case 'v': piece.currentRotation = 180
					default: piece.currentRotation = 270
				}

				piece.pos = piecePos

				g.grid.lockPiece(&piece)
			}

			piecesRow = append(piecesRow, &piece)
			piecePos.x += s
		}

		piecesMatrix = append(piecesMatrix, piecesRow)
		piecePos.y += s
	}

	return piecesMatrix
}

// TestGridLockUnlockPieces tests the lockPieces and unlockPieces methods of Grid.
func TestGridLockUnlockPieces(t *testing.T) {
	game := NewGame()

	gridDesc1 := []string {
	// 0   1   2
		"_   _   _",    // 0
		">L  >T  <H", } // 1
	piecesMat1 := fillGrid(game, gridDesc1);

	gridDesc2 := []string {
	// 0   1   2   3   4   5   6
		"_   >H  ^T",   // 0
		"_   _   _", }  // 1
	piecesMat2 := fillGrid(game, gridDesc2);

	if len(game.grid.lockedPieces) != 5 {
		t.Errorf("Expected 5 locked pieces. Got %d instead.", len(game.grid.lockedPieces))
	}

	if game.grid.lockedPieces[0] != piecesMat2[0][1] { t.Errorf("Expected 1st locked piece d %v. Got %v instead.", piecesMat2[0][1], game.grid.lockedPieces[0]) }
	if game.grid.lockedPieces[1] != piecesMat2[0][2] { t.Errorf("Expected 2nd locked piece d %v. Got %v instead.", piecesMat2[0][2], game.grid.lockedPieces[1]) }
	if game.grid.lockedPieces[2] != piecesMat1[1][0] { t.Errorf("Expected 3rd locked piece d %v. Got %v instead.", piecesMat1[0][0], game.grid.lockedPieces[2]) }
	if game.grid.lockedPieces[3] != piecesMat1[1][1] { t.Errorf("Expected 4th locked piece d %v. Got %v instead.", piecesMat1[0][1], game.grid.lockedPieces[3]) }
	if game.grid.lockedPieces[4] != piecesMat1[1][2] { t.Errorf("Expected 5th locked piece d %v. Got %v instead.", piecesMat1[0][2], game.grid.lockedPieces[4]) }

	// unlock (remove) 2 pieces
	p1 := piecesMat2[0][2]
	p2 := piecesMat1[1][1]
	game.grid.unlockPiece(p1)
	game.grid.unlockPiece(p2)

	if len(game.grid.lockedPieces) != 3 {
		t.Errorf("Expected 3 locked pieces. Got %d instead.", len(game.grid.lockedPieces))
	}

	if game.grid.lockedPieces[0] != piecesMat2[0][1] { t.Errorf("Expected 1st locked piece d %v. Got %v instead.", piecesMat2[0][1], game.grid.lockedPieces[0]) }
	if game.grid.lockedPieces[1] != piecesMat1[1][0] { t.Errorf("Expected 2nd locked piece d %v. Got %v instead.", piecesMat1[0][0], game.grid.lockedPieces[1]) }
	if game.grid.lockedPieces[2] != piecesMat1[1][2] { t.Errorf("Expected 3rd locked piece d %v. Got %v instead.", piecesMat1[0][2], game.grid.lockedPieces[2]) }

	// swap positions of two pieces
	p1.pos, p2.pos = p2.pos, p1.pos

	// lock (add) the 2 pieces
	game.grid.lockPiece(p1)
	game.grid.lockPiece(p2)

	if len(game.grid.lockedPieces) != 5 {
		t.Errorf("Expected 5 locked pieces. Got %d instead.", len(game.grid.lockedPieces))
	}

	if game.grid.lockedPieces[0] != piecesMat2[0][1] { t.Errorf("Expected 1st locked piece d %v. Got %v instead.", piecesMat2[0][1], game.grid.lockedPieces[0]) }
	if game.grid.lockedPieces[1] != p2               { t.Errorf("Expected 2nd locked piece d %v. Got %v instead.", p2, game.grid.lockedPieces[1]) }
	if game.grid.lockedPieces[2] != piecesMat1[1][0] { t.Errorf("Expected 3rd locked piece d %v. Got %v instead.", piecesMat1[0][0], game.grid.lockedPieces[2]) }
	if game.grid.lockedPieces[3] != p1               { t.Errorf("Expected 4th locked piece d %v. Got %v instead.", p1, game.grid.lockedPieces[3]) }
	if game.grid.lockedPieces[4] != piecesMat1[1][2] { t.Errorf("Expected 5th locked piece d %v. Got %v instead.", piecesMat1[0][2], game.grid.lockedPieces[4]) }
}

// TestGameJoinAndScorePieces tests the joinAndScorePieces method of Game.
func TestGameJoinAndScorePieces(t *testing.T) {
	game := NewGame()
	fellow := allBodies[ slices.IndexFunc(allBodies, func(b *Body) bool { return b.name == "Fellow" }) ]

	// grid status
	gridDesc := []string {
	// 0   1   2   3   4   5   6
		"_   _   ^T  _   <H",           // 0
		"_   >H  >H  _   vT  <T  <L",   // 1
		">L  >T  <H  <T  <L  ^L  vH", } // 2
	piecesMat := fillGrid(game, gridDesc);

	origScore := game.score
	game.joinAndScorePieces([]*Piece{ piecesMat[2][3] }) // <T in the bottom row

	// 1 body joins and disappears:
	//          ^T      <H
	//      >H  >H      vT  <T  <L
	//  >L  >T              ^L  vH

	// pieces fallen:
	//
	//      >H  ^T      <H  <T  <L
	//  >L  >T  >H      vT  ^L  vH
	
	// 2 more bodies join and disappear:
	//
	//      >H  ^T
	//                  vT  ^L  vH

	// pieces fallen:
	//
	//      >H  ^T      vT  ^L  vH

	if origScore + 3 * fellow.score != game.score {
		t.Errorf("Expected score (%d) is %d", game.score, origScore + 2 * fellow.score)
	}

	if len(game.grid.lockedPieces) != 5 {
		t.Errorf("Expected 5 locked pieces. Got %d instead.", len(game.grid.lockedPieces))
	}

	if game.grid.lockedPieces[0] != piecesMat[1][1] { t.Errorf("Expected 1st locked piece d %v. Got %v instead.", piecesMat[1][1], game.grid.lockedPieces[0]) }
	if game.grid.lockedPieces[1] != piecesMat[0][2] { t.Errorf("Expected 2nd locked piece d %v. Got %v instead.", piecesMat[0][2], game.grid.lockedPieces[1]) }
	if game.grid.lockedPieces[2] != piecesMat[1][4] { t.Errorf("Expected 3rd locked piece d %v. Got %v instead.", piecesMat[1][4], game.grid.lockedPieces[2]) }
	if game.grid.lockedPieces[3] != piecesMat[2][5] { t.Errorf("Expected 4th locked piece d %v. Got %v instead.", piecesMat[2][5], game.grid.lockedPieces[3]) }
	if game.grid.lockedPieces[4] != piecesMat[2][6] { t.Errorf("Expected 5th locked piece d %v. Got %v instead.", piecesMat[2][6], game.grid.lockedPieces[4]) }
}

// TestGameGeneratePiece tests the generatePiece methods of Game.
func TestGameGeneratePiece(t *testing.T) {
	game := NewGame()
	for i := 0; i < 100000; i++ {
		game.generatePiece()
	}

	for a := 0; a < len(allPieces)-1; a++ {
		spawnA := game.spawnStat[allPieces[a].pieceType]
		probA, ok := game.spawnProb[allPieces[a].pieceType]
		if !ok {
			probA = 1.0
		}

		for b := a+1; b < len(allPieces); b++ {
			spawnB := game.spawnStat[allPieces[b].pieceType]
			probB, ok := game.spawnProb[allPieces[b].pieceType]
			if !ok {
				probB = 1.0
			}

			expectedSpawnA := float32(spawnB) * probA / probB

			// allow may 10% error
			if expectedSpawnA < float32(spawnA) * 0.9 || float32(spawnA) * 1.1 < expectedSpawnA {
				t.Errorf("Incorrect nr of generated pieces: '%s'(%f%%):%d '%s'(%f%%):%d", allPieces[a].pieceType, 100*probA, spawnA, allPieces[b].pieceType, 100*probB, spawnB)
			}
//		t.Logf("Verify spawn, pieces idx:(%d,%d), spawn nr:(%d,%d), rel diff:%f", a, b, spawnB, spawnA, expectedSpawnA / float32(spawnA))
		}
	}
}
