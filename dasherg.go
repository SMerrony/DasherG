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
	"runtime/pprof"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/glib"
	"github.com/mattn/go-gtk/gtk"
)

const (
	appID        = "uk.co.merrony.dasherg"
	appTitle     = "DasherG"
	appComment   = "A Data General DASHER terminal emulator"
	appCopyright = "Copyright ©2017 S.Merrony"
	appVersion   = "0.1 alpha"
	appWebsite   = "https://github.com/SMerrony/aosvs-tools"

	fontFile     = "D410-b-12.bdf"
	hostBuffSize = 2048
	keyBuffSize  = 200

	updateCrtNormal      = 1
	updateCrtBlink       = 2
	blinkPeriodMs        = 500
	statusUpdatePeriodMs = 500

	// gtkLoopMs = 5
)

var appAuthors = []string{"Stephen Merrony"}

var (
	status   *Status
	terminal *terminalT

	fromHostChan          = make(chan []byte, hostBuffSize)
	keyboardChan          = make(chan byte, keyBuffSize)
	localListenerStopChan = make(chan bool)
	updateCrtChan         = make(chan int, hostBuffSize)

	gc              *gdk.GC
	crt             *gtk.DrawingArea
	colormap        *gdk.Colormap
	offScreenPixmap *gdk.Pixmap
	win             *gtk.Window
	gdkWin          *gdk.Window

	//alive      bool
	blinkState bool

	green              *gdk.Color
	blinkTicker        = time.NewTicker(time.Millisecond * blinkPeriodMs)
	statusUpdateTicker = time.NewTicker(time.Millisecond * statusUpdatePeriodMs)

	// widgets needing global access
	fKeyLabs                                          [20][4]*gtk.Label
	serialConnectMenuItem, serialDisconnectMenuItem   *gtk.MenuItem
	networkConnectMenuItem, networkDisconnectMenuItem *gtk.MenuItem
	onlineLabel, hostLabel, emuStatusLabel            *gtk.Label
)

var (
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	hostFlag   = flag.String("host", "", "Host to connect with")
)

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

	gtk.Init(nil)
	green = gdk.NewColorRGB(0, 255, 0)
	bdfLoad(fontFile)
	go localListener()
	status = &Status{}
	status.setup()
	terminal = &terminalT{}
	terminal.setup(status, updateCrtChan)
	win = gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	setupWindow(win)
	win.ShowAll()
	gdkWin = crt.GetWindow()

	if *hostFlag != "" {
		hostParts := strings.Split(*hostFlag, ":")
		if len(hostParts) != 2 {
			log.Fatalf("-host flag must contain host and port separated by a colon, you passed %s", *hostFlag)
		}
		hostPort, err := strconv.Atoi(hostParts[1])
		if err != nil || hostPort < 0 {
			log.Fatalf("port must be a positive integer on -host flag, you passed %s", hostParts[1])
		}
		if openTelnetConn(hostParts[0], hostPort) {
			localListenerStopChan <- true
		}
	}
	go updateCrt(crt, terminal)

	gtk.Main()
}

func setupWindow(win *gtk.Window) {
	win.SetTitle(appTitle)
	win.Connect("destroy", func() {
		gtk.MainQuit()
		//os.Exit(0)
	})
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
	vbox.PackStart(buildFkeyMatrix(), false, false, 0)
	crt = buildCrt()
	//go updateCrt(crt, terminal)
	go terminal.run()
	go func() {
		for _ = range blinkTicker.C {
			updateCrtChan <- updateCrtBlink
		}
	}()
	vbox.PackStart(crt, false, false, 1)
	statusBox := buildStatusBox()
	vbox.PackEnd(statusBox, false, false, 0)
	win.Add(vbox)
}

func localListener() {
	key := make([]byte, 2)
	for {
		select {
		case kev := <-keyboardChan:
			key[0] = kev
			fromHostChan <- key
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
	loggingMenuItem.Connect("activate", toggleLogging)
	subMenu.Append(loggingMenuItem)

	sendFileMenuItem := gtk.NewMenuItemWithLabel("Send File")
	subMenu.Append(sendFileMenuItem)

	quitMenuItem := gtk.NewMenuItemWithLabel("Quit")
	subMenu.Append(quitMenuItem)
	quitMenuItem.Connect("activate", func() {
		pprof.StopCPUProfile()
		gtk.MainQuit()
		//os.Exit(0)
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
	var emuGroup *glib.SList
	emulationMenuItem.SetSubmenu(subMenu)
	d200MenuItem := gtk.NewRadioMenuItemWithLabel(emuGroup, "D200") //gtk.NewCheckMenuItemWithLabel("D200")
	d200MenuItem.Connect("activate", func() { status.emulation = d200 })
	emuGroup = d200MenuItem.GetGroup()
	subMenu.Append(d200MenuItem)
	d210MenuItem := gtk.NewRadioMenuItemWithLabel(emuGroup, "D210") //gtk.NewCheckMenuItemWithLabel("D210")
	d210MenuItem.Connect("activate", func() { status.emulation = d210 })
	emuGroup = d210MenuItem.GetGroup()
	subMenu.Append(d210MenuItem)
	d211MenuItem := gtk.NewRadioMenuItemWithLabel(emuGroup, "D211") //gtk.NewCheckMenuItemWithLabel("D211")
	d211MenuItem.Connect("activate", func() { status.emulation = d211 })
	emuGroup = d211MenuItem.GetGroup()
	subMenu.Append(d211MenuItem)
	resizeMenuItem := gtk.NewMenuItemWithLabel("Resize")
	subMenu.Append(resizeMenuItem)
	selfTestMenuItem := gtk.NewMenuItemWithLabel("Self-Test")
	subMenu.Append(selfTestMenuItem)
	selfTestMenuItem.Connect("activate", func() { terminal.selfTest(fromHostChan) })
	loadTemplateMenuItem := gtk.NewMenuItemWithLabel("Load Template")
	subMenu.Append(loadTemplateMenuItem)

	serialMenuItem := gtk.NewMenuItemWithLabel("Serial")
	menuBar.Append(serialMenuItem)
	subMenu = gtk.NewMenu()
	serialMenuItem.SetSubmenu(subMenu)
	serialConnectMenuItem = gtk.NewMenuItemWithLabel("Connect")
	subMenu.Append(serialConnectMenuItem)
	serialDisconnectMenuItem = gtk.NewMenuItemWithLabel("Disconnect")
	subMenu.Append(serialDisconnectMenuItem)
	serialDisconnectMenuItem.SetSensitive(false)

	networkMenuItem := gtk.NewMenuItemWithLabel("Network")
	menuBar.Append(networkMenuItem)
	subMenu = gtk.NewMenu()
	networkMenuItem.SetSubmenu(subMenu)
	networkConnectMenuItem = gtk.NewMenuItemWithLabel("Connect")
	subMenu.Append(networkConnectMenuItem)
	networkConnectMenuItem.Connect("activate", openNetDialog)
	networkDisconnectMenuItem = gtk.NewMenuItemWithLabel("Disconnect")
	subMenu.Append(networkDisconnectMenuItem)
	networkDisconnectMenuItem.Connect("activate", closeRemote)
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

func buildFkeyMatrix() *gtk.Table {
	fkeyMatrix := gtk.NewTable(5, 19, false)

	locPrBut := gtk.NewButtonWithLabel("LocPr")
	locPrBut.SetTooltipText("Local Print")
	locPrBut.SetCanFocus(false)
	//locPrBut.Connect("clicked", func() { keyboardChan <- dasherPrintScreen })
	fkeyMatrix.AttachDefaults(locPrBut, 0, 1, 0, 1)
	breakBut := gtk.NewButtonWithLabel("Break")
	breakBut.SetCanFocus(false)
	fkeyMatrix.AttachDefaults(breakBut, 0, 1, 4, 5)
	holdBut := gtk.NewButtonWithLabel("Hold")
	holdBut.SetCanFocus(false)
	fkeyMatrix.AttachDefaults(holdBut, 18, 19, 0, 1)
	erPgBut := gtk.NewButtonWithLabel("Er Pg")
	erPgBut.SetTooltipText("Erase Page")
	erPgBut.SetCanFocus(false)
	erPgBut.Connect("clicked", func() { keyboardChan <- dasherErasePage })
	fkeyMatrix.AttachDefaults(erPgBut, 18, 19, 1, 2)
	crBut := gtk.NewButtonWithLabel("CR")
	crBut.SetTooltipText("Carriage Return")
	crBut.SetCanFocus(false)
	crBut.Connect("clicked", func() { keyboardChan <- dasherCR })
	fkeyMatrix.AttachDefaults(crBut, 18, 19, 2, 3)
	erEOLBut := gtk.NewButtonWithLabel("ErEOL")
	erEOLBut.SetTooltipText("Erase to End Of Line")
	erEOLBut.SetCanFocus(false)
	erEOLBut.Connect("clicked", func() { keyboardChan <- dasherEraseEol })
	fkeyMatrix.AttachDefaults(erEOLBut, 18, 19, 3, 4)

	var fKeyButs [20]*gtk.Button

	for f := 1; f <= 5; f++ {
		fKeyButs[f] = gtk.NewButtonWithLabel(fmt.Sprintf("F%d", f))
		fKeyButs[f].SetCanFocus(false)
		fkeyMatrix.AttachDefaults(fKeyButs[f], uint(f), uint(f)+1, 4, 5)
		for l := 0; l < 4; l++ {
			fKeyLabs[f][l] = gtk.NewLabel("")
			frm := gtk.NewFrame("")
			frm.Add(fKeyLabs[f][l])
			fkeyMatrix.AttachDefaults(frm, uint(f), uint(f)+1, uint(l), uint(l)+1)
		}
	}
	csfLab := gtk.NewLabel("Ctrl-Shft")
	fkeyMatrix.AttachDefaults(csfLab, 6, 7, 0, 1)
	cfLab := gtk.NewLabel("Ctrl")
	fkeyMatrix.AttachDefaults(cfLab, 6, 7, 1, 2)
	sLab := gtk.NewLabel("Shift")
	fkeyMatrix.AttachDefaults(sLab, 6, 7, 2, 3)

	for f := 6; f <= 10; f++ {
		fKeyButs[f] = gtk.NewButtonWithLabel(fmt.Sprintf("F%d", f))
		fKeyButs[f].SetCanFocus(false)
		fkeyMatrix.AttachDefaults(fKeyButs[f], uint(f)+1, uint(f)+2, 4, 5)
		for l := 0; l < 4; l++ {
			fKeyLabs[f][l] = gtk.NewLabel("")
			frm := gtk.NewFrame("")
			frm.Add(fKeyLabs[f][l])
			fkeyMatrix.AttachDefaults(frm, uint(f)+1, uint(f)+2, uint(l), uint(l)+1)
		}
	}
	csfLab2 := gtk.NewLabel("")
	csfLab2.SetMarkup("<span size=\"x-small\">Ctrl-Shift</span>")
	fkeyMatrix.AttachDefaults(csfLab2, 12, 13, 0, 1)

	cfLab2 := gtk.NewLabel("Ctrl")
	fkeyMatrix.AttachDefaults(cfLab2, 12, 13, 1, 2)
	sLab2 := gtk.NewLabel("Shift")
	fkeyMatrix.AttachDefaults(sLab2, 12, 13, 2, 3)

	for f := 11; f <= 15; f++ {
		fKeyButs[f] = gtk.NewButtonWithLabel(fmt.Sprintf("F%d", f))
		fKeyButs[f].SetCanFocus(false)
		fkeyMatrix.AttachDefaults(fKeyButs[f], uint(f)+2, uint(f)+3, 4, 5)
		for l := 0; l < 4; l++ {
			fKeyLabs[f][l] = gtk.NewLabel("")
			frm := gtk.NewFrame("")
			frm.Add(fKeyLabs[f][l])
			fkeyMatrix.AttachDefaults(frm, uint(f)+2, uint(f)+3, uint(l), uint(l)+1)
		}
	}
	return fkeyMatrix
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
		networkConnectMenuItem.SetSensitive(false)
		networkDisconnectMenuItem.SetSensitive(true)
	}
}

func closeRemote() {
	closeTelnetConn()
	networkConnectMenuItem.SetSensitive(true)
	networkDisconnectMenuItem.SetSensitive(false)
	go localListener()
}

func toggleLogging() {
	if status.logging {
		status.logFile.Close()
		status.logging = false
	} else {
		fd := gtk.NewFileChooserDialog("DasherG Logfile", win, gtk.FILE_CHOOSER_ACTION_SAVE,
			"_Cancel", gtk.RESPONSE_CANCEL, "_Open", gtk.RESPONSE_ACCEPT)
		res := fd.Run()
		if res == gtk.RESPONSE_ACCEPT {
			filename := fd.GetFilename()
			status.logFile, err = os.Create(filename)
			if err != nil {
				log.Printf("WARNING: Could not open log file %s\n", filename)
				status.logging = false
			} else {
				status.logging = true
			}
		}
		fd.Destroy()
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
		gc.SetForeground(gc.GetColormap().AllocColorRGB(0, 65535, 0))
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

	// abusing this event to signal update actually required
	crt.Connect("client-event", drawCrt)
	return crt
}

func updateCrt(crt *gtk.DrawingArea, t *terminalT) {
	//var cIx int

	for {
		updateType := <-updateCrtChan
		switch updateType {
		case updateCrtBlink:
			blinkState = !blinkState
			fallthrough
		case updateCrtNormal:
			//glib.IdleAdd(func() { crt.Emit("custom-event") })
			crt.Emit("client-event")
		}
		//fmt.Println("updateCrt called")
	}
}

func drawCrt() {
	glib.IdleAdd(func() {
		var cIx int
		drawable := offScreenPixmap.GetDrawable()
		terminal.rwMutex.RLock()
		for line := 0; line < terminal.visibleLines; line++ {
			for col := 0; col < terminal.visibleCols; col++ {
				cIx = int(terminal.display[line][col].charValue)
				if cIx > 31 && cIx < 128 {
					switch {
					case terminal.blinkEnabled && blinkState && terminal.display[line][col].blink:
						drawable.DrawPixbuf(gc, bdfFont[32].pixbuf, 0, 0, col*charWidth, line*charHeight, charWidth, charHeight, 0, 0, 0)
					case terminal.display[line][col].reverse:
						drawable.DrawPixbuf(gc, bdfFont[cIx].reversePixbuf, 0, 0, col*charWidth, line*charHeight, charWidth, charHeight, 0, 0, 0)
					case terminal.display[line][col].dim:
						drawable.DrawPixbuf(gc, bdfFont[cIx].dimPixbuf, 0, 0, col*charWidth, line*charHeight, charWidth, charHeight, 0, 0, 0)
					default:
						drawable.DrawPixbuf(gc, bdfFont[cIx].pixbuf, 0, 0, col*charWidth, line*charHeight, charWidth, charHeight, 0, 0, 0)
					}
				}
				// underscore?
				if terminal.display[line][col].underscore {
					drawable.DrawLine(gc, col*charWidth, ((line+1)*charHeight)-1, (col+1)*charWidth, ((line+1)*charHeight)-1)
				}
			} // end for col
		} // end for line
		// draw the cursor - if on-screen
		if terminal.cursorX < terminal.visibleCols && terminal.cursorY < terminal.visibleLines {
			cIx := int(terminal.display[terminal.cursorY][terminal.cursorX].charValue)
			if terminal.display[terminal.cursorY][terminal.cursorX].reverse {
				drawable.DrawPixbuf(gc, bdfFont[cIx].pixbuf, 0, 0, terminal.cursorX*charWidth, terminal.cursorY*charHeight, charWidth, charHeight, 0, 0, 0)
			} else {
				drawable.DrawPixbuf(gc, bdfFont[cIx].reversePixbuf, 0, 0, terminal.cursorX*charWidth, terminal.cursorY*charHeight, charWidth, charHeight, 0, 0, 0)
			}
		}

		terminal.rwMutex.RUnlock()
		gdkWin.Invalidate(nil, false)
	})
}

func buildStatusBox() *gtk.HBox {
	statusBox := gtk.NewHBox(true, 2)

	onlineLabel = gtk.NewLabel("")
	olf := gtk.NewFrame("")
	olf.Add(onlineLabel)
	statusBox.Add(olf)

	hostLabel = gtk.NewLabel("")
	hlf := gtk.NewFrame("")
	hlf.Add(hostLabel)
	statusBox.Add(hlf)

	emuStatusLabel = gtk.NewLabel("")
	esf := gtk.NewFrame("")
	esf.Add(emuStatusLabel)
	statusBox.Add(esf)
	statusBox.Connect("client-event", updateStatusBox)

	go func() {
		for _ = range statusUpdateTicker.C {
			//glib.IdleAdd(func() { statusBox.Emit("property-notify-event") })
			statusBox.Emit("client-event")
		}
	}()
	return statusBox
}

// updateStatusBox to be run regularly - N.B. on the main thread!
func updateStatusBox() {
	glib.IdleAdd(func() {
		switch status.connected {
		case disconnected:
			onlineLabel.SetText("Local (Offline)")
			hostLabel.SetText("")
		case serialConnected:
			fmt.Println("Serial not yet supported")
			hostLabel.SetText("")
		case telnetConnected:
			onlineLabel.SetText("Online (Telnet)")
			hostLabel.SetText(status.remoteHost + ":" + status.remotePort)
		}
		emuStat := "D" + strconv.Itoa(status.emulation) + " (" +
			strconv.Itoa(status.visLines) + "x" + strconv.Itoa(status.visCols) + ")"

		emuStatusLabel.SetText(emuStat)
	})
}
