// specialKeys.go

// Copyright ©2025 Steve Merrony

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
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func buildSpecialKeyRow(win fyne.Window) *fyne.Container {
	breakBut := widget.NewButton("Break", func() {
		if terminal.connectionType == serialConnected {
			serialSession.sendSerialBreakChan <- true
		}
	})
	c1But := widget.NewButton("C1", func() { keyboardChan <- dasherC1 })
	c2But := widget.NewButton("C2", func() { keyboardChan <- dasherC2 })
	c3But := widget.NewButton("C3", func() { keyboardChan <- dasherC3 })
	c4But := widget.NewButton("C4", func() { keyboardChan <- dasherC4 })
	crBut := widget.NewButton("CR", func() { keyboardChan <- dasherCR })
	erPgBut := widget.NewButton("Er Pg", func() { keyboardChan <- dasherErasePage })
	erEOLBut := widget.NewButton("ErEOL", func() { keyboardChan <- dasherEraseEol })
	locPrBut := widget.NewButton("LocPr", func() { localPrint(win) })
	holdBut := widget.NewButton("Hold", func() {
		terminal.rwMutex.Lock()
		terminal.holding = !terminal.holding
		terminal.rwMutex.Unlock()
	})
	grid := container.New(layout.NewGridLayout(17),
		breakBut,
		layout.NewSpacer(), layout.NewSpacer(), layout.NewSpacer(), layout.NewSpacer(), layout.NewSpacer(), layout.NewSpacer(), layout.NewSpacer(),
		c1But, c2But, c3But, c4But,
		locPrBut,
		erPgBut,
		erEOLBut,
		crBut,
		holdBut)

	return grid
}
