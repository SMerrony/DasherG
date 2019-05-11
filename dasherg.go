// dasherg.go

// Copyright (C) 2017,2018  Steve Merrony

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
	"image"
	"image/color"
	"image/draw"
	"image/png"
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

//go:generate go-bindata -prefix "resources/" -pkg main -o resources.go resources/...

const (
	appID        = "uk.co.merrony.dasherg"
	appTitle     = "DasherG"
	appComment   = "A Data General DASHER terminal emulator"
	appCopyright = "Copyright ©2017, 2018 S.Merrony"
	appVersion   = "0.95 beta"
	appWebsite   = "https://github.com/SMerrony/DasherG"
	fontFile     = "D410-b-12.bdf"
	helpURL      = "https://github.com/SMerrony/DasherG"

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
	expectChan            = make(chan byte, hostBuffSize)
	traceExpect           bool

	gc              *gdk.GC
	crt             *gtk.DrawingArea
	zoom            = zoomNormal
	offScreenPixmap *gdk.Pixmap
	win             *gtk.Window
	gdkWin          *gdk.Window
	iconPixbuf      *gdkpixbuf.Pixbuf

	// widgets needing global access
	serialConnectMenuItem, serialDisconnectMenuItem      *gtk.MenuItem
	networkConnectMenuItem, networkDisconnectMenuItem    *gtk.MenuItem
	onlineLabel, hostLabel, loggingLabel, emuStatusLabel *gtk.Label
	expectDialog                                         *gtk.FileChooserDialog
)

var (
	cpuprofile      = flag.String("cpuprofile", "", "write cpu profile to file")
	cputrace        = flag.String("cputrace", "", "write trace to file")
	hostFlag        = flag.String("host", "", "Host to connect with")
	traceExpectFlag = flag.Bool("tracescript", false, "Print trace of Mini-Expect script to STDOUT")
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

	if *traceExpectFlag {
		traceExpect = true
	}

	gtk.Init(nil)

	// get the application and dialog icon
	iconPixbuf = gdkpixbuf.NewPixbufFromData(iconPNG)

	bdfLoad(fontFile, zoomNormal)
	go localListener(keyboardChan, fromHostChan)
	terminal = new(terminalT)
	terminal.setup(fromHostChan, updateCrtChan, expectChan)
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

	// testing... I don't know why doing this in terminal.setup above is being lost
	terminal.emulation = d210

	gtk.Main()
}

func setupWindow(win *gtk.Window) {
	win.SetTitle(appTitle)
	win.Connect("destroy", func() {
		gtk.MainQuit()
		//os.Exit(0)
	})
	//win.SetDefaultSize(800, 600)
	go keyEventHandler(keyboardChan)
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
	win.SetIcon(iconPixbuf)
}

func localListener(kbdChan <-chan byte, frmHostChan chan<- []byte) {
	for {
		key := make([]byte, 2)
		select {
		case kev := <-kbdChan:
			key[0] = kev
			frmHostChan <- key
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

	subMenu.Append(gtk.NewSeparatorMenuItem())

	expectFileMenuItem := gtk.NewMenuItemWithLabel("Run mini-Expect Script")
	expectFileMenuItem.Connect("activate", chooseExpectScript)
	subMenu.Append(expectFileMenuItem)

	sendFileMenuItem := gtk.NewMenuItemWithLabel("Send (Text) File")
	sendFileMenuItem.Connect("activate", sendFile)
	subMenu.Append(sendFileMenuItem)

	subMenu.Append(gtk.NewSeparatorMenuItem())

	quitMenuItem := gtk.NewMenuItemWithLabel("Quit")
	subMenu.Append(quitMenuItem)
	quitMenuItem.Connect("activate", func() {
		pprof.StopCPUProfile()
		gtk.MainQuit()
		//os.Exit(0)
	})

	editMenuItem := gtk.NewMenuItemWithLabel("Edit")
	menuBar.Append(editMenuItem)
	subMenu = gtk.NewMenu()
	editMenuItem.SetSubmenu(subMenu)
	pasteItem := gtk.NewMenuItemWithLabel("Paste")
	pasteItem.Connect("activate", pasteClipboard)
	subMenu.Append(pasteItem)

	viewMenuItem := gtk.NewMenuItemWithLabel("View")
	menuBar.Append(viewMenuItem)
	subMenu = gtk.NewMenu()
	viewMenuItem.SetSubmenu(subMenu)
	viewHistoryItem := gtk.NewMenuItemWithLabel("History")
	viewHistoryItem.Connect("activate", func() { showHistory(terminal) })
	subMenu.Append(viewHistoryItem)
	loadTemplateMenuItem := gtk.NewMenuItemWithLabel("Load Func. Key Template")
	loadTemplateMenuItem.Connect("activate", loadFKeyTemplate)
	subMenu.Append(loadTemplateMenuItem)

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
	subMenu.Append(gtk.NewSeparatorMenuItem())
	resizeMenuItem := gtk.NewMenuItemWithLabel("Resize")
	resizeMenuItem.Connect("activate", resizeDialog)
	subMenu.Append(resizeMenuItem)
	subMenu.Append(gtk.NewSeparatorMenuItem())
	selfTestMenuItem := gtk.NewMenuItemWithLabel("Self-Test")
	subMenu.Append(selfTestMenuItem)
	selfTestMenuItem.Connect("activate", func() { terminal.selfTest(fromHostChan) })

	serialMenuItem := gtk.NewMenuItemWithLabel("Serial")
	menuBar.Append(serialMenuItem)
	subMenu = gtk.NewMenu()
	serialMenuItem.SetSubmenu(subMenu)
	serialConnectMenuItem = gtk.NewMenuItemWithLabel("Connect")
	serialConnectMenuItem.Connect("activate", openSerialDialog)
	subMenu.Append(serialConnectMenuItem)
	serialDisconnectMenuItem = gtk.NewMenuItemWithLabel("Disconnect")
	serialDisconnectMenuItem.Connect("activate", closeSerial)
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
	onlineHelpMenuItem.Connect("activate", func() { openBrowser(helpURL) })
	subMenu.Append(onlineHelpMenuItem)
	subMenu.Append(gtk.NewSeparatorMenuItem())
	aboutMenuItem := gtk.NewMenuItemWithLabel("About")
	subMenu.Append(aboutMenuItem)
	aboutMenuItem.Connect("activate", aboutDialog)

	return menuBar
}

func aboutDialog() {
	ad := gtk.NewAboutDialog()
	ad.SetProgramName(appTitle)
	ad.SetAuthors(appAuthors)
	ad.SetIcon(iconPixbuf)
	ad.SetLogo(iconPixbuf)
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
	nd.SetTitle("DasherG - Telnet Host")
	nd.SetIcon(iconPixbuf)
	ca := nd.GetVBox()
	hostLab := gtk.NewLabel("Host:")
	ca.PackStart(hostLab, true, true, 5)
	hostEntry := gtk.NewEntry()
	hostEntry.SetText(lastHost)
	ca.PackStart(hostEntry, true, true, 5)
	portLab := gtk.NewLabel("Port:")
	ca.PackStart(portLab, true, true, 5)
	portEntry := gtk.NewEntry()
	portEntry.SetActivatesDefault(true) // hitting ENTER will cause default (OK) response
	if lastPort != 0 {
		portEntry.SetText(strconv.Itoa(lastPort))
	}
	ca.PackStart(portEntry, true, true, 5)

	nd.AddButton("Cancel", gtk.RESPONSE_CANCEL)
	nd.AddButton("OK", gtk.RESPONSE_OK)
	nd.SetDefaultResponse(gtk.RESPONSE_OK)
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
			if openTelnetConn(host, port) {
				localListenerStopChan <- true
				networkConnectMenuItem.SetSensitive(false)
				serialConnectMenuItem.SetSensitive(false)
				networkDisconnectMenuItem.SetSensitive(true)
			} else {
				ed := gtk.NewMessageDialog(win, gtk.DIALOG_DESTROY_WITH_PARENT, gtk.MESSAGE_ERROR,
					gtk.BUTTONS_CLOSE, "Could not connect to remote host")
				ed.Run()
				ed.Destroy()
			}
		}
	}

	nd.Destroy()
}

func closeRemote() {
	closeTelnetConn()
	glib.IdleAdd(func() {
		networkDisconnectMenuItem.SetSensitive(false)
		serialConnectMenuItem.SetSensitive(true)
		networkConnectMenuItem.SetSensitive(true)
	})
	go localListener(keyboardChan, fromHostChan)
}

func openSerialDialog() {
	sd := gtk.NewDialog()
	sd.SetTitle("DasherG - Serial Port")
	sd.SetIcon(iconPixbuf)
	ca := sd.GetVBox()
	table := gtk.NewTable(5, 2, false)
	table.SetColSpacings(5)
	table.SetRowSpacings(5)
	portLab := gtk.NewLabel("Port:")
	table.AttachDefaults(portLab, 0, 1, 0, 1)
	portEntry := gtk.NewEntry()
	table.AttachDefaults(portEntry, 1, 2, 0, 1)
	baudLab := gtk.NewLabel("Baud:")
	table.AttachDefaults(baudLab, 0, 1, 1, 2)
	baudCombo := gtk.NewComboBoxText()
	baudCombo.AppendText("300")
	baudCombo.AppendText("1200")
	baudCombo.AppendText("2400")
	baudCombo.AppendText("9600")
	baudCombo.AppendText("19200")
	baudCombo.SetActive(3)
	table.AttachDefaults(baudCombo, 1, 2, 1, 2)
	bitsLab := gtk.NewLabel("Data bits:")
	table.AttachDefaults(bitsLab, 0, 1, 2, 3)
	bitsCombo := gtk.NewComboBoxText()
	bitsCombo.AppendText("7")
	bitsCombo.AppendText("8")
	bitsCombo.SetActive(1)
	table.AttachDefaults(bitsCombo, 1, 2, 2, 3)
	parityLab := gtk.NewLabel("Parity:")
	table.AttachDefaults(parityLab, 0, 1, 3, 4)
	parityCombo := gtk.NewComboBoxText()
	parityCombo.AppendText("None")
	parityCombo.AppendText("Even")
	parityCombo.AppendText("Odd")
	parityCombo.SetActive(0)
	table.AttachDefaults(parityCombo, 1, 2, 3, 4)
	stopLab := gtk.NewLabel("Stop bits:")
	table.AttachDefaults(stopLab, 0, 1, 4, 5)
	stopCombo := gtk.NewComboBoxText()
	stopCombo.AppendText("1")
	//stopCombo.AppendText("1.5")
	stopCombo.AppendText("2")
	stopCombo.SetActive(0)
	table.AttachDefaults(stopCombo, 1, 2, 4, 5)
	ca.PackStart(table, true, true, 5)
	sd.AddButton("Cancel", gtk.RESPONSE_CANCEL)
	sd.AddButton("OK", gtk.RESPONSE_OK)
	sd.SetDefaultResponse(gtk.RESPONSE_OK)
	sd.ShowAll()
	response := sd.Run()

	if response == gtk.RESPONSE_OK {
		baud, _ := strconv.Atoi(baudCombo.GetActiveText())
		bits, _ := strconv.Atoi(bitsCombo.GetActiveText())
		stopBits, _ := strconv.Atoi(stopCombo.GetActiveText())
		if openSerialPort(portEntry.GetText(), baud, bits, parityCombo.GetActiveText(), stopBits) {
			localListenerStopChan <- true
			serialConnectMenuItem.SetSensitive(false)
			networkConnectMenuItem.SetSensitive(false)
			serialDisconnectMenuItem.SetSensitive(true)
		} else {
			ed := gtk.NewMessageDialog(win, gtk.DIALOG_DESTROY_WITH_PARENT, gtk.MESSAGE_ERROR,
				gtk.BUTTONS_CLOSE, "Could not connect via serial port")
			ed.Run()
			ed.Destroy()
		}
	}
	sd.Destroy()
}

func closeSerial() {
	closeSerialPort()
	glib.IdleAdd(func() {
		serialDisconnectMenuItem.SetSensitive(false)
		networkConnectMenuItem.SetSensitive(true)
		serialConnectMenuItem.SetSensitive(true)
	})
	go localListener(keyboardChan, fromHostChan)
}

func showHistory(t *terminalT) {
	hd := gtk.NewDialog()
	hd.SetTitle("DasherG - Terminal History")
	hd.SetIcon(iconPixbuf)
	ca := hd.GetVBox()
	scrolledWindow := gtk.NewScrolledWindow(nil, nil)
	tv := gtk.NewTextView()
	tv.ModifyFontEasy("monospace")
	scrolledWindow.Add(tv)
	tb := tv.GetBuffer()
	var iter gtk.TextIter
	tb.GetStartIter(&iter)
	for _, line := range t.history {
		if len(line) > 0 {
			tb.Insert(&iter, line+"\n")
		}
	}
	tv.SetEditable(false)
	tv.SetSizeRequest(t.visibleCols*charWidth, t.visibleLines*charHeight)
	ca.PackStart(scrolledWindow, true, true, 1)
	hd.AddButton("OK", gtk.RESPONSE_OK)
	hd.SetDefaultResponse(gtk.RESPONSE_OK)
	hd.ShowAll()
	hd.Run()
	hd.Destroy()
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
				if terminal.display[line][col].dirty || (terminal.blinkEnabled && terminal.display[line][col].blink) {
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
						drawable.DrawLine(gc, col*charWidth, ((line+1)*charHeight)-1, (col+1)*charWidth-1, ((line+1)*charHeight)-1)
					}
					terminal.display[line][col].dirty = false
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
			terminal.display[terminal.cursorY][terminal.cursorX].dirty = true // this ensures that the old cursor pos is redrawn on the next refresh
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
		onlineLabel.SetText("Online (Serial)")
		hostLabel.SetText(terminal.serialPort)
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
	if terminal.holding {
		emuStat += " (Hold)"
	}
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

func chooseExpectScript() {
	expectDialog = gtk.NewFileChooserDialog("DasherG mini-Expect Script to run", win, gtk.FILE_CHOOSER_ACTION_OPEN, "_Cancel", gtk.RESPONSE_CANCEL, "_Run", gtk.RESPONSE_ACCEPT)
	res := expectDialog.Run()
	if res == gtk.RESPONSE_ACCEPT {
		expectFile, err := os.Open(expectDialog.GetFilename())
		if err != nil {
			errDialog := gtk.NewMessageDialog(win, gtk.DIALOG_DESTROY_WITH_PARENT, gtk.MESSAGE_ERROR,
				gtk.BUTTONS_CLOSE, "Could not open or read mini-Expect script file")
			errDialog.Run()
			errDialog.Destroy()
		} else {
			go expectRunner(expectFile, expectChan, keyboardChan, terminal)
			expectDialog.Destroy()
		}
	}
}

func localPrint() {
	fd := gtk.NewFileChooserDialog("DasherG Screen-Dump", win, gtk.FILE_CHOOSER_ACTION_SAVE,
		"_Cancel", gtk.RESPONSE_CANCEL, "_Save", gtk.RESPONSE_ACCEPT)
	fd.SetFilename("DASHER.png")
	res := fd.Run()
	if res == gtk.RESPONSE_ACCEPT {
		filename := fd.GetFilename()
		dumpFile, err := os.Create(filename)
		if err != nil {
			fmt.Printf("ERROR: Could not create file <%s> for screen-dump\n", filename)
		} else {
			defer dumpFile.Close()
			img := image.NewNRGBA(image.Rect(0, 0, (terminal.visibleCols+1)*fontWidth, (terminal.visibleLines+1)*fontHeight))
			bg := image.NewUniform(color.RGBA{255, 255, 255, 255})   // prepare white for background
			grey := image.NewUniform(color.RGBA{128, 128, 128, 255}) // prepare grey for foreground
			blk := image.NewUniform(color.RGBA{0, 0, 0, 255})        // prepare black for foreground
			draw.Draw(img, img.Bounds(), bg, image.ZP, draw.Src)     // fill the background
			for line := 0; line < terminal.visibleLines; line++ {
				for col := 0; col < terminal.visibleCols; col++ {
					for x := 0; x < fontWidth; x++ {
						for y := 0; y < fontHeight; y++ {
							switch {
							case terminal.display[line][col].dim:
								if bdfFont[terminal.display[line][col].charValue].pixels[x][y] {
									img.Set(col*fontWidth+x, (line+1)*fontHeight-y, grey)
								}
							case terminal.display[line][col].reverse:
								if !bdfFont[terminal.display[line][col].charValue].pixels[x][y] {
									img.Set(col*fontWidth+x, (line+1)*fontHeight-y, blk)
								}
							default:
								if bdfFont[terminal.display[line][col].charValue].pixels[x][y] {
									img.Set(col*fontWidth+x, (line+1)*fontHeight-y, blk)
								}
							}
						}
					}
					if terminal.display[line][col].underscore {
						for x := 0; x < fontWidth; x++ {
							img.Set(col*fontWidth+x, (line+1)*fontHeight, blk)
						}
					}
				}
			}
			if err := png.Encode(dumpFile, img); err != nil {
				fmt.Printf("ERROR: Could not save PNG screen-dump, %v\n", err)
			}
			dumpFile.Close()
		}
	}
	fd.Destroy()
}

func pasteClipboard() {
	clipboard := gtk.NewClipboardGetForDisplay(gdk.DisplayGetDefault(), gdk.SELECTION_CLIPBOARD)
	text := clipboard.WaitForText()
	if len(text) == 0 {
		ed := gtk.NewMessageDialog(win, gtk.DIALOG_DESTROY_WITH_PARENT, gtk.MESSAGE_ERROR,
			gtk.BUTTONS_CLOSE, "Nothing in Clipboard to Paste")
		ed.Run()
	} else {
		for _, ch := range text {
			keyboardChan <- byte(ch)
		}
	}
}
