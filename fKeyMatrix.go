// fKeyMatrix.go

// Copyright ©2017-2021 Steve Merrony

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
	"image/color"
	"os"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

var fnButs [16]*widget.Button
var csFLabs, cFLabs, sFLabs, fLabs [16]*canvas.Text
var templLabs [2]*canvas.Text

func fnButton(number int) *widget.Button {
	str := strconv.Itoa(number)
	fnButs[number] = widget.NewButton("F"+str, nil)
	return fnButs[number]
}

func smallText(text string) *canvas.Text {
	s := text
	return &canvas.Text{
		Alignment: fyne.TextAlignCenter,
		Color:     color.White,
		Text:      s,
		TextSize:  9.0,
		TextStyle: fyne.TextStyle{
			Bold:      false,
			Italic:    false,
			Monospace: false,
			TabWidth:  0,
		},
	}
}

func buildFkeyMatrix(win fyne.Window) *fyne.Container {

	templLabs[0] = smallText("")
	templLabs[1] = smallText("")

	locPrBut := widget.NewButton("LocPr", func() { localPrint(win) })
	holdBut := widget.NewButton("Hold", func() {
		terminal.rwMutex.Lock()
		terminal.holding = !terminal.holding
		terminal.rwMutex.Unlock()
	})
	erPgBut := widget.NewButton("Er Pg", func() { keyboardChan <- dasherErasePage })
	crBut := widget.NewButton("CR", func() { keyboardChan <- dasherCR })
	erEOLBut := widget.NewButton("ErEOL", func() { keyboardChan <- dasherEraseEol })
	breakBut := widget.NewButton("Break", func() {
		if terminal.connectionType == serialConnected {
			serialSession.sendSerialBreakChan <- true
		}
	})

	hbox := container.NewHBox()
	var vboxes [19]*fyne.Container
	for col := 0; col < 19; col++ {
		vboxes[col] = container.NewVBox()
		hbox.Add(vboxes[col])
	}

	vboxes[0].Add(locPrBut)
	vboxes[0].Add(crBut)
	// vboxes[0].Add(smallText(""))
	// vboxes[0].Add(smallText(""))
	vboxes[0].Add(breakBut)

	for col := 1; col <= 5; col++ {
		csFLabs[col] = smallText("")
		vboxes[col].Add(csFLabs[col])
		cFLabs[col] = smallText("")
		vboxes[col].Add(cFLabs[col])
		sFLabs[col] = smallText("")
		vboxes[col].Add(sFLabs[col])
		fLabs[col] = smallText("")
		vboxes[col].Add(fLabs[col])
		vboxes[col].Add(fnButton(col))
	}

	vboxes[6].Add(smallText("Ctrl-Sh"))
	vboxes[6].Add(smallText("Ctrl"))
	vboxes[6].Add(smallText("Shift"))
	vboxes[6].Add(smallText(""))
	vboxes[6].Add(templLabs[0])

	for col := 7; col <= 11; col++ {
		csFLabs[col-1] = smallText("")
		vboxes[col].Add(csFLabs[col-1])
		cFLabs[col-1] = smallText("")
		vboxes[col].Add(cFLabs[col-1])
		sFLabs[col-1] = smallText("")
		vboxes[col].Add(sFLabs[col-1])
		fLabs[col-1] = smallText("")
		vboxes[col].Add(fLabs[col-1])
		vboxes[col].Add(fnButton(col - 1))
	}

	vboxes[12].Add(smallText("Ctrl-Sh"))
	vboxes[12].Add(smallText("Ctrl"))
	vboxes[12].Add(smallText("Shift"))
	vboxes[12].Add(smallText(""))
	vboxes[12].Add(templLabs[1])

	for col := 13; col <= 17; col++ {
		csFLabs[col-2] = smallText("")
		vboxes[col].Add(csFLabs[col-2])
		cFLabs[col-2] = smallText("")
		vboxes[col].Add(cFLabs[col-2])
		sFLabs[col-2] = smallText("")
		vboxes[col].Add(sFLabs[col-2])
		fLabs[col-2] = smallText("")
		vboxes[col].Add(fLabs[col-2])
		vboxes[col].Add(fnButton(col - 2))
	}

	vboxes[18].Add(holdBut)
	vboxes[18].Add(erPgBut)
	// vboxes[18].Add(crBut)
	vboxes[18].Add(erEOLBut)
	// vboxes[18].Add(smallText(""))

	return hbox
}

func loadFKeyTemplate(win fyne.Window) {
	fd := dialog.NewFileOpen(func(urirc fyne.URIReadCloser, e error) {
		if urirc != nil {
			file, err := os.Open(urirc.URI().Path())
			if err != nil {
				dialog.ShowError(err, win)
			} else {
				defer file.Close()
				// clear the labels
				for f := 1; f <= 15; f++ {
					csFLabs[f].Text = ""
					cFLabs[f].Text = ""
					sFLabs[f].Text = ""
					fLabs[f].Text = ""
				}
				// read all the labels in order from the template file
				lineScanner := bufio.NewScanner(file)
				lineScanner.Scan()
				tlab := lineScanner.Text()
				templLabs[0].Text = tlab
				templLabs[1].Text = tlab
				for k := 1; k <= 15; k++ {
					for r := 3; r >= 0; r-- {
						lineScanner.Scan()
						lab := lineScanner.Text()
						if lab != "" {
							// flab := strings.Replace(lab, "\\", "\n", -1)
							flab := strings.Replace(lab, "\\", " ", -1)
							switch r {
							case 0:
								csFLabs[k].Text = flab
							case 1:
								cFLabs[k].Text = flab
							case 2:
								sFLabs[k].Text = flab
							case 3:
								fLabs[k].Text = flab
							}
						}
					}
				}
			}
		}
	}, win)
	fd.Resize(fyne.Size{Width: 600, Height: 600})
	fd.SetDismissText("Load Template")
	fd.Show()
}
