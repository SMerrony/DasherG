// dasherg.go

// Copyright © 2017-2021,2025  Steve Merrony

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
	_ "embed"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"os/exec"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	// _ "net/http/pprof" // debugging
)

const (
	appTitle     = "DasherG"
	appComment   = "A Data General DASHER terminal emulator"
	appCopyright = "Copyright ©2017-2021,2025,2026 S.Merrony"
	appSemVer    = "0.17.0" // TODO Update SemVer on each release!
	appWebsite   = "https://github.com/SMerrony/DasherG"
	helpURL      = "https://github.com/SMerrony/DasherG"

	hostBuffSize = 2048
	keyBuffSize  = 200

	updateCrtNormal = 1 // crt is 'dirty' and needs updating
	updateCrtBlink  = 2 // crt blink state needs flipping
	blinkPeriodMs   = 500
	// crtRefreshMs influences the responsiveness of the display. 50ms = 20Hz or 20fps
	crtRefreshMs         = 50
	statusUpdatePeriodMs = 500
	logLines             = 1000
)

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

	selectionRegion struct {
		isActive                           bool
		startRow, startCol, endRow, endCol int
	}

	zoom     = ZoomNormal
	w        fyne.Window
	crtImg   *crtMouseable
	amber    = color.RGBA{0xff, 0xbf, 0x00, 0xff}
	dimAmber = color.RGBA{0x88, 0x5f, 0x00, 0xff}
	green    = color.RGBA{0x00, 0xff, 0x00, 0xff}
	dimGreen = color.RGBA{0x00, 0x80, 0x00, 0xff}
	white    = color.RGBA{0xff, 0xff, 0xff, 0xff}
	dimWhite = color.RGBA{0x88, 0x88, 0x88, 0xff}

	fontColour, fontDimColour color.RGBA

	// widgets needing global access
	onlineLabel2, hostLabel2, loggingLabel2, emuStatusLabel2                           *widget.Label
	serialConnectItem, serialDisconnectItem, networkConnectItem, networkDisconnectItem *fyne.MenuItem
	topVbox, labelGrid                                                                 *fyne.Container
	specialThemeOverride, labelThemeOverride, funcThemeOverride                        *container.ThemeOverride
)

var (
	amberFlag       = flag.Bool("amber", false, "Use Amber font instead of default Green")
	cpuprofile      = flag.String("cpuprofile", "", "Write cpu profile to file")
	cputrace        = flag.String("cputrace", "", "Write trace to file")
	hostFlag        = flag.String("host", "", "Host to connect with")
	traceExpectFlag = flag.Bool("tracescript", false, "Print trace of Mini-Expect script on STDOUT")
	xmodemTraceFlag = flag.Bool("tracexmodem", false, "Show details of XMODEM file transfers on STDOUT")
	versionFlag     = flag.Bool("version", false, "Display version number and exit")
	whiteFlag       = flag.Bool("white", false, "Use White font")
)

//go:embed resources/D410-b-12.bdf
var fontData []byte

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

	a := app.New()
	// a.Settings().SetTheme(&ourTheme{})

	fontColour = green
	fontDimColour = dimGreen
	if *amberFlag {
		fontColour = amber
		fontDimColour = dimAmber
	}
	if *whiteFlag {
		fontColour = white
		fontDimColour = dimWhite
	}

	bdfLoad(fontData, ZoomNormal, fontColour, fontDimColour)
	go localListener(keyboardChan, fromHostChan)
	terminal = new(terminalT)
	terminal.setup(fromHostChan, updateCrtChan, expectChan, fontColour)
	w = a.NewWindow(appTitle)
	setupWindow(w)

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
			lastTelnetHost = hostParts[0]
			lastTelnetPort = hostPort
		}
	}

	go terminal.updateListener()

	go func() {
		for {
			drawCrt()
			time.Sleep(crtRefreshMs * time.Millisecond)
		}
	}()

	w.ShowAndRun()
}

func setupWindow(w fyne.Window) {
	w.SetIcon(resourceDGlogoOrangePng)
	w.SetMainMenu(buildMenu())

	go keyEventHandler(keyboardChan)

	if deskCanvas, ok := w.Canvas().(desktop.Canvas); ok {
		deskCanvas.SetOnKeyDown(func(ev *fyne.KeyEvent) {
			keyDownEventChan <- ev
		})
		deskCanvas.SetOnKeyUp(func(ev *fyne.KeyEvent) {
			keyUpEventChan <- ev
		})
	}

	crtImg = buildCrt()
	go terminal.run()

	go func() {
		for {
			updateCrtChan <- updateCrtBlink
			time.Sleep(blinkPeriodMs * time.Millisecond)
		}
	}()

	setContent(w)
}

func setContent(w fyne.Window) {
	specialKeyGrid := buildSpecialKeyRow(w)
	labelGrid = buildEmptyLabelGrid()
	fkGrid := buildFuncKeyRow()
	specialThemeOverride = container.NewThemeOverride(specialKeyGrid, &buttonTheme{})
	labelThemeOverride = container.NewThemeOverride(labelGrid, &fkeyLabelTheme{})
	funcThemeOverride = container.NewThemeOverride(fkGrid, &fkeyTheme{})
	topVbox = container.NewVBox(specialThemeOverride, labelThemeOverride, funcThemeOverride)
	labelThemeOverride.Hide()
	statusBox := buildStatusBox()
	content := container.NewBorder(
		topVbox,
		statusBox,
		nil, nil,
		container.NewHBox(layout.NewSpacer(),
			container.NewVBox(layout.NewSpacer(), crtImg, layout.NewSpacer()),
			// scrollSlider,
			layout.NewSpacer()),
	)
	w.SetContent(content)
}

func localListener(kbdChan <-chan byte, frmHostChan chan<- []byte) {
	fmt.Println("INFO: localListener starting")
	for {
		key := make([]byte, 2)
		select {
		case kev := <-kbdChan:
			key[0] = kev
			// fmt.Printf("DEBUG: localListener sending <%c>\n", kev)
			frmHostChan <- key
		case <-localListenerStopChan:
			fmt.Println("INFO: localListener stopped")
			return
		}
	}
}

func buildMenu() (mainMenu *fyne.MainMenu) {

	// file
	loggingItem := fyne.NewMenuItem("Logging", func() { fileLogging(w) })
	expectItem := fyne.NewMenuItem("Run mini-Expect Sctipt", func() { fileChooseExpectScript(w) })
	sendFileItem := fyne.NewMenuItem("Send (Text) File", func() { fileSendText(w) })
	xmodemRcvItem := fyne.NewMenuItem("XMODEM-CRC - Receive File", func() { fileXmodemReceive(w) })
	xmodemSendItem := fyne.NewMenuItem("XMODEM-CRC - Send File", func() { fileXmodemSend(w) })
	xmodemSend1kItem := fyne.NewMenuItem("XMODEM-CRC - Send File (1kB packets)", func() { fileXmodemSend1k(w) })
	fileMenu := fyne.NewMenu("File",
		loggingItem, fyne.NewMenuItemSeparator(),
		expectItem, fyne.NewMenuItemSeparator(),
		sendFileItem, fyne.NewMenuItemSeparator(),
		xmodemRcvItem, xmodemSendItem, xmodemSend1kItem)

	// edit
	pasteItem := fyne.NewMenuItem("Paste", func() { editPaste(w) })
	editMenu := fyne.NewMenu("Edit", pasteItem)

	// view
	historyItem := fyne.NewMenuItem("History", func() { viewHistory() })
	loadTemplateItem := fyne.NewMenuItem("Load Func. Key Template", func() {
		labelGrid.Show()
		loadFKeyTemplate(w)
	})
	hideTemplateItem := fyne.NewMenuItem("Hide Func. Key Template", func() {
		labelThemeOverride.Hide()
		w.Resize(fyne.Size{Width: 100, Height: 100})
	})
	viewMenu := fyne.NewMenu("View", historyItem, fyne.NewMenuItemSeparator(), loadTemplateItem, hideTemplateItem)

	// emulation
	d200Item := fyne.NewMenuItem("D200", func() { terminal.setEmulation(d200) })
	d210Item := fyne.NewMenuItem("D210", func() { terminal.setEmulation(d210) })
	resizeItem := fyne.NewMenuItem("Resize Terminal", func() { emulationResize(w) })
	selfTestItem := fyne.NewMenuItem("Self-Test", func() { terminal.selfTest(fromHostChan) })
	emulationMenu := fyne.NewMenu("Emulation",
		d200Item, d210Item, fyne.NewMenuItemSeparator(),
		resizeItem, fyne.NewMenuItemSeparator(),
		selfTestItem,
	)

	// serial
	serialConnectItem = fyne.NewMenuItem("Connect", func() { serialConnect(w) })
	serialDisconnectItem = fyne.NewMenuItem("Disconnect", serialClose)
	serialDisconnectItem.Disabled = true
	serialMenu := fyne.NewMenu("Serial", serialConnectItem, serialDisconnectItem)

	// network
	networkConnectItem = fyne.NewMenuItem("Connect", func() { telnetOpen(w) })
	networkDisconnectItem = fyne.NewMenuItem("Disconnect", telnetClose)
	networkDisconnectItem.Disabled = true
	networkMenu := fyne.NewMenu("Network", networkConnectItem, networkDisconnectItem)

	// help
	onlineHelpItem := fyne.NewMenuItem("Online Help", func() { openBrowser(helpURL) })
	aboutItem := fyne.NewMenuItem("About", helpAbout)
	helpMenu := fyne.NewMenu("Help", onlineHelpItem, fyne.NewMenuItemSeparator(), aboutItem)

	mainMenu = fyne.NewMainMenu(
		fileMenu,
		editMenu,
		viewMenu,
		emulationMenu,
		serialMenu,
		networkMenu,
		helpMenu,
	)
	return mainMenu
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

// getSelection returns a DG-ASCII string containing the mouse-selected portion of the screen
func getSelection() string {
	startCharPosn := selectionRegion.startCol + selectionRegion.startRow*terminal.display.visibleCols
	endCharPosn := selectionRegion.endCol + selectionRegion.endRow*terminal.display.visibleCols
	selection := ""
	if startCharPosn <= endCharPosn {
		// normal (forward) selection
		col := selectionRegion.startCol
		for row := selectionRegion.startRow; row <= selectionRegion.endRow; row++ {
			for col < terminal.display.visibleCols {
				selection += string(terminal.display.cells[row][col].charValue)
				terminal.displayDirty[row][col] = true
				if row == selectionRegion.endRow && col == selectionRegion.endCol {
					return selection
				}
				col++
			}
			selection += string(rune(dasherNewLine))
			col = 0
		}
	}
	return selection
}

func buildStatusBox() (statBox *fyne.Container) {

	onlineLabel2 = widget.NewLabel("")
	hostLabel2 = widget.NewLabel("")
	loggingLabel2 = widget.NewLabel("")
	emuStatusLabel2 = widget.NewLabel("")

	statBox = container.New(layout.NewHBoxLayout(),
		onlineLabel2,
		layout.NewSpacer(),
		hostLabel2,
		layout.NewSpacer(),
		loggingLabel2,
		layout.NewSpacer(),
		emuStatusLabel2,
	)

	go func() {
		for {
			updateStatusBox()
			time.Sleep(statusUpdatePeriodMs * time.Millisecond)
		}
	}()

	return statBox
}

func updateStatusBox() {
	fyne.Do(func() {
		terminal.rwMutex.RLock()
		switch terminal.connectionType {
		case disconnected:
			onlineLabel2.SetText("Local (Offline)")
			hostLabel2.SetText("")
		case serialConnected:
			onlineLabel2.SetText("Online (Serial)")
			serParms := terminal.serialPort + " @ " + serialSession.getParms()
			hostLabel2.SetText(serParms)
		case telnetConnected:
			onlineLabel2.SetText("Online (Telnet)")
			hostLabel2.SetText(terminal.remoteHost + ":" + terminal.remotePort)
		}
		if terminal.logging {
			loggingLabel2.SetText("Logging")
		} else {
			loggingLabel2.SetText("")
		}
		emuStat := "D" + strconv.Itoa(int(terminal.emulation)) + " (" +
			strconv.Itoa(terminal.display.visibleLines) + "x" + strconv.Itoa(terminal.display.visibleCols) + ")"
		if terminal.holding {
			emuStat += " (Hold)"
		}
		terminal.rwMutex.RUnlock()
		emuStatusLabel2.SetText(emuStat)
	})
}

func localPrint(win fyne.Window) {
	fd := dialog.NewFileSave(func(uriwc fyne.URIWriteCloser, e error) {
		if uriwc != nil {
			filename := uriwc.URI().Path()
			dumpFile, err := os.Create(filename)
			if err != nil {
				dialog.ShowError(err, win)
			} else {
				defer dumpFile.Close()
				img := image.NewNRGBA(image.Rect(0, 0, (terminal.display.visibleCols+1)*fontWidth, (terminal.display.visibleLines+1)*fontHeight))
				bg := image.NewUniform(color.RGBA{255, 255, 255, 255})    // prepare white for background
				grey := image.NewUniform(color.RGBA{128, 128, 128, 255})  // prepare grey for foreground
				blk := image.NewUniform(color.RGBA{0, 0, 0, 255})         // prepare black for foreground
				draw.Draw(img, img.Bounds(), bg, image.Point{}, draw.Src) // fill the background
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
			}
		}
	}, win)
	fd.SetFileName("DASHER.png")
	fd.Resize(fyne.Size{Width: 600, Height: 600})
	fd.SetDismissText("Dump Screen")
	fd.Show()
}
