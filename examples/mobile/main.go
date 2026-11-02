// Mobile shows what phones and tablets ask of native UI: the keyboards of
// text fields, gestures, sheets and the other patterns of touch screens.
//
//	go run ./cmd/mygo dev -platform ios examples/mobile
//	go run ./cmd/mygo dev -platform android examples/mobile
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type app struct {
	email, password, code, phone, url, amount, search, notes string
	submitted                                                string
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

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).Padding(16).Gap(12).Children(func() {
			ui.Text(c, "Keyboards").FontSize(22).Bold()
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
				a.submitted = a.search
			}
			if a.submitted != "" {
				ui.Textf(c, "Searched for %s", a.submitted).TextColor(t.TextMuted)
			}
			ui.Text(c, "Notes").FontSize(13).TextColor(t.TextMuted)
			ui.TextArea(c, &a.notes).Label("Notes").Capitalize(ui.CapitalizeSentences).Height(120)
		})
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Mobile", Width: 420, Height: 800, Content: ui.View(a.view)})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
