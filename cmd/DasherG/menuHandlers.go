// menuHandlers.go - part of DasherG

// Copyright ©2019-2021,2025,2026 Steve Merrony

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
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func editPaste(win fyne.Window) {
	text := win.Clipboard().Content()
	if len(text) == 0 {
		dialog.ShowInformation("DasherG", "Nothing in Clipboard to Paste", win)
	} else {
		for _, ch := range text {
			keyboardChan <- byte(ch)
		}
	}
}

func emulationResize(win fyne.Window) {

	var selectedCols int
	colsRadio := widget.NewRadioGroup([]string{"80", "81", "120", "132", "135"},
		func(selected string) { selectedCols, _ = strconv.Atoi(selected) })

	var selectedLines int
	linesRadio := widget.NewRadioGroup([]string{"24", "25", "36", "48", "66"},
		func(selected string) { selectedLines, _ = strconv.Atoi(selected) })

	terminal.rwMutex.RLock()
	colsRadio.SetSelected(strconv.Itoa(terminal.display.visibleCols))
	linesRadio.SetSelected(strconv.Itoa(terminal.display.visibleLines))
	terminal.rwMutex.RUnlock()

	zoomRadio := widget.NewRadioGroup([]string{ZoomLarge, ZoomNormal, ZoomSmaller, ZoomTiny},
		func(selected string) { zoom = selected })
	zoomRadio.SetSelected(zoom)

	formItems := []*widget.FormItem{
		widget.NewFormItem("Columns", colsRadio),
		widget.NewFormItem("Lines", linesRadio),
		widget.NewFormItem("Zoom", zoomRadio),
	}

	dialog.ShowForm("Resize Terminal", "Resize", "Cancel", formItems,
		func(b bool) {
			if b {
				terminal.rwMutex.Lock()
				terminal.display.visibleCols = selectedCols
				terminal.display.visibleLines = selectedLines
				bdfLoad(fontData, zoom, green, dimGreen)
				terminal.rwMutex.Unlock()
				crtImg = buildCrt()
				terminal.resize()
				setContent(win)
			}
		}, win)

}

func fileChooseExpectScript(win fyne.Window) {
	ed := dialog.NewFileOpen(func(urirc fyne.URIReadCloser, e error) {
		if urirc != nil {
			expectFile, err := os.Open(urirc.URI().Path())
			if err != nil {
				dialog.ShowError(err, win)
				log.Printf("WARNING: Could not open mini-Expect file %s\n", urirc.URI().Path())
			} else {
				go expectRunner(expectFile, expectChan, keyboardChan, terminal)
			}
		}
	}, win)
	ed.Resize(fyne.Size{Width: 600, Height: 600})
	ed.SetConfirmText("Execute")
	ed.Show()
}

func fileLogging(win fyne.Window) {
	if terminal.logging {
		terminal.logFile.Close()
		terminal.logging = false
	} else {
		dialog.ShowFileSave(func(uriwc fyne.URIWriteCloser, e error) {
			if uriwc != nil {
				filename := uriwc.URI().Path()
				var err error
				terminal.logFile, err = os.Create(filename)
				if err != nil {
					dialog.ShowError(err, win)
					log.Printf("WARNING: Could not open log file %s\n", filename)
					terminal.logging = false
				} else {
					terminal.logging = true
				}
			}
		}, win)
	}
}

func fileSendText(win fyne.Window) {
	fsd := dialog.NewFileOpen(func(urirc fyne.URIReadCloser, e error) {
		if urirc != nil {
			bytes, err := os.ReadFile(urirc.URI().Path())
			if err != nil {
				dialog.ShowError(err, win)
				log.Printf("WARNING: Could not open or read text file %s\n", urirc.URI().Path())
			} else {
				for _, b := range bytes {
					keyboardChan <- b
				}
			}
		}
	}, win)
	fsd.Resize(fyne.Size{Width: 600, Height: 600})
	fsd.SetConfirmText("Execute")
	fsd.Show()
}

func fileXmodemReceive(win fyne.Window) {
	fsd := dialog.NewFileSave(func(urirc fyne.URIWriteCloser, e error) {
		if urirc != nil {
			f, err := os.Create(urirc.URI().Path())
			if err != nil {
				dialog.ShowError(err, win)
			} else {
				defer f.Close()
				terminal.setRawMode(true)
				blob, err := XModemReceive(terminal.rawChan, keyboardChan)
				if err != nil {
					dialog.ShowError(err, win)
				} else {
					f.Write(blob)
				}
				terminal.setRawMode(false)
			}
		}
	}, win)
	fsd.Resize(fyne.Size{Width: 600, Height: 600})
	fsd.SetConfirmText("Receive")
	fsd.Show()
}

func fileXmodemSend(win fyne.Window) {
	fsd := dialog.NewFileOpen(func(urirc fyne.URIReadCloser, e error) {
		if urirc != nil {
			f, err := os.Open(urirc.URI().Path())
			if err != nil {
				dialog.ShowError(err, win)
			} else {
				defer f.Close()
				terminal.setRawMode(true)
				err := XmodemSendShort(terminal.rawChan, keyboardChan, f)
				if err != nil {
					dialog.ShowError(err, win)
				}
				terminal.setRawMode(false)
			}
		}
	}, win)
	fsd.Resize(fyne.Size{Width: 600, Height: 600})
	fsd.SetConfirmText("Receive")
	fsd.Show()
}

func fileXmodemSend1k(win fyne.Window) {
	fsd := dialog.NewFileOpen(func(urirc fyne.URIReadCloser, e error) {
		if urirc != nil {
			f, err := os.Open(urirc.URI().Path())
			if err != nil {
				dialog.ShowError(err, win)
			} else {
				defer f.Close()
				terminal.setRawMode(true)
				err := XmodemSendLong(terminal.rawChan, keyboardChan, f)
				if err != nil {
					dialog.ShowError(err, win)
				}
				terminal.setRawMode(false)
			}
		}
	}, win)
	fsd.Resize(fyne.Size{Width: 600, Height: 600})
	fsd.SetConfirmText("Receive (1k)")
	fsd.Show()
}

func viewHistory() {
	app := fyne.CurrentApp()
	viewWin := app.NewWindow("History")
	textGrid := widget.NewTextGrid()
	textGrid.SetText(terminal.displayHistory.getAllAsPlainString())
	scroller := container.NewVScroll(textGrid)
	viewWin.SetContent(scroller)
	viewWin.Resize(fyne.NewSize(640, 480))
	viewWin.Show()
}

func helpAbout() {
	info := fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s\n\n%s", appTitle, appComment, appSemVer, appWebsite, appCopyright)
	dialog.ShowInformation("About", info, w)
}

func serialClose() {
	serialSession.closeSerialPort()
	serialConnectItem.Disabled = false
	serialDisconnectItem.Disabled = true
	networkConnectItem.Disabled = false
	networkDisconnectItem.Disabled = true
	go localListener(keyboardChan, fromHostChan)
}

func serialConnect(win fyne.Window) {
	portEntry := widget.NewEntry()
	var selectedBaud int
	baudSelect := widget.NewSelect([]string{"300", "1200", "2400", "9600", "19200"},
		func(selected string) {
			selectedBaud, _ = strconv.Atoi(selected)
		})
	baudSelect.SetSelected("9600")
	var selectedBits int
	bitsSelect := widget.NewSelect([]string{"7", "8"},
		func(selected string) {
			selectedBits, _ = strconv.Atoi(selected)
		})
	bitsSelect.SetSelected("8")
	var selectedParity string
	paritySelect := widget.NewSelect([]string{"None", "Odd", "Even"},
		func(selected string) {
			selectedParity = selected
		})
	paritySelect.SetSelected("None")
	var selectedStopBits int
	stopBitsSelect := widget.NewSelect([]string{"1", "2"},
		func(selected string) {
			selectedStopBits, _ = strconv.Atoi(selected)
		})
	stopBitsSelect.SetSelected("1")
	formItems := []*widget.FormItem{
		widget.NewFormItem("Port", portEntry),
		widget.NewFormItem("Baud", baudSelect),
		widget.NewFormItem("Data Bits", bitsSelect),
		widget.NewFormItem("Parity", paritySelect),
		widget.NewFormItem("Stop Bits", stopBitsSelect),
	}
	dialog.ShowForm("DasherG - Serial Port", "Connect", "Cancel", formItems,
		func(b bool) {
			if b {
				if serialSession.openSerialPort(portEntry.Text, selectedBaud, selectedBits, selectedParity, selectedStopBits) {
					localListenerStopChan <- true
					serialConnectItem.Disabled = true
					networkConnectItem.Disabled = true
					networkDisconnectItem.Disabled = true
					serialDisconnectItem.Disabled = false
				} else {
					err := errors.New("could not connect via serial port")
					dialog.ShowError(err, win)
				}
			}
		}, win)
}

func telnetClose() {
	if telnetClosing {
		return
	}
	telnetClosing = true
	telnetSession.closeTelnetConn()
	serialConnectItem.Disabled = false
	serialDisconnectItem.Disabled = true
	networkConnectItem.Disabled = false
	networkDisconnectItem.Disabled = true
	go localListener(keyboardChan, fromHostChan)
	telnetClosing = false
}

func telnetOpen(win fyne.Window) {
	hostEntry := widget.NewEntry()
	hostEntry.SetText(lastTelnetHost)
	portEntry := widget.NewEntry()
	if lastTelnetPort != 0 {
		portEntry.SetText(strconv.Itoa(lastTelnetPort))
	}
	formItems := []*widget.FormItem{
		widget.NewFormItem("Host", hostEntry),
		widget.NewFormItem("Port", portEntry),
	}
	dialog.ShowForm("DasherG - Telnet Host", "Connect", "Cancel", formItems, func(b bool) {
		if b {
			host := hostEntry.Text
			port, err := strconv.Atoi(portEntry.Text)
			if err != nil || port < 0 || len(host) == 0 {
				err = errors.New("must enter valid host and numeric port")
				dialog.ShowError(err, win)
				return
			}
			telnetSession = newTelnetSession()
			if telnetSession.openTelnetConn(host, port) {
				localListenerStopChan <- true
				serialConnectItem.Disabled = true
				networkConnectItem.Disabled = true
				networkDisconnectItem.Disabled = false
				serialDisconnectItem.Disabled = true
				lastTelnetHost = host
				lastTelnetPort = port
			} else {
				err = errors.New("could not connect to remote host")
				dialog.ShowError(err, win)
			}
		}
	}, win)
}
