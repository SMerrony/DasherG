// menuHandlers.go - part of DasherG

// Copyright (C) 2019  Steve Merrony

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
	"io/ioutil"
	"log"
	"os"
	"strconv"

	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/glib"
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

func emulationResize() {
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

func fileChooseExpectScript() {
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

func fileLogging() {
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
	ad := gtk.NewAboutDialog()
	ad.SetProgramName(appTitle)
	ad.SetAuthors(appAuthors)
	ad.SetIcon(iconPixbuf)
	ad.SetLogo(iconPixbuf)
	ad.SetVersion(appSemVer)
	ad.SetCopyright(appCopyright)
	ad.SetWebsite(appWebsite)
	ad.Run()
	ad.Destroy()
}

func serialClose() {
	serialSession.closeSerialPort()
	glib.IdleAdd(func() {
		serialDisconnectMenuItem.SetSensitive(false)
		networkConnectMenuItem.SetSensitive(true)
		serialConnectMenuItem.SetSensitive(true)
	})
	go localListener(keyboardChan, fromHostChan)
}

func serialConnect() {
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
		if serialSession.openSerialPort(portEntry.GetText(), baud, bits, parityCombo.GetActiveText(), stopBits) {
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

func telnetClose() {
	closeTelnetConn()
	glib.IdleAdd(func() {
		networkDisconnectMenuItem.SetSensitive(false)
		serialConnectMenuItem.SetSensitive(true)
		networkConnectMenuItem.SetSensitive(true)
	})
	go localListener(keyboardChan, fromHostChan)
}

func telnetOpen() {
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

func viewHistory(t *terminalT) {
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
