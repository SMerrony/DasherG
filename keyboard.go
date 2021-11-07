// Copyright ©2017-2021 Steve Merrony

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
	"fyne.io/fyne/v2"
)

var (
	keyDownEventChan          = make(chan *fyne.KeyEvent, keyBuffSize)
	keyUpEventChan            = make(chan *fyne.KeyEvent, keyBuffSize)
	ctrlPressed, shiftPressed bool
)

func keyEventHandler(kbdChan chan<- byte) {
	for {
		select {
		case keyPressEvent := <-keyDownEventChan:
			// fmt.Println("keyEventHandler got press event")
			switch keyPressEvent.Name {
			case "LeftControl", "RightControl":
				ctrlPressed = true
			case "LeftShift", "RightShift":
				shiftPressed = true
			case "CapsLock":
				shiftPressed = !shiftPressed
			}

		case keyReleaseEvent := <-keyUpEventChan:
			// fmt.Printf("keyEventHandler got release event for <%s> with code: %d\n", keyReleaseEvent.Name, keyReleaseEvent.Physical)
			switch keyReleaseEvent.Name {
			case "LeftControl", "RightControl":
				ctrlPressed = false
			case "LeftShift", "RightShift":
				shiftPressed = false

			case fyne.KeyReturn:
				kbdChan <- dasherNewLine

			case fyne.KeyEscape:
				kbdChan <- '\033'

			case fyne.KeyHome:
				kbdChan <- dasherHome

			case fyne.KeyDelete: // the DEL key must map to 127 which is the DASHER DEL code
				kbdChan <- modify(127)

			case fyne.KeyF1:
				kbdChan <- dasherCmd
				kbdChan <- modify(113)
			case fyne.KeyF2:
				kbdChan <- dasherCmd
				kbdChan <- modify(114)
			case fyne.KeyF3:
				kbdChan <- dasherCmd
				kbdChan <- modify(115)
			case fyne.KeyF4:
				kbdChan <- dasherCmd
				kbdChan <- modify(116)
			case fyne.KeyF5:
				kbdChan <- dasherCmd
				kbdChan <- modify(117)

			case fyne.KeyF6:
				kbdChan <- dasherCmd
				kbdChan <- modify(118)
			case fyne.KeyF7:
				kbdChan <- dasherCmd
				kbdChan <- modify(119)
			case fyne.KeyF8:
				kbdChan <- dasherCmd
				kbdChan <- modify(120)
			case fyne.KeyF9:
				kbdChan <- dasherCmd
				kbdChan <- modify(121)
			case fyne.KeyF10:
				kbdChan <- dasherCmd
				kbdChan <- modify(122)

			case fyne.KeyF11:
				kbdChan <- dasherCmd
				kbdChan <- modify(123)
			case fyne.KeyF12:
				kbdChan <- dasherCmd
				kbdChan <- modify(124)
			case "F13":
				kbdChan <- dasherCmd
				kbdChan <- modify(125)
			case "F14":
				kbdChan <- dasherCmd
				kbdChan <- modify(126)
			case "F15":
				kbdChan <- dasherCmd
				kbdChan <- modify(112)

				// Cursor keys
			case fyne.KeyDown:
				kbdChan <- dasherCursorDown
			case fyne.KeyLeft:
				kbdChan <- dasherCursorLeft
			case fyne.KeyRight:
				kbdChan <- dasherCursorRight
			case fyne.KeyUp:
				kbdChan <- dasherCursorUp

			case fyne.KeySpace:
				kbdChan <- ' '

			default:
				// TODO special case for #, remove when Fyne adds KeyName for KeyHash
				if keyReleaseEvent.Physical.ScanCode == 51 {
					if shiftPressed {
						kbdChan <- '~'
					} else {
						kbdChan <- '#'
					}
					continue
				}
				keyByte := byte(keyReleaseEvent.Name[0])
				switch {
				case keyByte >= 'A' && keyByte <= 'Z':
					if !shiftPressed {
						keyByte += 32
					}
				case keyByte >= '0' && keyByte <= '9':
					if shiftPressed {
						switch keyByte {
						case '0':
							keyByte = ')'
						case '1':
							keyByte = '!'
						case '2':
							keyByte = '"'
						case '3':
							keyByte = '#' // US-style keyboard...
						case '4':
							keyByte = '$'
						case '5':
							keyByte = '%'
						case '6':
							keyByte = '^'
						case '7':
							keyByte = '&'
						case '8':
							keyByte = '*'
						case '9':
							keyByte = '('
						}
					}
				case shiftPressed:
					switch keyByte {
					case '`', '\\':
						keyByte = '|'
					case '-':
						keyByte = '_'
					case '=':
						keyByte = '+'
					case '[':
						keyByte = '{'
					case ']':
						keyByte = '}'
					case ';':
						keyByte = ':'
					case '\'':
						keyByte = '@'
					case '#':
						keyByte = '~'
					case ',':
						keyByte = '<'
					case '.':
						keyByte = '>'
					case '/':
						keyByte = '?'
					}
				}
				if ctrlPressed {
					keyByte &= 31 //mask off lower 5 bits
					//fmt.Printf("Keystroke modified to <%d>\n", keyByte)
				}
				kbdChan <- keyByte
			}
		}
	}
}

func modify(k byte) byte {
	var modifier byte
	if shiftPressed {
		modifier -= 16
	}
	if ctrlPressed {
		modifier -= 64
	}
	return k + modifier
}
