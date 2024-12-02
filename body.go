package main

import (
	"log"
)

/*
represents a piece of the body
*/
type BodyPiece struct {
	pos         Pos       // relative position in the body local CS (origin of the CS is not fixed, can be one of the body piece)
	rotation    int       // rotation 0:right, 90:top, 180:left, 270:bottom
	pieceType   string    // type of the piece. must match to one of the globally available piece types
}

/*
represents a body which is composed of one of more pieces.
the pieces are arranged in a specific geometry.
note that any specific piece type can be used one or more times in the geometry.
*/
type Body struct {
	name            string            // name of the body
	score           int               //
	bodyPieces      []BodyPiece       // pieces which can be joined
	pieceTypeToIdx  map[string][]int  // maps piece type to indices in bodyPieces
}

func (b *Body) init() {
	// already initialized?
	if b.pieceTypeToIdx != nil {
		return
	}

	b.pieceTypeToIdx = make(map[string][]int)

	for idx, bodyPiece := range b.bodyPieces {
		idxList := b.pieceTypeToIdx[bodyPiece.pieceType]
		b.pieceTypeToIdx[bodyPiece.pieceType] = append(idxList, idx)
	}

	log.Printf("Body.init() name:%s pieceTypeToIdx:%v", b.name, b.pieceTypeToIdx)
}

/*
checks if the Body is located in the game's grid at the location of a locked piece
*/
func (body *Body) matchAtLockedPiece(g *Game, lockedPiece *Piece) []Pos {
	log.Printf(" Body[%s].matchAtLockedPiece(lockedPiece:%s,pos:%v,rot:%d)", body.name, lockedPiece.pieceType, lockedPiece.pos, lockedPiece.currentRotation)
	// check if the body contains at least one body piece having the same type as the locked piece?
	idxList, ok := body.pieceTypeToIdx[lockedPiece.pieceType]
	if !ok { return nil }

	// enumerate the body pieces of the body having the required type. try to match the body to the grid at that body piece
	for _, idx := range idxList {
		bodyPiece := &body.bodyPieces[idx]
		posList := body.matchBodyPieceAtLockedPiece(g, bodyPiece, lockedPiece)
		if posList != nil && 0 < len(posList) {
			return posList
		}
	}
	return nil
}

/*
checks if the Body is located in the game's grid at the location of a locked piece.
the check assumes that the locked piece is located at a specific body piece.
*/
func (body *Body) matchBodyPieceAtLockedPiece(g *Game, bodyPiece *BodyPiece, lockedPiece *Piece) []Pos {
	bodyCsOrigin := bodyPiece.pos // fix the origin of the body CS
	bodyCsRotation := bodyPiece.rotation - lockedPiece.currentRotation

	log.Printf("  Body[%s].matchBodyPieceAtLockedPiece(bodyPiece:%v) - bodyCs:%v,%ddeg", body.name, bodyPiece, bodyCsOrigin, bodyCsRotation)

	matchedPosList := make([]Pos, 0)
	for _, bp := range body.bodyPieces {
		relPosBodyCs := subPos(bp.pos, bodyCsOrigin)
		relPosGridCs := rotatePos(relPosBodyCs, bodyCsRotation)
		posGridCs := addPos(lockedPiece.pos, relPosGridCs)

		if !isOverlap(posGridCs, Size{1, 1}, Pos{0, 0}, gridSize) {
			log.Printf("   Checking '%s'@%v... Outside the grid.", bp.pieceType, posGridCs)
			return nil
		}

		piece := g.grid[posGridCs.x][posGridCs.y]
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
		if !angleDegEq(piece.currentRotation, bp.rotation - bodyCsRotation) {
			log.Printf("  Checking '%s'@%v... Piece orientation %d mismatches to expected %d.", bp.pieceType, posGridCs, piece.currentRotation, bp.rotation + bodyCsRotation)
			return nil
		}

		log.Printf("  Checking '%s'@%v... Body piece matches", bp.pieceType, posGridCs)
		matchedPosList = append(matchedPosList, posGridCs)
	}

	log.Printf("   Body matched. Returning pos list %v", matchedPosList)
	return matchedPosList
}
