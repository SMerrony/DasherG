// Copyright (C) 2017  Steve Merrony

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

type cell struct {
	charValue                                byte
	blink, dim, reverse, underscore, protect bool
	dirty                                    bool
}

func (cell *cell) set(cv byte, bl, dm, rev, under, prot bool) {
	cell.charValue = cv
	cell.blink = bl
	cell.dim = dm
	cell.reverse = rev
	cell.underscore = under
	cell.protect = prot
	cell.dirty = true
}

func (cell *cell) clearToSpace() {
	cell.charValue = ' '
	cell.blink = false
	cell.dim = false
	cell.reverse = false
	cell.underscore = false
	cell.protect = false
	cell.dirty = true
}

func (cell *cell) clearToSpaceIfUnprotected() {
	if !cell.protect {
		cell.clearToSpace()
	}
}

func (cell *cell) copy(fromcell *cell) {
	cell.charValue = fromcell.charValue
	cell.blink = fromcell.blink
	cell.dim = fromcell.dim
	cell.reverse = fromcell.reverse
	cell.underscore = fromcell.underscore
	cell.protect = fromcell.protect
	cell.dirty = true
}
