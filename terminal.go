package main

import "fmt"

const (
	defaultLines, defaultCols       = 24, 80
	maxVisibleLines, maxVisibleCols = 66, 135
	totalLines, totalCols           = 96, 208
)

type Terminal struct {
	visibleLines, visibleCols                    int
	cursorX, cursorY                             int
	rollEnabled, blinkEnabled, protectionEnabled bool
	display                                      [totalLines][totalCols]Cell

	status     *Status
	updateChan chan bool

	inCommand, inExtendedCommand,
	readingWindowAddressX, readingWindowAddressY,
	blinking, dimmed, reversedVideo, underscored, protectd bool
	newXaddress, newYaddress                    int
	inTelnetCommand, gotTelnetDo, gotTelnetWill bool
	telnetCmd, doAction, willAction             byte
}

func (t *Terminal) setup(pStatus *Status, update chan bool) {
	t.status = pStatus
	t.updateChan = update
	t.visibleLines = defaultLines
	t.visibleCols = defaultCols
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
	t.display[1][1].charValue = '1'
	t.display[2][2].charValue = '2'
	t.display[12][39].charValue = 'O'
	t.display[12][40].charValue = 'K'
	t.updateChan <- true
	fmt.Printf("Terminal setup done\n")
}

func (t *Terminal) clearLine(line int) {
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

func (t *Terminal) clearScreen() {
	for row := 0; row < t.visibleLines; row++ {
		t.clearLine(row)
	}
}

func (t *Terminal) eraseUnprotectedToEndOfScreen() {
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

func (t *Terminal) scrollUp(rows int) {
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
		// clear the bottom row
		t.clearLine(totalLines - 1)
	}
}

func (t *Terminal) selfTest(hostChan chan []byte) {
	var (
		testLineHRule1 = "123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012"
		testLineHRule2 = "         1         2         3         4         5         6         7         8         9         10        11        12        13"
		testLine1      = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567489!\"$%^."
		testLineN      = "3 Normal : "
		testLineD      = "4 Dim    : "
		testLineB      = "5 Blink  : "
		testLineU      = "6 Under  : "
		testLineR      = "7 Reverse: "
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

	hostChan <- []byte(testLineU)
	hostChan <- []byte{dasherUnderline}
	hostChan <- []byte(testLine1)
	hostChan <- []byte{dasherNormal}
	hostChan <- []byte("\n")

	hostChan <- []byte(testLineR)
	hostChan <- []byte{dasherCmd}
	hostChan <- []byte("D")
	hostChan <- []byte(testLine1)
	hostChan <- []byte{dasherCmd}
	hostChan <- []byte("E")
	hostChan <- []byte("\n")
}

func (t *Terminal) processHostData(hostData []byte) {
	var (
		skipChar bool
		ch       byte
	)

	for _, ch = range hostData {
		skipChar = false

		// check for Telnet command
		if t.status.connection == telnetConnected && ch == telnetCmdIAC {
			if t.inTelnetCommand {
				// special case - the host really wants to send a 255 - let it through
				t.inTelnetCommand = false
			} else {
				t.inTelnetCommand = true
				skipChar = true
				continue
			}
		}

		if t.status.connection == telnetConnected && t.inTelnetCommand {
			switch ch {
			case telnetCmdDO:
				t.gotTelnetDo = true
				skipChar = true
			case telnetCmdWILL:
				t.gotTelnetWill = true
				skipChar = true
			case telnetCmdAO, telnetCmdAYT, telnetCmdBRK, telnetCmdDM, telnetCmdDONT, telnetCmdEC, telnetCmdEL, telnetCmdGA, telnetCmdIP, telnetCmdNOP, telnetCmdSB, telnetCmdSE:
				skipChar = true
			}
		}
		if skipChar {
			continue
		}

		if t.status.connection == telnetConnected && t.gotTelnetDo {
			// whatever the host asks us to do we will refuse
			// FIXME send the message
			t.gotTelnetDo = false
			t.inTelnetCommand = false
			skipChar = true
		}

		if t.status.connection == telnetConnected && t.gotTelnetWill {
			// whatever the host offers to do we will refuse
			// FIXME send the message
			t.gotTelnetWill = false
			t.inTelnetCommand = false
			skipChar = true
		}
		if skipChar {
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
			continue
		}

		// FIXME lots of code omitted

		switch ch {
		case dasherBlinkOn:
			t.blinking = true
			skipChar = true
		case dasherBlinkOff:
			t.blinking = false
			skipChar = true
		case dasherDimOn:
			t.dimmed = true
			skipChar = true
		case dasherDimOff:
			t.dimmed = false
			skipChar = true
		case dasherErasePage:
			t.clearScreen()
			t.cursorX = 0
			t.cursorY = 0
			skipChar = true
		case dasherUnderline:
			t.underscored = true
			skipChar = true
		case dasherNormal:
			t.underscored = false
			skipChar = true
		}

		if skipChar {
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
			continue
		}

		// finally, put the char in the displayable char matrix
		t.display[t.cursorY][t.cursorX].set(ch, t.blinking, t.dimmed, t.reversedVideo, t.underscored, t.protectd)
		t.cursorX++
	}

	if !skipChar {
		t.status.dirty = true

	}
	if t.status.dirty {
		t.updateChan <- true
	}
}
