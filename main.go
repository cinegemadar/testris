package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"os"
	"slices"
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
	scale        = 16 // Unified scale factor for cells and sprites
)

var (
	gridSize         = Size{30, 30}
	borderColor      = color.RGBA{R: 70, G: 255, B: 255, A: 255}
	boundingBoxColor = color.RGBA{R: 255, G: 255, B: 0, A: 255}
	sidebarColor     = color.RGBA{R: 130, G: 130, B: 130, A: 255}
	backgroundColor  = color.RGBA{R: 0, G: 0, B: 0, A: 255}
)

type Piece struct {
	image           *ebiten.Image // Single image for the piece
	currentRotation int           // Current rotation in degrees (0, 90, 180, 270)
	size            Size          // Dimensions of the piece
	pieceType       string        // Head, Torso, Leg
	pos             Pos           // Position of the piece (top left corner)
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
	for g.canMove(g.activePiece, 0, 1) {
		g.activePiece.pos.y++
	}
	g.lockPiece(g.activePiece)
	g.joinAndScorePieces([]Pos{g.activePiece.pos})
	g.spawnNewPiece()
}

func (g *Game) movePiece(direction int, pressed *bool, key ebiten.Key) {
	g.handleKeyPress(key, pressed, func() {
		if g.canMove(g.activePiece, direction, 0) {
			g.activePiece.pos.x += direction
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
	grid                [][]*Piece // Store piece references for each grid cell
	lockedPieces        []*Piece   // Array to store locked pieces
	activePiece         *Piece
	nextPiece           *Piece
	score               int
	frameCount          int
	gameOver            bool
	rotateKeyPressed    bool
	moveLeftKeyPressed  bool
	moveRightKeyPressed bool
	dropKeyPressed      bool
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

var allPieces []Piece
var allBodies []*Body

func init() {
	allPieces = []Piece{
		{image: mustLoadImage("assets/head.png"), currentRotation: 0, size: Size{3, 3}, pieceType: "Head"},
		{image: mustLoadImage("assets/torso.png"), currentRotation: 0, size: Size{3, 3}, pieceType: "Torso"},
		{image: mustLoadImage("assets/leg.png"), currentRotation: 0, size: Size{3, 3}, pieceType: "Leg"},
		{image: mustLoadImage("assets/bomb.png"), currentRotation: 0, size: Size{3, 3}, pieceType: "Bomb"},
	}

	allBodies = []*Body{
		&Body{ // bar shape, consists of 4 parts
			name:  "longi",
			score: 2000,
			bodyPieces: []BodyPiece{ // defined as vertical bar
				BodyPiece{pos: Pos{0, 0}, rotation: 0, pieceType: "Head"},
				BodyPiece{pos: Pos{0, 3}, rotation: 0, pieceType: "Torso"},
				BodyPiece{pos: Pos{0, 6}, rotation: 0, pieceType: "Torso"},
				BodyPiece{pos: Pos{0, 9}, rotation: 0, pieceType: "Leg"},
			},
		},
		&Body{ // bar shape, consists of 3 parts
			name:  "fellow",
			score: 1000,
			bodyPieces: []BodyPiece{ // defined as vertical bar
				BodyPiece{pos: Pos{0, 0}, rotation: 0, pieceType: "Head"},
				BodyPiece{pos: Pos{0, 3}, rotation: 0, pieceType: "Torso"},
				BodyPiece{pos: Pos{0, 6}, rotation: 0, pieceType: "Leg"},
			},
		},
	}
}

/*
NewGame creates and returns a new Game instance with initialized pieces
and game state.
*/
func NewGame() *Game {
	// allocate grid
	theGrid := make([][]*Piece, gridSize.w)
	for i := 0; i < gridSize.w; i++ {
		theGrid[i] = make([]*Piece, gridSize.h)
	}

	return &Game{
		grid:        theGrid,
		activePiece: generatePiece(),
		nextPiece:   generatePiece(),
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
		if isWithinBounds(Pos{x, y}, Size{0, 0}, Pos{sidebarX + 10, 160}, Pos{sidebarX + 110, 180}) {
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
		if !g.canMove(g.activePiece, 0, 1) {
			g.lockPiece(g.activePiece)
			g.joinAndScorePieces([]Pos{g.activePiece.pos})
			g.spawnNewPiece()
		} else {
			g.activePiece.pos.y++
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
		if g.canMove(g.activePiece, direction, 0) {
			g.activePiece.pos.x += direction
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
	g.applyRotationToPiece(op, g.activePiece)
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

		g.applyRotationToPiece(op, lp)
		screen.DrawImage(lp.image, op)
	}
}

/*
applyRotationToPiece applies the current rotation to a piece and prepares it
for drawing on the screen.

Parameters:
- op: The ebiten.DrawImageOptions to apply transformations.
- piece: The Piece to apply the rotation to.
*/
func (g *Game) applyRotationToPiece(op *ebiten.DrawImageOptions, piece *Piece) {
	op.GeoM.Scale(scale, scale)

	// Center the rotation point (relative to the piece).
	x, y := grid2ScrPos(float32(piece.pos.x), float32(piece.pos.y))
	w, h := grid2ScrSize(float32(piece.size.w)/2, float32(piece.size.h)/2)
	centerX, centerY := x+w, y+h

	// Translate to the center of the piece.
	op.GeoM.Translate(float64(-w), float64(-h))

	// Rotate around the center.
	op.GeoM.Rotate(-getRotationTheta(piece.currentRotation))

	// Translate the piece back to its grid position.
	op.GeoM.Translate(float64(centerX), float64(centerY))
}

/*
drawBoundingBox draws a bounding box around the active piece for visual
reference.

Parameters:
- screen: The ebiten.Image to draw the bounding box onto.
*/
func (g *Game) drawBoundingBox(screen *ebiten.Image) {
	x, y := grid2ScrPos(float32(g.activePiece.pos.x), float32(g.activePiece.pos.y))
	w, h := grid2ScrSize(float32(g.activePiece.size.w), float32(g.activePiece.size.h))
	vector.StrokeRect(screen, x, y, w+1, h+1, 1, boundingBoxColor, false)
}

/*
drawBorder draws a border around the game area.

Parameters:
- screen: The ebiten.Image to draw the border onto.
*/
func drawBorder(screen *ebiten.Image) {
	x, y := grid2ScrPos(0.5, 0.5)
	w, h := grid2ScrSize(float32(gridSize.w-1), float32(gridSize.h-1))
	vector.StrokeRect(screen, x, y, w, h, scale, boundingBoxColor, false)
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
func (g *Game) canMove(piece *Piece, dx, dy int) bool {
	newPos := Pos{piece.pos.x + dx, piece.pos.y + dy}
	size := rotateSize(piece.size, piece.currentRotation)

	if !isWithinBounds(newPos, size, Pos{1, 1}, Pos{gridSize.w - 1, gridSize.h - 1}) {
		return false
	}

	for _, piece := range g.lockedPieces {
		if piece.isColliding(newPos, size) {
			return false
		}
	}

	return true
}

/*
lockPiece locks the active piece in its current position on the grid,
adding it to the list of locked pieces.
*/
func (g *Game) lockPiece(piece *Piece) {
	g.lockedPieces = append(g.lockedPieces, piece)

	// add references to the locked piece in the grid
	g.changePieceInGrid(piece, true)
}

/*
add/remove references to the locked piece in the grid
*/
func (g *Game) changePieceInGrid(piece *Piece, add bool) {
	rotatedSize := rotateSize(piece.size, piece.currentRotation)
	for x := piece.pos.x; x < piece.pos.x+rotatedSize.w; x++ {
		for y := piece.pos.y; y < piece.pos.y+rotatedSize.h; y++ {
			if add {
				g.grid[x][y] = piece
			} else {
				g.grid[x][y] = nil
			}
		}
	}
}

/*
spawnNewPiece make the next piece to be the active piece and
creates the next active piece from the available pieces.
*/
func (g *Game) spawnNewPiece() {
	if g.activePiece.pos.y == 0 && !g.canMove(g.activePiece, 0, 1) {
		g.endGame()
		return
	}

	g.activePiece = g.nextPiece
	g.nextPiece = generatePiece()
}

func (g *Game) joinAndScorePieces(positions []Pos) {
	log.Printf("joinAndScorePieces(pos: %v)", positions)

	for 0 < len(positions) {
		pos := positions[0]
		positions = positions[1:]

		piece := g.grid[pos.x][pos.y]

		if piece != nil {
			for _, body := range allBodies {
				posList := body.matchAtLockedPiece(g, piece)

				g.score += body.score
				g.removePieces(posList)

				// todo: compact pieces
			}
		}
	}
}

func (g *Game) removePieces(positions []Pos) {
	for _, pos := range positions {
		piece := g.grid[pos.x][pos.y]
		// remove references to the locked piece in the grid
		g.changePieceInGrid(piece, false)

		idx := slices.Index(g.lockedPieces, piece)
		if idx < 0 {
			log.Fatal("Piece is not found in grid!")
		} else {
			// remove item
			newLen := len(g.lockedPieces) - 1
			g.lockedPieces[idx] = g.lockedPieces[newLen]
			g.lockedPieces = g.lockedPieces[:newLen]
		}
	}
}

/*
generatePiece creates a new piece from the available pieces and
positions it at the top of the grid.
*/
func generatePiece() *Piece {
	newPiece := allPieces[rand.Intn(len(allPieces))]
	newPiece.pos.x = gridSize.w / 2
	newPiece.pos.y = 0
	newPiece.currentRotation = rand.Intn(4) * 90

	return &newPiece
}

/*
main initializes the game window and starts the game loop.
*/
func main() {
	log.SetFlags(log.Ltime)
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("TESTRis - Fixed Piece Spawning and Locking")

	// initialze bodies
	for _, body := range allBodies {
		body.init()
	}

	game := NewGame()
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
