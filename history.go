// Copyright (C) 2021,2025 Stephen Merrony
//
// This file is part of DasherG.
//
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
//

package main

import (
	"sync"
)

type historyT struct {
	mutex     sync.RWMutex
	cells     [logLines][totalCols]cell
	firstLine int
	lastLine  int
	// some pre-filled structures for efficiency
	emptyCell cell
	emptyLine [totalCols]cell
}

func createHistory() (h *historyT) {
	h = new(historyT)
	h.emptyCell.clearToSpace()
	for c := range h.emptyLine {
		h.emptyLine[c] = h.emptyCell
	}
	h.firstLine = 0
	h.lastLine = 0
	return h
}

func (h *historyT) append(screenLine [totalCols]cell) {
	h.mutex.Lock()
	h.lastLine++
	if h.lastLine == logLines {
		h.lastLine = 0 // wrap-around if at end
	}
	// has the tail hit the head of the circular buffer?
	if h.lastLine == h.firstLine {
		h.firstLine++ // advance the head pointer
		if h.firstLine == logLines {
			h.firstLine = 0 // but reset if at capacity
		}
	}
	for i, srcCell := range screenLine {
		h.cells[h.lastLine][i].copyFrom(srcCell)
	}
	h.mutex.Unlock()
}

// func (h *historyT) getNthLine(n int) (screenLine [totalCols]cell) {
// 	h.mutex.RLock()
// 	if h.firstLine == h.lastLine { // there's no history yet
// 		screenLine = h.emptyLine
// 	} else {
// 		bufferLineIx := h.lastLine - n
// 		if bufferLineIx < 0 {
// 			bufferLineIx += logLines
// 		}
// 		screenLine = h.cells[bufferLineIx]
// 	}
// 	h.mutex.RUnlock()
// 	return screenLine
// }

// Get all of the history as a plain string with no special formatting.
func (h *historyT) getAllAsPlainString() (text string) {
	h.mutex.RLock()
	text = "   *** Start of History ***\n"
	hLine := h.firstLine
	if h.firstLine > h.lastLine {
		for hLine < logLines {
			for cIx := range terminal.display.visibleCols {
				text += string(h.cells[hLine][cIx].charValue)
			}
			text += "\n"
			hLine++
		}
		hLine = 0
	}
	for hLine <= h.lastLine {
		for cIx := range terminal.display.visibleCols {
			text += string(h.cells[hLine][cIx].charValue)
		}
		text += "\n"
		hLine++
	}

	text += "\n   *** End of History ***"
	h.mutex.RUnlock()
	return text
}
