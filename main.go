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
	borderThickness = 10.0
	screenWidth     = 800
	screenHeight    = 600
	gridSize        = 27
	cellSize        = 20
	spriteScale     = 8 // Scale factor for sprites
)

type Piece struct {
	image           *ebiten.Image // Single image for the piece
	currentRotation float64       // Current rotation in degrees (0, 90, 180, 270)
	width, height   int           // Dimensions of the piece
}

func (g *Game) resetKeyPressFlagsExcept(except ...string) {

	g.moveLeftKeyPressed = false
	g.moveRightKeyPressed = false
	g.rotateKeyPressed = false

	for _, flag := range except {
		switch flag {
		case "moveLeftKeyPressed":
			g.moveLeftKeyPressed = true
		case "moveRightKeyPressed":
			g.moveRightKeyPressed = true
		case "rotateKeyPressed":
			g.rotateKeyPressed = true
		}
	}

}

func (g *Game) Reset() {
	*g = *NewGame()
}

type Game struct {
	grid                [gridSize][gridSize]*ebiten.Image // Store image references for each grid cell
	activePiece         *Piece
	nextPiece           *Piece
	pieceX, pieceY      int
	score               int
	frameCount          int
	rotateKeyPressed    bool
	moveLeftKeyPressed  bool
	moveRightKeyPressed bool
}

func LoadImage(path string) *ebiten.Image {
	img, _, err := ebitenutil.NewImageFromFile(path)
	if err != nil {
		log.Fatalf("Failed to load image: %s", path)
	}
	return img
}

func loadPieces() []*Piece {
	return []*Piece{
		{image: LoadImage("assets/head.png"), currentRotation: 0, width: 3, height: 3},
		{image: LoadImage("assets/torso.png"), currentRotation: 0, width: 3, height: 3},
		{image: LoadImage("assets/leg.png"), currentRotation: 0, width: 3, height: 3},
	}
}

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

func (g *Game) Update() error {
	g.frameCount++

	// Handle user input for single actions per key press
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		if !g.moveLeftKeyPressed && g.canMove(-1, 0) {
			g.pieceX--
		}
		g.moveLeftKeyPressed = true
	} else {
		g.moveLeftKeyPressed = false
	}

	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		if !g.moveRightKeyPressed && g.canMove(1, 0) {
			g.pieceX++
		}
		g.moveRightKeyPressed = true
	} else {
		g.moveRightKeyPressed = false
	}

	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		if !g.rotateKeyPressed {
			g.activePiece.currentRotation = math.Mod(g.activePiece.currentRotation+90, 360)
		}
		g.rotateKeyPressed = true
	} else {
		g.rotateKeyPressed = false
	}

	// Drop the piece every few frames
	if g.frameCount%10 == 0 {
		if !g.canMove(0, 1) { // Check if the piece can move down
			g.lockPiece()
			g.spawnNewPiece()
		} else {
			g.pieceY++
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 30, G: 30, B: 30, A: 255}) // Lighter background for debugging

	// Draw the sidebar with a distinct background color for debugging
	sidebarX := screenWidth - 140
	vector.DrawFilledRect(screen, float32(sidebarX), 0, 140, screenHeight, color.RGBA{R: 70, G: 70, B: 70, A: 255}, false)

	vector.DrawFilledRect(screen, 0, float32(screenHeight)-float32(borderThickness), float32(screenWidth), float32(borderThickness), color.White, false) // Bottom border
	vector.DrawFilledRect(screen, 0, 0, float32(borderThickness), float32(screenHeight), color.White, false)                                             // Left border
	vector.DrawFilledRect(screen, float32(sidebarX)-float32(borderThickness), 0, float32(borderThickness), float32(screenHeight), color.White, false)    // Right border
	for y := 0; y < gridSize; y++ {
		for x := 0; x < gridSize; x++ {
			if g.grid[y][x] != nil {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(spriteScale, spriteScale) // Apply scaling to locked pieces
				op.GeoM.Translate(float64(x*cellSize), float64(y*cellSize))
				op.GeoM.Translate(-float64(g.grid[y][x].Bounds().Dx())/2, -float64(g.grid[y][x].Bounds().Dy())/2) // Center rotation
				screen.DrawImage(g.grid[y][x], op)
			}
		}
	}

	// Draw the active piece
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(spriteScale, spriteScale) // Scale the sprite

	// Translate to the center of the piece for rotation
	// op.GeoM.Translate(float64(g.activePiece.width*cellSize)/2, float64(g.activePiece.height*cellSize)/2)

	// Apply rotation around the center
	op.GeoM.Rotate(g.activePiece.currentRotation * (math.Pi / 180))

	// Translate back to the piece's position on the grid
	op.GeoM.Translate(float64(g.pieceX*cellSize)-float64(g.activePiece.width*cellSize)/2, float64(g.pieceY*cellSize)-float64(g.activePiece.height*cellSize)/2)
	screen.DrawImage(g.activePiece.image, op)

	// Draw "Next Piece"
	ebitenutil.DebugPrintAt(screen, "NEXT PIECE", sidebarX+10, 20)
	op.GeoM.Reset()
	op.GeoM.Scale(spriteScale, spriteScale) // Apply scaling to the next piece
	op.GeoM.Translate(float64(sidebarX+40), 50)
	screen.DrawImage(g.nextPiece.image, op)

	// Draw restart button
	ebitenutil.DebugPrintAt(screen, "RESTART", sidebarX+10, 160)

	// Draw score
	ebitenutil.DebugPrintAt(screen, "SCORE", sidebarX+10, 120)
	ebitenutil.DebugPrintAt(screen, string(rune(g.score)), sidebarX+10, 140)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func (g *Game) canMove(dx, dy int) bool {
	newX, newY := g.pieceX+dx, g.pieceY+dy

	// Calculate the corner positions based on the current rotation
	corners := [][2]int{
		{0, 0}, // Upper left
		{g.activePiece.width - 1, 0}, // Upper right
		{0, g.activePiece.height - 1}, // Bottom left
		{g.activePiece.width - 1, g.activePiece.height - 1}, // Bottom right
	}

	for _, corner := range corners {
		x, y := corner[0], corner[1]
		rotatedX, rotatedY := x, y
		switch g.activePiece.currentRotation {
		case 90:
			rotatedX, rotatedY = y, g.activePiece.width-1-x
		case 180:
			rotatedX, rotatedY = g.activePiece.width-1-x, g.activePiece.height-1-y
		case 270:
			rotatedX, rotatedY = g.activePiece.height-1-y, x
		}

		// Check if the corner is within bounds
		if newX+rotatedX < 0 || newX+rotatedX >= gridSize || newY+rotatedY >= gridSize {
			return false
		}

		// Check collision at the corner
		if g.grid[newY+rotatedY][newX+rotatedX] != nil {
			return false
		}
	}

	return true
}

func (g *Game) lockPiece() {
	// Create a new image to represent the locked piece
	lockedPieceImage := ebiten.NewImage(g.activePiece.width*cellSize, g.activePiece.height*cellSize)

	// Draw the active piece onto the locked piece image with the correct transformations
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Rotate(g.activePiece.currentRotation * (math.Pi / 180))
	op.GeoM.Translate(float64(g.activePiece.width*cellSize/2), float64(g.activePiece.height*cellSize/2))
	lockedPieceImage.DrawImage(g.activePiece.image, op)

	// Store the locked piece image in the middle cell of its position
	midX := g.pieceX + g.activePiece.width/2
	midY := g.pieceY + g.activePiece.height/2
	if midX >= 0 && midX < gridSize && midY >= 0 && midY < gridSize {
		g.grid[midY][midX] = lockedPieceImage
	}
}

func (g *Game) spawnNewPiece() {
	pieces := loadPieces()

	g.activePiece = g.nextPiece
	g.nextPiece = pieces[rand.Intn(len(pieces))]
	g.pieceX = gridSize / 2
	g.pieceY = 0
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("TESTRis - Fixed Piece Spawning and Locking")

	game := NewGame()
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
