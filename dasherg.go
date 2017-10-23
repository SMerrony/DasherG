// dasherg.go

// Copyright (C) 2017  Steve Merrony

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package main

import (
	"fmt"
	"os"

	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/glib"
	"github.com/mattn/go-gtk/gtk"
)

const (
	appID        = "uk.co.merrony.dasherg"
	appTitle     = "DasherG"
	appCopyright = "2017 S.Merrony"
	appVersion   = "0.1 alpha"

	fontFile     = "D410-b-12.bdf"
	hostBuffSize = 2048
	keyBuffSize  = 200
)

var appAuthors = []string{"Stephen Merrony"}

var (
	status                 *Status
	terminal               *Terminal
	hostChan, keyboardChan chan []byte
	updateChan             chan bool

	gc              *gdk.GC
	crt             *gtk.DrawingArea
	colormap        *gdk.Colormap
	offScreenPixmap *gdk.Pixmap
	win             *gtk.Window
	gdkWin          *gdk.Window
)

func main() {
	glib.ThreadInit(nil)
	gdk.ThreadsInit()
	gdk.ThreadsEnter()
	gtk.Init(&os.Args)
	bdfLoad(fontFile)
	hostChan = make(chan []byte, hostBuffSize)
	keyboardChan = make(chan []byte, keyBuffSize)
	updateChan = make(chan bool, hostBuffSize)
	status = &Status{}
	status.setup()
	terminal = &Terminal{}
	terminal.setup(status, updateChan)
	win = gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	setupWindow(win)
	win.SetTitle(appTitle)
	win.Connect("destroy", gtk.MainQuit)
	win.ShowAll()
	gdkWin = crt.GetWindow()

	gtk.Main()
}

func setupWindow(win *gtk.Window) {
	win.SetTitle(appTitle)
	win.Connect("destroy", func(ctx *glib.CallbackContext) { gtk.MainQuit() }, "foo")
	win.SetDefaultSize(800, 600)
	vbox := gtk.NewVBox(false, 1)
	vbox.PackStart(buildMenu(), false, false, 0)
	//crt, crtBuff := buildCrt()
	crt = buildCrt()
	go updateCrt(crt, terminal)
	go hostListener()
	vbox.PackStart(crt, false, false, 1)
	statusBar := buildStatusBar()
	vbox.PackEnd(statusBar, false, false, 0)
	win.Add(vbox)
}

func hostListener() {
	for b := range hostChan {
		terminal.processHostData(b)
	}
}

func buildMenu() *gtk.MenuBar {
	menuBar := gtk.NewMenuBar()

	fileMenuItem := gtk.NewMenuItemWithLabel("File")
	menuBar.Append(fileMenuItem)
	subMenu := gtk.NewMenu()
	fileMenuItem.SetSubmenu(subMenu)
	loggingMenuItem := gtk.NewMenuItemWithLabel("Logging")
	subMenu.Append(loggingMenuItem)

	sendFileMenuItem := gtk.NewMenuItemWithLabel("Send File")
	subMenu.Append(sendFileMenuItem)

	quitMenuItem := gtk.NewMenuItemWithLabel("Quit")
	subMenu.Append(quitMenuItem)
	quitMenuItem.Connect("activate", func() { gtk.MainQuit() })

	viewMenuItem := gtk.NewMenuItemWithLabel("View")
	menuBar.Append(viewMenuItem)
	subMenu = gtk.NewMenu()
	viewMenuItem.SetSubmenu(subMenu)
	viewHistoryItem := gtk.NewMenuItemWithLabel("History")
	subMenu.Append(viewHistoryItem)

	emulationMenuItem := gtk.NewMenuItemWithLabel("Emulation")
	menuBar.Append(emulationMenuItem)
	subMenu = gtk.NewMenu()
	emulationMenuItem.SetSubmenu(subMenu)
	d200MenuItem := gtk.NewCheckMenuItemWithLabel("D200")
	subMenu.Append(d200MenuItem)
	d210MenuItem := gtk.NewCheckMenuItemWithLabel("D210")
	subMenu.Append(d210MenuItem)
	d211MenuItem := gtk.NewCheckMenuItemWithLabel("D211")
	subMenu.Append(d211MenuItem)
	resizeMenuItem := gtk.NewMenuItemWithLabel("Resize")
	subMenu.Append(resizeMenuItem)
	selfTestMenuItem := gtk.NewMenuItemWithLabel("Self-Test")
	subMenu.Append(selfTestMenuItem)
	selfTestMenuItem.Connect("activate", func() { terminal.selfTest(hostChan) })
	loadTemplateMenuItem := gtk.NewMenuItemWithLabel("Load Template")
	subMenu.Append(loadTemplateMenuItem)

	serialMenuItem := gtk.NewMenuItemWithLabel("Serial")
	menuBar.Append(serialMenuItem)
	subMenu = gtk.NewMenu()
	serialMenuItem.SetSubmenu(subMenu)
	serialConnectMenuItem := gtk.NewMenuItemWithLabel("Connect")
	subMenu.Append(serialConnectMenuItem)
	serialDisconnectMenuItem := gtk.NewMenuItemWithLabel("Disconnect")
	subMenu.Append(serialDisconnectMenuItem)
	serialDisconnectMenuItem.SetSensitive(false)

	networkMenuItem := gtk.NewMenuItemWithLabel("Network")
	menuBar.Append(networkMenuItem)
	subMenu = gtk.NewMenu()
	networkMenuItem.SetSubmenu(subMenu)
	networkConnectMenuItem := gtk.NewMenuItemWithLabel("Connect")
	subMenu.Append(networkConnectMenuItem)
	networkDisconnectMenuItem := gtk.NewMenuItemWithLabel("Disconnect")
	subMenu.Append(networkDisconnectMenuItem)
	networkDisconnectMenuItem.SetSensitive(false)

	helpMenuItem := gtk.NewMenuItemWithLabel("Help")
	menuBar.Append(helpMenuItem)
	subMenu = gtk.NewMenu()
	helpMenuItem.SetSubmenu(subMenu)
	onlineHelpMenuItem := gtk.NewMenuItemWithLabel("Online Help")
	subMenu.Append(onlineHelpMenuItem)
	aboutMenuItem := gtk.NewMenuItemWithLabel("About")
	subMenu.Append(aboutMenuItem)
	aboutMenuItem.Connect("activate", aboutDialog)

	return menuBar
}

func aboutDialog() {
	ad := gtk.NewAboutDialog()
	ad.SetName(appTitle)
	ad.SetAuthors(appAuthors)
	ad.SetVersion(appVersion)
	ad.SetCopyright(appCopyright)
	ad.Run()
	ad.Destroy()
}

// func buildCrt() (*gtk.TextView, *gtk.TextBuffer) {
// 	crt := gtk.NewTextView()
// 	crt.SetEditable(false)
// 	crt.ModifyFontEasy("Monospace 12px")
// 	crt.ModifyBG(gtk.STATE_NORMAL, gdk.NewColor("black"))
// 	crt.ModifyFG(gtk.STATE_NORMAL, gdk.NewColorRGB(0, 255, 0))
// 	crt.SetSizeRequest(80*10, 24*12)
// 	crtBuff := crt.GetBuffer()

// 	crtBuff.SetText("Hello")
// 	return crt, crtBuff
// }

func buildCrt() *gtk.DrawingArea {
	crt := gtk.NewDrawingArea()
	crt.SetSizeRequest(80*charWidth, 24*charHeight)

	crt.Connect("configure-event", func() {
		if offScreenPixmap != nil {
			offScreenPixmap.Unref()
		}
		//allocation := crt.GetAllocation()
		offScreenPixmap = gdk.NewPixmap(crt.GetWindow().GetDrawable(), 80*charWidth, 24*charHeight, 24)

		gc = gdk.NewGC(offScreenPixmap.GetDrawable())
		//gc.SetRgbFgColor(gdk.NewColor("black"))
		offScreenPixmap.GetDrawable().DrawRectangle(gc, true, 0, 0, -1, -1)
		//gc.SetRgbFgColor(gdk.NewColor("red"))
		//gc.SetRgbBgColor(gdk.NewColor("black"))
		fmt.Println("configure-event handled")
	})

	crt.Connect("expose-event", func() {
		// if pixmap == nil {
		// 	return
		// }
		gdkWin.GetDrawable().DrawDrawable(gc, offScreenPixmap.GetDrawable(), 0, 0, 0, 0, -1, -1)
		fmt.Println("expose-event handled")
	})
	return crt
}

func drawCrt() {
	fmt.Println("drawCrt called")
}

// func updateCrt(buff *gtk.TextBuffer, t *Terminal) {
// 	var s string
// 	for {
// 		_ = <-updateChan
// 		text := bytes.NewBufferString(s)
// 		for line := 0; line < t.visibleLines; line++ {
// 			for col := 0; col < t.visibleCols; col++ {
// 				text.WriteByte(t.display[line][col].charValue)
// 			}
// 			text.WriteByte('\n')
// 		}
// 		gdk.ThreadsEnter()
// 		buff.SetText(text.String())
// 		gdk.ThreadsLeave()
// 		//fmt.Printf("CRT text replaced with...\n%s\n", text.String())
// 	}
// }
func updateCrt(crt *gtk.DrawingArea, t *Terminal) {
	var cIx int
	gWin := crt.GetWindow()
	drawable := gWin.GetDrawable()
	gc := gdk.NewGC(drawable)
	_ = gc
	for {
		_ = <-updateChan
		gdk.ThreadsEnter()
		for line := 0; line < t.visibleLines; line++ {
			for col := 0; col < t.visibleCols; col++ {
				cIx = int(t.display[line][col].charValue)
				_ = cIx
				//drawable.DrawPixbuf(gc, bdfFont[cIx].pixbuf, 0, 0, col*charWidth, line*charHeight, charWidth, charHeight, 0, 0, 0)
			}
		}
		gdk.ThreadsLeave()
		fmt.Println("updateCrt called")
	}
}
func buildStatusBar() *gtk.Statusbar {
	statusBar := gtk.NewStatusbar()

	return statusBar
}
