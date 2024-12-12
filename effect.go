package main

import (
	"log"
	"math"
	"math/rand"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

//
// ------------ WaveEffect ------------
//
type WaveEffectComp struct {
	state ComponentState
	isBlocking bool
	rect Rect
	center Pos
	pixelSize int
	pixelSize_2 int
	waveFill float64
	windowSize float64
	waveStart float64
	drawOrder int
	lifetimeFrameCnt int
	ageFrameCnt int
}

func NewWaveEffect(isBlocking bool, rect Rect, pixelSize int, waveFill float64, lifetimeFrameCnt int, drawOrder int) *WaveEffectComp {
	return &WaveEffectComp {
		isBlocking: isBlocking,
		rect: rect,
		pixelSize: pixelSize,
		pixelSize_2: pixelSize/2,
		waveFill: waveFill,
		lifetimeFrameCnt: lifetimeFrameCnt,
		drawOrder: drawOrder,
	}
}

func (w *WaveEffectComp) activate(isActive bool) {
	newState := StateInactive
	if isActive && w.isBlocking {
		newState = StateBlocking
	} else if isActive {
		newState = StateActive
	}

	if w.state != newState && isActive {
		log.Printf("Activating wave effect")
		w.ageFrameCnt = 0
	}

	w.state = newState
}

func (w *WaveEffectComp) reset() {
	w.state = StateInactive
}

func (w *WaveEffectComp) update(paused bool, frameCnt int) {
	if w.state == StateInactive {
		return
	}

	if w.ageFrameCnt < w.lifetimeFrameCnt {
		w.ageFrameCnt++
	} else {
		log.Printf("Inactivating wave effect")
		w.state = StateInactive
	}
}

func (w *WaveEffectComp) draw(screen *ebiten.Image) {
	if w.state != StateInactive {
		agePercent := float64(w.ageFrameCnt) / float64(w.lifetimeFrameCnt)

		w.windowSize = 2*math.Pi/w.waveFill
		w.waveStart = -2*math.Pi+(2*math.Pi+w.windowSize)*agePercent

		pLR := Pos{w.rect.pos.x+w.rect.size.w, w.rect.pos.y+w.rect.size.h}
		for x := w.rect.pos.x; x < pLR.x; x += w.pixelSize {
			for y := w.rect.pos.y; y < pLR.y; y += w.pixelSize {
				dx := float64(w.center.x - x - w.pixelSize_2)
				dy := float64(w.center.y - y - w.pixelSize_2)
				dstPcnt := math.Sqrt(dx*dx+dy*dy) / float64(w.rect.size.w)
				intensity := w.getWaveIntensity(dstPcnt, agePercent)
				if 0 < intensity {
					color := waveEffectColor
					color.A = intensity
					vector.DrawFilledRect(screen, float32(x), float32(y), float32(w.pixelSize), float32(w.pixelSize), color, false)
				}
			}
		}
	}
}

func (w *WaveEffectComp) getDrawOrder() int {
  return w.drawOrder
}

func (w *WaveEffectComp) getState() ComponentState {
	return w.state
}

func (w *WaveEffectComp) getWaveIntensity(distancePercent float64, agePercent float64) uint8 {
	x := w.windowSize*distancePercent

	// length of wave is 2Pi
	//
	//   |                         xxxx
	//   |                       xx    xx
	//   |                      x        x
	//   |                     x          x
	//   |                   xx            xx
	//   |_________________xx________________xx_______________________________
	//   |                |                    |             |             |
	//   0             waveStart        waveStart+2Pi        x        windowSize
	//

	if w.waveStart < x && x < w.waveStart+2*math.Pi {
		v := math.Min((1 + math.Sin(x-w.waveStart-math.Pi/2)) * 128, 255) // return in interval [0,25555]
		return uint8(v)
	}	else {
		return 0
	}
}

func (w *WaveEffectComp) setCenter(center Pos) {
	w.center = center
}

//
// ------------ RockEffect ------------
//
type RockState struct {
	pos    Pos // position displacement of the piece
	orient int // angular displacement of the piece
}

type RockEffectComp struct {
	state            ComponentState
	isBlocking       bool
	target           []*Piece    // pieces to be rocked
	lifetimeFrameCnt int         // length of the effect
	nofRock          int         // number of rock events during the effect
	drawOrder        int
	ageFrameCnt      int
	rockCnt          int         // should rock the target when this counter increases
	rockState        []RockState // displacement of actual rocking for each target piece
	completedCallback func()
}

func NewRockEffect(isBlocking bool, lifetimeFrameCnt int, nofRock int, drawOrder int) *RockEffectComp {
	return &RockEffectComp {
		isBlocking: isBlocking,
		lifetimeFrameCnt: lifetimeFrameCnt,
		nofRock: nofRock,
		drawOrder: drawOrder,
	}
}

func (r *RockEffectComp) activate(isActive bool) {
	newState := StateInactive
	if isActive && r.isBlocking {
		newState = StateBlocking
	} else if isActive {
		newState = StateActive
	}

	if r.state != newState && isActive {
		log.Printf("Activating rock effect")
		r.ageFrameCnt = 0
		r.rockCnt = -1
		r.rockState = make([]RockState, len(r.target), len(r.target))
	}

	r.state = newState
}

func (r *RockEffectComp) reset() {
	r.state = StateInactive
}

func (r *RockEffectComp) update(paused bool, frameCnt int) {
	if r.state == StateInactive {
		return
	}

	if r.ageFrameCnt < r.lifetimeFrameCnt {
		r.ageFrameCnt++
	} else {
		log.Printf("Inactivating rock effect")
		r.state = StateInactive

		if r.completedCallback != nil {
			r.completedCallback()
		}
	}
}

func (r *RockEffectComp) draw(screen *ebiten.Image) {
	if r.state != StateInactive {
		// determine the rock event
		rockCnt := (r.nofRock+1)*r.ageFrameCnt/r.lifetimeFrameCnt

		// check if time to rock
		if r.rockCnt < rockCnt {
			r.rockCnt = rockCnt

			// generate new random rock displacement
			for idx := 0; idx < len(r.rockState); idx++ {
				r.rockState[idx].pos.x = rand.Intn(scale/4) - scale/8
				r.rockState[idx].pos.y = rand.Intn(scale/4) - scale/8
				r.rockState[idx].orient = rand.Intn(21) - 10 // +- 10 deg
			}
		}

		for idx, piece := range r.target {
			op := &ebiten.DrawImageOptions{}
			piece.currentRotation += r.rockState[idx].orient
			applyRotationToPiece(op, piece)
			piece.currentRotation -= r.rockState[idx].orient
			op.GeoM.Translate(float64(r.rockState[idx].pos.x), float64(r.rockState[idx].pos.y))
			screen.DrawImage(piece.image, op)
		}
	}
}

func (r *RockEffectComp) getDrawOrder() int {
  return r.drawOrder
}

func (r *RockEffectComp) getState() ComponentState {
	return r.state
}

func (r *RockEffectComp) setTarget(target []*Piece) {
  r.target = target
}

func (r *RockEffectComp) setCompletedCallback(completed func()) {
  r.completedCallback = completed
}
