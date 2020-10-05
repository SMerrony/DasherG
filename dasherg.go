// dasherg.go

// Copyright ©2017-2020  Steve Merrony

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
	"log"
	"os/exec"
	"runtime"

	// _ "net/http/pprof" // debugging

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
	appCopyright = "Copyright ©2017-2020 S.Merrony"
	appSemVer    = "v0.10.3" // TODO Update SemVer on each release!
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
	telnetSession         *telnetSessionT
	serialSession         = newSerialSession()
	lastTelnetHost        string
	lastTelnetPort        int
	telnetClosing         bool
	traceExpect           bool

	gc              *gdk.GC
	crt             *gtk.DrawingArea
	scroller        *gtk.VScrollbar
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
	cpuprofile      = flag.String("cpuprofile", "", "Write cpu profile to file")
	cputrace        = flag.String("cputrace", "", "Write trace to file")
	hostFlag        = flag.String("host", "", "Host to connect with")
	traceExpectFlag = flag.Bool("tracescript", false, "Print trace of Mini-Expect script on STDOUT")
	versionFlag     = flag.Bool("version", false, "Display version number and exit")
	xmodemTraceFlag = flag.Bool("xmodemtrace", false, "Show details of XMODEM file transfers on STDOUT")
)

func main() {

	flag.Parse()
	if *versionFlag {
		fmt.Println(appTitle, appSemVer)
		os.Exit(0)
	}

	// debugging...
	// runtime.SetMutexProfileFraction(1)
	// go func() {
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()

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
		telnetSession = newTelnetSession()
		if telnetSession.openTelnetConn(hostParts[0], hostPort) {
			localListenerStopChan <- true
			networkConnectMenuItem.SetSensitive(false)
			serialConnectMenuItem.SetSensitive(false)
			networkDisconnectMenuItem.SetSensitive(true)
			lastTelnetHost = hostParts[0]
			lastTelnetPort = hostPort
		}
	}

	go terminal.updateListener()

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
	scroller = buildScrollbar()
	hbox := gtk.NewHBox(false, 1)
	hbox.PackStart(crt, false, false, 1)
	hbox.PackEnd(scroller, false, false, 1)
	vbox.PackStart(hbox, false, false, 1)
	statusBox := buildStatusBox()
	vbox.PackEnd(statusBox, false, false, 0)
	win.Add(vbox)
	win.SetIcon(iconPixbuf)
}

func localListener(kbdChan <-chan byte, frmHostChan chan<- []byte) {
	fmt.Println("INFO: localListener starting")
	for {
		key := make([]byte, 2)
		select {
		case kev := <-kbdChan:
			key[0] = kev
			frmHostChan <- key
		case <-localListenerStopChan:
			fmt.Println("INFO: localListener stopped")
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
	loggingMenuItem.Connect("activate", fileLogging)
	subMenu.Append(loggingMenuItem)

	subMenu.Append(gtk.NewSeparatorMenuItem())

	expectFileMenuItem := gtk.NewMenuItemWithLabel("Run mini-Expect Script")
	expectFileMenuItem.Connect("activate", fileChooseExpectScript)
	subMenu.Append(expectFileMenuItem)

	subMenu.Append(gtk.NewSeparatorMenuItem())

	sendFileMenuItem := gtk.NewMenuItemWithLabel("Send (Text) File")
	sendFileMenuItem.Connect("activate", fileSendText)
	subMenu.Append(sendFileMenuItem)

	subMenu.Append(gtk.NewSeparatorMenuItem())

	xmodemRcvMenuItem := gtk.NewMenuItemWithLabel("XMODEM-CRC - Receive File")
	xmodemRcvMenuItem.Connect("activate", fileXmodemReceive)
	subMenu.Append(xmodemRcvMenuItem)

	xmodemSendMenuItem := gtk.NewMenuItemWithLabel("XMODEM-CRC - Send File")
	xmodemSendMenuItem.Connect("activate", fileXmodemSend)
	subMenu.Append(xmodemSendMenuItem)

	xmodemSend1kMenuItem := gtk.NewMenuItemWithLabel("XMODEM-CRC - Send File (1k packets)")
	xmodemSend1kMenuItem.Connect("activate", fileXmodemSend1k)
	subMenu.Append(xmodemSend1kMenuItem)

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
	pasteItem.Connect("activate", editPaste)
	subMenu.Append(pasteItem)

	emulationMenuItem := gtk.NewMenuItemWithLabel("Emulation")
	menuBar.Append(emulationMenuItem)
	subMenu = gtk.NewMenu()
	var emuGroup *glib.SList
	emulationMenuItem.SetSubmenu(subMenu)

	d200MenuItem := gtk.NewRadioMenuItemWithLabel(emuGroup, "D200")
	emuGroup = d200MenuItem.GetGroup()
	if terminal.emulation == d200 {
		d200MenuItem.SetActive(true)
	}
	subMenu.Append(d200MenuItem)

	d210MenuItem := gtk.NewRadioMenuItemWithLabel(emuGroup, "D210")
	if terminal.emulation == d210 {
		d210MenuItem.SetActive(true)
	}
	subMenu.Append(d210MenuItem)

	// for some reason, the 1st of these gets triggered at startup...
	d210MenuItem.Connect("activate", func() { terminal.setEmulation(d210) })
	d200MenuItem.Connect("activate", func() { terminal.setEmulation(d200) })

	subMenu.Append(gtk.NewSeparatorMenuItem())
	resizeMenuItem := gtk.NewMenuItemWithLabel("Resize")
	resizeMenuItem.Connect("activate", emulationResize)
	subMenu.Append(resizeMenuItem)
	subMenu.Append(gtk.NewSeparatorMenuItem())
	selfTestMenuItem := gtk.NewMenuItemWithLabel("Self-Test")
	subMenu.Append(selfTestMenuItem)
	selfTestMenuItem.Connect("activate", func() { terminal.selfTest(fromHostChan) })
	loadTemplateMenuItem := gtk.NewMenuItemWithLabel("Load Func. Key Template")
	loadTemplateMenuItem.Connect("activate", loadFKeyTemplate)
	subMenu.Append(loadTemplateMenuItem)

	serialMenuItem := gtk.NewMenuItemWithLabel("Serial")
	menuBar.Append(serialMenuItem)
	subMenu = gtk.NewMenu()
	serialMenuItem.SetSubmenu(subMenu)
	serialConnectMenuItem = gtk.NewMenuItemWithLabel("Connect")
	serialConnectMenuItem.Connect("activate", serialConnect)
	subMenu.Append(serialConnectMenuItem)
	serialDisconnectMenuItem = gtk.NewMenuItemWithLabel("Disconnect")
	serialDisconnectMenuItem.Connect("activate", serialClose)
	subMenu.Append(serialDisconnectMenuItem)
	serialDisconnectMenuItem.SetSensitive(false)

	networkMenuItem := gtk.NewMenuItemWithLabel("Network")
	menuBar.Append(networkMenuItem)
	subMenu = gtk.NewMenu()
	networkMenuItem.SetSubmenu(subMenu)
	networkConnectMenuItem = gtk.NewMenuItemWithLabel("Connect")
	subMenu.Append(networkConnectMenuItem)
	networkConnectMenuItem.Connect("activate", telnetOpen)
	networkDisconnectMenuItem = gtk.NewMenuItemWithLabel("Disconnect")
	subMenu.Append(networkDisconnectMenuItem)
	networkDisconnectMenuItem.Connect("activate", telnetClose)
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
	aboutMenuItem.Connect("activate", helpAbout)

	return menuBar
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

func buildScrollbar() (sb *gtk.VScrollbar) {
	adj := gtk.NewAdjustment(historyLines, 0.0, historyLines, 1.0, 1.0, 1.0)
	sb = gtk.NewVScrollbar(adj)
	sb.Connect("value-changed", handleScrollbarChangedEvent)
	return sb
}

func handleScrollbarChangedEvent(ctx *glib.CallbackContext) {
	posn := int(scroller.GetValue())
	//fmt.Printf("Scrollbar event: Value: %d\n", posn)
	if posn == historyLines-1 {
		terminal.cancelScrollBack()
	} else {
		terminal.scrollBack(historyLines - posn)
	}
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
	switch terminal.connectionType {
	case disconnected:
		onlineLabel.SetText("Local (Offline)")
		hostLabel.SetText("")
	case serialConnected:
		onlineLabel.SetText("Online (Serial)")
		serParms := terminal.serialPort + " @ " + serialSession.getParms()
		hostLabel.SetText(serParms)
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
		strconv.Itoa(terminal.display.visibleLines) + "x" + strconv.Itoa(terminal.display.visibleCols) + ")"
	if terminal.holding {
		emuStat += " (Hold)"
	}
	terminal.rwMutex.RUnlock()
	emuStatusLabel.SetText(emuStat)
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
			img := image.NewNRGBA(image.Rect(0, 0, (terminal.display.visibleCols+1)*fontWidth, (terminal.display.visibleLines+1)*fontHeight))
			bg := image.NewUniform(color.RGBA{255, 255, 255, 255})   // prepare white for background
			grey := image.NewUniform(color.RGBA{128, 128, 128, 255}) // prepare grey for foreground
			blk := image.NewUniform(color.RGBA{0, 0, 0, 255})        // prepare black for foreground
			draw.Draw(img, img.Bounds(), bg, image.ZP, draw.Src)     // fill the background
			for line := 0; line < terminal.display.visibleLines; line++ {
				for col := 0; col < terminal.display.visibleCols; col++ {
					for x := 0; x < fontWidth; x++ {
						for y := 0; y < fontHeight; y++ {
							switch {
							case terminal.display.cells[line][col].dim:
								if bdfFont[terminal.display.cells[line][col].charValue].pixels[x][y] {
									img.Set(col*fontWidth+x, (line+1)*fontHeight-y, grey)
								}
							case terminal.display.cells[line][col].reverse:
								if !bdfFont[terminal.display.cells[line][col].charValue].pixels[x][y] {
									img.Set(col*fontWidth+x, (line+1)*fontHeight-y, blk)
								}
							default:
								if bdfFont[terminal.display.cells[line][col].charValue].pixels[x][y] {
									img.Set(col*fontWidth+x, (line+1)*fontHeight-y, blk)
								}
							}
						}
					}
					if terminal.display.cells[line][col].underscore {
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
