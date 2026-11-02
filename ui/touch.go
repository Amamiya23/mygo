package ui

import (
	"math"
	"runtime"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// A finger on a touch screen presses what it touches, as a mouse does,
// until it moves. Then it pans what scrolls under it along the way it went,
// as UIKit's scroll views and Android's do, and the press ends without a
// click; or, for what drags along that way, drags it. Past the content's
// ends the content follows less and less far, and goes back as the finger
// lifts (overscrollKind). Lifted while it moves, it flings the content on,
// which slows down as UIKit's decelerates
// (UIScrollView.DecelerationRate.normal), from frame to frame of the
// window's clock. Held still it long-presses.

// Desktops send no touches: what handles them is reached only where the
// platform has touch screens (touch_mobile.go), or a Tester touches, for
// apps of desktops not to link it.
var (
	touchInput func(rt *engine, ev platform.SurfaceEvent, x, y float32) bool
	touchFrame func(rt *engine, now time.Time)
	// insetRoot keeps the content out from under what covers the
	// window's edges, which only phones' do.
	insetRoot     func(rt *engine, root *node)
	extendScrolls func(rt *engine, root *node, h float32)
)

// enableTouch has the engines handle touches.
func enableTouch() {
	touchInput = (*engine).touchEvent
	touchFrame = (*engine).touchStep
	insetRoot = (*engine).padInsets
	extendScrolls = (*engine).extendUnderBottom
}

// overscroll is how content goes past its ends under a finger.
type overscroll uint8

const (
	// overscrollNone stops it at them, as in browsers, whose pages have
	// their own.
	overscrollNone overscroll = iota
	// overscrollBounce has it follow the finger less and less far, and
	// spring back, as iOS's scroll views.
	overscrollBounce
	// overscrollStretch does so over a shorter way, as Android's content
	// yields at its edges.
	overscrollStretch
)

// overscrollKind is the overscroll of the system's scroll views; tests set it.
var overscrollKind = func() overscroll {
	switch runtime.GOOS {
	case "ios":
		return overscrollBounce
	case "android":
		return overscrollStretch
	}
	return overscrollNone
}()

// touchSlop is how far a finger moves, in DIPs, before it pans rather
// than presses: UIKit's and Android's touch slop, about a millimeter.
const touchSlop = 8

// flingDecay is how much of its speed a fling keeps each millisecond,
// UIKit's normal deceleration.
const flingDecay = 0.998

// flingStop is the speed, in DIPs a second, under which a fling stops.
const flingStop = 20

// pressDelay is how long a finger on what scrolls waits before it presses,
// Android's tap timeout: a drag that starts sooner scrolls, pressing
// nothing, as UIKit's scroll views delay their content's touches.
const pressDelay = 100 * time.Millisecond

// longPressDelay is how long a finger stays before it long-presses, as
// UIKit's long press recognizer and Android's timeout: half a second.
const longPressDelay = 500 * time.Millisecond

// springOmega is how fast overscrolled content springs back, in radians a
// second: critically damped, as a spring of stiffness 100 and mass 0.5,
// which takes about half a second, as a UIScrollView's bounce does.
const springOmega = 14.14

// indicatorHold is how long a scroll indicator shows after the content last
// moved, and indicatorFade how long it takes to go.
const (
	indicatorHold = 700 * time.Millisecond
	indicatorFade = 250 * time.Millisecond
)

// touchPan is the finger on the screen.
type touchPan struct {
	down    bool
	panning bool
	// swallow drops what the finger does: it went down to stop a fling,
	// which takes no tap.
	swallow bool
	// waiting is a press held back until pressDelay passed (at), on what
	// scrolls, with its clicks.
	waiting bool
	at      time.Time
	clicks  int
	x0, y0  float32 // where it went down
	lx, ly  float32 // where it was last
	// target is what the finger pans, and axes the way it locked to.
	target uint64
	axes   Orientation
	// holds is the element waiting for a long press; dragging says a long
	// press gave the finger to the mouse's handling, to drag what it holds.
	holds    uint64
	dragging bool
	// samples are its last positions, for its speed as it lifts.
	samples [6]touchSample
	n       int
}

type touchSample struct {
	t    time.Time
	x, y float32
}

// fling is content going on after a finger lifted as it panned.
type fling struct {
	on     bool
	target uint64
	x, y   float32 // where, for what handles its input
	vx, vy float32 // DIPs a second
	at     time.Time
}

// touchEvent handles the pointer events of a finger, and reports whether
// it took the event, which the mouse's handling then skips.
func (rt *engine) touchEvent(ev platform.SurfaceEvent, x, y float32) bool {
	t := &rt.touch
	switch ev.Kind {
	case platform.PointerDown:
		*t = touchPan{down: true, x0: x, y0: y, lx: x, ly: y, at: rt.now()}
		t.sample(rt.now(), x, y)
		rt.catchBounces()
		if rt.flinging.on {
			// A finger stops content flinging, and taps nothing.
			rt.flinging.on = false
			t.swallow = true
			return true
		}
		t.holds = rt.longPressTarget(x, y)
		if rt.overScroller(x, y) {
			t.waiting, t.clicks = true, ev.Clicks
			rt.requestFrame()
			return true
		}
		rt.requestFrame()
		return false
	case platform.PointerMove:
		if !t.down {
			return false
		}
		if t.swallow {
			return true
		}
		if t.dragging {
			return false
		}
		if !t.panning {
			dx, dy := x-t.x0, y-t.y0
			if math.Hypot(float64(dx), float64(dy)) < touchSlop {
				return t.waiting
			}
			t.holds = 0
			axis := Vertical
			if abs32(dx) > abs32(dy) {
				axis = Horizontal
			}
			if rt.dragsPress(axis) {
				return t.waiting
			}
			t.panning = true
			if t.waiting {
				t.waiting = false // it never pressed
			} else {
				rt.cancelPress()
			}
			t.target, t.axes = rt.panTarget(t.x0, t.y0, axis)
			// From where it went down, past the slop: the content does
			// not jump.
			t.lx, t.ly = x, y
		}
		dx, dy := t.lx-x, t.ly-y
		t.lx, t.ly = x, y
		t.sample(rt.now(), x, y)
		rt.pan(t.target, t.axes, t.x0, t.y0, dx, dy, false)
		return true
	case platform.PointerUp:
		if !t.down {
			return false
		}
		was := *t
		t.down, t.waiting, t.holds = false, false, 0
		if was.swallow {
			return true
		}
		if was.dragging {
			return false
		}
		if was.waiting {
			// A tap quicker than the delay: it presses, and lets go.
			rt.pressNow(was)
			return false
		}
		if !was.panning {
			return false
		}
		t.sample(rt.now(), x, y)
		vx, vy := t.velocity(rt.now())
		s := rt.states[was.target]
		if was.axes&Horizontal == 0 {
			vx = 0
		}
		if was.axes&Vertical == 0 {
			vy = 0
		}
		if s != nil && s.overRaw != ([2]float32{}) {
			if r := s.refresh; r != nil && -s.over[1] >= refreshPull && !r.Refreshing {
				r.Refreshing = true
				s.refreshFired = true
			}
			// The content moves slower than the finger, on the band.
			rt.startBounce(s, vx*s.bandSlope(0), vy*s.bandSlope(1))
		} else if math.Hypot(float64(vx), float64(vy)) > flingStop && !rt.preferences().ReduceMotion {
			rt.flinging = fling{on: true, target: was.target, x: was.x0, y: was.y0, vx: vx, vy: vy, at: rt.now()}
			rt.requestFrame()
		}
		return true
	case platform.PointerLeave:
		return t.panning || t.swallow
	}
	return false
}

// overScroller reports whether a finger at x, y is on what scrolls, whose
// press waits for pressDelay, rather than on what it drags itself, as a
// slider, which it presses at once.
func (rt *engine) overScroller(x, y float32) bool {
	scrolls := false
	for _, id := range rt.hitChain(x, y) {
		s := rt.states[id]
		if s == nil {
			continue
		}
		if !scrolls && s.flags&(flagDraggable|flagTrackPointer) != 0 {
			return false
		}
		if s.flags&flagScrollY != 0 && s.contentH > float64(s.h) || s.flags&flagScrollX != 0 && s.contentW > float64(s.w) {
			scrolls = true
		}
	}
	return scrolls
}

// pressNow presses where a finger waited.
func (rt *engine) pressNow(t touchPan) {
	rt.pointerMove(t.x0, t.y0)
	rt.pointerDown(t.x0, t.y0, 0, rt.mods, max(t.clicks, 1))
}

func (t *touchPan) sample(now time.Time, x, y float32) {
	copy(t.samples[1:], t.samples[:len(t.samples)-1])
	t.samples[0] = touchSample{now, x, y}
	t.n = min(t.n+1, len(t.samples))
}

// velocity is the finger's speed over its last tenth of a second, in DIPs
// a second the content moves: against the finger.
func (t *touchPan) velocity(now time.Time) (float32, float32) {
	last := t.samples[0]
	first := last
	for i := 1; i < t.n; i++ {
		if now.Sub(t.samples[i].t) > 100*time.Millisecond {
			break
		}
		first = t.samples[i]
	}
	dt := float32(last.t.Sub(first.t).Seconds())
	if dt <= 0 || now.Sub(last.t) > 100*time.Millisecond {
		return 0, 0
	}
	return (first.x - last.x) / dt, (first.y - last.y) / dt
}

// dragsPress reports whether the element pressed takes a finger moving along
// axis itself rather than panning: what drags that way (Draggable), and what
// follows the pointer, as a slider, unless it lies across that way and
// something around it scrolls along it, for a slider not to stop a list.
func (rt *engine) dragsPress(axis Orientation) bool {
	p := rt.pressed
	if p == nil {
		return false
	}
	if p.flags&flagDraggable != 0 {
		if r := p.dragRec; r != nil && r.axes != 0 && r.axes&axis == 0 {
			return false
		}
		return true
	}
	if p.flags&flagTrackPointer != 0 {
		crosswise := axis == Vertical && p.w > p.h || axis == Horizontal && p.h > p.w
		if crosswise {
			if id, _ := rt.panTarget(rt.touch.x0, rt.touch.y0, axis); id != 0 {
				return false
			}
		}
		return true
	}
	return false
}

// cancelPress ends the press of a finger that pans, without a click.
func (rt *engine) cancelPress() {
	p := rt.pressed
	if p == nil {
		return
	}
	if p.input != nil {
		rt.deliver(p, InputEvent{Kind: InputPointerUp, Button: rt.pressButton, Mods: rt.mods, Clicks: rt.pressClicks})
	}
	if p.editor != nil {
		p.editor.release()
	}
	rt.selection.dragging = false
	p.pressed = false
	rt.pressed = nil
	rt.setHover(nil)
	rt.requestFrame()
}

// panTarget returns the innermost scroll container under x, y that scrolls
// along axis, which the finger keeps until it lifts, and the axes it pans:
// along axis, or both where the container scrolls both ways. It returns 0
// when there is none.
func (rt *engine) panTarget(x, y float32, axis Orientation) (uint64, Orientation) {
	for _, id := range rt.hitChain(x, y) {
		s := rt.states[id]
		if s == nil {
			continue
		}
		v := s.flags&flagScrollY != 0 && (s.contentH > float64(s.h) || s.refresh != nil)
		h := s.flags&flagScrollX != 0 && s.contentW > float64(s.w)
		switch {
		case v && h:
			return id, Horizontal | Vertical
		case v && axis == Vertical:
			return id, Vertical
		case h && axis == Horizontal:
			return id, Horizontal
		}
	}
	return 0, axis
}

// pan scrolls by dx, dy what a finger at x, y moves: the innermost element
// handling its input takes it, as it takes a touchpad's scrolling, or else
// the container the finger locked to, and past its ends its overscroll. It
// reports whether something moved.
func (rt *engine) pan(target uint64, axes Orientation, x, y, dx, dy float32, momentum bool) bool {
	chain := rt.appendHitChain(rt.chain[:0], x, y)
	rt.chain = chain[:0]
	ev := InputEvent{Kind: InputScroll, DX: dx, DY: dy, Mods: rt.mods, Precise: true, Momentum: momentum}
	for _, id := range chain {
		if s := rt.states[id]; s != nil && s.input != nil && s.flags&flagDisabled == 0 && rt.deliver(s, ev) {
			return true
		}
	}
	s := rt.states[target]
	if s == nil {
		return false
	}
	sx, sy := s.vector(x, y, dx, dy)
	if axes&Horizontal == 0 {
		sx = 0
	}
	if axes&Vertical == 0 {
		sy = 0
	}
	s.wheel = nil
	moved := false
	if sx != 0 && rt.panAxis(s, 0, sx, momentum) {
		moved = true
	}
	if sy != 0 && rt.panAxis(s, 1, sy, momentum) {
		moved = true
	}
	if moved {
		s.barsUntil = rt.now().Add(indicatorHold)
		rt.requestFrame()
	}
	return moved
}

// panAxis moves s along an axis, 0 across and 1 down: what its overscroll
// takes back first, then its content, and the rest into its overscroll, as a
// finger does. A fling's rest becomes a spring's speed instead (flingStep).
func (rt *engine) panAxis(s *state, axis int, d float32, momentum bool) bool {
	raw := &s.overRaw[axis]
	if *raw != 0 {
		if n := *raw + d; n*(*raw) > 0 {
			*raw = n
			s.setOver(axis)
			return true
		} else {
			d = n
			*raw = 0
			s.setOver(axis)
		}
	}
	x0, y0 := s.scrollX, s.scrollY
	if axis == 0 {
		scrollBy(s, d, 0)
	} else {
		scrollBy(s, 0, d)
	}
	var went float32
	if axis == 0 {
		went = float32(s.scrollX - x0)
		if s.rtl {
			went = -went
		}
	} else {
		went = float32(s.scrollY - y0)
	}
	left := d - went
	if abs32(left) < 1e-3 {
		return went != 0
	}
	if momentum || !s.canOverscroll(axis) {
		return went != 0
	}
	*raw = left
	s.setOver(axis)
	return true
}

// canOverscroll reports whether a finger takes s past its ends along axis.
func (s *state) canOverscroll(axis int) bool {
	if overscrollKind == overscrollNone {
		return false
	}
	if axis == 1 && s.refresh != nil {
		return true
	}
	if axis == 0 {
		return s.contentW > float64(s.w)
	}
	return s.contentH > float64(s.h)
}

// setOver shows the overscroll for how far the finger went past the end:
// less and less far, as on a rubber band as long as the view, half that for
// Android's stretch. Pull to refresh goes further, for the indicator.
func (s *state) setOver(axis int) {
	dim := s.w
	if axis == 1 {
		dim = s.h
	}
	if overscrollKind == overscrollStretch && s.refresh == nil {
		dim *= 0.25
	}
	s.over[axis] = Rubberband(s.overRaw[axis], 0, 0, dim)
}

// bandSlope is how far the content moves for each DIP the finger does, past
// the end along axis.
func (s *state) bandSlope(axis int) float32 {
	dim := s.w
	if axis == 1 {
		dim = s.h
	}
	if dim <= 0 {
		return 0
	}
	const c = 0.55
	k := abs32(s.overRaw[axis])*c/dim + 1
	return 1 / (k * k)
}

// rawFor is how far a finger goes past the end to show over, setOver's
// inverse, for a finger that catches content on its way back.
func (s *state) rawFor(axis int, over float32) float32 {
	dim := s.w
	if axis == 1 {
		dim = s.h
	}
	if overscrollKind == overscrollStretch && s.refresh == nil {
		dim *= 0.25
	}
	a := abs32(over)
	if dim <= 0 || a >= dim*0.999 {
		return over
	}
	const c = 0.55
	r := dim / c * (1/(1-a/dim) - 1)
	if over < 0 {
		return -r
	}
	return r
}

// startBounce has overscrolled content spring back to its ends from where it
// shows, at a speed in DIPs a second.
func (rt *engine) startBounce(s *state, vx, vy float32) {
	s.overRaw = [2]float32{}
	s.overV = [2]float32{vx, vy}
	if rt.preferences().ReduceMotion {
		s.over, s.overV = [2]float32{}, [2]float32{}
		rt.requestFrame()
		return
	}
	for _, id := range rt.bouncing {
		if id == s.id {
			rt.requestFrame()
			return
		}
	}
	rt.bouncing = append(rt.bouncing, s.id)
	rt.requestFrame()
	rt.animating = true // called building a frame, as well
}

// catchBounces has a finger going down hold what springs back where it shows.
func (rt *engine) catchBounces() {
	for _, id := range rt.bouncing {
		if s := rt.states[id]; s != nil {
			for axis := range s.over {
				s.overRaw[axis] = s.rawFor(axis, s.over[axis])
			}
			s.overV = [2]float32{}
		}
	}
	rt.bouncing = rt.bouncing[:0]
}

// bounceStep moves what springs back on by dt seconds: critically damped,
// from where it shows at its speed.
func (rt *engine) bounceStep(dt float32) {
	kept := rt.bouncing[:0]
	for _, id := range rt.bouncing {
		s := rt.states[id]
		if s == nil {
			continue
		}
		rest := s.refreshHold()
		going := false
		for axis := range s.over {
			to := float32(0)
			if axis == 1 {
				to = rest
			}
			x := s.over[axis] - to
			v := s.overV[axis]
			if x == 0 && v == 0 {
				continue
			}
			const w = springOmega
			b := v + w*x
			e := float32(math.Exp(float64(-w * dt)))
			nx := (x + b*dt) * e
			nv := (b - w*(x+b*dt)) * e
			if abs32(nx) < 0.1 && abs32(nv) < 5 {
				s.over[axis], s.overV[axis] = to, 0
				continue
			}
			s.over[axis], s.overV[axis] = nx+to, nv
			going = true
		}
		if going {
			kept = append(kept, id)
		}
	}
	rt.bouncing = kept
	if len(kept) > 0 {
		rt.animating = true
	}
}

// touchStep moves on what a finger left going, to the frame's time.
func (rt *engine) touchStep(now time.Time) {
	if t := &rt.touch; t.down {
		if t.waiting {
			if now.Sub(t.at) >= pressDelay {
				t.waiting = false
				rt.pressNow(*t)
			} else {
				rt.animating = true
			}
		}
		if t.holds != 0 && !t.panning {
			if now.Sub(t.at) >= longPressDelay {
				rt.longPress(t)
			} else {
				rt.animating = true
			}
		}
	}
	dt := float32(now.Sub(rt.bounceAt).Seconds())
	rt.bounceAt = now
	if len(rt.bouncing) > 0 {
		rt.animating = true
		if dt > 0 {
			rt.bounceStep(min(dt, 0.05))
		}
	}
	if !rt.barsFadeAt.IsZero() && now.Before(rt.barsFadeAt) {
		rt.animating = true
	}
	rt.flingStep(now)
}

// flingStep moves flinging content on to now, slowing it down, and stops
// it once it is slow or can go no further, where a spring takes its speed.
func (rt *engine) flingStep(now time.Time) {
	f := &rt.flinging
	if !f.on {
		return
	}
	ms := float32(now.Sub(f.at).Seconds() * 1000)
	f.at = now
	if ms <= 0 {
		rt.animating = true
		return
	}
	// The distance of a speed decaying by flingDecay each millisecond.
	k := float32(math.Pow(flingDecay, float64(ms)))
	scale := (1 - k) / -float32(math.Log(flingDecay)) / 1000
	dx, dy := f.vx*scale, f.vy*scale
	f.vx, f.vy = f.vx*k, f.vy*k
	axes := Orientation(0)
	if f.vx != 0 {
		axes |= Horizontal
	}
	if f.vy != 0 {
		axes |= Vertical
	}
	moved := rt.pan(f.target, axes, f.x, f.y, dx, dy, true)
	if !moved {
		if s := rt.states[f.target]; s != nil && !rt.preferences().ReduceMotion {
			vx, vy := float32(0), float32(0)
			if s.canOverscroll(0) {
				vx = f.vx
			}
			if s.canOverscroll(1) {
				vy = f.vy
			}
			if vx != 0 || vy != 0 {
				rt.startBounce(s, vx, vy)
			}
		}
		f.on = false
		return
	}
	if math.Hypot(float64(f.vx), float64(f.vy)) < flingStop {
		f.on = false
		return
	}
	rt.animating = true
}

// longPressTarget returns the innermost element under x, y that a finger held
// still does something to: what asked (LongPressed), a context menu, what a
// finger holds to drag, text it selects.
func (rt *engine) longPressTarget(x, y float32) uint64 {
	for _, id := range rt.hitChain(x, y) {
		s := rt.states[id]
		if s == nil || s.flags&flagDisabled != 0 {
			continue
		}
		if s.longPress || s.drags() || s.flags&(flagContextMenu|flagEditable|flagSelectable) != 0 {
			return id
		}
	}
	return 0
}

// longPress is a finger held still for longPressDelay.
func (rt *engine) longPress(t *touchPan) {
	s := rt.states[t.holds]
	t.holds = 0
	if s == nil {
		return
	}
	rt.haptic("medium")
	x, y := t.lx, t.ly
	switch {
	case s.longPress:
		s.longPressed = true
		// It takes the press: no click as the finger lifts.
		rt.cancelPress()
		t.swallow = true
		rt.requestFrame()
	case s.drags():
		// What it holds now follows it (dragFrom): moves go to the mouse's
		// handling.
		t.dragging = true
		rt.pointerMove(x, y)
	case s.flags&flagEditable != 0 || s.flags&flagSelectable != 0:
		// A word chosen, as a double click does.
		rt.cancelPress()
		rt.pointerDown(x, y, 0, rt.mods, 2)
		rt.pointerUp(0, 2)
		t.swallow = true
		rt.showEditMenu(s)
	default:
		// The context menu, as the secondary button does.
		rt.cancelPress()
		t.swallow = true
		rt.pointerDown(x, y, 1, rt.mods, 1)
		rt.pointerUp(1, 1)
	}
}

// haptic plays a feel of the system's, on a device that has them.
func (rt *engine) haptic(kind string) {
	if h, ok := rt.host.(interface{ haptic(string) }); ok {
		h.haptic(kind)
	}
}

// showEditMenu shows the system's edit menu at the selection of s (cut, copy,
// paste, select all), after a long press selected a word.
func (rt *engine) showEditMenu(s *state) {
	h, ok := rt.host.(interface{ showEditMenu(x, y float64) })
	if !ok {
		return
	}
	r := transformRect(s.world, Rect{X: s.x, Y: s.y, W: s.w, H: s.h})
	w := rt.toWindow(platform.RectF{X: float64(r.X + r.W/2), Y: float64(r.Y), W: 0, H: 0})
	h.showEditMenu(w.X, w.Y)
}

// lowMemory lets go of what the next frame makes again.
func (rt *engine) lowMemory() {
	rt.text.Forget()
	rt.requestFrame()
}

// LongPressed reports whether a finger held the element still for half a
// second since the last frame, on a touch screen. The press then clicks
// nothing as it lifts.
func (e *node) LongPressed() bool {
	if !e.hasState() {
		return false
	}
	e.st.longPress = true
	e.flags |= flagClickable
	if e.disabled() || !e.st.longPressed {
		return false
	}
	e.st.longPressed = false
	e.c.rt.consumed = true
	return true
}
