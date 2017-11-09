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
	"io/ioutil"
	"log"
	"os/exec"
	"runtime"
	// _ "net/http/pprof"
	"os"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"strings"
	"unsafe"

	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/gdkpixbuf"
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
	helpURL      = "https://github.com/SMerrony/aosvs-tools/tree/master/dasherg"
	iconFile     = "DGlogoOrange.png"
	hostBuffSize = 2048
	keyBuffSize  = 200

	updateCrtNormal = 1 // crt is 'dirty' and needs updating
	updateCrtBlink  = 2 // crt blink state needs flipping
	blinkPeriodMs   = 500
	// crtRefreshMs influences the responsiveness of the display. 50ms = 20Hz or 20fps
	crtRefreshMs         = 50
	statusUpdatePeriodMs = 500

	zoomLarge = iota
	zoomNormal
	zoomSmaller
	zoomTiny
)

var appAuthors = []string{"Stephen Merrony"}

var (
	terminal *terminalT

	fromHostChan          = make(chan []byte, hostBuffSize)
	keyboardChan          = make(chan byte, keyBuffSize)
	localListenerStopChan = make(chan bool)
	updateCrtChan         = make(chan int, hostBuffSize)

	gc              *gdk.GC
	crt             *gtk.DrawingArea
	zoom            = zoomNormal
	colormap        *gdk.Colormap
	offScreenPixmap *gdk.Pixmap
	win             *gtk.Window
	gdkWin          *gdk.Window

	// widgets needing global access
	fKeyLabs                                             [20][4]*gtk.Label
	serialConnectMenuItem, serialDisconnectMenuItem      *gtk.MenuItem
	networkConnectMenuItem, networkDisconnectMenuItem    *gtk.MenuItem
	onlineLabel, hostLabel, loggingLabel, emuStatusLabel *gtk.Label
)

var (
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	cputrace   = flag.String("cputrace", "", "write trace to file")
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

	if *cputrace != "" {
		f, err := os.Create(*cputrace)
		if err != nil {
			log.Fatal(err)
		}
		_ = trace.Start(f)
		defer trace.Stop()
	}

	gtk.Init(nil)
	bdfLoad(fontFile, zoomNormal)
	go localListener()
	terminal = new(terminalT)
	terminal.setup(updateCrtChan)
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
	glib.TimeoutAdd(crtRefreshMs, func() bool {
		drawCrt()
		return true
	})

	gtk.Main()
}

func setupWindow(win *gtk.Window) {
	win.SetTitle(appTitle)
	win.Connect("destroy", func() {
		gtk.MainQuit()
		//os.Exit(0)
	})
	//win.SetDefaultSize(800, 600)
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
	go terminal.run()
	glib.TimeoutAdd(blinkPeriodMs, func() bool {
		updateCrtChan <- updateCrtBlink
		return true
	})
	vbox.PackStart(crt, false, false, 1)
	statusBox := buildStatusBox()
	vbox.PackEnd(statusBox, false, false, 0)
	win.Add(vbox)
	win.SetIconFromFile(iconFile)
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

	sendFileMenuItem := gtk.NewMenuItemWithLabel("Send (Text) File")
	sendFileMenuItem.Connect("activate", sendFile)
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
	d200MenuItem.Connect("activate", func() { terminal.emulation = d200 })
	emuGroup = d200MenuItem.GetGroup()
	subMenu.Append(d200MenuItem)
	d210MenuItem := gtk.NewRadioMenuItemWithLabel(emuGroup, "D210") //gtk.NewCheckMenuItemWithLabel("D210")
	if terminal.emulation == d210 {
		d210MenuItem.SetActive(true)
	}
	d210MenuItem.Connect("activate", func() { terminal.emulation = d210 })
	emuGroup = d210MenuItem.GetGroup()
	subMenu.Append(d210MenuItem)
	d211MenuItem := gtk.NewRadioMenuItemWithLabel(emuGroup, "D211") //gtk.NewCheckMenuItemWithLabel("D211")
	if terminal.emulation == d211 {
		d211MenuItem.SetActive(true)
	}
	d211MenuItem.Connect("activate", func() { terminal.emulation = d211 })
	emuGroup = d211MenuItem.GetGroup()
	subMenu.Append(d211MenuItem)
	resizeMenuItem := gtk.NewMenuItemWithLabel("Resize")
	resizeMenuItem.Connect("activate", resizeDialog)
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
	networkDisconnectMenuItem.Connect("activate", closeTelnetConn)
	networkDisconnectMenuItem.SetSensitive(false)

	helpMenuItem := gtk.NewMenuItemWithLabel("Help")
	menuBar.Append(helpMenuItem)
	subMenu = gtk.NewMenu()
	helpMenuItem.SetSubmenu(subMenu)
	onlineHelpMenuItem := gtk.NewMenuItemWithLabel("Online Help")
	onlineHelpMenuItem.Connect("activate", func() { openBrowser(helpURL) })
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
	csfLab2 := gtk.NewLabel("Ctrl-Shft")
	//csfLab2.SetMarkup("<span size=\"x-small\">Ctrl-Shift</span>")
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

	var keyEv gdk.EventKey
	keyEv.Window = unsafe.Pointer(win)
	keyEv.Type = 9 // gdk.KEY_RELEASE
	fKeyButs[1].Connect("clicked", func() {
		keyEv.Keyval = gdk.KEY_F1
		keyReleaseEventChan <- &keyEv
	})
	fKeyButs[2].Connect("clicked", func() {
		keyEv.Keyval = gdk.KEY_F2
		keyReleaseEventChan <- &keyEv
	})
	fKeyButs[3].Connect("clicked", func() {
		keyEv.Keyval = gdk.KEY_F3
		keyReleaseEventChan <- &keyEv
	})
	fKeyButs[4].Connect("clicked", func() {
		keyEv.Keyval = gdk.KEY_F4
		keyReleaseEventChan <- &keyEv
	})
	fKeyButs[5].Connect("clicked", func() {
		keyEv.Keyval = gdk.KEY_F5
		keyReleaseEventChan <- &keyEv
	})
	fKeyButs[6].Connect("clicked", func() {
		keyEv.Keyval = gdk.KEY_F6
		keyReleaseEventChan <- &keyEv
	})
	fKeyButs[7].Connect("clicked", func() {
		keyEv.Keyval = gdk.KEY_F7
		keyReleaseEventChan <- &keyEv
	})
	fKeyButs[8].Connect("clicked", func() {
		keyEv.Keyval = gdk.KEY_F8
		keyReleaseEventChan <- &keyEv
	})
	fKeyButs[9].Connect("clicked", func() {
		keyEv.Keyval = gdk.KEY_F9
		keyReleaseEventChan <- &keyEv
	})
	fKeyButs[10].Connect("clicked", func() {
		keyEv.Keyval = gdk.KEY_F10
		keyReleaseEventChan <- &keyEv
	})
	fKeyButs[11].Connect("clicked", func() {
		keyEv.Keyval = gdk.KEY_F11
		keyReleaseEventChan <- &keyEv
	})
	fKeyButs[12].Connect("clicked", func() {
		keyEv.Keyval = gdk.KEY_F12
		keyReleaseEventChan <- &keyEv
	})
	fKeyButs[13].Connect("clicked", func() {
		keyEv.Keyval = gdk.KEY_F13
		keyReleaseEventChan <- &keyEv
	})
	fKeyButs[14].Connect("clicked", func() {
		keyEv.Keyval = gdk.KEY_F14
		keyReleaseEventChan <- &keyEv
	})
	fKeyButs[15].Connect("clicked", func() {
		keyEv.Keyval = gdk.KEY_F15
		keyReleaseEventChan <- &keyEv
	})

	return fkeyMatrix
}

func aboutDialog() {
	ad := gtk.NewAboutDialog()
	ad.SetProgramName(appTitle)
	ad.SetAuthors(appAuthors)
	ad.SetIconFromFile(iconFile)
	logo, _ := gdkpixbuf.NewPixbufFromFile(iconFile)
	ad.SetLogo(logo)
	ad.SetVersion(appVersion)
	ad.SetCopyright(appCopyright)
	ad.SetWebsite(appWebsite)
	ad.Run()
	ad.Destroy()
}

func resizeDialog() {
	rd := gtk.NewDialog()
	rd.SetTitle("Resize Terminal")
	vb := rd.GetVBox()
	table := gtk.NewTable(3, 3, false)
	cLab := gtk.NewLabel("Columns")
	table.AttachDefaults(cLab, 0, 1, 0, 1)
	colsCombo := gtk.NewComboBoxText()
	colsCombo.AppendText("80")
	colsCombo.AppendText("81")
	colsCombo.AppendText("120")
	colsCombo.AppendText("132")
	colsCombo.AppendText("135")
	switch terminal.visibleCols {
	case 80:
		colsCombo.SetActive(0)
	case 81:
		colsCombo.SetActive(1)
	case 120:
		colsCombo.SetActive(2)
	case 132:
		colsCombo.SetActive(3)
	case 135:
		colsCombo.SetActive(4)
	}
	table.AttachDefaults(colsCombo, 1, 2, 0, 1)
	lLab := gtk.NewLabel("Lines")
	table.AttachDefaults(lLab, 0, 1, 1, 2)
	linesCombo := gtk.NewComboBoxText()
	linesCombo.AppendText("24")
	linesCombo.AppendText("25")
	linesCombo.AppendText("36")
	linesCombo.AppendText("48")
	linesCombo.AppendText("66")
	terminal.rwMutex.RLock()
	switch terminal.visibleLines {
	case 24:
		linesCombo.SetActive(0)
	case 25:
		linesCombo.SetActive(1)
	case 36:
		linesCombo.SetActive(2)
	case 48:
		linesCombo.SetActive(3)
	case 66:
		linesCombo.SetActive(4)
	}
	terminal.rwMutex.RUnlock()
	table.AttachDefaults(linesCombo, 1, 2, 1, 2)
	zLab := gtk.NewLabel("Zoom")
	table.AttachDefaults(zLab, 0, 1, 2, 3)
	zoomCombo := gtk.NewComboBoxText()
	zoomCombo.AppendText("Large")
	zoomCombo.AppendText("Normal")
	zoomCombo.AppendText("Smaller")
	zoomCombo.AppendText("Tiny")
	switch zoom {
	case zoomLarge:
		zoomCombo.SetActive(0)
	case zoomNormal:
		zoomCombo.SetActive(1)
	case zoomSmaller:
		zoomCombo.SetActive(2)
	case zoomTiny:
		zoomCombo.SetActive(3)
	}
	table.AttachDefaults(zoomCombo, 1, 2, 2, 3)
	vb.PackStart(table, false, false, 1)

	rd.AddButton("Cancel", gtk.RESPONSE_CANCEL)
	rd.AddButton("OK", gtk.RESPONSE_OK)
	rd.ShowAll()
	response := rd.Run()
	if response == gtk.RESPONSE_OK {
		terminal.rwMutex.Lock()
		terminal.visibleCols, _ = strconv.Atoi(colsCombo.GetActiveText())
		terminal.visibleLines, _ = strconv.Atoi(linesCombo.GetActiveText())
		switch zoomCombo.GetActiveText() {
		case "Large":
			zoom = zoomLarge
		case "Normal":
			zoom = zoomNormal
		case "Smaller":
			zoom = zoomSmaller
		case "Tiny":
			zoom = zoomTiny
		}
		bdfLoad(fontFile, zoom)

		crt.SetSizeRequest(terminal.visibleCols*charWidth, terminal.visibleLines*charHeight)
		terminal.rwMutex.Unlock()
		terminal.resize()
		win.Resize(800, 600) // this is effectively a minimum size, user can override
	}
	rd.Destroy()
}

func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}

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

func toggleLogging() {
	if terminal.logging {
		terminal.logFile.Close()
		terminal.logging = false
	} else {
		fd := gtk.NewFileChooserDialog("DasherG Logfile", win, gtk.FILE_CHOOSER_ACTION_SAVE,
			"_Cancel", gtk.RESPONSE_CANCEL, "_Open", gtk.RESPONSE_ACCEPT)
		res := fd.Run()
		if res == gtk.RESPONSE_ACCEPT {
			filename := fd.GetFilename()
			terminal.logFile, err = os.Create(filename)
			if err != nil {
				log.Printf("WARNING: Could not open log file %s\n", filename)
				terminal.logging = false
			} else {
				terminal.logging = true
			}
		}
		fd.Destroy()
	}
}

func buildCrt() *gtk.DrawingArea {
	crt = gtk.NewDrawingArea()
	terminal.rwMutex.RLock()
	crt.SetSizeRequest(terminal.visibleCols*charWidth, terminal.visibleLines*charHeight)
	terminal.rwMutex.RUnlock()

	crt.Connect("configure-event", func() {
		if offScreenPixmap != nil {
			offScreenPixmap.Unref()
		}
		//allocation := crt.GetAllocation()
		terminal.rwMutex.RLock()
		offScreenPixmap = gdk.NewPixmap(crt.GetWindow().GetDrawable(),
			terminal.visibleCols*charWidth, terminal.visibleLines*charHeight*charHeight, 24)
		terminal.rwMutex.RUnlock()
		gc = gdk.NewGC(offScreenPixmap.GetDrawable())
		offScreenPixmap.GetDrawable().DrawRectangle(gc, true, 0, 0, -1, -1)
		gc.SetForeground(gc.GetColormap().AllocColorRGB(0, 65535, 0))
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

func updateCrt(crt *gtk.DrawingArea, t *terminalT) {
	//var cIx int

	for {
		updateType := <-updateCrtChan
		t.rwMutex.Lock()
		switch updateType {
		case updateCrtBlink:
			t.blinkState = !t.blinkState
			fallthrough
		case updateCrtNormal:
			terminal.terminalUpdated = true
		}
		t.rwMutex.Unlock()
		//fmt.Println("updateCrt called")
	}
}

func drawCrt() {
	terminal.rwMutex.Lock()
	if terminal.terminalUpdated {
		var cIx int
		drawable := offScreenPixmap.GetDrawable()
		for line := 0; line < terminal.visibleLines; line++ {
			for col := 0; col < terminal.visibleCols; col++ {
				cIx = int(terminal.display[line][col].charValue)
				if cIx > 31 && cIx < 128 {
					switch {
					case terminal.blinkEnabled && terminal.blinkState && terminal.display[line][col].blink:
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
				//fmt.Printf("Drawing cursor at %d,%d\n", terminal.cursorX*charWidth, terminal.cursorY*charHeight)
				drawable.DrawPixbuf(gc, bdfFont[cIx].reversePixbuf, 0, 0, terminal.cursorX*charWidth, terminal.cursorY*charHeight, charWidth, charHeight, 0, 0, 0)
			}
		}
		terminal.terminalUpdated = false
		gdkWin.Invalidate(nil, false)
	}
	terminal.rwMutex.Unlock()
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

	loggingLabel = gtk.NewLabel("")
	lf := gtk.NewFrame("")
	lf.Add(loggingLabel)
	statusBox.Add(lf)

	emuStatusLabel = gtk.NewLabel("")
	esf := gtk.NewFrame("")
	esf.Add(emuStatusLabel)
	statusBox.Add(esf)

	glib.TimeoutAdd(statusUpdatePeriodMs, func() bool {
		updateStatusBox()
		return true
	})

	return statusBox
}

// updateStatusBox to be run regularly - N.B. on the main thread!
func updateStatusBox() {
	terminal.rwMutex.RLock()
	switch terminal.connected {
	case disconnected:
		onlineLabel.SetText("Local (Offline)")
		hostLabel.SetText("")
	case serialConnected:
		fmt.Println("Serial not yet supported")
		hostLabel.SetText("")
	case telnetConnected:
		onlineLabel.SetText("Online (Telnet)")
		hostLabel.SetText(terminal.remoteHost + ":" + terminal.remotePort)
	}
	if terminal.logging {
		loggingLabel.SetText("Logging")
	} else {
		loggingLabel.SetText("")
	}
	emuStat := "D" + strconv.Itoa(int(terminal.emulation)) + " (" +
		strconv.Itoa(terminal.visibleLines) + "x" + strconv.Itoa(terminal.visibleCols) + ")"
	terminal.rwMutex.RUnlock()
	emuStatusLabel.SetText(emuStat)
}

func sendFile() {
	sd := gtk.NewFileChooserDialog("DasherG File to send", win, gtk.FILE_CHOOSER_ACTION_OPEN, "_Cancel", gtk.RESPONSE_CANCEL, "_Send", gtk.RESPONSE_ACCEPT)
	res := sd.Run()
	if res == gtk.RESPONSE_ACCEPT {
		fileName := sd.GetFilename()
		bytes, err := ioutil.ReadFile(fileName)
		if err != nil {
			ed := gtk.NewMessageDialog(win, gtk.DIALOG_DESTROY_WITH_PARENT, gtk.MESSAGE_ERROR,
				gtk.BUTTONS_CLOSE, "Could not open or read file to send")
			ed.Run()
			ed.Destroy()
		} else {
			for _, b := range bytes {
				keyboardChan <- b
			}
		}
	}
	sd.Destroy()
}
