// Copyright ©2017-2020  Steve Merrony

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
	"fmt"
	"os"
	"sync"
	"time"
)

const (
	defaultLines, defaultCols       = 24, 80
	maxVisibleLines, maxVisibleCols = 66, 135
	totalLines, totalCols           = 96, 208
	historyLines                    = 1000 // N.B. c.f. scrollback parameters
	holdPauseMs                     = 500
)

type emulType int

const (
	disconnected    = 0
	serialConnected = 1
	telnetConnected = 2

	d200 emulType = 200
	d210 emulType = 210
	// d211 emulType = 211
)

// terminalT encapsulates most of the emulation behaviour itself.
// The display[][] matrix represents the currently-displayed state of the
// terminal (which is actually displayed elsewhere)
type terminalT struct {
	rwMutex                            sync.RWMutex // deadlock.RWMutex //sync.RWMutex
	fromHostChan                       <-chan []byte
	expectChan                         chan<- byte
	rawChan                            chan byte
	emulation                          emulType
	connectionType                     int
	remoteHost, remotePort, serialPort string
	// visibleLines, visibleCols                    int
	cursorX, cursorY                             int
	rollEnabled, blinkEnabled, protectionEnabled bool
	blinkState                                   bool
	holding, logging, scrolledBack               bool
	expecting                                    bool
	rawMode                                      bool // in rawMode all host data is passed straight through to rawChan
	selectionRegion                              selectionRegionT
	logFile                                      *os.File

	// display is the 2D array of cells containing the terminal 'contents'
	display                  displayT
	displayDirty             [totalLines][totalCols]bool
	savedDisplay             displayT // used when scrolling back over history
	displayHistory           [historyLines][totalCols]cell
	historyStart, historyEnd int

	updateCrtChan chan int
	// terminalUpdated indicates that a visual refresh is required
	terminalUpdated bool

	// some pre-filled structures for efficiency
	emptyCell cell
	emptyLine [totalCols]cell
	dirtyLine [totalCols]bool

	inCommand, inExtendedCommand,
	readingWindowAddressX, readingWindowAddressY,
	blinking, dimmed, reversedVideo, underscored, protectd bool
	newXaddress, newYaddress                    int
	inTelnetCommand, gotTelnetDo, gotTelnetWill bool
	telnetCmd, doAction, willAction             byte
}

type selectionRegionT struct {
	isActive                           bool
	startRow, startCol, endRow, endCol int
}

func (t *terminalT) setup(fromHostChan <-chan []byte, update chan int, expectChan chan<- byte) {
	t.rwMutex.Lock()
	t.fromHostChan = fromHostChan
	t.expectChan = expectChan
	t.rawChan = make(chan byte)
	t.emulation = d210
	t.updateCrtChan = update
	t.display.visibleLines = defaultLines
	t.display.visibleCols = defaultCols
	t.savedDisplay.visibleLines = defaultLines
	t.savedDisplay.visibleCols = defaultCols
	t.emptyCell.clearToSpace()
	for c := range t.emptyLine {
		t.emptyLine[c] = t.emptyCell
	}
	for c := range t.dirtyLine {
		t.dirtyLine[c] = true
	}
	for l := range t.displayHistory {
		t.displayHistory[l] = t.emptyLine
	}
	t.rollEnabled = true
	t.blinkEnabled = true
	t.scrolledBack = false
	t.clearScreen()
	t.display.cells[12][39].charValue = 'O'
	t.display.cells[12][40].charValue = 'K'
	t.rwMutex.Unlock()
	t.updateCrtChan <- updateCrtNormal
}

// updateListener is to be run as a Goroutine, it listens for update notifications and marks
// the terminal as needing a redraw
func (t *terminalT) updateListener() {
	for {
		updateType := <-updateCrtChan
		t.rwMutex.Lock()
		if updateType == updateCrtBlink {
			t.blinkState = !t.blinkState
		}
		t.terminalUpdated = true
		t.rwMutex.Unlock()
	}
}

func (t *terminalT) getSelection() string {
	startCharPosn := t.selectionRegion.startCol + t.selectionRegion.startRow*terminal.display.visibleCols
	endCharPosn := t.selectionRegion.endCol + t.selectionRegion.endRow*terminal.display.visibleCols
	selection := ""
	if startCharPosn <= endCharPosn {
		// normal (forward) selection
		col := t.selectionRegion.startCol
		for row := t.selectionRegion.startRow; row <= t.selectionRegion.endRow; row++ {
			for col < terminal.display.visibleCols {
				selection += string(terminal.display.cells[row][col].charValue)
				terminal.displayDirty[row][col] = true
				if row == t.selectionRegion.endRow && col == t.selectionRegion.endCol {
					return selection
				}
				col++
			}
			selection += string(rune(dasherNewLine))
			col = 0
		}
	}
	return selection
}

func (t *terminalT) setEmulation(e emulType) {
	// fmt.Printf("DEBUG: setEmuiation() called, old: %v, new: %v", t.emulation, e)
	t.rwMutex.Lock()
	t.emulation = e
	t.rwMutex.Unlock()
}

func (t *terminalT) setRawMode(raw bool) {
	t.rwMutex.Lock()
	t.rawMode = raw
	t.rwMutex.Unlock()
}

func (t *terminalT) clearLine(line int) {
	//t.display.cells[line] = t.emptyLine
	t.display.clearLine(line)
	t.displayDirty[line] = t.dirtyLine
	t.inCommand = false
	t.readingWindowAddressX = false
	t.readingWindowAddressY = false
	t.blinking = false
	t.dimmed = false
	t.reversedVideo = false
	t.underscored = false
}

func (t *terminalT) clearScreen() {
	t.scrollUp(t.display.visibleLines + 1)
	t.inCommand = false
	t.readingWindowAddressX = false
	t.readingWindowAddressY = false
	t.blinking = false
	t.dimmed = false
	t.reversedVideo = false
	t.underscored = false
}

func (t *terminalT) resize() {
	t.clearScreen()
	t.cursorX = 0
	t.cursorY = 0
}

func (t *terminalT) eraseUnprotectedToEndOfScreen() {
	// clear remainder of line
	for x := t.cursorX; x < t.display.visibleCols; x++ {
		t.display.cells[t.cursorY][x].clearToSpaceIfUnprotected()
		t.displayDirty[t.cursorY][x] = true
	}
	// clear all lines below
	for y := t.cursorY + 1; y < t.display.visibleLines; y++ {
		for x := 0; x < t.display.visibleCols; x++ {
			t.display.cells[y][x].clearToSpaceIfUnprotected()
			t.displayDirty[y][x] = true
		}
	}
}

// addToHistory inserts a display line into the circular history buffer
// It is assumed that the terminal mutex has been locked by the caller
func (t *terminalT) addToHistory(screenLine [totalCols]cell) {
	t.historyEnd++
	// end of buffer?
	if t.historyEnd == historyLines {
		// wrap-around
		t.historyEnd = 0
	}
	// has the tail hit the head?
	if t.historyEnd == t.historyStart {
		t.historyStart++ // advance the head
		if t.historyStart == historyLines {
			// wrap-around if at limit
			t.historyStart = 0
		}
	}
	t.displayHistory[t.historyEnd] = screenLine
	// fmt.Printf("addToHistory added real index: %d [start: %d]\n", t.historyEnd, t.historyStart)
}

// getNthHistoryLine returns a line of cells from the history
func (t *terminalT) getNthHistoryLine(n int) (screenLine [totalCols]cell) {
	var hline int
	if t.historyStart == t.historyEnd { // there's no history yet
		screenLine = t.emptyLine
	} else {
		hline = t.historyEnd - n
		if hline < 0 {
			hline += historyLines
		}
		screenLine = t.displayHistory[hline]
	}
	// fmt.Printf("getNthHistoryLine called with %d, returning history ix: %d\n", n, hline)
	return screenLine
}

func (t *terminalT) scrollUp(rows int) {
	for times := 0; times < rows; times++ {
		t.addToHistory(t.display.cells[0])
		// move each line up a row
		for r := 1; r < t.display.visibleLines; r++ {
			// t.display.cells[r-1] = t.display.cells[r]
			t.display.copyLine(r, r-1)
			t.displayDirty[r-1] = t.dirtyLine
		}
		t.clearLine(t.display.visibleLines - 1)
	}
}

func (t *terminalT) scrollDown(topLine string) {
	// move every visible row down
	for r := t.display.visibleLines; r > 0; r-- {
		// t.display.cells[r+1] = t.display.cells[r]
		t.display.copyLine(r, r+1)
		t.displayDirty[r+1] = t.dirtyLine
	}
	t.clearLine(0)
	for c := 0; c < len(topLine); c++ {
		t.display.cells[0][c].set(byte(topLine[c]), false, false, false, false, false)
		t.displayDirty[0][c] = true
	}
}

func (t *terminalT) scrollBack(startLine int) {
	t.rwMutex.Lock()
	// fmt.Printf("scrollBack - startLine: %d, scrolledBack: %v\n", startLine, t.scrolledBack)
	if !t.scrolledBack { // new scrollback
		// save live screen
		t.display.copyTo(&t.savedDisplay)
		// t.savedDisplay.debugPrint()
		t.scrolledBack = true
	}

	// testing...
	// t.display.clearVisible()
	// for l := 0; l < t.display.visibleLines; l++ {
	// 	t.displayDirty[l] = t.dirtyLine
	// }

	// there are two cases: we are already scrolled back beyond the 'live' screen, or we are partially showing it
	if startLine < t.display.visibleLines {
		// the partial case
		onScreenLine := 0
		for hl := startLine; hl >= 0; hl-- {
			t.display.cells[onScreenLine] = t.getNthHistoryLine(hl)
			t.displayDirty[onScreenLine] = t.dirtyLine
			onScreenLine++
		}
		liveLine := 0
		for onScreenLine < t.display.visibleLines {
			t.display.cells[onScreenLine] = t.savedDisplay.cells[liveLine]
			t.displayDirty[onScreenLine] = t.dirtyLine
			liveLine++
			onScreenLine++
		}
	} else {
		// all 'history' - we can cheat
		for l := 0; l < t.display.visibleLines; l++ {
			histLine := startLine - l
			t.display.cells[l] = t.getNthHistoryLine(histLine)
			t.displayDirty[l] = t.dirtyLine
		}
	}
	t.terminalUpdated = true
	t.rwMutex.Unlock()
}

// cancelScrollBack restores the 'live' screen after scrollbacks (may) have happened
func (t *terminalT) cancelScrollBack() {
	// fmt.Println("cancelScrollBack called")
	t.rwMutex.Lock()
	t.savedDisplay.copyTo(&t.display)
	for l := 0; l < t.display.visibleLines; l++ {
		t.displayDirty[l] = t.dirtyLine
	}
	t.scrolledBack = false
	t.terminalUpdated = true
	t.rwMutex.Unlock()
}

func (t *terminalT) selfTest(hostChan chan []byte) {
	const (
		testLineHRule1 = "123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012245"
		testLineHRule2 = "         1         2         3         4         5         6         7         8         9         10        11        12        13    "
		testLineChars  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!\"$%^.&"
		testLineN      = "3 Normal : "
		testLineD      = "4 Dim    : "
		testLineB      = "5 Blink  : "
		testLineR      = "6 Reverse: "
		testLineU      = "7 Under  : "
	)

	hostChan <- []byte{dasherErasePage}
	hostChan <- []byte(testLineHRule1[:t.display.visibleCols])
	hostChan <- []byte(testLineHRule2[:t.display.visibleCols])
	hostChan <- []byte(testLineN)
	hostChan <- []byte(testLineChars)
	hostChan <- []byte("\n")

	hostChan <- []byte(testLineD)
	hostChan <- []byte{dasherDimOn}
	hostChan <- []byte(testLineChars)
	hostChan <- []byte{dasherDimOff}
	hostChan <- []byte("\n")

	hostChan <- []byte(testLineB)
	hostChan <- []byte{dasherBlinkOn}
	hostChan <- []byte(testLineChars)
	hostChan <- []byte{dasherBlinkOff}
	hostChan <- []byte("\n")

	hostChan <- []byte(testLineR)
	hostChan <- []byte{dasherCmd}
	hostChan <- []byte("D")
	hostChan <- []byte(testLineChars)
	hostChan <- []byte{dasherCmd}
	hostChan <- []byte("E")
	hostChan <- []byte("\n")

	hostChan <- []byte(testLineU)
	hostChan <- []byte{dasherUnderline}
	hostChan <- []byte(testLineChars)
	hostChan <- []byte{dasherNormal}

	for i := 8; i <= t.display.visibleLines; i++ {
		hostChan <- []byte(fmt.Sprintf("\n%d", i))
	}
}

func (t *terminalT) run() {
	var (
		skipChar, sb bool
		ch           byte
	)
	for hostData := range t.fromHostChan {
		// pause if we are HOLDing
		t.rwMutex.RLock()
		for t.holding {
			t.rwMutex.RUnlock()
			time.Sleep(holdPauseMs * time.Millisecond)
			t.rwMutex.RLock()
		}
		sb = t.scrolledBack
		t.rwMutex.RUnlock()

		if sb {
			t.cancelScrollBack()
		}

		for _, ch = range hostData {

			t.rwMutex.Lock()

			if t.rawMode {
				t.rawChan <- ch
				t.rwMutex.Unlock()
				continue
			}

			skipChar = false
			// check for Telnet command
			if t.connectionType == telnetConnected && ch == telnetCmdIAC {
				if t.inTelnetCommand {
					// special case - the host really wants to send a 255 - let it through
					t.inTelnetCommand = false
				} else {
					t.inTelnetCommand = true
					skipChar = true
					t.rwMutex.Unlock()
					continue
				}
			}

			if t.connectionType == telnetConnected && t.inTelnetCommand {
				switch ch {
				case telnetCmdDO:
					t.gotTelnetDo = true
					skipChar = true
				case telnetCmdWILL:
					t.gotTelnetWill = true
					skipChar = true
				case telnetCmdAO, telnetCmdAYT, telnetCmdBRK, telnetCmdDM, telnetCmdDONT, telnetCmdEC,
					telnetCmdEL, telnetCmdGA, telnetCmdIP, telnetCmdNOP, telnetCmdSB, telnetCmdSE:
					skipChar = true
				}
			}
			if skipChar {
				t.rwMutex.Unlock()
				continue
			}

			if t.connectionType == telnetConnected && t.gotTelnetDo {
				// whatever the host asks us to do we will refuse
				keyboardChan <- telnetCmdIAC
				keyboardChan <- telnetCmdWONT
				keyboardChan <- ch
				t.gotTelnetDo = false
				t.inTelnetCommand = false
				skipChar = true
			}

			if t.connectionType == telnetConnected && t.gotTelnetWill {
				// whatever the host offers to do we will refuse
				keyboardChan <- telnetCmdIAC
				keyboardChan <- telnetCmdDONT
				keyboardChan <- ch
				t.gotTelnetWill = false
				t.inTelnetCommand = false
				skipChar = true
			}
			if skipChar {
				t.rwMutex.Unlock()
				continue
			}

			if t.readingWindowAddressX {
				t.newXaddress = int(ch & 0x7f)
				if t.newXaddress >= t.display.visibleCols {
					t.newXaddress -= t.display.visibleCols
				}
				if t.newXaddress == 127 {
					// special case - x stays the same - see D410 User Manual p.3-25
					t.newXaddress = t.cursorX
				}
				t.readingWindowAddressX = false
				t.readingWindowAddressY = true
				skipChar = true
				t.rwMutex.Unlock()
				continue
			}

			if t.readingWindowAddressY {
				t.newYaddress = int(ch & 0x7f)
				t.cursorX = t.newXaddress
				t.cursorY = t.newYaddress
				if t.newYaddress == 127 {
					// special case - y stays the same - see D410 User Manual p.3-25
					t.newYaddress = t.cursorY
				}
				if t.cursorY >= t.display.visibleLines {
					// see end of p.3-24 in D410 User Manual
					if t.rollEnabled {
						t.scrollUp(t.cursorY - (t.display.visibleLines - 1))
					}
					t.cursorY -= t.display.visibleLines
				}
				t.readingWindowAddressY = false
				skipChar = true
				t.rwMutex.Unlock()
				continue
			}

			// logging?
			if t.logging {
				t.logFile.Write([]byte{ch})
			}

			// Short commands
			if t.inCommand {
				switch ch {
				case 'C': // requires response
					t.sendModelID()
					skipChar = true
				case 'D':
					t.reversedVideo = true
					skipChar = true
				case 'E':
					t.reversedVideo = false
					skipChar = true
				case 'F':
					if t.emulation >= d210 {
						t.inExtendedCommand = true
						skipChar = true
					}
				default:
					fmt.Printf("WARNING: unrecognised Break-CMD code - char <%c>\n", ch)
				}

				t.inCommand = false
				t.rwMutex.Unlock()
				continue
			}

			// D210 commands
			if t.inExtendedCommand {
				switch ch {
				case 'F':
					t.eraseUnprotectedToEndOfScreen()
					// fmt.Println("Erase Unprot to end of screen")
					skipChar = true
					t.inExtendedCommand = false
				default:
					fmt.Printf("WARNING: unrecognised extended Break-CMD F code - char <%c>\n", ch)
				}
			}
			if skipChar {
				t.rwMutex.Unlock()
				continue
			}

			switch ch {
			case dasherNul:
				skipChar = true
			case dasherBell:
				fmt.Println("\x07BELL!")
				skipChar = true
			case dasherBlinkOn:
				t.blinking = true
				skipChar = true
			case dasherBlinkOff:
				t.blinking = false
				skipChar = true
			case dasherBlinkDisable:
				t.blinkEnabled = false
				skipChar = true
			case dasherBlinkEnable:
				t.blinkEnabled = true
				skipChar = true
			case dasherCursorDown:
				if t.cursorY < t.display.visibleLines-1 {
					t.cursorY++
				} else {
					t.cursorY = 0
				}
				skipChar = true
			case dasherCursorLeft:
				if t.cursorX > 0 {
					t.cursorX--
				} else {
					t.cursorX = t.display.visibleCols - 1
					if t.cursorY > 0 {
						t.cursorY--
					} else {
						t.cursorY = t.display.visibleLines - 1
					}
				}
				skipChar = true
			case dasherCursorRight:
				if t.cursorX < t.display.visibleCols-1 {
					t.cursorX++
				} else {
					t.cursorX = 0
					if t.cursorY < t.display.visibleLines-2 {
						t.cursorY++
					} else {
						t.cursorY = 0
					}
				}
				skipChar = true
			case dasherCursorUp:
				if t.cursorY > 0 {
					t.cursorY--
				} else {
					t.cursorY = t.display.visibleLines - 1
				}
				skipChar = true
			case dasherDimOn:
				t.dimmed = true
				skipChar = true
			case dasherDimOff:
				t.dimmed = false
				skipChar = true
			case dasherEraseEol:
				for col := t.cursorX; col < t.display.visibleCols; col++ {
					//t.display.cells[t.cursorY][col].clearToSpace()
					t.display.cells[t.cursorY][col] = t.emptyCell
					t.displayDirty[t.cursorY][col] = true
				}
				skipChar = true
			case dasherErasePage:
				t.clearScreen()
				t.cursorX = 0
				t.cursorY = 0
				skipChar = true
			case dasherHome:
				t.cursorX = 0
				t.cursorY = 0
				skipChar = true
			case dasherReadWindowAdd: // REQUIRES RESPONSE - see D410 User Manual p.3-18
				keyboardChan <- 31
				keyboardChan <- byte(t.cursorX)
				keyboardChan <- byte(t.cursorY)
				skipChar = true
			case dasherRevVideoOff:
				if t.emulation >= d210 {
					t.reversedVideo = false
					skipChar = true
				}
			case dasherRevVideoOn:
				if t.emulation >= d210 {
					t.reversedVideo = true
					skipChar = true
				}
			case dasherRollDisable:
				t.rollEnabled = false
				skipChar = true
			case dasherRollEnable:
				t.rollEnabled = true
				skipChar = true
			case dasherUnderline:
				t.underscored = true
				skipChar = true
			case dasherNormal:
				t.underscored = false
				skipChar = true
			case dasherTab:
				t.cursorX++
				for (t.cursorX+1)%8 != 0 {
					if t.cursorX >= t.display.visibleCols-1 {
						t.cursorX = 0
					} else {
						t.cursorX++
					}
				}
				skipChar = true
			case dasherWriteWindowAddr:
				t.readingWindowAddressX = true
				skipChar = true
			case dasherCmd:
				t.inCommand = true
				skipChar = true
			}

			if skipChar {
				t.rwMutex.Unlock()
				continue
			}

			// wrap due to hitting margin or new line?
			if t.cursorX == t.display.visibleCols || ch == dasherNewLine {
				if t.cursorY == t.display.visibleLines-1 { // hit bottom of screen
					if t.rollEnabled {
						t.scrollUp(1)
					} else {
						t.cursorY = 0
						t.clearLine(t.cursorY)
					}
				} else {
					t.cursorY++
					if !t.rollEnabled {
						t.clearLine(t.cursorY)
					}
				}
				t.cursorX = 0
			}

			// CR?
			if ch == dasherCR || ch == dasherNewLine {
				t.cursorX = 0
				if t.expecting {
					t.expectChan <- ch
				}
				t.rwMutex.Unlock()
				continue
			}

			// finally, put the char in the displayable char matrix
			if ch > 0 && int(ch) < len(bdfFont) && bdfFont[ch].loaded {
				t.display.cells[t.cursorY][t.cursorX].set(ch, t.blinking, t.dimmed, t.reversedVideo, t.underscored, t.protectd)
			} else {
				t.display.cells[t.cursorY][t.cursorX].set(127, t.blinking, t.dimmed, t.reversedVideo, t.underscored, t.protectd)
			}
			t.displayDirty[t.cursorY][t.cursorX] = true
			t.cursorX++
			if t.expecting {
				t.expectChan <- ch
			}
			t.rwMutex.Unlock()
		}
		t.updateCrtChan <- updateCrtNormal
	}
}

func (t *terminalT) sendModelID() {
	switch t.emulation {
	case d200:
		keyboardChan <- 036  // Header 1
		keyboardChan <- 0157 // Header 2             (="o")
		keyboardChan <- 043  // model report follows (="#")
		keyboardChan <- 041  // D100/D200            (="!")
		keyboardChan <- 0132 // 0b01011010 see p.2-7 of D100/D200 User Manual (="Z")
		keyboardChan <- 003  // firmware code
	case d210:
		keyboardChan <- 036  // Header 1
		keyboardChan <- 0157 // Header 2             (="o")
		keyboardChan <- 043  // model report follows (="#")
		keyboardChan <- 050  // D210                 (="(")
		keyboardChan <- 0121 // 0b01010001 See p.3-9 of D210/D211 User Manual
		keyboardChan <- 0132 // firmware code
		// case d211:
		// 	keyboardChan <- 036  // Header 1
		// 	keyboardChan <- 0157 // Header 2             (="o")
		// 	keyboardChan <- 043  // model report follows (="#")
		// 	keyboardChan <- 050  // D210                 (="(")
		// 	keyboardChan <- 0131 // 0b01010001 See p.3-9 of D210/D211 User Manual
		// 	keyboardChan <- 0172 // firmware code

		//    case 410:  // This is from p.3-17 of the D410 User Manual
		//        keyboardChan <- 036  // Header 1
		//        keyboardChan <- 0157 // Header 2             (="o")
		//        keyboardChan <- 043  // model report follows (="#")
		//        keyboardChan <- 052  // D410                 (="*")
		//        keyboardChan <- 89   // Status - 0b01011001
		//        keyboardChan <- 89   // Keyboard - 0b01011001 (="Y")

	}
}
