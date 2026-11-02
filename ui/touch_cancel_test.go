package ui

import (
	"testing"
	"time"
)

// A finger the system takes away throws nothing, however fast it went.
func TestTouchCancelThrowsNothing(t *testing.T) {
	defer func(k overscroll) { overscrollKind = k }(overscrollKind)
	overscrollKind = overscrollBounce
	var s ScrollState
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Fill().TrackScroll(&s).Children(func() {
			coreBox(c).Height(5000).Shrink(0)
		})
	}, 200, 200)
	tt.Frame()
	tt.TouchDown(100, 150)
	tt.Advance(time.Millisecond)
	tt.TouchMove(100, 140)
	tt.Advance(16 * time.Millisecond)
	tt.TouchMove(100, 60)
	tt.Frame()
	at := s.Y
	if at <= 0 {
		t.Fatalf("panning: at %v", at)
	}
	tt.TouchCancel()
	for range 30 {
		tt.Advance(16 * time.Millisecond)
	}
	if s.Y != at {
		t.Errorf("taken away at %v, it went on to %v", at, s.Y)
	}
}

// Nor does it go back, however far it went.
func TestTouchCancelStaysOnPage(t *testing.T) {
	defer func(k overscroll) { overscrollKind = k }(overscrollKind)
	overscrollKind = overscrollBounce
	r := NewRouter("/a")
	tt := coreNewTester(func(c *context) {
		r.coreView(c, func(rt *Route) { coreBox(c).Fill() })
	}, 400, 400)
	r.Push("/a/b")
	tt.Frame()
	tt.Advance(time.Second)
	tt.TouchDown(10, 200)
	tt.Advance(time.Millisecond)
	tt.TouchMove(30, 200)
	tt.Frame()
	tt.Advance(16 * time.Millisecond)
	tt.TouchMove(350, 200)
	tt.Frame()
	tt.TouchCancel()
	for range 60 {
		tt.Advance(16 * time.Millisecond)
	}
	if r.Path() != "/a/b" || r.swipe != 0 || r.settling || r.swiping {
		t.Errorf("taken away: at %s, swipe %v, settling %v, swiping %v", r.Path(), r.swipe, r.settling, r.swiping)
	}
}

// Nor does it refresh, however far it pulled.
func TestTouchCancelRefreshesNothing(t *testing.T) {
	defer func(k overscroll) { overscrollKind = k }(overscrollKind)
	overscrollKind = overscrollBounce
	var st RefreshState
	refreshes := 0
	tt := coreNewTester(func(c *context) {
		e := coreScroll(c).Fill()
		if e.PullToRefresh(&st) {
			refreshes++
		}
		e.Children(func() {
			coreBox(c).Height(1000).Shrink(0)
		})
	}, 200, 200)
	tt.Frame()
	tt.TouchDown(100, 10)
	tt.Advance(time.Millisecond)
	for y := float32(30); y <= 190; y += 20 {
		tt.TouchMove(100, y)
		tt.Frame()
		tt.Advance(16 * time.Millisecond)
	}
	tt.TouchCancel()
	for range 60 {
		tt.Advance(16 * time.Millisecond)
	}
	if refreshes != 0 {
		t.Errorf("refreshed %d times", refreshes)
	}
}

// Going back as the system's gesture ends leaves the page where it belongs.
func TestBackGestureCommits(t *testing.T) {
	r := NewRouter("/a")
	tt := coreNewTester(func(c *context) {
		r.coreView(c, func(rt *Route) { coreBox(c).Fill() })
	}, 400, 400)
	r.Push("/a/b")
	tt.Frame()
	tt.Advance(time.Second)
	rt := tt.rt
	rt.backGesture("started", 0)
	rt.backGesture("progress", 0.5)
	tt.Frame()
	if !r.swiping || r.swipe == 0 {
		t.Fatalf("following: swiping %v, %v", r.swiping, r.swipe)
	}
	r.Back()
	for range 60 {
		tt.Advance(16 * time.Millisecond)
	}
	if r.Path() != "/a" || r.swiping || r.swipe != 0 || r.settling || rt.swipeRouter != nil {
		t.Errorf("gone back: at %s, swiping %v, swipe %v, settling %v, gesture of %v", r.Path(), r.swiping, r.swipe, r.settling, rt.swipeRouter)
	}
}

// A tap while something springs back doesn't leave it stretched.
func TestTapCatchingBounceLetsItGo(t *testing.T) {
	defer func(k overscroll) { overscrollKind = k }(overscrollKind)
	overscrollKind = overscrollBounce
	var top float32
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Fill().Children(func() {
			top = coreBox(c).Height(1000).Shrink(0).st.y
		})
	}, 200, 200)
	tt.Frame()
	tt.TouchDown(100, 50)
	tt.Advance(time.Millisecond)
	tt.TouchMove(100, 60)
	tt.Advance(16 * time.Millisecond)
	tt.TouchMove(100, 150)
	tt.Frame()
	tt.Advance(200 * time.Millisecond)
	tt.TouchUp(100, 150)
	tt.Advance(32 * time.Millisecond)
	tt.Frame()
	if top <= 0 {
		t.Fatalf("springing back: at %v", top)
	}
	tt.TouchDown(100, 100)
	tt.Advance(16 * time.Millisecond)
	tt.TouchUp(100, 100)
	for range 200 {
		tt.Advance(16 * time.Millisecond)
	}
	tt.Frame()
	if top != 0 {
		t.Errorf("after the tap, it stays at %v", top)
	}
}

// Something else in the place of what pulled to refresh doesn't refresh it, as
// two tabs at the same place of the tree.
func TestPullToRefreshGoesWithIt(t *testing.T) {
	defer func(k overscroll) { overscrollKind = k }(overscrollKind)
	overscrollKind = overscrollBounce
	var r RefreshState
	pulls := true
	tt := coreNewTester(func(c *context) {
		e := coreScroll(c).Fill()
		if pulls {
			e.PullToRefresh(&r)
		}
		e.Children(func() { coreBox(c).Height(50).Shrink(0) })
	}, 200, 200)
	tt.Frame()
	pulls = false
	tt.Frame()
	tt.TouchDown(100, 20)
	tt.Advance(time.Millisecond)
	for y := float32(40); y <= 190; y += 15 {
		tt.TouchMove(100, y)
		tt.Advance(16 * time.Millisecond)
	}
	tt.Advance(200 * time.Millisecond)
	tt.TouchUp(100, 190)
	tt.Frame()
	if r.Refreshing {
		t.Error("a pull on what took its place refreshed it")
	}
}
