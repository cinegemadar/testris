package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const highScoreFileName = "highscore.txt"

const (
	screenWidth  = 800
	screenHeight = 600
	sidebarWidth = 140
	speed        = 10
	gridSize     = 30
	scale        = 16 // Unified scale factor for cells and sprites
)

var (
	borderColor      = color.RGBA{R: 70, G: 255, B: 255, A: 255}
	boundingBoxColor = color.RGBA{R: 255, G: 255, B: 0, A: 255}
	sidebarColor     = color.RGBA{R: 130, G: 130, B: 130, A: 255}
	backgroundColor  = color.RGBA{R: 0, G: 0, B: 0, A: 255}
)

type Piece struct {
	image           *ebiten.Image // Single image for the piece
	currentRotation int           // Current rotation in degrees (0, 90, 180, 270)
	width, height   int           // Dimensions of the piece
	pieceType       string        // Head, Torso, Leg
	x, y            int           // Position of the piece
	highScore       int
	dropKeyPressed  bool
}

/*
handleKeyRelease centralizes the handling of key releases to trigger actions.

Parameters:
- key: The ebiten key to check.
- pressed: A pointer to a boolean indicating if the key was previously pressed.
- action: The action to perform if the key is released.
*/
func (g *Game) handleKeyRelease(key ebiten.Key, pressed *bool, action func()) {
	if !ebiten.IsKeyPressed(key) {
		if *pressed {
			action()
		}
		*pressed = false
	} else {
		*pressed = true
	}
}

/*
dropPiece moves the active piece as far down as possible.
*/
func (g *Game) dropPiece() {
	for g.canMove(0, 1) {
		g.pieceY++
	}
	g.lockPiece()
	g.spawnNewPiece()
}

func isWithinBounds(x, y, width, height, minX, maxX, minY, maxY int) bool {
	return x+width <= maxX && x >= minX && y+height <= maxY && y >= minY
}

func isColliding(newX, newY, width, height int, piece *Piece) bool {
	return newX < piece.x+width && newX+width > piece.x && newY < piece.y+height && newY+height > piece.y
}

func (g *Game) movePiece(direction int, pressed *bool, key ebiten.Key) {
	g.handleKeyPress(key, pressed, func() {
		if g.canMove(direction, 0) {
			g.pieceX += direction
		}
	})
}

func mustLoadImage(path string) *ebiten.Image {
	img, err := LoadImage(path)
	if err != nil {
		log.Fatal(err)
	}
	return img
}

/*
loadTopScores loads and returns the top 5 scores from the highscore.txt file.
*/
func readScoresFromFile() []int {
	data, err := os.ReadFile(highScoreFileName)
	if err != nil {
		if os.IsNotExist(err) {
			return []int{}
		}
		log.Printf("Failed to read high scores: %v", err)
		return []int{}
	}

	scoreStrings := strings.Split(string(data), "\n")
	var scores []int
	for _, scoreStr := range scoreStrings {
		if scoreStr == "" {
			continue
		}
		score, err := strconv.Atoi(scoreStr)
		if err == nil {
			scores = append(scores, score)
		}
	}
	return scores
}

func (g *Game) loadTopScores() []int {
	scores := readScoresFromFile()
	sort.Sort(sort.Reverse(sort.IntSlice(scores)))
	if len(scores) > 5 {
		scores = scores[:5]
	}
	return scores
}

/*
saveScore appends the current score to the highscore.txt file.
*/
func (g *Game) saveScore(score int) {
	file, err := os.OpenFile(highScoreFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open high score file: %v", err)
		return
	}
	defer file.Close()

	if _, err := file.WriteString(fmt.Sprintf("%d\n", score)); err != nil {
		log.Printf("Failed to write score: %v", err)
	}
}

/*
endGame handles the end of the game, saving the score and checking for a new high score.
*/
func (g *Game) endGame() {
	g.gameOver = true
	// Save the current score to the highscore file
	g.saveScore(g.score)

	if g.score >= g.loadHighScore() {
		ebitenutil.DebugPrintAt(ebiten.NewImage(screenWidth, screenHeight), "New High Score!", screenWidth/2-50, screenHeight/2+40)
		log.Println("New high score achieved!")
	}
}

/*
loadHighScore loads the high score from a file.
*/
func (g *Game) loadHighScore() int {
	scores := readScoresFromFile()
	var highScore int
	for _, score := range scores {
		if score > highScore {
			highScore = score
		}
	}
	return highScore
}

type Game struct {
	grid                [gridSize][gridSize]*ebiten.Image // Store image references for each grid cell
	lockedPieces        []*Piece                          // Array to store locked pieces
	activePiece         *Piece
	nextPiece           *Piece
	pieceX, pieceY      int // Position of the active piece
	score               int
	frameCount          int
	gameOver            bool
	rotateKeyPressed    bool
	moveLeftKeyPressed  bool
	moveRightKeyPressed bool
	dropKeyPressed      bool
}

/*
getRotationTheta converts degrees to radians.

Parameters:
- deg: The angle in degrees to be converted.

Returns:
- The angle in radians.
*/
func getRotationTheta(deg int) float64 {
	return float64(deg) * (math.Pi / 180)
}

/*
Reset reinitializes the game state to start a new game.
*/
func (g *Game) Reset() {
	*g = *NewGame()
}

/*
LoadImage loads an image from the specified file path.

Parameters:
- path: The file path to the image.

Returns:
- A pointer to the loaded ebiten.Image.
*/
func LoadImage(path string) (*ebiten.Image, error) {
	img, _, err := ebitenutil.NewImageFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load image from path %s: %w", path, err)
	}
	return img, nil
}

var allPieces []*Piece

func init() {
	allPieces = []*Piece{
		{image: mustLoadImage("assets/head.png"), currentRotation: 0, width: 3, height: 3, pieceType: "Head"},
		{image: mustLoadImage("assets/torso.png"), currentRotation: 0, width: 3, height: 3, pieceType: "Torso"},
		{image: mustLoadImage("assets/leg.png"), currentRotation: 0, width: 3, height: 3, pieceType: "Leg"},
		{image: mustLoadImage("assets/bomb.png"), currentRotation: 0, width: 3, height: 3, pieceType: "Bomb"},
	}
}

/*
NewGame creates and returns a new Game instance with initialized pieces
and game state.
*/
func NewGame() *Game {
	return &Game{
		grid:        [gridSize][gridSize]*ebiten.Image{},
		activePiece: allPieces[rand.Intn(len(allPieces))],
		nextPiece:   allPieces[rand.Intn(len(allPieces))],
		pieceX:      gridSize / 2,
		pieceY:      0,
	}
}

/*
Update handles the game logic for each frame, including user input,
piece movement, and game state updates.

Returns:
- An error if the update fails, otherwise nil.
*/
func (g *Game) Update() error {
	g.frameCount++

	g.restart() // Always check for restart

	if !g.gameOver {
		g.movePieceInDirection(-1, ebiten.KeyArrowLeft, &g.moveLeftKeyPressed)
		g.movePieceInDirection(1, ebiten.KeyArrowRight, &g.moveRightKeyPressed)
		g.rotate()
		g.drop()
		g.handleKeyRelease(ebiten.KeyEnter, &g.dropKeyPressed, g.dropPiece)
	}

	return nil
}

/*
Restart the game if the restart button is clicked on the sidebar.
*/
func (g *Game) restart() {
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		sidebarX := screenWidth - sidebarWidth
		if isWithinBounds(x, y, 0, 0, sidebarX+10, sidebarX+110, 160, 180) {
			g.Reset()
		}
	}
}

/*
drop moves the active piece down the grid at a regular interval,
locking it in place if it cannot move further.
*/
func (g *Game) drop() {
	if g.frameCount%speed == 0 {
		if !g.canMove(0, 1) {
			g.lockPiece()
			g.spawnNewPiece()
		} else {
			g.pieceY++
		}
	}
}

/*
handleKeyPress centralizes the handling of key presses to reduce redundancy.

Parameters:
- key: The ebiten key to check.
- pressed: A pointer to a boolean indicating if the key was previously pressed.
- action: The action to perform if the key is pressed.
*/
func (g *Game) handleKeyPress(key ebiten.Key, pressed *bool, action func()) {
	if ebiten.IsKeyPressed(key) {
		if !*pressed {
			action()
		}
		*pressed = true
	} else {
		*pressed = false
	}
}

/*
rotate handles the rotation of the active piece when the space key is pressed.
*/
func (g *Game) rotate() {
	g.handleKeyPress(ebiten.KeySpace, &g.rotateKeyPressed, func() {
		g.activePiece.currentRotation = (g.activePiece.currentRotation + 90) % 360
	})
}

/*
movePieceInDirection moves the active piece one cell in the specified direction
when the corresponding arrow key is pressed.

Parameters:
- direction: The direction to move the piece (-1 for left, 1 for right).
- key: The ebiten key to check for the direction.
- pressed: A pointer to a boolean indicating if the key was previously pressed.
*/
func (g *Game) movePieceInDirection(direction int, key ebiten.Key, pressed *bool) {
	g.handleKeyPress(key, pressed, func() {
		if g.canMove(direction, 0) {
			g.pieceX += direction
		}
	})
}

/*
Draw renders the current game state to the screen, including the active piece,
locked pieces, and sidebar information.

Parameters:
- screen: The ebiten.Image to draw the game state onto.
*/
func (g *Game) Draw(screen *ebiten.Image) {
	// Fill only the game area with the background color
	vector.DrawFilledRect(screen, 0, 0, float32(screenWidth-sidebarWidth), float32(screenHeight), backgroundColor, false)

	if g.gameOver {
		if g.score >= g.loadHighScore() {
			ebitenutil.DebugPrintAt(screen, "New High Score!", screenWidth/2-50, screenHeight/2)
		} else {
			ebitenutil.DebugPrintAt(screen, "GAME OVER", screenWidth/2-50, screenHeight/2)
		}
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Score: %d", g.score), screenWidth/2-50, screenHeight/2+20)
	}

	sidebarX := screenWidth - sidebarWidth
	vector.DrawFilledRect(screen, float32(sidebarX), 0, sidebarWidth, screenHeight, sidebarColor, false)
	drawBorder(screen)
	g.drawLockedPieces(screen)
	g.drawBoundingBox(screen)

	op := &ebiten.DrawImageOptions{}
	g.applyRotationToPiece(op, g.activePiece, g.pieceX, g.pieceY)
	screen.DrawImage(g.activePiece.image, op)
	g.drawSidebar(screen)
}

/*
drawSidebar renders the sidebar, including the next piece, restart button,
and score.

Parameters:
- screen: The ebiten.Image to draw the sidebar onto.
*/
func (g *Game) drawSidebar(screen *ebiten.Image) {
	sidebarX := screenWidth - sidebarWidth

	// Draw "Next Piece"
	ebitenutil.DebugPrintAt(screen, "NEXT PIECE", sidebarX+10, 20)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale) // Apply scaling to the next piece
	op.GeoM.Translate(float64(sidebarX+40), 50)
	screen.DrawImage(g.nextPiece.image, op)

	// Draw restart button
	ebitenutil.DebugPrintAt(screen, "RESTART", sidebarX+10, 160)

	// Draw top 5 scores
	ebitenutil.DebugPrintAt(screen, "TOP 5 SCORES", sidebarX+10, 200)
	topScores := g.loadTopScores()
	for i, score := range topScores {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d: %d", i+1, score), sidebarX+10, 220+i*20)
	}

	// Draw current score
	ebitenutil.DebugPrintAt(screen, "SCORE", sidebarX+10, 120)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d", g.score), sidebarX+10, 140)
}

/*
drawLockedPieces renders all locked pieces on the grid.

Parameters:
- screen: The ebiten.Image to draw the locked pieces onto.
*/
func (g *Game) drawLockedPieces(screen *ebiten.Image) {
	for _, lp := range g.lockedPieces {
		op := &ebiten.DrawImageOptions{}

		// Calculate the top-left corner of the locked piece in screen coordinates.

		g.applyRotationToPiece(op, lp, lp.x, lp.y)
		screen.DrawImage(lp.image, op)
	}
}

/*
applyRotationToPiece applies the current rotation to a piece and prepares it
for drawing on the screen.

Parameters:
- op: The ebiten.DrawImageOptions to apply transformations.
- piece: The Piece to apply the rotation to.
- posX: The x position of the piece on the grid.
- posY: The y position of the piece on the grid.
*/
func (g *Game) applyRotationToPiece(op *ebiten.DrawImageOptions, piece *Piece, posX, posY int) {
	op.GeoM.Scale(scale, scale)

	// Center the rotation point (relative to the piece).
	centerX := float64((posX * scale) + (piece.width*scale)/2)
	centerY := float64((posY * scale) + (piece.height*scale)/2)

	// Translate to the center of the piece.
	op.GeoM.Translate(-float64(piece.width*scale)/2, -float64(piece.height*scale)/2)

	// Rotate around the center.
	op.GeoM.Rotate(getRotationTheta(piece.currentRotation))

	// Translate the piece back to its grid position.
	op.GeoM.Translate(centerX, centerY)
}

/*
drawBoundingBox draws a bounding box around the active piece for visual
reference.

Parameters:
- screen: The ebiten.Image to draw the bounding box onto.
*/
func (g *Game) drawBoundingBox(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, float32(g.pieceX*scale), float32(g.pieceY*scale), float32(g.activePiece.width*scale), 1, boundingBoxColor, false)
	vector.DrawFilledRect(screen, float32(g.pieceX*scale), float32((g.pieceY+g.activePiece.height)*scale), float32(g.activePiece.width*scale), 1, boundingBoxColor, false)
	vector.DrawFilledRect(screen, float32(g.pieceX*scale), float32(g.pieceY*scale), 1, float32(g.activePiece.height*scale), boundingBoxColor, false)
	vector.DrawFilledRect(screen, float32((g.pieceX+g.activePiece.width)*scale), float32(g.pieceY*scale), 1, float32(g.activePiece.height*scale), boundingBoxColor, false)
}

/*
drawBorder draws a border around the game area.

Parameters:
- screen: The ebiten.Image to draw the border onto.
*/
func drawBorder(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, 0, 0, float32(gridSize*scale), float32(scale), borderColor, false)
	vector.DrawFilledRect(screen, 0, float32(gridSize*scale)-float32(scale), float32(gridSize*scale), float32(scale), borderColor, false)
	vector.DrawFilledRect(screen, 0, 0, float32(scale), float32(gridSize*scale), borderColor, false)
	vector.DrawFilledRect(screen, float32(gridSize*scale)-float32(scale), 0, float32(scale), float32(gridSize*scale), borderColor, false)
}

/*
Layout returns the logical screen dimensions for the game.

Parameters:
- outsideWidth: The width of the outside environment.
- outsideHeight: The height of the outside environment.

Returns:
- The width and height of the game screen.
*/
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

/*
canMove checks if the active piece can move to a new position on the grid.

Parameters:
- dx: The change in the x-direction.
- dy: The change in the y-direction.

Returns:
- True if the piece can move to the new position, otherwise false.
*/
func (g *Game) canMove(dx, dy int) bool {
	newX := g.pieceX + dx
	newY := g.pieceY + dy

	if !isWithinBounds(newX, newY, g.activePiece.width, g.activePiece.height, 1, gridSize-1, 1, gridSize-1) {
		return false
	}

	for _, piece := range g.lockedPieces {
		if isColliding(newX, newY, g.activePiece.width, g.activePiece.height, piece) {
			return false
		}
	}

	return true
}

/*
lockPiece locks the active piece in its current position on the grid,
adding it to the list of locked pieces.
*/
func (g *Game) lockPiece() {
	lockedPiece := *g.activePiece
	lockedPiece.x = g.pieceX
	lockedPiece.y = g.pieceY
	g.lockedPieces = append(g.lockedPieces, &lockedPiece)
}

/*
spawnNewPiece selects a new active piece from the available pieces and
positions it at the top of the grid.
*/
func (g *Game) spawnNewPiece() {
	if g.pieceY == 0 && !g.canMove(0, 1) {
		g.endGame()
		return
	}

	g.activePiece = g.nextPiece
	g.nextPiece = allPieces[rand.Intn(len(allPieces))]
	g.pieceX = gridSize / 2
	g.pieceY = 0
}

/*
main initializes the game window and starts the game loop.
*/
func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("TESTRis - Fixed Piece Spawning and Locking")

	game := NewGame()
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
