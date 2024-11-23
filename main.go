package main

import (
	"image/color"
	"log"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	borderThickness = 5
	screenWidth     = 800
	screenHeight    = 600
	sidebarWidth    = 140
	speed           = 5
	gridSize        = 30
	cellSize        = 16
	spriteScale     = 16 // Scale factor for sprites
)

var (
	borderColor      = color.RGBA{R: 70, G: 255, B: 255, A: 255}
	boundingBoxColor = color.RGBA{R: 255, G: 255, B: 0, A: 255}
	sidebarColor     = color.RGBA{R: 130, G: 130, B: 130, A: 255}
	backgroundColor  = color.RGBA{R: 0, G: 0, B: 0, A: 255}
)

type LockedPiece struct {
	piece *Piece // The locked piece itself
	x, y  int    // The position where it is locked
}

type Piece struct {
	image           *ebiten.Image // Single image for the piece
	currentRotation int           // Current rotation in degrees (0, 90, 180, 270)
	width, height   int           // Dimensions of the piece
	piece_type      string        // Head, Torso, Leg
}

type Game struct {
	grid                [gridSize][gridSize]*ebiten.Image // Store image references for each grid cell
	lockedPieces        []LockedPiece                     // Array to store locked pieces with positions
	activePiece         *Piece
	nextPiece           *Piece
	pieceX, pieceY      int // Position of the active piece
	score               int
	frameCount          int
	rotateKeyPressed    bool
	moveLeftKeyPressed  bool
	moveRightKeyPressed bool
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
func LoadImage(path string) *ebiten.Image {
	img, _, err := ebitenutil.NewImageFromFile(path)
	if err != nil {
		log.Fatalf("Failed to load image: %s", path)
	}
	return img
}

/*
loadPieces initializes and returns a slice of Piece pointers,
each representing a different type of game piece.
*/
func loadPieces() []*Piece {
	return []*Piece{
		{image: LoadImage("assets/head.png"), currentRotation: 0, width: 3, height: 3, piece_type: "Head"},
		{image: LoadImage("assets/torso.png"), currentRotation: 0, width: 3, height: 3, piece_type: "Torso"},
		{image: LoadImage("assets/leg.png"), currentRotation: 0, width: 3, height: 3, piece_type: "Leg"},
	}
}

/*
NewGame creates and returns a new Game instance with initialized pieces
and game state.
*/
func NewGame() *Game {
	allPieces := loadPieces()

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

	// Handle user input for single actions per key press.
	// This is a chain of responsibility; not all actions below are triggered in each iteration.
	g.moveLeft()
	g.moveRight()
	g.rotate()

	// Check for mouse click to restart the game
	g.restart()

	// Drop the piece every few frames
	g.drop()

	return nil
}

/*
Restart the game if the restart button is clicked on the sidebar.
*/
func (g *Game) restart() {
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		sidebarX := screenWidth - sidebarWidth
		if x >= sidebarX+10 && x <= sidebarX+110 && y >= 160 && y <= 180 {
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
rotate handles the rotation of the active piece when the space key is pressed.
*/
func (g *Game) rotate() {
	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		if !g.rotateKeyPressed {
			g.activePiece.currentRotation = (g.activePiece.currentRotation + 90) % 360
		}
		g.rotateKeyPressed = true
	} else {
		g.rotateKeyPressed = false
	}
}

/*
moveRight moves the active piece one cell to the right if possible
when the right arrow key is pressed.
*/
func (g *Game) moveRight() {
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		if !g.moveRightKeyPressed && g.canMove(1, 0) {
			g.pieceX++
		}
		g.moveRightKeyPressed = true
	} else {
		g.moveRightKeyPressed = false
	}
}

/*
moveLeft moves the active piece one cell to the left if possible
when the left arrow key is pressed.
*/
func (g *Game) moveLeft() {
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		if !g.moveLeftKeyPressed && g.canMove(-1, 0) {
			g.pieceX--
		}
		g.moveLeftKeyPressed = true
	} else {
		g.moveLeftKeyPressed = false
	}
}

/*
Draw renders the current game state to the screen, including the active piece,
locked pieces, and sidebar information.

Parameters:
- screen: The ebiten.Image to draw the game state onto.
*/
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(backgroundColor) // Lighter background

	// Draw the sidebar.
	sidebarX := screenWidth - sidebarWidth
	vector.DrawFilledRect(screen, float32(sidebarX), 0, sidebarWidth, screenHeight, sidebarColor, false)

	// Draw the border around the game area.
	drawBorder(screen)

	// Draw the locked pieces.
	g.drawLockedPieces(screen)

	// Draw the bounding box around the active piece.
	g.drawBoundingBox(screen)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(spriteScale, spriteScale) // Scale the sprite

	g.applyRotation(op, screen)

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
	op.GeoM.Scale(spriteScale, spriteScale) // Apply scaling to the next piece
	op.GeoM.Translate(float64(sidebarX+40), 50)
	screen.DrawImage(g.nextPiece.image, op)

	// Draw restart button
	ebitenutil.DebugPrintAt(screen, "RESTART", sidebarX+10, 160)

	// Draw score
	ebitenutil.DebugPrintAt(screen, "SCORE", sidebarX+10, 120)
	ebitenutil.DebugPrintAt(screen, string(rune(g.score)), sidebarX+10, 140)
}

/*
drawLockedPieces renders all locked pieces on the grid.

Parameters:
- screen: The ebiten.Image to draw the locked pieces onto.
*/
func (g *Game) drawLockedPieces(screen *ebiten.Image) {
	for _, lp := range g.lockedPieces {
		op := &ebiten.DrawImageOptions{}

			// Scale the locked piece using spriteScale (consistent with active piece scaling).
		// op.GeoM.Scale(
		// 	float64(spriteScale)/float64(lp.piece.image.Bounds().Dx()),
		// 	float64(spriteScale)/float64(lp.piece.image.Bounds().Dy()),
		// )

		// Calculate the center of the locked piece in screen coordinates.
		centerX := float64(lp.x*cellSize) + float64(cellSize)/2
		centerY := float64(lp.y*cellSize) + float64(cellSize)/2

		op.GeoM.Scale(
			float64(spriteScale),
			float64(spriteScale),
		)
		// Translate to the center, rotate, and translate back.
		op.GeoM.Translate(-float64(cellSize)/2, -float64(cellSize)/2) // Move to the center.
		op.GeoM.Rotate(getRotationTheta(lp.piece.currentRotation))    // Apply rotation.
		op.GeoM.Translate(centerX, centerY)                           // Translate to locked position.

		// Draw the locked piece.
		screen.DrawImage(lp.piece.image, op)
	}
}

/*
applyRotation applies the current rotation to the active piece and draws it
on the screen.

Parameters:
- op: The ebiten.DrawImageOptions to apply transformations.
- screen: The ebiten.Image to draw the rotated piece onto.
*/
func (g *Game) applyRotation(op *ebiten.DrawImageOptions, screen *ebiten.Image) {
	// Center the rotation point (relative to the piece).
	centerX := float64((g.pieceX * cellSize) + (g.activePiece.width*cellSize)/2)
	centerY := float64((g.pieceY * cellSize) + (g.activePiece.height*cellSize)/2)

	// Translate to the center of the piece.
	op.GeoM.Translate(-float64(g.activePiece.width*cellSize)/2, -float64(g.activePiece.height*cellSize)/2)

	// Rotate around the center.
	op.GeoM.Rotate(getRotationTheta(g.activePiece.currentRotation))

	// Translate the piece back to its grid position.
	op.GeoM.Translate(centerX, centerY)

	// Draw the rotated piece.
	screen.DrawImage(g.activePiece.image, op)
}

/*
drawBoundingBox draws a bounding box around the active piece for visual
reference.

Parameters:
- screen: The ebiten.Image to draw the bounding box onto.
*/
func (g *Game) drawBoundingBox(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, float32(g.pieceX*cellSize), float32(g.pieceY*cellSize), float32(g.activePiece.width*cellSize), 1, boundingBoxColor, false)
	vector.DrawFilledRect(screen, float32(g.pieceX*cellSize), float32((g.pieceY+g.activePiece.height)*cellSize), float32(g.activePiece.width*cellSize), 1, boundingBoxColor, false)
	vector.DrawFilledRect(screen, float32(g.pieceX*cellSize), float32(g.pieceY*cellSize), 1, float32(g.activePiece.height*cellSize), boundingBoxColor, false)
	vector.DrawFilledRect(screen, float32((g.pieceX+g.activePiece.width)*cellSize), float32(g.pieceY*cellSize), 1, float32(g.activePiece.height*cellSize), boundingBoxColor, false)
}

/*
drawBorder draws a border around the game area.

Parameters:
- screen: The ebiten.Image to draw the border onto.
*/
func drawBorder(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, 0, 0, float32(gridSize*cellSize), float32(borderThickness), borderColor, false)
	vector.DrawFilledRect(screen, 0, float32(gridSize*cellSize)-float32(borderThickness), float32(gridSize*cellSize), float32(borderThickness), borderColor, false)
	vector.DrawFilledRect(screen, 0, 0, float32(borderThickness), float32(gridSize*cellSize), borderColor, false)
	vector.DrawFilledRect(screen, float32(gridSize*cellSize)-float32(borderThickness), 0, float32(borderThickness), float32(gridSize*cellSize), borderColor, false)
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
	// Calculate the new position of the active piece
	newX := g.pieceX + dx
	newY := g.pieceY + dy

	// Ensure the piece stays within bounds.
	if newX < 0 || newX+g.activePiece.width > gridSize {
		return false
	}
	if newY < 0 || newY+g.activePiece.height > gridSize {
		return false
	}

	// Check for collisions with locked pieces
	for _, lp := range g.lockedPieces {
		// Loop through each cell of the active piece's grid
		for y := 0; y < g.activePiece.height; y++ {
			for x := 0; x < g.activePiece.width; x++ {
				if lp.x == newX+x && lp.y == newY+y {
					return false
				}
			}
		}
	}

	return true
}

/*
lockPiece locks the active piece in its current position on the grid,
adding it to the list of locked pieces.
*/
func (g *Game) lockPiece() {
	// Create a copy of the active piece with its current state
	lockedPiece := &Piece{
		image:           ebiten.NewImageFromImage(g.activePiece.image), // Deep copy of the image
		currentRotation: g.activePiece.currentRotation,
		width:           g.activePiece.width,  // This remains constant for squares
		height:          g.activePiece.height, // This remains constant for squares
		piece_type:      g.activePiece.piece_type,
	}

	// Append the locked piece and its position to the array
	g.lockedPieces = append(g.lockedPieces, LockedPiece{
		piece: lockedPiece,
		x:     g.pieceX, // Current X position in grid
		y:     g.pieceY, // Current Y position in grid
	})
}

/*
spawnNewPiece selects a new active piece from the available pieces and
positions it at the top of the grid.
*/
func (g *Game) spawnNewPiece() {
	pieces := loadPieces()

	g.activePiece = g.nextPiece
	g.nextPiece = pieces[rand.Intn(len(pieces))]
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
