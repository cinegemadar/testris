package main

import (
	"log"
)

/*
represents a piece of the body
*/
type BodyPiece struct {
	pos       Pos    // relative position in the body local CS (origin of the CS is not fixed, can be one of the body piece)
	rotation  int    // rotation 0:right, 90:top, 180:left, 270:bottom
	pieceType string // type of the piece. must match to one of the globally available piece types
}

/*
represents a body which is composed of one of more pieces.
the pieces are arranged in a specific geometry.
note that any specific piece type can be used one or more times in the geometry.
*/
type Body struct {
	name           string           // name of the body
	score          int              //
	bodyPieces     []BodyPiece      // pieces which can be joined
	pieceTypeToIdx map[string][]int // maps piece type to indices in bodyPieces
}

func (b *Body) init() {
	// already initialized?
	print(b.name)
	if b.pieceTypeToIdx != nil {
		return
	}

	b.pieceTypeToIdx = make(map[string][]int)

	for idx, bodyPiece := range b.bodyPieces {
		idxList := b.pieceTypeToIdx[bodyPiece.pieceType]
		b.pieceTypeToIdx[bodyPiece.pieceType] = append(idxList, idx)
	}

	log.Printf("Body.init() name:'%s' pieceTypeToIdx:%v", b.name, b.pieceTypeToIdx)
}

/*
Returns the bounding box of a body in bodyPiece CS.
*/
func (b *Body) getBoundingBox() (Pos, Size) {
	minPos := Pos{}
	maxPos := Pos{}

	for i, bp := range b.bodyPieces {
		piece := getPieceByType(bp.pieceType)
		rotatedSize := rotateSize(piece.size, piece.currentRotation)

		if i == 0 || bp.pos.x < minPos.x {
			minPos.x = bp.pos.x
		}
		if i == 0 || bp.pos.y < minPos.y {
			minPos.y = bp.pos.y
		}
		if i == 0 || maxPos.x < bp.pos.x+rotatedSize.w {
			maxPos.x = bp.pos.x + rotatedSize.w
		}
		if i == 0 || maxPos.y < bp.pos.y+rotatedSize.h {
			maxPos.y = bp.pos.y + rotatedSize.h
		}
	}

	return minPos, Size{maxPos.x - minPos.x, maxPos.y - minPos.y}
}

/*
checks if the Body is located in the game's grid at the location of a locked piece
*/
func (body *Body) matchAtLockedPiece(grid *GridComp, lockedPiece *Piece) []*Piece {
	log.Printf(" Body[%s].matchAtLockedPiece(lockedPiece:%s,pos:%v,rot:%d)", body.name, lockedPiece.pieceType, lockedPiece.pos, lockedPiece.currentRotation)
	// check if the body contains at least one body piece having the same type as the locked piece?
	idxList, ok := body.pieceTypeToIdx[lockedPiece.pieceType]
	if !ok {
		return nil
	}

	// enumerate the body pieces of the body having the required type. try to match the body to the grid at that body piece
	for _, idx := range idxList {
		bodyPiece := &body.bodyPieces[idx]
		pieceList := body.matchBodyPieceAtLockedPiece(grid, bodyPiece, lockedPiece)
		if 0 < len(pieceList) {
			return pieceList
		}
	}
	return nil
}

/*
checks if the Body is located in the game's grid at the location of a locked piece.
the check assumes that the locked piece is located at a specific body piece.
*/
func (body *Body) matchBodyPieceAtLockedPiece(grid *GridComp, bodyPiece *BodyPiece, lockedPiece *Piece) []*Piece {
	bodyCsOrigin := bodyPiece.pos // fix the origin of the body CS
	bodyCsRotation := bodyPiece.rotation - lockedPiece.currentRotation

	log.Printf("  Body[%s].matchBodyPieceAtLockedPiece(bodyPiece:%v) - bodyCs:%v,%ddeg", body.name, bodyPiece, bodyCsOrigin, bodyCsRotation)

	var matchedPieceList []*Piece
	for _, bp := range body.bodyPieces {
		relPosBodyCs := subPos(bp.pos, bodyCsOrigin)
		relPosGridCs := rotatePos(relPosBodyCs, bodyCsRotation)
		posGridCs := addPos(lockedPiece.pos, relPosGridCs)

		if !isOverlap(posGridCs, Size{1, 1}, Pos{0, 0}, gridSize) {
			log.Printf("   Checking '%s'@%v... Outside the grid.", bp.pieceType, posGridCs)
			return nil
		}

		piece := grid.getPiece(posGridCs)
		if piece == nil {
			log.Printf("  Checking '%s'@%v... Empty grid location.", bp.pieceType, posGridCs)
			return nil
		}
		if piece.pieceType != bp.pieceType {
			log.Printf("  Checking '%s'@%v... Type %s does not match.", bp.pieceType, posGridCs, piece.pieceType)
			return nil
		}

		if piece.pos != posGridCs {
			log.Printf("  Checking '%s'@%v... Position %v is slightly off", bp.pieceType, posGridCs, piece.pos)
			return nil
		}
		if !angleDegEq(piece.currentRotation, bp.rotation-bodyCsRotation) {
			log.Printf("  Checking '%s'@%v... Piece orientation %d mismatches to expected %d.", bp.pieceType, posGridCs, piece.currentRotation, bp.rotation+bodyCsRotation)
			return nil
		}

		log.Printf("  Checking '%s'@%v... Body piece matches", bp.pieceType, posGridCs)
		matchedPieceList = append(matchedPieceList, piece)
	}

	log.Printf("   Body matched. Returning piece list %v", matchedPieceList)
	return matchedPieceList
}
