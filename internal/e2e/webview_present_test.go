//go:build !js

package e2e

import (
	"bytes"
	"image"
	"image/png"
	"math"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// screenPixel returns the color the display shows at (x, y), in DIPs, of
// a window of native UI, web views under it included, where a platform
// can read it.
func screenPixel(w *mygo.Window, x, y float64) (r, g, b uint8, ok bool) {
	if r, g, b, ok := screenColor(w, x, y); ok {
		return r, g, b, true
	}
	data, ok := surfaceOnScreen(w)
	if !ok || data == nil {
		return 0, 0, 0, false
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return 0, 0, 0, false
	}
	s := deviceScale(w)
	c := img.At(int(x*s), int(y*s))
	if !(image.Point{int(x * s), int(y * s)}).In(img.Bounds()) {
		return 0, 0, 0, false
	}
	cr, cg, cb, _ := c.RGBA()
	return uint8(cr >> 8), uint8(cg >> 8), uint8(cb >> 8), true
}

func reddish(r, g, b uint8) bool { return r > 180 && g < 80 && b < 80 }

// TestContentWindowWebViewRoundedScaled shows a red page in a rounded web
// view over a window of native UI with nothing painted around it, as a
// peek panel over a live window does, in a panel that then scales by
// half: the page shows inside its corners only, keeps its layout as it
// shows scaled, and takes clicks where it shows, at its own coordinates,
// but in the corners cut off.
func TestContentWindowWebViewRoundedScaled(t *testing.T) {
	var view *mygo.WebView
	var scale atomic.Value // float32
	scale.Store(float32(1))
	var frame atomic.Pointer[ui.Rect]
	w := newWindow(t, mygo.WindowOptions{Title: "Web view rounded", Width: 400, Height: 300, Content: ui.View(func(c *ui.Context) {
		c.Root().Background(ui.Transparent)
		k := scale.Load().(float32)
		ui.Box(c).Absolute().Left(60).Top(50).Size(280, 200).Scale(k, k).Children(func() {
			if view != nil {
				e := ui.WebView(c, view).Grow(1).Radius(40)
				e.Draw(func(*ui.Painter, ui.Rect) {
					// Where it shows, scaled.
					x0, y0 := e.LocalToWindow(0, 0)
					x1, y1 := e.LocalToWindow(280, 200)
					frame.Store(&ui.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0})
				})
			}
		})
	})})
	v, err := w.NewWebView(mygo.WebViewOptions{})
	if err != nil {
		t.Fatal(err)
	}
	w.Update(func() { view = v })
	v.Page().LoadHTML(`<!doctype html><body style="margin:0;height:100vh;background:#f00"
	onmousedown="window.downs.push([event.clientX, event.clientY])">
<script>
window.downs = []; window.resizes = 0
onresize = () => window.resizes++
requestAnimationFrame(() => requestAnimationFrame(() => { window.ready = true }))
</script></body>`, "")
	waitForPage(t, v.Page(), "window.ready")
	w.Focus()
	at := func(want float32) ui.Rect {
		t.Helper()
		eventually(t, "the web view placed", func() bool {
			w.Invalidate()
			f := frame.Load()
			return f != nil && math.Abs(float64(f.W-280*want)) < 0.5
		})
		return *frame.Load()
	}
	shows := func(what string, f ui.Rect) {
		t.Helper()
		if _, _, _, ok := screenPixel(w, float64(f.X+f.W/2), float64(f.Y+f.H/2)); !ok {
			return // the display cannot be read here
		}
		corners := [][2]float32{{f.X + 3, f.Y + 3}, {f.X + f.W - 4, f.Y + 3}, {f.X + f.W - 4, f.Y + f.H - 4}, {f.X + 3, f.Y + f.H - 4}}
		eventually(t, what+": the page inside its rounded corners, and not outside them", func() bool {
			if r, g, b, _ := screenPixel(w, float64(f.X+f.W/2), float64(f.Y+f.H/2)); !reddish(r, g, b) {
				return false
			}
			for _, c := range corners {
				if r, g, b, _ := screenPixel(w, float64(c[0]), float64(c[1])); reddish(r, g, b) {
					return false
				}
			}
			return true
		})
	}
	downs := func() [][2]float64 {
		d, _ := mygo.EvalAs[[][2]float64](v.Page(), "window.downs")
		return d
	}

	f := at(1)
	shows("unscaled", f)
	if !click(w, float64(f.X+f.W/2), float64(f.Y+f.H/2)) {
		t.Skip("click automation not available on this platform")
	}
	eventually(t, "the click on the page", func() bool { return len(downs()) == 1 })

	scale.Store(float32(0.5))
	f = at(0.5)
	shows("scaled by half", f)
	if n, _ := mygo.EvalAs[float64](v.Page(), "innerWidth"); n != 280 {
		t.Errorf("scaled by half, the page is %v wide: want 280, as unscaled", n)
	}
	// WebKitGTK zooms the page to show it scaled, which lays it out again
	// at the same width, telling it of a resize.
	if n, _ := mygo.EvalAs[float64](v.Page(), "window.resizes"); n != 0 && runtime.GOOS != "linux" {
		t.Errorf("scaled by half, the page resized %v times: want it only shown smaller", n)
	}
	// A click a quarter of the way in lands a quarter of the way into the
	// page, at its own coordinates.
	click(w, float64(f.X+f.W/4), float64(f.Y+f.H/4))
	eventually(t, "the click on the scaled page", func() bool { return len(downs()) == 2 })
	if d := downs()[1]; math.Abs(d[0]-70) > 2 || math.Abs(d[1]-50) > 2 {
		t.Errorf("the click a quarter of the way into the scaled page reached it at %v: want 70, 50", d)
	}
	// In a corner cut off, the click is the content's.
	click(w, float64(f.X+2), float64(f.Y+2))
	click(w, float64(f.X+f.W/2), float64(f.Y+f.H/2))
	eventually(t, "the click after the one in the corner", func() bool { return len(downs()) >= 3 })
	if n := len(downs()); n != 3 {
		t.Errorf("the page got %d clicks: want none in its corner cut off", n-2)
	}
}
