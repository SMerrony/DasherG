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
//

package main

import "fmt"
import "sync"

const (
	defaultLines, defaultCols       = 24, 80
	maxVisibleLines, maxVisibleCols = 66, 135
	totalLines, totalCols           = 96, 208
)

// terminalT encapsulates most of the emulation bevahiour itself.
// The display[][] matrix represents the currently-displayed state of the
// terminal (which is actually displayed elsewhere)
type terminalT struct {
	rwMutex                                      sync.RWMutex
	visibleLines, visibleCols                    int
	cursorX, cursorY                             int
	rollEnabled, blinkEnabled, protectionEnabled bool
	display                                      [totalLines][totalCols]cell

	status        *Status
	updateCrtChan chan int

	inCommand, inExtendedCommand,
	readingWindowAddressX, readingWindowAddressY,
	blinking, dimmed, reversedVideo, underscored, protectd bool
	newXaddress, newYaddress                    int
	inTelnetCommand, gotTelnetDo, gotTelnetWill bool
	telnetCmd, doAction, willAction             byte
}

func (t *terminalT) setup(pStatus *Status, update chan int) {
	t.rwMutex.Lock()
	t.status = pStatus
	t.updateCrtChan = update
	t.visibleLines = t.status.visLines
	t.visibleCols = t.status.visCols
	t.cursorX = 0
	t.cursorY = 0
	t.rollEnabled = true
	t.blinkEnabled = true
	t.protectionEnabled = false
	t.inCommand = false
	t.inExtendedCommand = false
	t.inTelnetCommand = false
	t.gotTelnetDo = false
	t.gotTelnetWill = false
	t.readingWindowAddressX = false
	t.readingWindowAddressY = false
	t.blinking = false
	t.dimmed = false
	t.reversedVideo = false
	t.underscored = false
	t.protectd = false
	t.clearScreen()
	// t.display[0][0].charValue = '0'
	// t.display[1][1].charValue = '1'
	// t.display[2][2].charValue = '2'
	t.display[12][39].charValue = 'O'
	t.display[12][40].charValue = 'K'
	t.rwMutex.Unlock()
	t.updateCrtChan <- updateCrtNormal
	fmt.Printf("terminalT setup done\n")
}

func (t *terminalT) clearLine(line int) {
	// TODO handle history here
	for cc := 0; cc < t.visibleCols; cc++ {
		t.display[line][cc].clearToSpace()
	}
	t.inCommand = false
	t.readingWindowAddressX = false
	t.readingWindowAddressY = false
	t.blinking = false
	t.dimmed = false
	t.reversedVideo = false
	t.underscored = false
}

func (t *terminalT) clearScreen() {
	for row := 0; row < t.visibleLines; row++ {
		t.clearLine(row)
	}
}

func (t *terminalT) resize() {
	t.clearScreen()
	t.visibleLines = t.status.visLines
	t.visibleCols = t.status.visCols
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
		//QString line;
		//for (int c = 0; c < visible_cols; c++) line.append( display[0][c].charValue );
		//history->addLine( line );
		// move each char up a row
		for r := 1; r < totalLines; r++ {
			for c := 0; c < t.visibleCols; c++ {
				t.display[r-1][c].copy(&t.display[r][c])
			}
		}
		t.clearLine(t.visibleLines - 1)
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
	for hostData := range fromHostChan {
		for _, ch = range hostData {

			t.rwMutex.Lock()
			skipChar = false
			// check for Telnet command
			if t.status.connected == telnetConnected && ch == telnetCmdIAC {
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

			if t.status.connected == telnetConnected && t.inTelnetCommand {
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

			if t.status.connected == telnetConnected && t.gotTelnetDo {
				// whatever the host asks us to do we will refuse
				keyboardChan <- telnetCmdIAC
				keyboardChan <- telnetCmdWONT
				keyboardChan <- ch
				t.gotTelnetDo = false
				t.inTelnetCommand = false
				skipChar = true
			}

			if t.status.connected == telnetConnected && t.gotTelnetWill {
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
			if t.status.logging {
				t.status.logFile.Write([]byte{ch})
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
				default:
					fmt.Println("Warning: unrecognise Break-CMD code")
				}

				// D210 commands
				if status.emulation >= d210 && ch == 'F' {
					t.inExtendedCommand = true
					skipChar = true
				}

				if status.emulation >= d210 && t.inExtendedCommand {
					switch ch {
					case 'F':
						t.eraseUnprotectedToEndOfScreen()
						skipChar = true
						t.inExtendedCommand = false
					}
				}

				t.inCommand = false
				t.rwMutex.Unlock()
				continue
			}

			switch ch {
			case dasherNul:
				skipChar = true
			case dasherBell:
				// TODO - how to handle this?
				fmt.Println("ignored BELL")
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
					t.display[t.cursorY][col].clearToSpace()
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
				if status.emulation >= d210 {
					t.reversedVideo = false
					skipChar = true
				}
			case dasherRevVideoOn:
				if status.emulation >= d210 {
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
				t.rwMutex.Unlock()
				continue
			}

			// finally, put the char in the displayable char matrix
			if ch > 0 && int(ch) < len(bdfFont) && bdfFont[ch].loaded {
				t.display[t.cursorY][t.cursorX].set(ch, t.blinking, t.dimmed, t.reversedVideo, t.underscored, t.protectd)
			} else {
				t.display[t.cursorY][t.cursorX].set(127, t.blinking, t.dimmed, t.reversedVideo, t.underscored, t.protectd)
			}
			t.cursorX++
			t.rwMutex.Unlock()
			//t.updateCrtChan <- updateCrtNormal
		}
		t.updateCrtChan <- updateCrtNormal
	}

	// if !skipChar {
	// 	t.status.dirty = true

	// }
	// if t.status.dirty {
	// 	t.updateChan <- true
	// }
}

func (t *terminalT) sendModelID() {
	switch status.emulation {
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
