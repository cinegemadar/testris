package main

import (
	"log"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type GridComp struct {
	size         Size
	content      [][]*Piece // Store piece references for each grid cell
	lockedPieces []*Piece   // Array to store locked pieces, sorted first by y then x coordinate
	state        ComponentState
	drawOrder    int
}

func NewGridComp(size Size, drawOrder int) *GridComp {
	// allocate grid
	theGrid := make([][]*Piece, size.w)
	for i := 0; i < gridSize.w; i++ {
		theGrid[i] = make([]*Piece, size.h)
	}

	return &GridComp {
		size: size,
		content: theGrid,
		drawOrder: drawOrder,
	}
}

func (g *GridComp) activate(isActive bool) {
	if isActive {
		g.state = StateActive
	} else {
		g.state = StateInactive
	}
}

func (g *GridComp) reset() {
	g.state = StateInactive

	for i := 0; i < g.size.w; i++ {
		g.content[i] = make([]*Piece, g.size.h)
	}

	g.lockedPieces = nil
}

func (g *GridComp) update(gamePaused bool, frameCnt int) {
	// grid has no dynamic behavior
}

func (g *GridComp) draw(screen *ebiten.Image) {
	if g.state != StateInactive {
		g.drawLockedPieces(screen)
		g.drawBorder(screen)
	}
}

func (g *GridComp) getDrawOrder() int {
  return g.drawOrder
}

func (g *GridComp) getState() ComponentState {
	return g.state
}

/*
drawLockedPieces renders all locked pieces on the grid.

Parameters:
- screen: The ebiten.Image to draw the locked pieces onto.
*/
func (g *GridComp) drawLockedPieces(screen *ebiten.Image) {
	for _, lp := range g.lockedPieces {
		op := &ebiten.DrawImageOptions{}

		// Calculate the top-left corner of the locked piece in screen coordinates.

		applyRotationToPiece(op, lp)
		screen.DrawImage(lp.image, op)
	}
}

/*
drawBorder draws a border around the game area.

Parameters:
- screen: The ebiten.Image to draw the border onto.
*/
func (g *GridComp) drawBorder(screen *ebiten.Image) {
	x, y := grid2ScrPos(0.5, -0.5)
	w, h := grid2ScrSize(float32(g.size.w-1), float32(g.size.h))
	// draw a rectangle with thick border. the top border is invisible (intentionally outside of the screen) intentionally.
	vector.StrokeRect(screen, x, y, w, h, scale, boundingBoxColor, false)
}

func (g *GridComp) getPiece(p Pos) *Piece {
	return g.content[p.x][p.y]
}

/*
canMove checks if the active piece can move to a new position on the grid.

Parameters:
- dx: The change in the x-direction.
- dy: The change in the y-direction.

Returns:
- True if the piece can move to the new position, otherwise false.
*/
func (g *GridComp) canMove(piece *Piece, dx, dy int) bool {
	newPos := Pos{piece.pos.x + dx, piece.pos.y + dy}
	size := rotateSize(piece.size, piece.currentRotation)

	if !isWithinBounds(newPos, size, Pos{1, 0}, Pos{g.size.w - 1, g.size.h}) {
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
drop moves the active piece as far down as possible.
*/
func (g *GridComp) drop(piece *Piece) {
	for g.canMove(piece, 0, 1) {
		piece.pos.y++
	}
}

func (g *GridComp) joinPieces(pieces []*Piece) []*Body {
	log.Printf("joinPieces(pieces: %v)", pieces)

	var bodies []*Body
	for 0 < len(pieces) {
		// dequeue first piece
		piece := pieces[0]
		pieces = pieces[1:]

		if piece != nil {
			for _, body := range allBodies {
				posList := body.matchAtLockedPiece(g, piece)

				if posList != nil {
					g.removePieces(posList)
					bodies = append(bodies, body)
				}
			}
		}
	}

	return bodies
}

func (g *GridComp) removePieces(positions []Pos) {
	for _, pos := range positions {
		piece := g.content[pos.x][pos.y]
		g.unlockPiece(piece)
	}
}

func (g *GridComp) compactGrid() []*Piece {
	fallenPieces := make([]*Piece, 0)

	for i := len(g.lockedPieces) - 1; 0 <= i; i-- {
		piece := g.lockedPieces[i]

		// check if piece can fall
		size := rotateSize(piece.size, piece.currentRotation)
		dy := size.h - 1
		for g.canMove(piece, 0, dy+1) {
			dy++
		}

		if size.h <= dy {
			log.Printf("Moving piece '%s'@%v down by %d", piece.pieceType, piece.pos, dy)
			g.unlockPiece(piece)
			piece.pos.y += dy
			g.lockPiece(piece)

			fallenPieces = append(fallenPieces, piece)
		}
	}

	return fallenPieces
}

/*
lockPiece locks the active piece in its current position on the grid,
adding it to the list of locked pieces.
*/
func (g *GridComp) lockPiece(piece *Piece) {
	// find in the sorted locked list
	idx := sort.Search(len(g.lockedPieces), func(i int) bool {
		return piece.pos.y < g.lockedPieces[i].pos.y || (piece.pos.y == g.lockedPieces[i].pos.y && piece.pos.x <= g.lockedPieces[i].pos.x)
	})

	if idx < len(g.lockedPieces) && g.lockedPieces[idx] == piece {
		var r *int
		*r = 42
		log.Fatalf("The piece %v is not expected in the locked list!", piece)
	}

	// insert to sorted list
	g.lockedPieces = append(g.lockedPieces, nil)
	copy(g.lockedPieces[idx+1:], g.lockedPieces[idx:])
	g.lockedPieces[idx] = piece

	// add references to the locked piece in the grid
	g.changePieceInGrid(piece, true)
}

/*
Remove piece from the locked list and grid matrix.
*/
func (g *GridComp) unlockPiece(lockedPiece *Piece) {
	// find in the sorted locked list
	idx := sort.Search(len(g.lockedPieces), func(i int) bool {
		return lockedPiece.pos.y < g.lockedPieces[i].pos.y || (lockedPiece.pos.y == g.lockedPieces[i].pos.y && lockedPiece.pos.x <= g.lockedPieces[i].pos.x)
	})

	if idx == len(g.lockedPieces) || g.lockedPieces[idx] != lockedPiece {
		log.Fatalf("The piece %v is expected in the locked list!", lockedPiece)
	}

	// remove from sorted list
	g.lockedPieces = append(g.lockedPieces[:idx], g.lockedPieces[idx+1:]...)

	// remove references to the locked piece in the grid
	g.changePieceInGrid(lockedPiece, false)
}

/*
add/remove references to the locked piece in the grid
*/
func (g *GridComp) changePieceInGrid(piece *Piece, add bool) {
	rotatedSize := rotateSize(piece.size, piece.currentRotation)
	for x := piece.pos.x; x < piece.pos.x+rotatedSize.w; x++ {
		for y := piece.pos.y; y < piece.pos.y+rotatedSize.h; y++ {
			if add {
				g.content[x][y] = piece
			} else {
				g.content[x][y] = nil
			}
		}
	}
}

func (g *GridComp) getPiecesBelow(piece *Piece) []*Piece {
	pieces := make([]*Piece, 0, 1) // empty, capacity=1

	below := addPos(piece.pos, Pos{0, 1})
	// is the location below the bomb within the grid?
	if isWithinBounds(below, Size{1, 1}, Pos{1, 1}, Pos{g.size.w - 1, g.size.h - 1}) {
		// remove (unlock) each piece below the bomb
		for i := 0; i < piece.size.w; i++ {
			piece := g.getPiece(below)
			if piece != nil {
				pieces = append(pieces, piece)
			}
			below.x++
		}
	}

	return pieces
}
