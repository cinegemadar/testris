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
	screenWidth  = 700
	screenHeight = 540
	gridSize     = 27
	cellSize     = 20
	spriteScale  = 5 // Scale factor for sprites
)

type Piece struct {
	image           *ebiten.Image // Single image for the piece
	currentRotation float64       // Current rotation in degrees (0, 90, 180, 270)
	width, height   int           // Dimensions of the piece
}

// Game represents the game state
type Game struct {
	grid           [gridSize][gridSize]*ebiten.Image // Store image references for each grid cell
	activePiece    *Piece
	nextPiece      *Piece
	pieceX, pieceY int
	score          int
	frameCount     int
}

// LoadImage loads an image from a file
func LoadImage(path string) *ebiten.Image {
	img, _, err := ebitenutil.NewImageFromFile(path)
	if err != nil {
		log.Fatalf("Failed to load image: %s", path)
	}
	return img
}

// Initialize a new game
func NewGame() *Game {
	// Load the piece images
	head := &Piece{image: LoadImage("assets/head.png"), currentRotation: 0, width: 3, height: 3}
	torso := &Piece{image: LoadImage("assets/torso.png"), currentRotation: 0, width: 3, height: 3}
	leg := &Piece{image: LoadImage("assets/leg.png"), currentRotation: 0, width: 3, height: 3}
	allPieces := []*Piece{head, torso, leg}

	return &Game{
		grid:        [gridSize][gridSize]*ebiten.Image{},
		activePiece: allPieces[rand.Intn(len(allPieces))],
		nextPiece:   allPieces[rand.Intn(len(allPieces))],
		pieceX:      gridSize / 2,
		pieceY:      0,
	}
}

// Update handles the game logic
func (g *Game) Update() error {
	g.frameCount++

	// Handle user input
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) && g.canMove(-1, 0) {
		g.pieceX--
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) && g.canMove(1, 0) {
		g.pieceX++
	}
	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		g.activePiece.currentRotation = math.Mod(g.activePiece.currentRotation+90, 360)
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

// Draw renders the game screen
func (g *Game) Draw(screen *ebiten.Image) {
	// Draw background
	screen.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 255}) // Black background

	// Draw the sidebar
	sidebarX := screenWidth - 140
	vector.DrawFilledRect(screen, float32(sidebarX), 0, 140, screenHeight, color.RGBA{R: 50, G: 50, B: 50, A: 255}, false)

	// Draw the border lines
	borderThickness := 5.0
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
	op.GeoM.Scale(spriteScale, spriteScale)                                                                         // Scale the sprite
	op.GeoM.Translate(-float64(g.activePiece.image.Bounds().Dx())/2, -float64(g.activePiece.image.Bounds().Dy())/2) // Center rotation
	op.GeoM.Rotate(g.activePiece.currentRotation * (3.14159265 / 180))                                              // Apply rotation
	op.GeoM.Translate(float64(g.pieceX*cellSize+cellSize/2), float64(g.pieceY*cellSize+cellSize/2))                 // Position the piece
	screen.DrawImage(g.activePiece.image, op)

	// Draw "Next Piece"
	ebitenutil.DebugPrintAt(screen, "NEXT PIECE", sidebarX+10, 20)
	op.GeoM.Reset()
	op.GeoM.Scale(spriteScale, spriteScale) // Apply scaling to the next piece
	op.GeoM.Translate(float64(sidebarX+40), 50)
	screen.DrawImage(g.nextPiece.image, op)

	// Draw score
	ebitenutil.DebugPrintAt(screen, "SCORE", sidebarX+10, 120)
	ebitenutil.DebugPrintAt(screen, string(rune(g.score)), sidebarX+10, 140)
}

// Layout determines the size of the screen
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// Check if the piece can move in a given direction
func (g *Game) canMove(dx, dy int) bool {
	newX, newY := g.pieceX+dx, g.pieceY+dy

	// Check bounds and collisions
	for y := 0; y < g.activePiece.height; y++ {
		for x := 0; x < g.activePiece.width; x++ {
			if newX+x < 0 || newX+x >= gridSize || newY+y >= gridSize {
				return false
			}
			if g.grid[newY+y][newX+x] != nil {
				return false
			}
		}
	}
	return true
}

// Lock the current piece into the grid
func (g *Game) lockPiece() {
	// Create a new image to represent the locked piece
	lockedPieceImage := ebiten.NewImage(g.activePiece.width*cellSize, g.activePiece.height*cellSize)

	// Draw the active piece onto the locked piece image with the correct transformations
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(spriteScale, spriteScale)
	op.GeoM.Rotate(g.activePiece.currentRotation * (math.Pi / 180))
	op.GeoM.Translate(float64(g.activePiece.width*cellSize/2), float64(g.activePiece.height*cellSize/2))
	lockedPieceImage.DrawImage(g.activePiece.image, op)

	// Store the locked piece image in the grid
	for y := 0; y < g.activePiece.height; y++ {
		for x := 0; x < g.activePiece.width; x++ {
			if g.pieceX+x >= 0 && g.pieceX+x < gridSize && g.pieceY+y >= 0 && g.pieceY+y < gridSize {
				g.grid[g.pieceY+y][g.pieceX+x] = lockedPieceImage
			}
		}
	}

	g.score++
	g.score++
}

// Spawn a new piece
func (g *Game) spawnNewPiece() {
	pieces := []*Piece{
		{image: LoadImage("assets/head.png"), currentRotation: 0, width: 3, height: 3},
		{image: LoadImage("assets/torso.png"), currentRotation: 0, width: 3, height: 3},
		{image: LoadImage("assets/leg.png"), currentRotation: 0, width: 3, height: 3},
	}

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
