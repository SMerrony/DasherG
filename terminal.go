// Copyright (C) 2017,2019  Steve Merrony

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
	historyLines                    = 2000
	holdPauseMs                     = 500
)

type emulType int

const (
	disconnected    = 0
	serialConnected = 1
	telnetConnected = 2

	d200 emulType = 200
	d210 emulType = 210
	d211 emulType = 211
)

// terminalT encapsulates most of the emulation bevahiour itself.
// The display[][] matrix represents the currently-displayed state of the
// terminal (which is actually displayed elsewhere)
type terminalT struct {
	rwMutex                                                          sync.RWMutex
	fromHostChan                                                     <-chan []byte
	expectChan                                                       chan<- byte
	rawChan                                                          chan byte
	emulation                                                        emulType
	connectionType                                                   int
	serialPort, serialBaud, serialBits, serialParity, serialStopBits string
	remoteHost, remotePort                                           string
	visibleLines, visibleCols                                        int
	cursorX, cursorY                                                 int
	rollEnabled, blinkEnabled, protectionEnabled                     bool
	blinkState                                                       bool
	holding, logging, scrolledBack                                   bool
	scrollBackTopLine                                                int
	expecting                                                        bool
	rawMode                                                          bool // in rawMode all host data is passed straight through to rawChan
	logFile                                                          *os.File

	// display is the 2D array of cells containing the terminal 'contents'
	display [totalLines][totalCols]cell
	history []string

	updateCrtChan chan int
	// terminalUpdated indicates that a visual refresh is required
	terminalUpdated bool

	// some empty structures for efficiency
	emptyCell cell
	emptyLine [totalCols]cell

	inCommand, inExtendedCommand,
	readingWindowAddressX, readingWindowAddressY,
	blinking, dimmed, reversedVideo, underscored, protectd bool
	newXaddress, newYaddress                    int
	inTelnetCommand, gotTelnetDo, gotTelnetWill bool
	telnetCmd, doAction, willAction             byte
}

func (t *terminalT) setup(fromHostChan <-chan []byte, update chan int, expectChan chan<- byte) {
	t.rwMutex.Lock()
	t.fromHostChan = fromHostChan
	t.expectChan = expectChan
	t.rawChan = make(chan byte)
	t.emulation = d210
	t.updateCrtChan = update
	t.visibleLines = defaultLines
	t.visibleCols = defaultCols
	t.emptyCell.clearToSpace()
	for c := range t.emptyLine {
		t.emptyLine[c] = t.emptyCell
	}
	t.rollEnabled = true
	t.blinkEnabled = true
	t.history = make([]string, historyLines)
	t.clearScreen()
	t.display[12][39].charValue = 'O'
	t.display[12][40].charValue = 'K'
	t.rwMutex.Unlock()
	t.updateCrtChan <- updateCrtNormal
}

func (t *terminalT) setRawMode(raw bool) {
	t.rwMutex.Lock()
	t.rawMode = raw
	t.rwMutex.Unlock()
}

func (t *terminalT) clearLine(line int) {
	t.display[line] = t.emptyLine
	t.inCommand = false
	t.readingWindowAddressX = false
	t.readingWindowAddressY = false
	t.blinking = false
	t.dimmed = false
	t.reversedVideo = false
	t.underscored = false
}

func (t *terminalT) clearScreen() {
	var line string
	for row := 0; row < t.visibleLines; row++ {
		// store in history
		line = ""
		for c := 0; c < t.visibleCols; c++ {
			line += string(t.display[row][c].charValue)
		}
		if len(t.history) >= historyLines {
			t.history = t.history[1:]
		}
		t.history = append(t.history, line)
		// clear the line
		t.display[row] = t.emptyLine
	}
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
	for x := t.cursorX; x < t.visibleCols; x++ {
		t.display[t.cursorY][x].clearToSpaceIfUnprotected()
	}
	// clear all lines below
	for y := t.cursorY + 1; y < t.visibleLines; y++ {
		for x := 0; x < t.visibleCols; x++ {
			t.display[y][x].clearToSpaceIfUnprotected()
		}
	}
}

func (t *terminalT) scrollUp(rows int) {
	for times := 0; times < rows; times++ {
		// store top line in history
		var line string
		for c := 0; c < t.visibleCols; c++ {
			line += string(t.display[0][c].charValue)
		}
		if len(t.history) >= historyLines {
			t.history = t.history[1:]
		}
		t.history = append(t.history, line)

		// move each char up a row
		for r := 1; r < t.visibleLines; r++ {
			for c := 0; c < t.visibleCols; c++ {
				t.display[r-1][c].copy(&t.display[r][c])
			}
		}
		t.clearLine(t.visibleLines - 1)
	}
}

func (t *terminalT) scrollDown(topLine string) {
	// move every visible row down
	for r := t.visibleLines; r > 0; r-- {
		for c := 0; c < t.visibleCols; c++ {
			t.display[r+1][c].copy(&t.display[r][c])
		}
	}
	t.clearLine(0)
	for c := 0; c < len(topLine); c++ {
		t.display[0][c].set(byte(topLine[c]), false, false, false, false, false)
	}
}

func (t *terminalT) scrollBack(rows int) {
	// determine the starting line in the history
	var startLine int
	if t.scrolledBack {
		startLine = t.scrollBackTopLine - rows
	} else {
		startLine = len(t.history) - rows
	}
	if startLine < 0 {
		startLine = 0
	}

}

func (t *terminalT) selfTest(hostChan chan []byte) {
	var (
		testLineHRule1 = "123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012245"
		testLineHRule2 = "         1         2         3         4         5         6         7         8         9         10        11        12        13    "
		testLine1      = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567489!\"$%^."
		testLineN      = "3 Normal : "
		testLineD      = "4 Dim    : "
		testLineB      = "5 Blink  : "
		testLineR      = "6 Reverse: "
		testLineU      = "7 Under  : "
		ba             []byte
	)
	ba = []byte{dasherErasePage}
	hostChan <- ba
	hostChan <- []byte(testLineHRule1[:t.visibleCols])
	hostChan <- []byte(testLineHRule2[:t.visibleCols])
	hostChan <- []byte(testLineN)
	hostChan <- []byte(testLine1)
	hostChan <- []byte("\n")

	hostChan <- []byte(testLineD)
	hostChan <- []byte{dasherDimOn}
	hostChan <- []byte(testLine1)
	hostChan <- []byte{dasherDimOff}
	hostChan <- []byte("\n")

	hostChan <- []byte(testLineB)
	hostChan <- []byte{dasherBlinkOn}
	hostChan <- []byte(testLine1)
	hostChan <- []byte{dasherBlinkOff}
	hostChan <- []byte("\n")

	hostChan <- []byte(testLineR)
	hostChan <- []byte{dasherCmd}
	hostChan <- []byte("D")
	hostChan <- []byte(testLine1)
	hostChan <- []byte{dasherCmd}
	hostChan <- []byte("E")
	hostChan <- []byte("\n")

	hostChan <- []byte(testLineU)
	hostChan <- []byte{dasherUnderline}
	hostChan <- []byte(testLine1)
	hostChan <- []byte{dasherNormal}

	for i := 8; i <= t.visibleLines; i++ {
		hostChan <- []byte(fmt.Sprintf("\n%d", i))
	}
}

func (t *terminalT) run() {
	var (
		skipChar bool
		ch       byte
	)
	for hostData := range t.fromHostChan {
		// pause if we are HOLDing
		t.rwMutex.RLock()
		for t.holding {
			t.rwMutex.RUnlock()
			time.Sleep(holdPauseMs * time.Millisecond)
			t.rwMutex.RLock()
		}
		t.rwMutex.RUnlock()

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
				if t.newXaddress >= t.visibleCols {
					t.newXaddress -= t.visibleCols
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
				if t.cursorY >= t.visibleLines {
					// see end of p.3-24 in D410 User Manual
					if t.rollEnabled {
						t.scrollUp(t.cursorY - (t.visibleLines - 1))
					}
					t.cursorY -= t.visibleLines
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
				if t.cursorY < t.visibleLines-1 {
					t.cursorY++
				} else {
					t.cursorY = 0
				}
				skipChar = true
			case dasherCursorLeft:
				if t.cursorX > 0 {
					t.cursorX--
				} else {
					t.cursorX = t.visibleCols - 1
					if t.cursorY > 0 {
						t.cursorY--
					} else {
						t.cursorY = t.visibleLines - 1
					}
				}
				skipChar = true
			case dasherCursorRight:
				if t.cursorX < t.visibleCols-1 {
					t.cursorX++
				} else {
					t.cursorX = 0
					if t.cursorY < t.visibleLines-2 {
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
					t.cursorY = t.visibleLines - 1
				}
				skipChar = true
			case dasherDimOn:
				t.dimmed = true
				skipChar = true
			case dasherDimOff:
				t.dimmed = false
				skipChar = true
			case dasherEraseEol:
				for col := t.cursorX; col < t.visibleCols; col++ {
					//t.display[t.cursorY][col].clearToSpace()
					t.display[t.cursorY][col] = t.emptyCell
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
			if t.cursorX == t.visibleCols || ch == dasherNewLine {
				if t.cursorY == t.visibleLines-1 { // hit bottom of screen
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
				t.display[t.cursorY][t.cursorX].set(ch, t.blinking, t.dimmed, t.reversedVideo, t.underscored, t.protectd)
			} else {
				t.display[t.cursorY][t.cursorX].set(127, t.blinking, t.dimmed, t.reversedVideo, t.underscored, t.protectd)
			}
			t.display[t.cursorY][t.cursorX].dirty = true
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
	case d211:
		keyboardChan <- 036  // Header 1
		keyboardChan <- 0157 // Header 2             (="o")
		keyboardChan <- 043  // model report follows (="#")
		keyboardChan <- 050  // D210                 (="(")
		keyboardChan <- 0131 // 0b01010001 See p.3-9 of D210/D211 User Manual
		keyboardChan <- 0172 // firmware code

		//    case 410:  // This is from p.3-17 of the D410 User Manual
		//        keyboardChan <- 036  // Header 1
		//        keyboardChan <- 0157 // Header 2             (="o")
		//        keyboardChan <- 043  // model report follows (="#")
		//        keyboardChan <- 052  // D410                 (="*")
		//        keyboardChan <- 89   // Status - 0b01011001
		//        keyboardChan <- 89   // Keyboard - 0b01011001 (="Y")

	}
}
