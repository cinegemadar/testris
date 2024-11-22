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
	speed           = 10
	gridSize        = 26
	cellSize        = 16
	spriteScale     = 16 // Scale factor for sprites
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

func getRotationTheta(deg int) float64 {
	return float64(deg) * (math.Pi / 180)
}
func (g *Game) Reset() {
	*g = *NewGame()
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
		{image: LoadImage("assets/head.png"), currentRotation: 0, width: 3, height: 3, piece_type: "Head"},
		{image: LoadImage("assets/torso.png"), currentRotation: 0, width: 3, height: 3, piece_type: "Torso"},
		{image: LoadImage("assets/leg.png"), currentRotation: 0, width: 3, height: 3, piece_type: "Leg"},
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
	g.moveLeft()
	g.moveRight()
	g.rotate()

	// Drop the piece every few frames
	g.drop()

	return nil
}

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

func (g *Game) rotate() {
	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		if !g.rotateKeyPressed {
			g.activePiece.currentRotation += g.activePiece.currentRotation + 90
			if g.activePiece.currentRotation > 720 {
				g.activePiece.currentRotation = 0
			}
		}
		g.rotateKeyPressed = true
	} else {
		g.rotateKeyPressed = false
	}
}

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

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 255}) // Lighter background

	// Draw the sidebar
	sidebarX := screenWidth - sidebarWidth
	vector.DrawFilledRect(screen, float32(sidebarX), 0, sidebarWidth, screenHeight, color.RGBA{R: 70, G: 70, B: 70, A: 255}, false)

	// Draw  border around the game area
	drawBorder(screen) // Right border

	// Draw the locked piece
	g.drawLockedPieces(screen)

	// Draw the bounding box around the active piece
	g.drawBoundingBox(screen) // Right border

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(spriteScale, spriteScale) // Scale the sprite

	g.applyRotation(op, screen)

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

func (g *Game) drawLockedPieces(screen *ebiten.Image) {
	for _, lp := range g.lockedPieces {

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(spriteScale, spriteScale) // Scale the sprite
		// Calculate the top-left corner of the piece in screen coordinates
		topLeftX := float64(lp.x * cellSize)
		topLeftY := float64(lp.y * cellSize)

		// Move to the center of the piece, rotate, and move back
		op.GeoM.Translate(topLeftX+float64(lp.piece.width*cellSize)/2, topLeftY+float64(lp.piece.height*cellSize)/2)
		op.GeoM.Rotate(getRotationTheta(lp.piece.currentRotation))
		op.GeoM.Translate(-float64(lp.piece.width*cellSize)/2, -float64(lp.piece.height*cellSize)/2)

		// Draw the locked piece
		screen.DrawImage(lp.piece.image, op)
	}
}

func (g *Game) applyRotation(op *ebiten.DrawImageOptions, screen *ebiten.Image) {
	// 1. Center the rotation point (relative to the piece)
	centerX := float64((g.pieceX * cellSize) + (g.activePiece.width*cellSize)/2)
	centerY := float64((g.pieceY * cellSize) + (g.activePiece.height*cellSize)/2)

	// Translate to the center of the piece
	op.GeoM.Translate(-float64(g.activePiece.width*cellSize)/2, -float64(g.activePiece.height*cellSize)/2)

	// Rotate around the center
	op.GeoM.Rotate(getRotationTheta(g.activePiece.currentRotation))

	// Translate the piece back to its grid position
	op.GeoM.Translate(centerX, centerY)

	// Draw the rotated piece
	screen.DrawImage(g.activePiece.image, op)
}

func (g *Game) drawBoundingBox(screen *ebiten.Image) {
	boundingBoxColor := color.RGBA{R: 255, G: 255, B: 0, A: 255}
	vector.DrawFilledRect(screen, float32(g.pieceX*cellSize), float32(g.pieceY*cellSize), float32(g.activePiece.width*cellSize), 1, boundingBoxColor, false)
	vector.DrawFilledRect(screen, float32(g.pieceX*cellSize), float32((g.pieceY+g.activePiece.height)*cellSize), float32(g.activePiece.width*cellSize), 1, boundingBoxColor, false)
	vector.DrawFilledRect(screen, float32(g.pieceX*cellSize), float32(g.pieceY*cellSize), 1, float32(g.activePiece.height*cellSize), boundingBoxColor, false)
	vector.DrawFilledRect(screen, float32((g.pieceX+g.activePiece.width)*cellSize), float32(g.pieceY*cellSize), 1, float32(g.activePiece.height*cellSize), boundingBoxColor, false)
}

func drawBorder(screen *ebiten.Image) {
	border_color := color.RGBA{R: 70, G: 255, B: 255, A: 255}
	vector.DrawFilledRect(screen, 0, 0, float32(gridSize*cellSize), float32(borderThickness), border_color, false)
	vector.DrawFilledRect(screen, 0, float32(gridSize*cellSize)-float32(borderThickness), float32(gridSize*cellSize), float32(borderThickness), border_color, false)
	vector.DrawFilledRect(screen, 0, 0, float32(borderThickness), float32(gridSize*cellSize), border_color, false)
	vector.DrawFilledRect(screen, float32(gridSize*cellSize)-float32(borderThickness), 0, float32(borderThickness), float32(gridSize*cellSize), border_color, false)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func (g *Game) canMove(dx, dy int) bool {
	// Calculate the new position of the bounding box after the move
	newX := g.pieceX + dx
	newY := g.pieceY + dy

	// Calculate the border thickness in cells
	borderCells := int(math.Ceil(float64(borderThickness) / float64(cellSize)))

	// Ensure the bounding box does not go outside the playable grid
	if newX < borderCells || newX+g.activePiece.width > gridSize-borderCells {
		return false
	}
	if newY < borderCells || newY+g.activePiece.height > gridSize-borderCells {
		return false
	}

	return true
}

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
