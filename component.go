package main

import (
	"fmt"
	"log"
	"reflect"
	"slices"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type ComponentState int

const (
	StateInactive ComponentState = iota // component does not simulate, nor render
	StateActive                         // component simulate and/or render
	StateBlocking                       // same as StateActive but the game is paused
)

type Component interface {
	activate(isActive bool)
	reset()
	update(gamePaused bool, frameCnt int) // frameCnt counts also in paused state
	draw(screen *ebiten.Image)
	getDrawOrder() int
	getState() ComponentState
}

type Components []Component

//
// ------------ component manager ------------
//
type ComponentMgr struct {
	compList       Components
	order2CompList map[int]Components
	sortedOrders   []int
}

func NewComponentMgr() *ComponentMgr {
	return &ComponentMgr {
		order2CompList: map[int]Components{},
	}
}

func (mgr *ComponentMgr) add(comp Component) {
	log.Printf("ComponentMgr.add() comp type is %s\n", reflect.TypeOf(comp))
	mgr.compList = append(mgr.compList, comp)

	// add to order hash
	drawOrder := comp.getDrawOrder()
	slice := mgr.order2CompList[drawOrder]
	mgr.order2CompList[drawOrder] = append(slice, comp)

	mgr.sortedOrders = make([]int, 0)
	for k, _ := range mgr.order2CompList {
		mgr.sortedOrders = append(mgr.sortedOrders, k)
	}
	sort.Ints(mgr.sortedOrders)
}

func (mgr *ComponentMgr) remove(comp Component) {
	drawOrder := comp.getDrawOrder()
	
	// remove from comps
	idx := slices.Index(mgr.compList, comp)
	mgr.compList[idx] = mgr.compList[len(mgr.compList)-1]
	mgr.compList = mgr.compList[:len(mgr.compList)-1]

	// remove from order hash
	slice := mgr.order2CompList[drawOrder]
	idx = slices.Index(slice, comp)
	if 0 <= idx {
		mgr.order2CompList[drawOrder] = slices.Delete(slice, idx, idx+1)
	}

	mgr.sortedOrders = make([]int, 0)
	for k, _ := range mgr.order2CompList {
		mgr.sortedOrders = append(mgr.sortedOrders, k)
	}
	sort.Ints(mgr.sortedOrders)
}

func (mgr *ComponentMgr) reset() {
	for _, c := range mgr.compList {
		c.reset()
	}
}

func (mgr *ComponentMgr) update(frameCnt int) {
	gamePaused := false
	for _, c := range mgr.compList {
		if c.getState() == StateBlocking {
			gamePaused = true
			break
		}
	}

	for _, c := range mgr.compList {
		if c.getState() != StateInactive {
			c.update(gamePaused, frameCnt)
		}
	}
}

func (mgr *ComponentMgr) draw(screen *ebiten.Image) {
	for _, order := range mgr.sortedOrders {
		components := mgr.order2CompList[order]
		for i := len(components)-1; 0<=i; i-- { // draw the active component last added
			comp := components[i]
			if comp.getState() != StateInactive {
				comp.draw(screen)
				break
			}
		}
	}
}

//
// ------------ dialog ------------
//
type DialogComp struct {
	state ComponentState
	isBlocking bool
	text []string
	screenPos Pos
	drawOrder int
	timeoutFrameCnt int
	countdownFrameCnt int
}

func NewModalDialog(text []string, screenPos Pos, drawOrder int) *DialogComp {
	return &DialogComp {
		isBlocking: true,
		text: text,
		screenPos: screenPos,
		drawOrder: drawOrder,
	}
}

func NewDialog(text []string, screenPos Pos, timeoutFrameCnt int, drawOrder int) *DialogComp {
	return &DialogComp {
		isBlocking: false,
		text: text,
		screenPos: screenPos,
		drawOrder: drawOrder,
		timeoutFrameCnt: timeoutFrameCnt,
	}
}

func (d *DialogComp) activate(isActive bool) {
	d.countdownFrameCnt = d.timeoutFrameCnt

	if isActive && d.isBlocking {
		d.state = StateBlocking
	} else if isActive {
		d.state = StateActive
	} else {
		d.state = StateInactive
	}
}

func (d *DialogComp) reset() {
	d.state = StateInactive
}

func (d *DialogComp) update(paused bool, frameCnt int) {
	if d.state == StateInactive {
		return
	}

	if 0 < d.timeoutFrameCnt {
		d.countdownFrameCnt--
		if d.countdownFrameCnt < 0 {
			d.state = StateInactive
		}
	}
}

func (d *DialogComp) draw(screen *ebiten.Image) {
	if d.state != StateInactive {
		for idx, t := range d.text {
			ebitenutil.DebugPrintAt(screen, t, d.screenPos.x, d.screenPos.y + idx * 20)
		}
	}
}

func (d *DialogComp) getDrawOrder() int {
  return d.drawOrder
}

func (d *DialogComp) getState() ComponentState {
	return d.state
}

//
// ------------ side bar ------------
//
type SideBarComp struct {
	state ComponentState
	pos Pos
	size Size
	drawOrder int
	input *UserInput
	restartAction func()
	restartTextBox Rect
	nextPiece *Piece
	score int
	speedLevel int
	topScores []int
}

func NewSideBar(input *UserInput, pos Pos, size Size, restartAction func(), drawOrder int) *SideBarComp {
	return &SideBarComp {
		pos: pos,
		size: size,
		drawOrder: drawOrder,
		input: input,
		restartAction: restartAction,
		restartTextBox: Rect{Pos{pos.x + 10, 160}, Size{100, 20}},
	}
}

func (s *SideBarComp) activate(isActive bool) {
	s.state = StateActive
}

func (s *SideBarComp) update(paused bool, frameCnt int) {
	if s.state == StateInactive {
		return
	}

	if s.input.isMouseLeftClick() {
		x, y := ebiten.CursorPosition()
		if isOverlap(Pos{x, y}, Size{1, 1}, s.restartTextBox.pos, s.restartTextBox.size) {
			s.restartAction()
		}
	}
}

func (s *SideBarComp) reset() {
	s.state = StateInactive
	s.nextPiece = nil
	s.score = 0
	s.speedLevel = 0
	s.topScores = []int{}
}

func (s *SideBarComp) draw(screen *ebiten.Image) {
	if s.state != StateInactive {
		s.drawSidebar(screen)
	}
}

func (s *SideBarComp) getDrawOrder() int {
  return s.drawOrder
}

func (s *SideBarComp) getState() ComponentState {
	return s.state
}

func (s *SideBarComp) setValues(nextPiece *Piece, score int, speedLevel int, topScores []int) {
	s.nextPiece = nextPiece
	s.score = score
	s.speedLevel = speedLevel
	s.topScores = topScores
}


/*
drawSidebar renders the sidebar, including the next piece, restart button,
and score.

Parameters:
- screen: The ebiten.Image to draw the sidebar onto.
*/
func (s *SideBarComp) drawSidebar(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, float32(s.pos.x), float32(s.pos.y), float32(s.size.w), float32(s.size.h), sidebarColor, false)

	// Draw "Next Piece"
	ebitenutil.DebugPrintAt(screen, "NEXT PIECE", s.pos.x+10, 20)
	op := &ebiten.DrawImageOptions{}
	imageScaleX, imageScaleY := s.nextPiece.getScale()
	op.GeoM.Scale(imageScaleX, imageScaleY) // Apply scaling to the next piece
	op.GeoM.Translate(float64(s.pos.x+40), 50)
	screen.DrawImage(s.nextPiece.image, op)

	// Draw restart button
	ebitenutil.DebugPrintAt(screen, "RESTART", s.restartTextBox.pos.x, s.restartTextBox.pos.y)

	// Draw top 5 scores
	ebitenutil.DebugPrintAt(screen, "TOP 5 SCORES", s.pos.x+10, 200)
	for i, score := range s.topScores {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d: %d", i+1, score), s.pos.x+10, 220+i*20)
	}

	// Draw current score
	ebitenutil.DebugPrintAt(screen, "SCORE", s.pos.x+10, 120)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d", s.score), s.pos.x+80, 120)

	// Draw current speed level
	ebitenutil.DebugPrintAt(screen, "SPEED", s.pos.x+10, 140)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d", s.speedLevel), s.pos.x+80, 140)

	// Draw hints about joint bodies
	hintPosLL := Pos{s.pos.x, screenHeight}
	hintRowHeight := 0
	for i := 0; i < len(allBodies); i++ {
		body := allBodies[len(allBodies)-1-i]

		ok, hintAreaSize := s.drawSidebarHint(screen, body, hintPosLL)

		// go to row above if no more space on the sidebar row
		if !ok {
			hintPosLL.x = s.pos.x
			hintPosLL.y -= hintRowHeight + 10
			_, hintAreaSize = s.drawSidebarHint(screen, body, hintPosLL)
		}

		if hintRowHeight < hintAreaSize.h {
			hintRowHeight = hintAreaSize.h
		}

		hintPosLL.x += hintAreaSize.w
	}
}

func (s *SideBarComp) drawSidebarHint(screen *ebiten.Image, body *Body, posLL Pos) (bool, Size) {
	hintTextAreaHeight := 40
	hintAreaSize := Size{70, hintTextAreaHeight} // text + pieces together

	// check if outside of screen
	if screenWidth < posLL.x+hintAreaSize.w {
		return false, hintAreaSize
	}

	// draw text
	hintTextPos := addPos(posLL, Pos{0, -hintTextAreaHeight})
	ebitenutil.DebugPrintAt(screen, body.name, hintTextPos.x, hintTextPos.y)                        // todo: render text at the center of the hint area
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d", body.score), hintTextPos.x, hintTextPos.y+20) // todo: render text at the center of the hint area

	// get dimension of the body
	boxPos, boxSize := body.getBoundingBox()
	hintAreaSize.h += boxSize.h * scale

	// draw text pieces
	bodyPosUL := addPos(posLL, Pos{hintAreaSize.w/2 - scale*boxSize.w/2, -hintAreaSize.h})
	for _, bp := range body.bodyPieces {
		piece := getPieceByType(bp.pieceType)
		w, h := grid2ScrSize(float32(piece.size.w)/2, float32(piece.size.h)/2)

		op := &ebiten.DrawImageOptions{}
		imageScaleX, imageScaleY := piece.getScale()
		op.GeoM.Scale(imageScaleX, imageScaleY) // Apply scaling to the next piece
		op.GeoM.Translate(float64(-w), float64(-h))
		op.GeoM.Rotate(-getRotationTheta(bp.rotation))
		op.GeoM.Translate(float64(bodyPosUL.x+(bp.pos.x-boxPos.x)*scale), float64(bodyPosUL.y+(bp.pos.y-boxPos.y)*scale))
		op.GeoM.Translate(float64(w), float64(h))
		screen.DrawImage(piece.image, op)
	}

	return true, hintAreaSize
}

//
// ------------ background ------------
//
type BackgroundComp struct {
	state ComponentState
	pos Pos
	size Size
	drawOrder int
}

func NewBackground(pos Pos, size Size, drawOrder int) *BackgroundComp {
	return &BackgroundComp {
		pos: pos,
		size: size,
		drawOrder: drawOrder,
	}
}

func (b *BackgroundComp) activate(isActive bool) {
	b.state = StateActive
}

func (b *BackgroundComp) update(paused bool, frameCnt int) {
	if b.state == StateInactive {
		return
	}	
}

func (b *BackgroundComp) reset() {
}

func (b *BackgroundComp) draw(screen *ebiten.Image) {
	if b.state != StateInactive {
		// Fill only the game area with the background color
		vector.DrawFilledRect(screen, float32(b.pos.x), float32(b.pos.y), float32(b.size.w), float32(b.size.h), backgroundColor, false)
	}
}

func (b *BackgroundComp) getDrawOrder() int {
  return b.drawOrder
}

func (b *BackgroundComp) getState() ComponentState {
	return b.state
}