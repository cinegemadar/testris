package main

import (
	"slices"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Piece struct {
	image           *ebiten.Image // Single image for the piece
	currentRotation int           // Current rotation in degrees (0, 90, 180, 270)
	size            Size          // Dimensions of the piece on the grid
	pieceType       string        // Head, Torso, Leg
	pos             Pos           // Position of the piece on the grid (top left corner)
}

/*
Returns scale which also considers dimensions of the image.
The size of the rendered image must be Piece.size on grid independently of the image resolution or size.
*/
func (piece *Piece) getScale() (float64, float64) {
	return scale * float64(piece.size.w) / float64(piece.image.Bounds().Max.X), scale * float64(piece.size.h) / float64(piece.image.Bounds().Max.Y)
}

func (piece *Piece) isBomb() bool {
	return piece.pieceType == "Bomb"
}

func getPieceByType(pieceType string) *Piece {
	idx := slices.IndexFunc(allPieces, func(p Piece) bool { return p.pieceType == pieceType })
	return &allPieces[idx]
}

/*
applyRotationToPiece applies the current rotation to a piece and prepares it
for drawing on the screen.

Parameters:
- op: The ebiten.DrawImageOptions to apply transformations.
- piece: The Piece to apply the rotation to.
*/
func applyRotationToPiece(op *ebiten.DrawImageOptions, piece *Piece) {
	imageScaleX, imageScaleY := piece.getScale()
	op.GeoM.Scale(imageScaleX, imageScaleY)

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

type PieceComp struct {
	p         *Piece
	grid      *GridComp
	input     *UserInput
	state     ComponentState
	drawOrder int
}

func NewPieceComp(grid *GridComp, input *UserInput, drawOrder int) *PieceComp {
	return &PieceComp {
		grid: grid,
		input: input,
		drawOrder: drawOrder,
	}
}

func (p *PieceComp) activate(isActive bool) {
	if isActive {
		p.state = StateActive
	} else {
		p.state = StateInactive
	}
}

func (p *PieceComp) reset() {
	p.state = StateInactive
}

func (p *PieceComp) update(paused bool, frameCnt int) {
	if p.state == StateInactive || paused || p.p == nil { // note that p.p can be nil while an effect is playing on the joined pieces
		return
	}
	
	piece := p.p

	if p.input.isKeyPressed("left") && p.grid.canMove(piece, -1, 0) {
		piece.pos.x -= 1
	}

	if p.input.isKeyPressed("right") && p.grid.canMove(piece, 1, 0) {
		piece.pos.x += 1
	}

	if p.input.isKeyPressed("rotate") && !piece.isBomb() {
		piece.currentRotation = (piece.currentRotation + 90) % 360
	}
}

func (p *PieceComp) draw(screen *ebiten.Image) {
	if p.state != StateInactive && p.p != nil { // note that p.p can be nil while an effect is playing on the joined pieces
		p.drawBoundingBox(screen)

		op := &ebiten.DrawImageOptions{}
		applyRotationToPiece(op, p.p)
		screen.DrawImage(p.p.image, op)
	}
}

/*
drawBoundingBox draws a bounding box around the active piece for visual
reference.

Parameters:
- screen: The ebiten.Image to draw the bounding box onto.
*/
func (p *PieceComp) drawBoundingBox(screen *ebiten.Image) {
	x, y := grid2ScrPos(float32(p.p.pos.x), float32(p.p.pos.y))
	w, h := grid2ScrSize(float32(p.p.size.w), float32(p.p.size.h))
	vector.StrokeRect(screen, x, y, w+1, h+1, 1, boundingBoxColor, false)
}

func (p *PieceComp) getDrawOrder() int {
  return p.drawOrder
}

func (p *PieceComp) getState() ComponentState {
	return p.state
}

