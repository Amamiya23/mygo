// Mobile shows what phones and tablets ask of native UI: gestures, sheets,
// wheels, swiped pages and rows, the keyboards of text fields, and what the
// system offers apps. Package ui/mobile builds most of it.
//
//	go run ./cmd/mygo dev -platform ios examples/mobile
//	go run ./cmd/mygo dev -platform android examples/mobile
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/mobile"
)

type app struct {
	win *mygo.Window
	tab int

	// Widgets.
	sheet, actions, drawer, picker, datePicker bool
	dark, wifi                                 bool
	size                                       int
	when                                       time.Time
	picked                                     string

	// Lists.
	refresh ui.RefreshState
	rows    []string
	nextRow int

	// Pages.
	pages mobile.PageState
	zoom  mobile.ZoomState

	// Keyboards.
	email, password, code, phone, url, amount, search, notes string
	searched                                                 string

	// System.
	bars   bool
	status string
}

func heading(c *ui.Context, s string) { ui.Text(c, s).FontSize(13).TextColor(c.Theme().TextMuted).PaddingX(16).Padding(16, 16, 4, 16) }

func (a *app) widgets(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).Padding(8, 0, 24, 0).Gap(16).Children(func() {
			mobile.Section(c, "Presented", func() {
				if mobile.ListRow(c.Key(1), mobile.RowOptions{Title: "Sheet", Subtitle: "Two detents; drag it", Disclosure: true}).Clicked() {
					a.sheet = true
				}
				if mobile.ListRow(c.Key(2), mobile.RowOptions{Title: "Action sheet", Disclosure: true}).Clicked() {
					a.actions = true
				}
				if mobile.ListRow(c.Key(3), mobile.RowOptions{Title: "Drawer", Disclosure: true}).Clicked() {
					a.drawer = true
				}
				if mobile.ListRow(c.Key(4), mobile.RowOptions{Title: "Wheel", Subtitle: a.pickedSize(), Disclosure: true}).Clicked() {
					a.picker = true
				}
				if mobile.ListRow(c.Key(5), mobile.RowOptions{Title: "Date", Subtitle: a.when.Format("2 Jan 2006, 15:04"), Disclosure: true}).Clicked() {
					a.datePicker = true
				}
			})
			mobile.Section(c, "Controls", func() {
				mobile.ListRow(c.Key(6), mobile.RowOptions{Title: "Wi-Fi", Trailing: func() { mobile.Switch(c, &a.wifi) }})
				mobile.ListRow(c.Key(7), mobile.RowOptions{Title: "Dark", Trailing: func() { mobile.Switch(c, &a.dark) }})
			})
			if a.picked != "" {
				ui.Textf(c, "Chose %s", a.picked).TextColor(t.TextMuted).PaddingX(16)
			}
		})
	})
	mobile.Sheet(c, &a.sheet, mobile.SheetOptions{Detents: []float32{0.45, 0.9}, Label: "Sheet"}, func() {
		ui.Scroll(c).Grow(1).Basis(0).Children(func() {
			ui.Column(c).Padding(8, 16).Gap(12).Children(func() {
				ui.Text(c, "A sheet").FontSize(22).Bold()
				ui.Text(c, "Drag the grabber between its two heights, or down and away. What it holds scrolls.").TextColor(t.TextMuted)
				for i := range 30 {
					ui.Textf(c.Key(i), "Row %d", i+1)
				}
			})
		})
	})
	choose := func(s string) func() { return func() { a.picked = s; c.Root() } }
	mobile.ActionSheet(c, &a.actions, mobile.ActionSheetOptions{Title: "Photo", Message: "What should happen to it?"},
		mobile.Action{Label: "Share", OnTap: choose("Share")},
		mobile.Action{Label: "Duplicate", OnTap: choose("Duplicate")},
		mobile.Action{Label: "Delete", Destructive: true, OnTap: choose("Delete")},
	)
	mobile.Drawer(c, &a.drawer, mobile.DrawerOptions{}, func() {
		ui.Column(c).Padding(16).Gap(12).Children(func() {
			ui.Text(c, "Navigation").FontSize(22).Bold()
			for _, s := range []string{"Inbox", "Sent", "Drafts", "Trash"} {
				mobile.ListRow(c.Key(s), mobile.RowOptions{Title: s})
			}
		})
	})
	mobile.Sheet(c, &a.picker, mobile.SheetOptions{Detents: []float32{0.38}, Label: "Size"}, func() {
		ui.Column(c).Padding(8, 16).Children(func() { mobile.Wheel(c, &a.size, sizes) })
	})
	mobile.Sheet(c, &a.datePicker, mobile.SheetOptions{Detents: []float32{0.6}, Label: "Date"}, func() {
		ui.Column(c).Padding(8, 16).Gap(8).Children(func() {
			mobile.DatePicker(c, &a.when)
			mobile.TimePicker(c, &a.when)
		})
	})
}

var sizes = []string{"Extra small", "Small", "Medium", "Large", "Extra large", "Huge"}

func (a *app) pickedSize() string { return sizes[a.size] }

func (a *app) lists(c *ui.Context) {
	t := c.Theme()
	l := ui.Scroll(c).Fill()
	if l.PullToRefresh(&a.refresh) {
		go func() {
			time.Sleep(1200 * time.Millisecond)
			a.win.Update(func() {
				a.nextRow++
				a.rows = append([]string{fmt.Sprintf("New row %d", a.nextRow)}, a.rows...)
				a.refresh.Refreshing = false
			})
		}()
	}
	l.Children(func() {
		ui.Column(c).Children(func() {
			ui.Text(c, "Pull down to refresh. Swipe a row aside. Hold one for its menu.").TextColor(t.TextMuted).Padding(12, 16)
			for i, r := range a.rows {
				i, r := i, r
				mobile.SwipeRow(c.Key(r), nil, mobile.SwipeOptions{
					Leading: []mobile.SwipeAction{{Label: "Pin"}},
					Trailing: []mobile.SwipeAction{{Label: "Delete", Destructive: true, OnTap: func() {
						a.rows = append(a.rows[:i:i], a.rows[i+1:]...)
					}}},
				}, func() {
					row := mobile.ListRow(c, mobile.RowOptions{Title: r, Subtitle: "Swipe me"})
					row.ContextMenu(func(m *ui.Menu) {
						m.Item("Copy")
						if m.Item("Delete").Chosen() {
							a.rows = append(a.rows[:i:i], a.rows[i+1:]...)
						}
					})
				})
				ui.Divider(c)
			}
		})
	})
}

var pageColors = []ui.Color{ui.Hex("#2563eb"), ui.Hex("#16a34a"), ui.Hex("#d97706"), ui.Hex("#db2777")}

func (a *app) pagesView(c *ui.Context) {
	ui.Column(c).Fill().Children(func() {
		mobile.PageView(c, &a.pages, len(pageColors), func(i int) {
			ui.Box(c).Fill().Background(pageColors[i]).Center().Children(func() {
				ui.Textf(c, "Page %d", i+1).FontSize(32).Bold().TextColor(ui.RGB(255, 255, 255))
			})
		}).Height(220).Shrink(0)
		mobile.PageDots(c, a.pages.Page, len(pageColors))
		heading(c, "Pinch to zoom, double tap, drag")
		mobile.Zoomable(c, &a.zoom, mobile.ZoomOptions{}, func() {
			ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
				const n = 8
				cw, ch := r.W/n, r.H/n
				for y := range n {
					for x := range n {
						col := ui.Hex("#dbeafe")
						if (x+y)%2 == 0 {
							col = ui.Hex("#1e3a8a")
						}
						p.Fill(ui.Rect{X: r.X + float32(x)*cw, Y: r.Y + float32(y)*ch, W: cw, H: ch}, col, 0)
					}
				}
			})
		}).FillWidth().Grow(1).Basis(0)
	})
}

func field(c *ui.Context, label string, value *string, style func(ui.Element) ui.Element) ui.Element {
	t := c.Theme()
	var in ui.Element
	ui.Column(c).Gap(4).Children(func() {
		ui.Text(c, label).FontSize(13).TextColor(t.TextMuted)
		in = style(ui.TextInput(c, value).Label(label))
	})
	return in
}

func (a *app) keyboards(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).Padding(16).Gap(12).Children(func() {
			field(c, "Email", &a.email, func(e ui.Element) ui.Element {
				return e.Keyboard(ui.KeyboardEmail).Autofill(ui.AutofillEmail).ReturnKey(ui.ReturnNext).Capitalize(ui.CapitalizeNone).Autocorrect(false)
			})
			field(c, "Password", &a.password, func(e ui.Element) ui.Element {
				return e.Password().Autofill(ui.AutofillPassword).ReturnKey(ui.ReturnNext)
			})
			field(c, "One-time code", &a.code, func(e ui.Element) ui.Element {
				return e.Keyboard(ui.KeyboardNumber).Autofill(ui.AutofillOneTimeCode)
			})
			field(c, "Phone", &a.phone, func(e ui.Element) ui.Element { return e.Keyboard(ui.KeyboardPhone).Autofill(ui.AutofillPhone) })
			field(c, "Website", &a.url, func(e ui.Element) ui.Element {
				return e.Keyboard(ui.KeyboardURL).ReturnKey(ui.ReturnGo).Autocorrect(false).Capitalize(ui.CapitalizeNone)
			})
			field(c, "Amount", &a.amount, func(e ui.Element) ui.Element { return e.Keyboard(ui.KeyboardDecimal) })
			s := field(c, "Search", &a.search, func(e ui.Element) ui.Element { return e.Keyboard(ui.KeyboardSearch).ReturnKey(ui.ReturnSearch) })
			if s.Submitted() {
				a.searched = a.search
			}
			if a.searched != "" {
				ui.Textf(c, "Searched for %s", a.searched).TextColor(t.TextMuted)
			}
			ui.Text(c, "Notes").FontSize(13).TextColor(t.TextMuted)
			ui.TextArea(c, &a.notes).Label("Notes").Capitalize(ui.CapitalizeSentences).Height(120)
		})
	})
}

func (a *app) system(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).Padding(8, 0, 24, 0).Gap(16).Children(func() {
			mobile.Section(c, "Feel", func() {
				for _, h := range []struct {
					name string
					play func()
				}{{"Selection", mygo.Haptics.Selection}, {"Light", mygo.Haptics.Light}, {"Heavy", mygo.Haptics.Heavy}, {"Success", mygo.Haptics.Success}, {"Error", mygo.Haptics.Error}} {
					if mobile.ListRow(c.Key(h.name), mobile.RowOptions{Title: h.name}).Clicked() {
						h.play()
					}
				}
			})
			mobile.Section(c, "The system", func() {
				if mobile.ListRow(c.Key("share"), mobile.RowOptions{Title: "Share", Subtitle: "Text and a link", Disclosure: true}).Clicked() {
					go func() {
						if err := mygo.Share(mygo.ShareOptions{Text: "Made with MyGo", URL: "https://github.com/egoist/mygo"}); err != nil {
							log.Print(err)
						}
					}()
				}
				row := mobile.ListRow(c.Key("bars"), mobile.RowOptions{Title: "Hide the system's bars", Trailing: func() { mobile.Switch(c.Key("bs"), &a.bars) }})
				_ = row
				if mobile.ListRow(c.Key("land"), mobile.RowOptions{Title: "Hold on its side"}).Clicked() {
					mygo.SetOrientations(mygo.OrientationLandscape)
				}
				if mobile.ListRow(c.Key("any"), mobile.RowOptions{Title: "Turn freely"}).Clicked() {
					mygo.SetOrientations(mygo.OrientationAny)
				}
				if mobile.ListRow(c.Key("status"), mobile.RowOptions{Title: "Network and battery", Subtitle: a.status}).Clicked() {
					go func() {
						n, b := mygo.Network.Status(), mygo.Power.Battery()
						a.win.Update(func() {
							a.status = fmt.Sprintf("online %v, metered %v; battery %.0f%%, charging %v", n.Online, n.Metered, b.Level*100, b.Charging)
						})
					}()
				}
			})
			ui.Text(c, "Native UI, drawn by MyGo.").FontSize(13).TextColor(t.TextMuted).PaddingX(16)
		})
	})
	mygo.SystemBars.SetHidden(a.bars)
}

func icon(path func(p *ui.Path, r ui.Rect)) func(p *ui.Painter, r ui.Rect, color ui.Color) {
	return func(p *ui.Painter, r ui.Rect, color ui.Color) {
		var l ui.Path
		path(&l, r)
		p.StrokePath(&l, 2, color)
	}
}

var tabs = []mobile.BarButton{
	{Label: "Widgets", Icon: icon(func(p *ui.Path, r ui.Rect) {
		p.MoveTo(r.X+3, r.Y+5).LineTo(r.X+21, r.Y+5).MoveTo(r.X+3, r.Y+12).LineTo(r.X+21, r.Y+12).MoveTo(r.X+3, r.Y+19).LineTo(r.X+21, r.Y+19)
	})},
	{Label: "Lists", Icon: icon(func(p *ui.Path, r ui.Rect) {
		p.MoveTo(r.X+4, r.Y+4).LineTo(r.X+20, r.Y+4).LineTo(r.X+20, r.Y+20).LineTo(r.X+4, r.Y+20).Close()
	})},
	{Label: "Pages", Icon: icon(func(p *ui.Path, r ui.Rect) {
		p.Circle(r.X+12, r.Y+12, 8)
	})},
	{Label: "Keyboards", Icon: icon(func(p *ui.Path, r ui.Rect) {
		p.MoveTo(r.X+2, r.Y+7).LineTo(r.X+22, r.Y+7).LineTo(r.X+22, r.Y+18).LineTo(r.X+2, r.Y+18).Close()
	})},
	{Label: "System", Icon: icon(func(p *ui.Path, r ui.Rect) {
		p.MoveTo(r.X+12, r.Y+3).LineTo(r.X+21, r.Y+21).LineTo(r.X+3, r.Y+21).Close()
	})},
}

func (a *app) view(c *ui.Context) {
	ui.Column(c).Fill().Children(func() {
		mobile.TopBar(c, mobile.TopBarOptions{Title: tabs[a.tab].Label})
		ui.Box(c).Grow(1).Basis(0).FillWidth().Children(func() {
			switch a.tab {
			case 0:
				a.widgets(c)
			case 1:
				a.lists(c)
			case 2:
				a.pagesView(c)
			case 3:
				a.keyboards(c)
			case 4:
				a.system(c)
			}
		})
		mobile.BottomBar(c, &a.tab, tabs...)
	})
}

func main() {
	a := &app{wifi: true, when: time.Now()}
	for i := range 30 {
		a.rows = append(a.rows, fmt.Sprintf("Row %d", i+1))
	}
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{Title: "Mobile", Width: 420, Height: 800, Content: ui.View(a.view)})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
