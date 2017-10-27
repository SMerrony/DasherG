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
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"sync"
	"time"
	"unsafe"

	"github.com/mattn/go-gtk/glib"

	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/gtk"
)

const (
	appID        = "uk.co.merrony.dasherg"
	appTitle     = "DasherG"
	appComment   = "A Data General DASHER terminal emulator"
	appCopyright = "Copyright 2017 S.Merrony"
	appVersion   = "0.1 alpha"
	appWebsite   = "https://github.com/SMerrony/aosvs-tools"

	fontFile     = "D410-b-12.bdf"
	hostBuffSize = 2048
	keyBuffSize  = 200

	updateCrtNormal = 1
	updateCrtBlink  = 2
)

var appAuthors = []string{"Stephen Merrony"}

var (
	status                *Status
	terminal              *Terminal
	mainFuncChan          = make(chan func(), 8)
	hostChan              = make(chan []byte, hostBuffSize)
	keyboardChan          = make(chan byte, keyBuffSize)
	localListenerStopChan = make(chan bool)
	updateCrtChan         = make(chan int, hostBuffSize)

	gc              *gdk.GC
	crt             *gtk.DrawingArea
	colormap        *gdk.Colormap
	offScreenPixmap *gdk.Pixmap
	win             *gtk.Window
	gdkWin          *gdk.Window
	gtkMutex        sync.Mutex
	alive           bool
	blinkState      bool

	green       *gdk.Color
	blinkTicker = time.NewTicker(time.Millisecond * 500)
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	runtime.LockOSThread() // we need main to run on the main thread for Gtk to work properly
	gtk.Init(&os.Args)
	green = gdk.NewColorRGB(0, 255, 0)
	bdfLoad(fontFile)
	go localListener()
	status = &Status{}
	status.setup()
	terminal = &Terminal{}
	terminal.setup(status, updateCrtChan)
	win = gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	setupWindow(win)
	win.ShowAll()
	gdkWin = crt.GetWindow()

	for {
		select {
		case f := <-mainFuncChan:
			gtkMutex.Lock()
			f()
			gtkMutex.Unlock()
		default:
			gtkMutex.Lock()
			if gtk.EventsPending() {
				alive = gtk.MainIterationDo(false)
			}
			time.Sleep(10 * time.Millisecond)
			gtkMutex.Unlock()
			if !alive {
				return
			}
		}
	}
}
func doOnMainThread(f func()) {
	done := make(chan bool, 1)
	mainFuncChan <- func() {
		f()
		done <- true
	}
	<-done
}

func setupWindow(win *gtk.Window) {
	win.SetTitle(appTitle)
	win.Connect("destroy", func() { os.Exit(0) })
	win.SetDefaultSize(800, 600)
	go keyEventHandler()
	win.Connect("key-press-event", func(ctx *glib.CallbackContext) {
		arg := ctx.Args(0)
		keyPressEventChan <- *(**gdk.EventKey)(unsafe.Pointer(&arg))
	})
	win.Connect("key-release-event", func(ctx *glib.CallbackContext) {
		arg := ctx.Args(0)
		keyReleaseEventChan <- *(**gdk.EventKey)(unsafe.Pointer(&arg))
	})
	vbox := gtk.NewVBox(false, 1)
	vbox.PackStart(buildMenu(), false, false, 0)
	crt = buildCrt()
	go updateCrt(crt, terminal)
	go terminal.run()
	go func() {
		for _ = range blinkTicker.C {
			updateCrtChan <- updateCrtBlink
		}
	}()
	vbox.PackStart(crt, false, false, 1)
	statusBar := buildStatusBar()
	vbox.PackEnd(statusBar, false, false, 0)
	win.Add(vbox)
}

func localListener() {
	key := make([]byte, 2)
	for {
		select {
		case kev := <-keyboardChan:
			key[0] = kev
			hostChan <- key
		case <-localListenerStopChan:
			fmt.Println("localListener stopped")
			return
		}
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
	quitMenuItem.Connect("activate", func() {
		pprof.StopCPUProfile()
		os.Exit(0)
	})

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
	networkConnectMenuItem.Connect("activate", openNetDialog)
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
	ad.SetProgramName(appTitle)
	ad.SetAuthors(appAuthors)
	ad.SetVersion(appVersion)
	ad.SetCopyright(appCopyright)
	ad.SetWebsite(appWebsite)
	ad.Run()
	ad.Destroy()
}

func openNetDialog() {
	nd := gtk.NewDialog()
	nd.SetTitle("Telnet Host")

	ca := nd.GetVBox()

	hostLab := gtk.NewLabel("Host:")
	ca.PackStart(hostLab, true, true, 0)
	hostEntry := gtk.NewEntry()
	ca.PackStart(hostEntry, true, true, 0)
	portLab := gtk.NewLabel("Host:")
	ca.PackStart(portLab, true, true, 0)
	portEntry := gtk.NewEntry()
	ca.PackStart(portEntry, true, true, 0)

	nd.AddButton("Cancel", gtk.RESPONSE_CANCEL)
	nd.AddButton("OK", gtk.RESPONSE_OK)
	nd.ShowAll()
	response := nd.Run()

	if response == gtk.RESPONSE_OK {
		host := hostEntry.GetText()
		port, err := strconv.Atoi(portEntry.GetText())
		if err != nil || port < 0 || len(host) == 0 {
			ed := gtk.NewMessageDialog(win, gtk.DIALOG_DESTROY_WITH_PARENT, gtk.MESSAGE_ERROR,
				gtk.BUTTONS_CLOSE, "Must enter valid host and numeric port")
			ed.Run()
			ed.Destroy()
		} else {
			openRemote(host, port)
		}
	}

	nd.Destroy()
}

func openRemote(host string, port int) {
	if openTelnetConn(host, port) {
		localListenerStopChan <- true
	}
}

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
		offScreenPixmap.GetDrawable().DrawRectangle(gc, true, 0, 0, -1, -1)
		fmt.Println("configure-event handled")
	})

	crt.Connect("expose-event", func() {
		// if pixmap == nil {
		// 	return
		// }
		gdkWin.GetDrawable().DrawDrawable(gc, offScreenPixmap.GetDrawable(), 0, 0, 0, 0, -1, -1)
		//fmt.Println("expose-event handled")
	})
	return crt
}

func updateCrt(crt *gtk.DrawingArea, t *Terminal) {
	var cIx int

	for {
		updateType := <-updateCrtChan
		switch updateType {
		case updateCrtBlink:
			blinkState = !blinkState
			fallthrough
		case updateCrtNormal:
			doOnMainThread(func() {
				drawable := offScreenPixmap.GetDrawable()
				t.rwMutex.RLock()
				for line := 0; line < t.visibleLines; line++ {
					for col := 0; col < t.visibleCols; col++ {
						cIx = int(t.display[line][col].charValue)
						if cIx > 31 && cIx < 128 {
							switch {
							case t.blinkEnabled && blinkState && t.display[line][col].blink:
								drawable.DrawPixbuf(gc, bdfFont[32].pixbuf, 0, 0, col*charWidth, line*charHeight, charWidth, charHeight, 0, 0, 0)
							case t.display[line][col].reverse:
								drawable.DrawPixbuf(gc, bdfFont[cIx].reversePixbuf, 0, 0, col*charWidth, line*charHeight, charWidth, charHeight, 0, 0, 0)
							case t.display[line][col].dim:
								drawable.DrawPixbuf(gc, bdfFont[cIx].dimPixbuf, 0, 0, col*charWidth, line*charHeight, charWidth, charHeight, 0, 0, 0)
							default:
								drawable.DrawPixbuf(gc, bdfFont[cIx].pixbuf, 0, 0, col*charWidth, line*charHeight, charWidth, charHeight, 0, 0, 0)
							}
						}
						// underscore?
						if t.display[line][col].underscore {
							//gc.SetRgbFgColor(green)
							gc.SetForeground(gdk.NewColor("red"))
							gc.SetBackground(gdk.NewColor("blue"))
							drawable.DrawLine(gc, col*charWidth, ((line+1)*charHeight)-1, (col+1)*charWidth, ((line+1)*charHeight)-1)
							//drawable.DrawRectangle(gc, true, col*charWidth, ((line+1)*charHeight)-5, charWidth, 17)
						}
					} // end for col
				} // end for line
				// draw the cursor - if on-screen
				if t.cursorX < t.visibleCols && t.cursorY < t.visibleLines {
					cIx := int(t.display[t.cursorY][t.cursorX].charValue)
					if t.display[t.cursorY][t.cursorX].reverse {
						drawable.DrawPixbuf(gc, bdfFont[cIx].pixbuf, 0, 0, t.cursorX*charWidth, t.cursorY*charHeight, charWidth, charHeight, 0, 0, 0)
					} else {
						drawable.DrawPixbuf(gc, bdfFont[cIx].reversePixbuf, 0, 0, t.cursorX*charWidth, t.cursorY*charHeight, charWidth, charHeight, 0, 0, 0)
					}
				}

				t.rwMutex.RUnlock()
				gdkWin.Invalidate(nil, false)
			})
		}
		//fmt.Println("updateCrt called")
	}
}

func buildStatusBar() *gtk.Statusbar {
	statusBar := gtk.NewStatusbar()

	return statusBar
}
