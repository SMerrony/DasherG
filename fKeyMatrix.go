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
	"os"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

var fnButs [15]*widget.Button
var csFLabs, cFLabs, sFLabs, fLabs [15]*widget.Label

func fnButton(number int) *widget.Button {
	str := strconv.Itoa(number + 1)
	fnButs[number] = widget.NewButton("F"+str, nil)
	return fnButs[number]
}

func smallLabel() *widget.Label {
	return widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{
		Bold:      false,
		Italic:    false,
		Monospace: false,
		TabWidth:  1,
	})
}

func smallLabelWithText(text string) *widget.Label {
	w := smallLabel()
	w.SetText(text)
	return w
}

func buildLabelGrid() *fyne.Container {

	// templLabs[0] = smallLabelWithText("")
	// templLabs[1] = smallLabelWithText("")

	grid := container.New(layout.NewGridLayout(17))
	// control-shift...
	for col := 0; col <= 4; col++ {
		csFLabs[col] = smallLabel()
		grid.Add(csFLabs[col])
	}
	grid.Add(smallLabelWithText("Ctrl\nShift"))
	for col := 6; col <= 10; col++ {
		csFLabs[col-1] = smallLabel()
		grid.Add(csFLabs[col-1])
	}
	grid.Add(smallLabelWithText("Ctrl\nShift"))
	for col := 12; col <= 16; col++ {
		csFLabs[col-2] = smallLabel()
		grid.Add(csFLabs[col-2])
	}
	// control...
	for col := 0; col <= 4; col++ {
		cFLabs[col] = smallLabel()
		grid.Add(cFLabs[col])
	}
	grid.Add(smallLabelWithText("Ctrl"))
	for col := 6; col <= 10; col++ {
		cFLabs[col-1] = smallLabel()
		grid.Add(cFLabs[col-1])
	}
	grid.Add(smallLabelWithText("Ctrl"))
	for col := 12; col <= 16; col++ {
		cFLabs[col-2] = smallLabel()
		grid.Add(cFLabs[col-2])
	}
	// shift...
	for col := 0; col <= 4; col++ {
		sFLabs[col] = smallLabel()
		grid.Add(sFLabs[col])
	}
	grid.Add(smallLabelWithText("Shift"))
	for col := 6; col <= 10; col++ {
		sFLabs[col-1] = smallLabel()
		grid.Add(sFLabs[col-1])
	}
	grid.Add(smallLabelWithText("Shift"))
	for col := 12; col <= 16; col++ {
		sFLabs[col-2] = smallLabel()
		grid.Add(sFLabs[col-2])
	}
	// plain...
	for col := 0; col <= 4; col++ {
		fLabs[col] = smallLabel()
		grid.Add(fLabs[col])
	}
	grid.Add(layout.NewSpacer())
	for col := 6; col <= 10; col++ {
		fLabs[col-1] = smallLabel()
		grid.Add(fLabs[col-1])
	}
	grid.Add(layout.NewSpacer())
	for col := 12; col <= 16; col++ {
		fLabs[col-2] = smallLabel()
		grid.Add(fLabs[col-2])
	}

	return grid
}

func buildFuncKeyRow() *fyne.Container {
	grid := container.New(layout.NewGridLayout(17))
	for col := 0; col <= 4; col++ {
		grid.Add(fnButton(col))
	}
	grid.Add(layout.NewSpacer())
	for col := 6; col <= 10; col++ {
		grid.Add(fnButton(col - 1))
	}
	grid.Add(layout.NewSpacer())
	for col := 12; col <= 16; col++ {
		grid.Add(fnButton(col - 2))
	}
	return grid
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
				for f := 0; f < 15; f++ {
					csFLabs[f].SetText("")
					cFLabs[f].SetText("")
					sFLabs[f].SetText("")
					fLabs[f].SetText("")
				}
				// read all the labels in order from the template file
				lineScanner := bufio.NewScanner(file)
				lineScanner.Scan()
				// tlab := lineScanner.Text()
				lineScanner.Text()
				// templLabs[0].Text(tlab)
				// templLabs[1].Text(tlab)
				for k := 0; k < 15; k++ {
					for r := 3; r >= 0; r-- {
						lineScanner.Scan()
						lab := lineScanner.Text()
						if lab != "" {
							flab := strings.Replace(lab, "\\", "\n", -1)
							// flab := strings.Replace(lab, "\\", " ", -1)
							switch r {
							case 0:
								csFLabs[k].SetText(flab)
							case 1:
								cFLabs[k].SetText(flab)
							case 2:
								sFLabs[k].SetText(flab)
							case 3:
								fLabs[k].SetText(flab)
							}
						}
					}
				}
			}
		}
	}, win)
	fd.Resize(fyne.Size{Width: 600, Height: 600})
	fd.Show()
}
