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
	spriteScale  = 10 // Scale factor for sprites
)

type Piece struct {
	image           *ebiten.Image // Single image for the piece
	currentRotation float64       // Current rotation in degrees (0, 90, 180, 270)
	width, height   int           // Dimensions of the piece
}

// Game represents the game state
type Game struct {
	grid           [gridSize][gridSize]color.Color // Store colors for each grid cell
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
		grid:        [gridSize][gridSize]color.Color{},
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

	// Draw the grid
	for y := 0; y < gridSize; y++ {
		for x := 0; x < gridSize; x++ {
			if g.grid[y][x] != nil {
				vector.DrawFilledRect(screen, float32(x*cellSize), float32(y*cellSize), float32(cellSize), float32(cellSize), g.grid[y][x], false)
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

	// Draw the sidebar
	sidebarX := screenWidth - 140
	vector.DrawFilledRect(screen, float32(sidebarX), 0, 140, screenHeight, color.RGBA{R: 50, G: 50, B: 50, A: 255}, false)

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
	bounds := g.activePiece.image.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Ensure the rotation is one of the valid discrete angles
	validAngles := []float64{0, 90, 180, 270}
	closestAngle := validAngles[0]
	minDiff := math.Abs(g.activePiece.currentRotation - validAngles[0])
	for _, angle := range validAngles {
		diff := math.Abs(g.activePiece.currentRotation - angle)
		if diff < minDiff {
			minDiff = diff
			closestAngle = angle
		}
	}
	angle := closestAngle * (3.14159265 / 180)
	cos := math.Cos(angle)
	sin := math.Sin(angle)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Transform the coordinates based on rotation
			// Calculate rotated coordinates
			xf := (float64(x)*cos - float64(y)*sin) / spriteScale
			yf := (float64(x)*sin + float64(y)*cos) / spriteScale
			// Check if the pixel is non-transparent
			_, _, _, a := g.activePiece.image.At(int(xf), int(yf)).RGBA()
			if a > 0 { // Non-transparent pixel
				gridX := g.pieceX + int(xf)
				gridY := g.pieceY + int(yf)

				// Lock this cell into the grid
				if gridX >= 0 && gridX < gridSize && gridY >= 0 && gridY < gridSize {
					g.grid[gridY][gridX] = g.activePiece.image.At(int(xf), int(yf))
				}
			}
		}
	}
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
