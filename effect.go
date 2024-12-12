package main

import (
	"math"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

//
// ------------ dialog ------------
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
	w.ageFrameCnt = 0

	if isActive && w.isBlocking {
		w.state = StateBlocking
	} else if isActive {
		w.state = StateActive
	} else {
		w.state = StateInactive
	}
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
	// length of wave i2 2 * Pi
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
	x := w.windowSize*distancePercent
	
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
