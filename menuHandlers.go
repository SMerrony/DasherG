// menuHandlers.go - part of DasherG

// Copyright ©2019-2021 Steve Merrony

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
	"io/ioutil"
	"log"
	"os"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/gtk"
)

func editPaste() {
	clipboard := gtk.NewClipboardGetForDisplay(gdk.DisplayGetDefault(), gdk.SELECTION_CLIPBOARD)
	text := clipboard.WaitForText()
	if len(text) == 0 {
		ed := gtk.NewMessageDialog(win, gtk.DIALOG_DESTROY_WITH_PARENT, gtk.MESSAGE_ERROR,
			gtk.BUTTONS_CLOSE, "Nothing in Clipboard to Paste")
		ed.Run()
		ed.Destroy()
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

	rd := gtk.NewDialog()
	rd.SetTitle("Resize Terminal")

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
				bdfLoad(fontFile, zoom, green, dimGreen)
				terminal.rwMutex.Unlock()
				crtImg = buildCrt()
				terminal.resize()
				setContent()
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
	ed.Resize(fyne.Size{600, 600})
	ed.SetDismissText("Execute")
	ed.Show()
}

func fileLogging(win fyne.Window) {
	if terminal.logging {
		terminal.logFile.Close()
		terminal.logging = false
	} else {
		fd := dialog.NewFileSave(func(uriwc fyne.URIWriteCloser, e error) {
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
		fd.Resize(fyne.Size{600, 600})
		fd.SetDismissText("Start Logging")
		fd.Show()
	}
}

func fileSendText() {
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

func fileXmodemReceive() {
	fsd := gtk.NewFileChooserDialog("DasherG XMODEM Receive File", win, gtk.FILE_CHOOSER_ACTION_SAVE, "_Cancel", gtk.RESPONSE_CANCEL, "_Receive", gtk.RESPONSE_ACCEPT)
	res := fsd.Run()
	if res == gtk.RESPONSE_ACCEPT {
		fileName := fsd.GetFilename()
		fsd.Destroy()
		f, err := os.Create(fileName)
		defer f.Close()
		if err != nil {
			ed := gtk.NewMessageDialog(win, gtk.DIALOG_DESTROY_WITH_PARENT, gtk.MESSAGE_ERROR,
				gtk.BUTTONS_CLOSE, "DasherG - XMODEM Could not create file to receive")
			ed.Run()
			ed.Destroy()
		} else {
			terminal.setRawMode(true)
			blob, err := XModemReceive(terminal.rawChan, keyboardChan)
			if err != nil {
				ed := gtk.NewMessageDialog(win, gtk.DIALOG_DESTROY_WITH_PARENT, gtk.MESSAGE_ERROR,
					gtk.BUTTONS_CLOSE, "DasherG - "+err.Error())
				ed.Run()
				ed.Destroy()
			} else {
				f.Write(blob)
			}
			terminal.setRawMode(false)
		}
	} else {
		fsd.Destroy()
	}
}

func fileXmodemSend() {
	fsd := gtk.NewFileChooserDialog("DasherG XMODEM Send File", win, gtk.FILE_CHOOSER_ACTION_OPEN, "_Cancel", gtk.RESPONSE_CANCEL, "_Send", gtk.RESPONSE_ACCEPT)
	res := fsd.Run()
	if res == gtk.RESPONSE_ACCEPT {
		fileName := fsd.GetFilename()
		fsd.Destroy()
		f, err := os.Open(fileName)
		defer f.Close()
		if err != nil {
			ed := gtk.NewMessageDialog(win, gtk.DIALOG_DESTROY_WITH_PARENT, gtk.MESSAGE_ERROR,
				gtk.BUTTONS_CLOSE, "DasherG - XMODEM Could not open file to send - "+err.Error())
			ed.Run()
			ed.Destroy()
		} else {
			terminal.setRawMode(true)
			err := XmodemSendShort(terminal.rawChan, keyboardChan, f)
			if err != nil {
				ed := gtk.NewMessageDialog(win, gtk.DIALOG_DESTROY_WITH_PARENT, gtk.MESSAGE_ERROR,
					gtk.BUTTONS_CLOSE, "DasherG - XMODEM Could not send file - "+err.Error())
				ed.Run()
				ed.Destroy()
			}
			terminal.setRawMode(false)
		}
	} else {
		fsd.Destroy()
	}
}

func fileXmodemSend1k() {
	fsd := gtk.NewFileChooserDialog("DasherG XMODEM Send File", win, gtk.FILE_CHOOSER_ACTION_OPEN, "_Cancel", gtk.RESPONSE_CANCEL, "_Send", gtk.RESPONSE_ACCEPT)
	res := fsd.Run()
	if res == gtk.RESPONSE_ACCEPT {
		fileName := fsd.GetFilename()
		fsd.Destroy()
		f, err := os.Open(fileName)
		defer f.Close()
		if err != nil {
			ed := gtk.NewMessageDialog(win, gtk.DIALOG_DESTROY_WITH_PARENT, gtk.MESSAGE_ERROR,
				gtk.BUTTONS_CLOSE, "DasherG - XMODEM Could not open file to send - "+err.Error())
			ed.Run()
			ed.Destroy()
		} else {
			terminal.setRawMode(true)
			err := XmodemSendLong(terminal.rawChan, keyboardChan, f)
			if err != nil {
				ed := gtk.NewMessageDialog(win, gtk.DIALOG_DESTROY_WITH_PARENT, gtk.MESSAGE_ERROR,
					gtk.BUTTONS_CLOSE, "DasherG - XMODEM Could not send file - "+err.Error())
				ed.Run()
				ed.Destroy()
			}
			terminal.setRawMode(false)
		}
	} else {
		fsd.Destroy()
	}

}

func helpAbout() {
	info := fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s", appTitle, appSemVer, appWebsite, appCopyright)
	dialog.ShowInformation("About", info, w)
}

func serialClose() {
	serialSession.closeSerialPort()
	// glib.IdleAdd(func() {
	// 	serialDisconnectMenuItem.SetSensitive(false)
	// 	networkConnectMenuItem.SetSensitive(true)
	// 	serialConnectMenuItem.SetSensitive(true)
	// })
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
					// serialConnectMenuItem.SetSensitive(false)
					// networkConnectMenuItem.SetSensitive(false)
					// serialDisconnectMenuItem.SetSensitive(true)
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
	// glib.IdleAdd(func() {
	// 	networkDisconnectMenuItem.SetSensitive(false)
	// 	serialConnectMenuItem.SetSensitive(true)
	// 	networkConnectMenuItem.SetSensitive(true)
	// })
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
				// networkConnectMenuItem.SetSensitive(false)
				// serialConnectMenuItem.SetSensitive(false)
				// networkDisconnectMenuItem.SetSensitive(true)
				lastTelnetHost = host
				lastTelnetPort = port
			} else {
				err = errors.New("could not connect to remote host")
				dialog.ShowError(err, win)
			}
		}
	}, win)
}
