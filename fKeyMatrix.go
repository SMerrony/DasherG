// fKeyMatrix.go

// Copyright ©2017,2019,2020 Steve Merrony

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
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/gtk"
)

// widgets needing global access
var (
	fKeyLabs     [20][4]*gtk.Label
	templateLabs [2]*gtk.Label
)

func buildFkeyMatrix() *gtk.Table {
	fkeyMatrix := gtk.NewTable(5, 19, false)

	locPrBut := gtk.NewButtonWithLabel("LocPr")
	locPrBut.SetTooltipText("Local Print")
	locPrBut.Connect("clicked", localPrint)
	locPrBut.SetCanFocus(false)
	fkeyMatrix.AttachDefaults(locPrBut, 0, 1, 0, 1)

	breakBut := gtk.NewButtonWithLabel("Break")
	breakBut.SetTooltipText("Send BREAK signal on Serial Connection")
	breakBut.Connect("clicked", func() {
		if terminal.connectionType == serialConnected {
			serialSession.sendSerialBreakChan <- true
		}
	})
	breakBut.SetCanFocus(false)
	fkeyMatrix.AttachDefaults(breakBut, 0, 1, 4, 5)

	holdBut := gtk.NewButtonWithLabel("Hold")
	holdBut.Connect("clicked", func() {
		terminal.rwMutex.Lock()
		terminal.holding = !terminal.holding
		terminal.rwMutex.Unlock()
	})
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

	templateLabs[0] = gtk.NewLabel("")
	fkeyMatrix.AttachDefaults(templateLabs[0], 6, 7, 4, 5)

	csfLab := gtk.NewLabel("")
	csfLab.SetMarkup("<span size=\"small\">Ctrl-Shift</span>")
	fkeyMatrix.AttachDefaults(csfLab, 6, 7, 0, 1)
	cfLab := gtk.NewLabel("")
	cfLab.SetMarkup("<span size=\"small\">Ctrl</span>")
	fkeyMatrix.AttachDefaults(cfLab, 6, 7, 1, 2)
	sLab := gtk.NewLabel("")
	sLab.SetMarkup("<span size=\"small\">Shift</span>")
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

	templateLabs[1] = gtk.NewLabel("")
	fkeyMatrix.AttachDefaults(templateLabs[1], 12, 13, 4, 5)

	csfLab2 := gtk.NewLabel("")
	csfLab2.SetMarkup("<span size=\"small\">Ctrl-Shift</span>")
	fkeyMatrix.AttachDefaults(csfLab2, 12, 13, 0, 1)
	cfLab2 := gtk.NewLabel("")
	cfLab2.SetMarkup("<span size=\"small\">Ctrl</span>")
	fkeyMatrix.AttachDefaults(cfLab2, 12, 13, 1, 2)
	sLab2 := gtk.NewLabel("")
	sLab2.SetMarkup("<span size=\"small\">Shift</span>")
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

var fnButs [16]*widget.Button
var csFLabs, cFLabs, sFLabs, fLabs [16]*widget.Label
var templLabs [2]*widget.Label

func fnButton(number int) *widget.Button {
	str := strconv.Itoa(number)
	fnButs[number] = widget.NewButton("F"+str, nil)
	return fnButs[number]
}

func buildFkeyMatrix2() *fyne.Container {

	fkeyGrid := container.NewGridWithColumns(19)
	locPrBut := widget.NewButton("LocPr", nil)
	holdBut := widget.NewButton("Hold", nil)
	erPgBut := widget.NewButton("Er Pg", nil)
	crBut := widget.NewButton("CR", nil)
	erEOLBut := widget.NewButton("ErEOL", nil)
	breakBut := widget.NewButton("Break", nil)

	// top row
	fkeyGrid.Add(locPrBut)
	for k := 1; k <= 5; k++ {
		csFLabs[k] = widget.NewLabel("")
		csFLabs[k].Wrapping = fyne.TextWrapWord
		fkeyGrid.Add(csFLabs[k])
	}
	fkeyGrid.Add(widget.NewLabel("CtlSh"))
	for k := 6; k <= 10; k++ {
		csFLabs[k] = widget.NewLabel("")
		csFLabs[k].Wrapping = fyne.TextWrapWord
		fkeyGrid.Add(csFLabs[k])
	}
	fkeyGrid.Add(widget.NewLabel("CtlSh"))
	for k := 11; k <= 15; k++ {
		csFLabs[k] = widget.NewLabel("")
		csFLabs[k].Wrapping = fyne.TextWrapWord
		fkeyGrid.Add(csFLabs[k])
	}
	fkeyGrid.Add(holdBut)

	// 2nd row
	fkeyGrid.Add(widget.NewLabel(""))
	for k := 1; k <= 5; k++ {
		cFLabs[k] = widget.NewLabel("")
		cFLabs[k].Wrapping = fyne.TextWrapWord
		fkeyGrid.Add(cFLabs[k])
	}
	fkeyGrid.Add(widget.NewLabel("Ctrl"))
	for k := 6; k <= 10; k++ {
		cFLabs[k] = widget.NewLabel("")
		cFLabs[k].Wrapping = fyne.TextWrapWord
		fkeyGrid.Add(cFLabs[k])
	}
	fkeyGrid.Add(widget.NewLabel("Ctrl"))
	for k := 11; k <= 15; k++ {
		cFLabs[k] = widget.NewLabel("")
		cFLabs[k].Wrapping = fyne.TextWrapWord
		fkeyGrid.Add(cFLabs[k])
	}
	fkeyGrid.Add(erPgBut)

	// 3rd row
	fkeyGrid.Add(widget.NewLabel(""))
	for k := 1; k <= 5; k++ {
		sFLabs[k] = widget.NewLabel("")
		sFLabs[k].Wrapping = fyne.TextWrapWord
		fkeyGrid.Add(sFLabs[k])
	}
	fkeyGrid.Add(widget.NewLabel("Shift"))
	for k := 6; k <= 10; k++ {
		sFLabs[k] = widget.NewLabel("")
		sFLabs[k].Wrapping = fyne.TextWrapWord
		fkeyGrid.Add(sFLabs[k])
	}
	fkeyGrid.Add(widget.NewLabel("Shift"))
	for k := 11; k <= 15; k++ {
		sFLabs[k] = widget.NewLabel("")
		sFLabs[k].Wrapping = fyne.TextWrapWord
		fkeyGrid.Add(sFLabs[k])
	}
	fkeyGrid.Add(crBut)

	// 4th row
	fkeyGrid.Add(widget.NewLabel(""))
	for k := 1; k <= 5; k++ {
		fLabs[k] = widget.NewLabel("")
		fLabs[k].Wrapping = fyne.TextWrapWord
		fkeyGrid.Add(fLabs[k])
	}
	fkeyGrid.Add(widget.NewLabel(""))
	for k := 6; k <= 10; k++ {
		fLabs[k] = widget.NewLabel("")
		fLabs[k].Wrapping = fyne.TextWrapWord
		fkeyGrid.Add(fLabs[k])
	}
	fkeyGrid.Add(widget.NewLabel(""))
	for k := 11; k <= 15; k++ {
		fLabs[k] = widget.NewLabel("")
		fLabs[k].Wrapping = fyne.TextWrapWord
		fkeyGrid.Add(fLabs[k])
	}
	fkeyGrid.Add(erEOLBut)

	// 5th (bottom) row
	fkeyGrid.Add(breakBut)
	for k := 1; k <= 5; k++ {
		fkeyGrid.Add(fnButton(k))
	}
	templLabs[0] = widget.NewLabel("")
	fkeyGrid.Add(templLabs[0])
	for k := 6; k <= 10; k++ {
		fkeyGrid.Add(fnButton(k))
	}
	templLabs[1] = widget.NewLabel("")
	fkeyGrid.Add(templLabs[1])
	for k := 11; k <= 15; k++ {
		fkeyGrid.Add(fnButton(k))
	}
	return fkeyGrid
}

func loadFKeyTemplate() {
	fkd := gtk.NewFileChooserDialog("DasherG Function Key Template", win, gtk.FILE_CHOOSER_ACTION_OPEN, "_Cancel", gtk.RESPONSE_CANCEL, "_Load", gtk.RESPONSE_ACCEPT)
	res := fkd.Run()
	if res == gtk.RESPONSE_ACCEPT {
		fileName := fkd.GetFilename()
		file, err := os.Open(fileName)
		if err != nil {
			ed := gtk.NewMessageDialog(win, gtk.DIALOG_DESTROY_WITH_PARENT, gtk.MESSAGE_ERROR,
				gtk.BUTTONS_CLOSE, "Could not open or read function key template file")
			ed.Run()
			ed.Destroy()
		} else {
			// clear the labels
			for k := 1; k <= 15; k++ {
				for r := 0; r < 4; r++ {
					fKeyLabs[k][r].SetText("")
					fKeyLabs[k][r].SetSingleLineMode(false)
					fKeyLabs[k][r].SetJustify(gtk.JUSTIFY_CENTER)
				}
			}
			// read all the labels in order from the template file
			lineScanner := bufio.NewScanner(file)
			lineScanner.Scan()
			tlab := lineScanner.Text()
			templateLabs[0].SetMarkup("<span weight=\"bold\" size=\"small\">" + tlab + "</span>")
			templateLabs[1].SetMarkup("<span weight=\"bold\" size=\"small\">" + tlab + "</span>")
			for k := 1; k <= 15; k++ {
				for r := 3; r >= 0; r-- {
					lineScanner.Scan()
					lab := lineScanner.Text()
					if lab != "" {
						flab := strings.Replace(lab, "\\", "\n", -1)
						fKeyLabs[k][r].SetMarkup("<span size=\"small\">" + flab + "</span>")
					}
				}
			}
			file.Close()
		}
	}
	fkd.Destroy()
}
