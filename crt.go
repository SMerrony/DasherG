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
	"image"
	"image/draw"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

func buildCrt2() (crtImage *canvas.Raster, backingImage *image.NRGBA) {
	backingImage = image.NewNRGBA(image.Rect(0, 0, terminal.visibleCols*charWidth, terminal.visibleLines*charHeight))
	crtImage = canvas.NewRasterFromImage(backingImage) // canvas.NewImageFromImage(backingImage)
	crtImage.SetMinSize(fyne.Size{float32(terminal.visibleCols * charWidth), float32(terminal.visibleLines * charHeight)})
	//crtImage.FillMode = canvas.ImageFillOriginal
	return crtImage, backingImage
}

func drawCrt() {
	terminal.rwMutex.Lock()
	if terminal.terminalUpdated {
		// fmt.Println("DEBUG: drawCrt2 running update...")
		var cIx int

		for line := 0; line < terminal.visibleLines; line++ {
			for col := 0; col < terminal.visibleCols; col++ {
				if terminal.displayDirty[line][col] || (terminal.blinkEnabled && terminal.display[line][col].blink) {
					cIx = int(terminal.display[line][col].charValue)
					if cIx > 31 && cIx < 128 {
						// fmt.Printf("DEBUG: drawCrt found updatable char <%c>\n", terminal.display[line][col].charValue)
						cellRect := image.Rect(col*charWidth, line*charHeight, col*charWidth+charWidth, line*charHeight+charHeight)
						switch {
						case terminal.blinkEnabled && terminal.blinkState && terminal.display[line][col].blink:
							draw.Draw(backingImg, cellRect, bdfFont[32].plainImg, image.ZP, draw.Src)
						case terminal.display[line][col].reverse:
							draw.Draw(backingImg, cellRect, bdfFont[cIx].revImg, image.ZP, draw.Src)
						case terminal.display[line][col].dim:
							draw.Draw(backingImg, cellRect, bdfFont[cIx].dimImg, image.ZP, draw.Src)
						default:
							draw.Draw(backingImg, cellRect, bdfFont[cIx].plainImg, image.ZP, draw.Src)
						}
					}
					// underscore?
					if terminal.display[line][col].underscore {
						for x := 0; x < charWidth; x++ {
							backingImg.Set(col*charWidth+x, ((line+1)*charHeight)-1, green)
						}
					}
					terminal.displayDirty[line][col] = false
				}
			} // end for col
		} // end for line
		// draw the cursor - if on-screen
		if terminal.cursorX < terminal.visibleCols && terminal.cursorY < terminal.visibleLines {
			cellRect := image.Rect(terminal.cursorX*charWidth, terminal.cursorY*charHeight, terminal.cursorX*charWidth+charWidth, terminal.cursorY*charHeight+charHeight)
			cIx := int(terminal.display[terminal.cursorY][terminal.cursorX].charValue)
			if cIx == 0 {
				cIx = 32
			}
			if terminal.display[terminal.cursorY][terminal.cursorX].reverse {
				draw.Draw(backingImg, cellRect, bdfFont[cIx].plainImg, image.ZP, draw.Src)
			} else {
				// fmt.Printf("Drawing cursor at %d,%d\n", terminal.cursorX*charWidth, terminal.cursorY*charHeight)
				draw.Draw(backingImg, cellRect, bdfFont[cIx].revImg, image.ZP, draw.Src)
			}
			terminal.displayDirty[terminal.cursorY][terminal.cursorX] = true // this ensures that the old cursor pos is redrawn on the next refresh
		}
		// shade any selected area
		if selectionRegion.isActive {
			startCharPosn := selectionRegion.startCol + selectionRegion.startRow*terminal.visibleCols
			endCharPosn := selectionRegion.endCol + selectionRegion.endRow*terminal.visibleCols
			if startCharPosn <= endCharPosn {
				// normal (forward) selection
				col := selectionRegion.startCol
				for row := selectionRegion.startRow; row <= selectionRegion.endRow; row++ {
					for col < terminal.visibleCols {
						// drawable.DrawLine(gc, col*charWidth, ((row+1)*charHeight)-1, (col+1)*charWidth-1, ((row+1)*charHeight)-1) // FIXME
						if row == selectionRegion.endRow && col == selectionRegion.endCol {
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
		w.Canvas().Refresh(crtImg)
	}
	terminal.rwMutex.Unlock()
}