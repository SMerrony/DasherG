// crt.go - part of DasherG

// Copyright ©2020  Steve Merrony

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
	"unsafe"

	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/glib"
	"github.com/mattn/go-gtk/gtk"
)

func buildCrt() *gtk.DrawingArea {
	var mne int
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
	})

	crt.Connect("expose-event", func() {
		gdkWin.GetDrawable().DrawDrawable(gc, offScreenPixmap.GetDrawable(), 0, 0, 0, 0, -1, -1)
		//fmt.Println("expose-event handled")
	})

	crt.SetCanFocus(true)
	crt.AddEvents(int(gdk.BUTTON_PRESS_MASK))
	crt.Connect("button-press-event", func(ctx *glib.CallbackContext) {
		arg := ctx.Args(0)
		btnPressEvent := *(**gdk.EventButton)(unsafe.Pointer(&arg))
		//fmt.Printf("DEBUG: Mouse clicked at %d, %d\t", btnPressEvent.X, btnPressEvent.Y)
		terminal.rwMutex.Lock()
		terminal.selectionRegion.startRow = int(btnPressEvent.Y) / charHeight
		terminal.selectionRegion.startCol = int(btnPressEvent.X) / charWidth
		terminal.selectionRegion.endRow = terminal.selectionRegion.startRow
		terminal.selectionRegion.endCol = terminal.selectionRegion.startCol
		terminal.selectionRegion.isActive = true
		terminal.rwMutex.Unlock()
		mne = crt.Connect("motion-notify-event", handleMotionNotifyEvent)
	})
	crt.AddEvents(int(gdk.BUTTON_RELEASE_MASK))
	crt.Connect("button-release-event", func(ctx *glib.CallbackContext) {
		arg := ctx.Args(0)
		btnPressEvent := *(**gdk.EventButton)(unsafe.Pointer(&arg))
		//fmt.Printf("DEBUG: Mouse released at %d, %d\t", btnPressEvent.X, btnPressEvent.Y)
		terminal.rwMutex.Lock()
		terminal.selectionRegion.endRow = int(btnPressEvent.Y) / charHeight
		terminal.selectionRegion.endCol = int(btnPressEvent.X) / charWidth
		sel := terminal.getSelection()
		terminal.selectionRegion.isActive = false
		terminal.rwMutex.Unlock()
		//fmt.Printf("DEBUG: Copied selection: <%s>\n", sel)
		clipboard := gtk.NewClipboardGetForDisplay(gdk.DisplayGetDefault(), gdk.SELECTION_CLIPBOARD)
		clipboard.SetText(sel)
		crt.HandlerDisconnect(mne)
	})
	crt.AddEvents(int(gdk.POINTER_MOTION_MASK))

	return crt
}

// handleMotionNotifyEvent is called every time the mouse moves after being clicked
// in the CRT.  It is no longer called once the mouse is released.
func handleMotionNotifyEvent(ctx *glib.CallbackContext) {
	arg := ctx.Args(0)
	btnPressEvent := *(**gdk.EventMotion)(unsafe.Pointer(&arg))
	row := int(btnPressEvent.Y) / charHeight
	col := int(btnPressEvent.X) / charWidth
	if row != terminal.selectionRegion.endRow || col != terminal.selectionRegion.endCol {
		// moved at least 1 cell...
		terminal.rwMutex.Lock()
		// fmt.Printf("DEBUG: Row: %d, Col: %d, Character: %c\n", row, col, terminal.display[row][col].charValue)
		terminal.selectionRegion.endCol = col
		terminal.selectionRegion.endRow = row
		terminal.rwMutex.Unlock()
	}
}

func drawCrt() {
	terminal.rwMutex.Lock()
	if terminal.terminalUpdated {
		var cIx int
		drawable := offScreenPixmap.GetDrawable()
		for line := 0; line < terminal.visibleLines; line++ {
			for col := 0; col < terminal.visibleCols; col++ {
				if terminal.displayDirty[line][col] || (terminal.blinkEnabled && terminal.display[line][col].blink) {
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
					terminal.displayDirty[line][col] = false
				}
			} // end for col
		} // end for line
		// draw the cursor - if on-screen
		if terminal.cursorX < terminal.visibleCols && terminal.cursorY < terminal.visibleLines {
			cIx := int(terminal.display[terminal.cursorY][terminal.cursorX].charValue)
			if cIx == 0 {
				cIx = 32
			}
			if terminal.display[terminal.cursorY][terminal.cursorX].reverse {
				drawable.DrawPixbuf(gc, bdfFont[cIx].pixbuf, 0, 0, terminal.cursorX*charWidth, terminal.cursorY*charHeight, charWidth, charHeight, 0, 0, 0)
			} else {
				//fmt.Printf("Drawing cursor at %d,%d\n", terminal.cursorX*charWidth, terminal.cursorY*charHeight)
				drawable.DrawPixbuf(gc, bdfFont[cIx].reversePixbuf, 0, 0, terminal.cursorX*charWidth, terminal.cursorY*charHeight, charWidth, charHeight, 0, 0, 0)
			}
			terminal.displayDirty[terminal.cursorY][terminal.cursorX] = true // this ensures that the old cursor pos is redrawn on the next refresh
		}
		// shade any selected area
		if terminal.selectionRegion.isActive {
			startCharPosn := terminal.selectionRegion.startCol + terminal.selectionRegion.startRow*terminal.visibleCols
			endCharPosn := terminal.selectionRegion.endCol + terminal.selectionRegion.endRow*terminal.visibleCols
			if startCharPosn <= endCharPosn {
				// normal (forward) selection
				col := terminal.selectionRegion.startCol
				for row := terminal.selectionRegion.startRow; row <= terminal.selectionRegion.endRow; row++ {
					for col < terminal.visibleCols {
						drawable.DrawLine(gc, col*charWidth, ((row+1)*charHeight)-1, (col+1)*charWidth-1, ((row+1)*charHeight)-1)
						if row == terminal.selectionRegion.endRow && col == terminal.selectionRegion.endCol {
							goto shadingDone
						}
						col++
					}
					col = 0
				}
			}
		}
	shadingDone:
		terminal.terminalUpdated = false
		gdkWin.Invalidate(nil, false)
	}
	terminal.rwMutex.Unlock()
}
