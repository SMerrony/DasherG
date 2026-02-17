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
//

package main

type displayT struct {
	cells                     [totalLines][totalCols]cell
	visibleLines, visibleCols int
}

// func (d *displayT) clearCell(line, col int) {
// 	d.cells[line][col].clearToSpace()
// }

// func (d *displayT) clearCellIfUnprotected(line, col int) {
// 	d.cells[line][col].clearToSpaceIfUnprotected()
// }

// func (d *displayT) clearLine(line int) {
// 	for col := 0; col < totalCols; col++ {
// 		d.cells[line][col].clearToSpace()
// 	}
// }

// func (d *displayT) clearVisible() {
// 	for line := 0; line < d.visibleLines; line++ {
// 		d.clearLine(line)
// 	}
// }

// func (d *displayT) copyLine(from, to int) {
// 	for col := 0; col < totalCols; col++ {
// 		d.cells[to][col].copyFrom(d.cells[from][col])
// 	}
// }

// func (d *displayT) copyTo(dest *displayT) {
// 	dest.visibleLines = d.visibleLines
// 	dest.visibleCols = d.visibleCols
// 	for line := 0; line < d.visibleLines; line++ {
// 		for col := 0; col < totalCols; col++ {
// 			dest.cells[line][col].copyFrom(d.cells[line][col])
// 		}
// 	}
// }

// func (d *displayT) debugPrint() {
// 	for line := 0; line < d.visibleLines; line++ {
// 		fmt.Printf("\nLine: %d: ", line)
// 		for col := 0; col < totalCols; col++ {
// 			fmt.Printf("%c", d.cells[line][col].charValue)
// 		}
// 	}
// }
