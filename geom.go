package main

import (
	"math"
)

type Pos struct { x, y int }  // position in the grid
type Size struct { w, h int } // dimensions of a piece in the grid

func angleDegEq(ang1, ang2 int) bool {
	return (ang1 - ang2) % 360 == 0
}

func rotatePos(pos Pos, angleDeg int) Pos {
	switch d := (angleDeg + 720) % 360; d {
		case 90:  return Pos{-pos.y, pos.x}
		case 180: return Pos{-pos.x, -pos.y}
		case 270: return Pos{pos.y, -pos.x}
		default: return pos
	}
}

func rotateSize(size Size, angleDeg int) Size {
	d := (angleDeg + 720) % 360
	if d == 90 || d == 270 {
		return Size{size.h, size.w}
	} else {
		return size
	}
}

func negPos(p Pos) Pos {
	return Pos{-p.x, -p.y}
}

func addPos(left Pos, right Pos) Pos {
	return Pos{left.x + right.x, left.y + right.y}
}

func subPos(left Pos, right Pos) Pos {
	return Pos{left.x - right.x, left.y - right.y}
}

func grid2ScrPos(x, y float32) (float32, float32) {
	return x * scale, y * scale
}

func grid2ScrSize(w, h float32) (float32, float32) {
	return w * scale, h * scale
}

func isWithinBounds(pos Pos, size Size, boundsMin, boundsMax Pos) bool {
	return pos.x + size.w <= boundsMax.x && pos.x >= boundsMin.x && pos.y + size.h <= boundsMax.y && pos.y >= boundsMin.y
}

func isOverlap(pos1 Pos, size1 Size, pos2 Pos, size2 Size) bool {
	return pos1.x < pos2.x + size2.w && pos2.x < pos1.x + size1.w && pos1.y < pos2.y + size2.h && pos2.y < pos1.y + size1.h
}

func (piece *Piece) isColliding(pos Pos, size Size) bool {
	return pos.x < piece.pos.x + size.w && piece.pos.x < pos.x + piece.size.w && pos.y < piece.pos.y + size.h && piece.pos.y < pos.y + piece.size.h
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
