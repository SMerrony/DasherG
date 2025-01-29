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
	"fmt"
	"image"
	"image/draw"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

var backingImg *image.NRGBA

type crtMouseable struct {
	widget.BaseWidget
	obj *canvas.Raster
}

func newCrtMouseable(img *image.NRGBA) *crtMouseable {
	crt := &crtMouseable{obj: canvas.NewRasterFromImage(img)}
	crt.ExtendBaseWidget(crt)
	return crt
}

func (c *crtMouseable) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.obj)
}

func (c *crtMouseable) MouseDown(me *desktop.MouseEvent) {
	// fmt.Printf("DEBUG: Mouse down at x: %f, y: %f\n", me.Position.X, me.Position.Y)
	selectionRegion.startCol = int(me.Position.X) / charWidth
	selectionRegion.startRow = int(me.Position.Y) / charHeight
	selectionRegion.isActive = false
	// fmt.Printf("DEBUG: Selection start - col: %d, row: %d\n", selectionRegion.startCol, selectionRegion.startRow)
}

func (c *crtMouseable) MouseUp(me *desktop.MouseEvent) {
	// fmt.Printf("DEBUG: Mouse up at x: %f, y: %f\n", me.Position.X, me.Position.Y)
	selectionRegion.endCol = int(me.Position.X) / charWidth
	selectionRegion.endRow = int(me.Position.Y) / charHeight
	sel := getSelection()
	fmt.Printf("DEBUG: Copied selection: <%s>\n", sel)
	w.Clipboard().SetContent(sel)
	selectionRegion.isActive = true
	// fmt.Printf("DEBUG: Selection end - col: %d, row: %d\n", selectionRegion.endCol, selectionRegion.endRow)
}

func buildCrt() (crtImage *crtMouseable) {
	terminal.rwMutex.RLock()
	backingImg = image.NewNRGBA(image.Rect(0, 0, terminal.display.visibleCols*charWidth, terminal.display.visibleLines*charHeight+1))
	draw.Draw(backingImg, backingImg.Bounds(), image.Black, image.ZP, draw.Src)
	crtImage = newCrtMouseable(backingImg) // canvas.NewImageFromImage(backingImage)
	crtImage.obj.SetMinSize(fyne.Size{Width: float32(terminal.display.visibleCols * charWidth),
		Height: float32(terminal.display.visibleLines*charHeight + 1)})
	// crtImage.FillMode = canvas.ImageFillOriginal
	terminal.rwMutex.RUnlock()
	return crtImage
}

func drawCrt() {
	terminal.rwMutex.Lock()
	if terminal.terminalUpdated {
		// fmt.Println("DEBUG: drawCrt2 running update...")
		var cIx int

		for line := 0; line < terminal.display.visibleLines; line++ {
			for col := 0; col < terminal.display.visibleCols; col++ {
				if terminal.displayDirty[line][col] || (terminal.blinkEnabled && terminal.display.cells[line][col].blink) {
					cIx = int(terminal.display.cells[line][col].charValue)
					if cIx > 31 && cIx < 128 {
						// fmt.Printf("DEBUG: drawCrt found updatable char <%c>\n", terminal.display.cells[line][col].charValue)
						cellRect := image.Rect(col*charWidth, line*charHeight+1, col*charWidth+charWidth, line*charHeight+charHeight)
						switch {
						case terminal.blinkEnabled && terminal.blinkState && terminal.display.cells[line][col].blink:
							draw.Draw(backingImg, cellRect, bdfFont[32].plainImg, image.Point{}, draw.Src)
						case terminal.display.cells[line][col].reverse:
							draw.Draw(backingImg, cellRect, bdfFont[cIx].revImg, image.Point{}, draw.Src)
						case terminal.display.cells[line][col].dim:
							draw.Draw(backingImg, cellRect, bdfFont[cIx].dimImg, image.Point{}, draw.Src)
						default:
							draw.Draw(backingImg, cellRect, bdfFont[cIx].plainImg, image.Point{}, draw.Src)
						}
					}
					// underscore?
					if terminal.display.cells[line][col].underscore {
						for x := 0; x < charWidth; x++ {
							backingImg.Set(col*charWidth+x, ((line+1)*charHeight)-1, green)
						}
					}
					terminal.displayDirty[line][col] = false
				}
			} // end for col
		} // end for line
		// draw the cursor - if on-screen
		if terminal.cursorX < terminal.display.visibleCols && terminal.cursorY < terminal.display.visibleLines {
			cellRect := image.Rect(terminal.cursorX*charWidth, terminal.cursorY*charHeight+1, terminal.cursorX*charWidth+charWidth, terminal.cursorY*charHeight+charHeight)
			cIx := int(terminal.display.cells[terminal.cursorY][terminal.cursorX].charValue)
			if cIx == 0 {
				cIx = 32
			}
			if terminal.display.cells[terminal.cursorY][terminal.cursorX].reverse {
				draw.Draw(backingImg, cellRect, bdfFont[cIx].plainImg, image.Point{}, draw.Src)
			} else {
				// fmt.Printf("Drawing cursor at %d,%d\n", terminal.cursorX*charWidth, terminal.cursorY*charHeight)
				draw.Draw(backingImg, cellRect, bdfFont[cIx].revImg, image.Point{}, draw.Src)
			}
			terminal.displayDirty[terminal.cursorY][terminal.cursorX] = true // this ensures that the old cursor pos is redrawn on the next refresh
		}
		// 	// shade any selected area
		// 	if selectionRegion.isActive {
		// 		startCharPosn := selectionRegion.startCol + selectionRegion.startRow*terminal.display.visibleCols
		// 		endCharPosn := selectionRegion.endCol + selectionRegion.endRow*terminal.display.visibleCols
		// 		if startCharPosn <= endCharPosn {
		// 			// normal (forward) selection
		// 			col := selectionRegion.startCol
		// 			for row := selectionRegion.startRow; row <= selectionRegion.endRow; row++ {
		// 				for col < terminal.display.visibleCols {
		// 					hLine(backingImg, col*charWidth, ((row+1)*charHeight)-1, (col+1)*charWidth-1)
		// 					//draw.DrawLine(gc, col*charWidth, ((row+1)*charHeight)-1, (col+1)*charWidth-1, ((row+1)*charHeight)-1)
		// 					if row == selectionRegion.endRow && col == selectionRegion.endCol {
		// 						goto shadingDone
		// 					}
		// 					col++
		// 				}
		// 				col = 0
		// 			}
		// 		}
		// 	}
		// shadingDone:
		terminal.terminalUpdated = false
		w.Canvas().Refresh(crtImg)
	}
	terminal.rwMutex.Unlock()
}

// func hLine(img *image.NRGBA, x1, y, x2 int) {
// 	for ; x1 <= x2; x1++ {
// 		img.Set(x1, y, color.White)
// 	}
// }
